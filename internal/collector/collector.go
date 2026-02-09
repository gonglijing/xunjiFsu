package collector

import (
	"container/heap"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

const defaultCollectIntervalMilliseconds = 5000

const (
	defaultDeviceSyncInterval  = 10 * time.Second
	defaultCommandPollInterval = 500 * time.Millisecond
)

// Collector 采集器
type Collector struct {
	driverExecutor       *driver.DriverExecutor
	northboundMgr        *northbound.NorthboundManager
	resourceLock         map[int64]chan struct{} // 每个资源一个串行锁
	mu                   sync.RWMutex
	running              bool
	stopChan             chan struct{}
	wakeChan             chan struct{}
	deviceSyncResetChan  chan time.Duration
	commandPollResetChan chan time.Duration
	deviceSyncInterval   time.Duration
	commandPollInterval  time.Duration
	wg                   sync.WaitGroup
	// 设备采集任务
	tasks    map[int64]*collectTask
	taskHeap *taskHeap // 优先队列，按下次采集时间排序
	// 最新值缓存（用于查询）
	latestDataCache map[int64]*models.CollectData
}

// collectTask 采集任务
type collectTask struct {
	device          *models.Device
	interval        time.Duration
	storageInterval time.Duration
	nextRun         time.Time
	lastRun         time.Time
	lastStored      time.Time
}

type deviceSyncAction int

const (
	deviceSyncActionNone deviceSyncAction = iota
	deviceSyncActionAdded
	deviceSyncActionUpdated
	deviceSyncActionRemoved
)

// taskHeap 任务优先队列（小根堆）
type taskHeap []*collectTask

func (h taskHeap) Len() int { return len(h) }
func (h taskHeap) Less(i, j int) bool {
	return h[i].nextRun.Before(h[j].nextRun)
}
func (h taskHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *taskHeap) Push(x interface{}) {
	*h = append(*h, x.(*collectTask))
}
func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	task := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return task
}

// NewCollector 创建采集器
func NewCollector(driverExecutor *driver.DriverExecutor, northboundMgr *northbound.NorthboundManager) *Collector {
	return NewCollectorWithIntervals(driverExecutor, northboundMgr, defaultDeviceSyncInterval, defaultCommandPollInterval)
}

func NewCollectorWithIntervals(driverExecutor *driver.DriverExecutor, northboundMgr *northbound.NorthboundManager, deviceSyncInterval, commandPollInterval time.Duration) *Collector {
	if deviceSyncInterval <= 0 {
		deviceSyncInterval = defaultDeviceSyncInterval
	}
	if commandPollInterval <= 0 {
		commandPollInterval = defaultCommandPollInterval
	}

	h := &taskHeap{}
	heap.Init(h)
	return &Collector{
		driverExecutor:       driverExecutor,
		northboundMgr:        northboundMgr,
		running:              false,
		stopChan:             make(chan struct{}),
		wakeChan:             make(chan struct{}, 1),
		deviceSyncResetChan:  make(chan time.Duration, 1),
		commandPollResetChan: make(chan time.Duration, 1),
		deviceSyncInterval:   deviceSyncInterval,
		commandPollInterval:  commandPollInterval,
		tasks:                make(map[int64]*collectTask),
		taskHeap:             h,
		resourceLock:         make(map[int64]chan struct{}),
		latestDataCache:      make(map[int64]*models.CollectData),
	}
}

// Start 启动采集器
func (c *Collector) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("collector is already running")
	}
	c.running = true
	c.stopChan = make(chan struct{})
	c.wakeChan = make(chan struct{}, 1)
	c.deviceSyncResetChan = make(chan time.Duration, 1)
	c.commandPollResetChan = make(chan time.Duration, 1)
	c.tasks = make(map[int64]*collectTask)
	c.latestDataCache = make(map[int64]*models.CollectData)
	h := &taskHeap{}
	heap.Init(h)
	c.taskHeap = h
	c.mu.Unlock()

	// 开机时加载所有 enable 的设备
	if err := c.loadEnabledDevices(); err != nil {
		log.Printf("Failed to load enabled devices: %v", err)
	}

	// 同步设备状态
	c.startAdjustableTickerWorker(c.deviceSyncInterval, c.deviceSyncResetChan, c.SyncDeviceStatus)

	// 北向下发命令轮询
	c.startAdjustableTickerWorker(c.commandPollInterval, c.commandPollResetChan, c.processNorthboundCommands)

	// 主循环
	c.wg.Add(1)
	go c.runLoop()

	log.Println("Collector started")
	return nil
}

// loadEnabledDevices 加载所有 enable 的设备到采集任务
func (c *Collector) loadEnabledDevices() error {
	devices, err := database.GetAllDevices()
	if err != nil {
		return fmt.Errorf("failed to get devices: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	loadedCount := 0
	for _, device := range devices {
		if device == nil || device.Enabled != 1 {
			continue
		}
		if c.upsertTaskLocked(device) {
			loadedCount++
		}
	}

	if loadedCount > 0 {
		c.notifyTaskChangedLocked()
	}

	log.Printf("Loaded %d enabled devices for collection", loadedCount)
	return nil
}

// runLoop 主采集循环（资源串行）
func (c *Collector) runLoop() {
	defer c.wg.Done()
	for {
		c.mu.Lock()
		task := c.peekNextCurrentTaskLocked()
		if task == nil {
			c.mu.Unlock()
			if c.waitForStopOrWake(0) {
				return
			}
			continue
		}

		delay := time.Until(task.nextRun)
		if delay > 0 {
			c.mu.Unlock()
			if c.waitForStopOrWake(delay) {
				return
			}
			continue
		}

		task = c.popNextCurrentTaskLocked()
		c.mu.Unlock()
		if task == nil {
			continue
		}

		if !c.isTaskCurrent(task) {
			continue
		}

		// 实际采集
		c.collectOnce(task)

		c.mu.Lock()
		if c.isTaskCurrentLocked(task) {
			task.nextRun = time.Now().Add(task.interval)
			heap.Push(c.taskHeap, task)
		}
		c.mu.Unlock()

		select {
		case <-c.stopChan:
			return
		default:
		}
	}
}

// collectOnce 执行单次采集
func (c *Collector) collectOnce(task *collectTask) {
	if task == nil || task.device == nil || c.driverExecutor == nil {
		return
	}

	device := task.device
	// 串行锁：同一资源串行访问
	lockCh := c.getResourceLock(device.ResourceID)
	lockCh <- struct{}{}
	defer func() { <-lockCh }()

	log.Printf("Collecting device %s (ID:%d)...", device.Name, device.ID)

	result, err := c.driverExecutor.Execute(device)
	if err != nil {
		log.Printf("Failed to collect device %s: %v", device.Name, err)
		return
	}

	collect := driverResultToCollectData(device, result)

	// 保存最新值到数据库（upsert 模式，只保存最新值）
	if err := database.InsertCollectDataWithOptions(collect, true); err != nil {
		log.Printf("Failed to insert data points: %v", err)
	}
	c.markTaskCollected(task, collect.Timestamp, true)

	// 缓存最新值（用于周期上传）
	c.cacheLatestData(collect)

	// 阈值计算 & 报警（但不上传北向，北向由周期任务负责）
	c.handleThreshold(collect)
}

// getResourceLock 返回该资源的串行锁
func (c *Collector) getResourceLock(resourceID *int64) chan struct{} {
	key := int64(0)
	if resourceID != nil {
		key = *resourceID
	}
	c.mu.Lock()
	ch, ok := c.resourceLock[key]
	if !ok {
		ch = make(chan struct{}, 1)
		c.resourceLock[key] = ch
	}
	c.mu.Unlock()
	return ch
}

// SyncDeviceStatus 同步设备状态（定时调用）
// 1. 检查数据库中设备状态变化
// 2. enable -> disable: 释放设备采集
// 3. disable -> enable: 加入管理，启动采集
// 4. 新的 enable 设备加入管理
func (c *Collector) SyncDeviceStatus() {
	devices, err := database.GetAllDevices()
	if err != nil {
		log.Printf("Failed to sync device status: %v", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	changed := false

	for _, device := range devices {
		action := c.syncDeviceTaskLocked(device)
		if action != deviceSyncActionNone {
			changed = true
		}
		c.logDeviceSyncAction(device, action)
	}

	if changed {
		c.notifyTaskChangedLocked()
	}
}

// Stop 停止采集器
func (c *Collector) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return fmt.Errorf("collector is not running")
	}
	c.running = false
	close(c.stopChan)
	c.mu.Unlock()

	c.wg.Wait()
	log.Println("Collector stopped")
	return nil
}

// IsRunning 检查采集器是否运行中
func (c *Collector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}

func (c *Collector) GetRuntimeIntervals() (time.Duration, time.Duration) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.deviceSyncInterval, c.commandPollInterval
}

func (c *Collector) SetRuntimeIntervals(deviceSyncInterval, commandPollInterval time.Duration) {
	c.mu.Lock()
	running := c.running
	if deviceSyncInterval > 0 {
		c.deviceSyncInterval = deviceSyncInterval
	}
	if commandPollInterval > 0 {
		c.commandPollInterval = commandPollInterval
	}
	deviceSyncResetChan := c.deviceSyncResetChan
	commandPollResetChan := c.commandPollResetChan
	deviceSyncCurrent := c.deviceSyncInterval
	commandPollCurrent := c.commandPollInterval
	c.mu.Unlock()

	if !running {
		return
	}

	if deviceSyncInterval > 0 {
		notifyIntervalChange(deviceSyncResetChan, deviceSyncCurrent)
	}
	if commandPollInterval > 0 {
		notifyIntervalChange(commandPollResetChan, commandPollCurrent)
	}
}

// AddDevice 添加设备到采集任务
func (c *Collector) AddDevice(device *models.Device) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}

	c.mu.Lock()

	if _, exists := c.tasks[device.ID]; exists {
		c.mu.Unlock()
		return fmt.Errorf("device %d already in collector", device.ID)
	}

	c.upsertTaskLocked(device)
	c.notifyTaskChangedLocked()
	c.mu.Unlock()
	return nil
}

// RemoveDevice 从采集任务移除设备
func (c *Collector) RemoveDevice(deviceID int64) error {
	c.mu.Lock()
	_, exists := c.tasks[deviceID]
	c.removeTaskLocked(deviceID)
	if exists {
		c.notifyTaskChangedLocked()
	}
	c.mu.Unlock()

	return nil
}

// UpdateDevice 更新设备采集配置
func (c *Collector) UpdateDevice(device *models.Device) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}

	c.mu.Lock()
	_, exists := c.tasks[device.ID]
	if !exists {
		c.mu.Unlock()
		return fmt.Errorf("device %d not in collector", device.ID)
	}

	c.upsertTaskLocked(device)
	c.notifyTaskChangedLocked()
	c.mu.Unlock()
	return nil
}

func (c *Collector) upsertTaskLocked(device *models.Device) bool {
	if device == nil {
		return false
	}
	current, exists := c.tasks[device.ID]
	task := newCollectTask(device, current)
	c.tasks[device.ID] = task
	heap.Push(c.taskHeap, task)
	return !exists
}

func (c *Collector) syncDeviceTaskLocked(device *models.Device) deviceSyncAction {
	if device == nil {
		return deviceSyncActionNone
	}

	_, exists := c.tasks[device.ID]
	if device.Enabled == 1 {
		created := c.upsertTaskLocked(device)
		if created {
			return deviceSyncActionAdded
		}
		return deviceSyncActionUpdated
	}

	if exists {
		c.removeTaskLocked(device.ID)
		return deviceSyncActionRemoved
	}

	return deviceSyncActionNone
}

func (c *Collector) logDeviceSyncAction(device *models.Device, action deviceSyncAction) {
	if device == nil {
		return
	}

	switch action {
	case deviceSyncActionAdded:
		log.Printf("Device %s (ID:%d) enabled, added to collection", device.Name, device.ID)
	case deviceSyncActionUpdated:
		log.Printf("Device %s (ID:%d) config updated", device.Name, device.ID)
	case deviceSyncActionRemoved:
		log.Printf("Device %s (ID:%d) disabled, removed from collection", device.Name, device.ID)
	}
}

func (c *Collector) removeTaskLocked(deviceID int64) {
	delete(c.tasks, deviceID)
}

func (c *Collector) notifyTaskChangedLocked() {
	select {
	case c.wakeChan <- struct{}{}:
	default:
	}
}

func (c *Collector) peekNextCurrentTaskLocked() *collectTask {
	for len(*c.taskHeap) > 0 {
		candidate := (*c.taskHeap)[0]
		if c.isTaskCurrentLocked(candidate) {
			return candidate
		}
		heap.Pop(c.taskHeap)
	}
	return nil
}

func (c *Collector) popNextCurrentTaskLocked() *collectTask {
	for len(*c.taskHeap) > 0 {
		candidate := heap.Pop(c.taskHeap).(*collectTask)
		if c.isTaskCurrentLocked(candidate) {
			return candidate
		}
	}
	return nil
}

func newCollectTask(device *models.Device, previous *collectTask) *collectTask {
	interval := resolveCollectInterval(device.CollectInterval)
	task := &collectTask{
		device:          device,
		interval:        interval,
		storageInterval: resolveStorageInterval(device.StorageInterval),
		nextRun:         time.Now().Add(interval),
		lastRun:         time.Time{},
	}
	if previous != nil {
		task.lastRun = previous.lastRun
		task.lastStored = previous.lastStored
	}
	return task
}

func (c *Collector) isTaskCurrentLocked(task *collectTask) bool {
	if task == nil || task.device == nil {
		return false
	}
	current, exists := c.tasks[task.device.ID]
	return exists && current == task
}

func (c *Collector) isTaskCurrent(task *collectTask) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isTaskCurrentLocked(task)
}

func (c *Collector) waitForStopOrWake(delay time.Duration) bool {
	if delay <= 0 {
		select {
		case <-c.stopChan:
			return true
		case <-c.wakeChan:
			return false
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return false
	case <-c.stopChan:
		return true
	case <-c.wakeChan:
		return false
	}
}

func (c *Collector) startTickerWorker(interval time.Duration, worker func()) {
	if worker == nil {
		return
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopChan:
				return
			case <-ticker.C:
				worker()
			}
		}
	}()
}

func (c *Collector) startAdjustableTickerWorker(interval time.Duration, resetChan <-chan time.Duration, worker func()) {
	if worker == nil {
		return
	}
	if interval <= 0 {
		interval = time.Second
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopChan:
				return
			case <-ticker.C:
				worker()
			case next := <-resetChan:
				if next <= 0 {
					continue
				}
				ticker.Stop()
				ticker = time.NewTicker(next)
			}
		}
	}()
}

func notifyIntervalChange(ch chan time.Duration, interval time.Duration) {
	if ch == nil || interval <= 0 {
		return
	}

	select {
	case ch <- interval:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- interval:
	default:
	}
}

func resolveCollectInterval(milliseconds int) time.Duration {
	if milliseconds <= 0 {
		milliseconds = defaultCollectIntervalMilliseconds
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func resolveStorageInterval(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = database.DefaultStorageIntervalSeconds
	}
	return time.Duration(seconds) * time.Second
}

func shouldStoreHistory(task *collectTask, collectedAt time.Time) bool {
	if task == nil {
		return false
	}

	interval := task.storageInterval
	if interval <= 0 {
		interval = resolveStorageInterval(0)
	}
	if task.lastStored.IsZero() {
		return true
	}
	if collectedAt.IsZero() {
		collectedAt = time.Now()
	}
	return collectedAt.Sub(task.lastStored) >= interval
}

func (c *Collector) markTaskCollected(task *collectTask, collectedAt time.Time, stored bool) {
	if task == nil {
		return
	}
	if collectedAt.IsZero() {
		collectedAt = time.Now()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isTaskCurrentLocked(task) {
		return
	}

	task.lastRun = collectedAt
	if stored {
		task.lastStored = collectedAt
	}
}

func parseFloatFieldValue(fields map[string]string, field string) (float64, bool) {
	if fields == nil {
		return 0, false
	}

	valueStr, ok := fields[field]
	if !ok {
		return 0, false
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(valueStr), 64)
	if err != nil {
		return 0, false
	}

	return value, true
}

// checkThresholds 检查阈值
func (c *Collector) checkThresholds(device *models.Device, data *models.CollectData) error {
	if device == nil || data == nil {
		return nil
	}

	thresholds, err := GetDeviceThresholds(device.ID)
	if err != nil {
		return err
	}

	checker := NewThresholdChecker()

	for _, threshold := range thresholds {
		if threshold == nil {
			continue
		}

		value, ok := parseFloatFieldValue(data.Fields, threshold.FieldName)
		if !ok {
			continue
		}

		if checker.Check(value, threshold.Operator, threshold.Value) {
			c.handleAlarm(device, threshold, value)
		}
	}

	return nil
}

// handleAlarm 处理报警
func (c *Collector) handleAlarm(device *models.Device, threshold *models.Threshold, actualValue float64) {
	// 创建报警日志
	logEntry := &models.AlarmLog{
		DeviceID:       device.ID,
		ThresholdID:    &threshold.ID,
		FieldName:      threshold.FieldName,
		ActualValue:    actualValue,
		ThresholdValue: threshold.Value,
		Operator:       threshold.Operator,
		Severity:       threshold.Severity,
		Message:        threshold.Message,
	}

	if _, err := database.CreateAlarmLog(logEntry); err != nil {
		log.Printf("Failed to create alarm log: %v", err)
	}

	// 发送到北向
	payload := &models.AlarmPayload{
		DeviceID:    device.ID,
		DeviceName:  device.Name,
		ProductKey:  device.ProductKey,
		DeviceKey:   device.DeviceKey,
		FieldName:   threshold.FieldName,
		ActualValue: actualValue,
		Threshold:   threshold.Value,
		Operator:    threshold.Operator,
		Severity:    threshold.Severity,
		Message:     threshold.Message,
	}

	c.northboundMgr.SendAlarm(payload)
}

// driverResultToCollectData 转换驱动结果为采集数据
func driverResultToCollectData(device *models.Device, res *driver.DriverResult) *models.CollectData {
	fields := map[string]string{}
	if len(res.Data) > 0 {
		for k, v := range res.Data {
			fields[k] = v
		}
	}
	for _, p := range res.Points {
		fields[p.FieldName] = fmt.Sprintf("%v", p.Value)
	}
	ts := res.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}
	productKey := device.ProductKey
	deviceKey := device.DeviceKey
	if productKey == "" || deviceKey == "" {
		gwProductKey, gwDeviceKey := database.GetGatewayIdentity()
		if productKey == "" {
			productKey = gwProductKey
		}
		if deviceKey == "" {
			deviceKey = gwDeviceKey
		}
	}

	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		ProductKey: productKey,
		DeviceKey:  deviceKey,
		Timestamp:  ts,
		Fields:     fields,
	}
}

// handleThresholdAndNorthbound 阈值 + 北向上传（已废弃，使用周期上传模式）
func (c *Collector) handleThresholdAndNorthbound(data *models.CollectData) {
	device, err := database.GetDeviceByID(data.DeviceID)
	if err == nil && device != nil {
		if err := c.checkThresholds(device, data); err != nil {
			log.Printf("check thresholds error: %v", err)
		}
	}
	// 改为周期上传，不再立即上传
}

// handleThreshold 仅检查阈值（用于采集时触发报警，不发送北向）
func (c *Collector) handleThreshold(data *models.CollectData) {
	device, err := database.GetDeviceByID(data.DeviceID)
	if err == nil && device != nil {
		if err := c.checkThresholds(device, data); err != nil {
			log.Printf("check thresholds error: %v", err)
		}
	}
}

// cacheLatestData 缓存最新数据（用于周期上传）
func (c *Collector) cacheLatestData(data *models.CollectData) {
	if data == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// 更新或添加最新值
	c.latestDataCache[data.DeviceID] = data
	log.Printf("Collector: cached latest data for device %d (%d fields)", data.DeviceID, len(data.Fields))
}

func (c *Collector) processNorthboundCommands() {
	if c.northboundMgr == nil {
		return
	}

	commands, err := c.northboundMgr.PullCommands(20)
	if err != nil {
		log.Printf("pull northbound commands failed: %v", err)
		return
	}
	if len(commands) == 0 {
		return
	}

	for _, command := range commands {
		if command == nil {
			continue
		}
		err := c.executeNorthboundCommand(command)
		if err != nil {
			log.Printf("execute northbound command failed: source=%s request_id=%s product_key=%s device_key=%s field=%s error=%v",
				command.Source, command.RequestID, command.ProductKey, command.DeviceKey, command.FieldName, err)
		}
		c.reportCommandResult(command, err)
	}
}

func (c *Collector) reportCommandResult(command *models.NorthboundCommand, execErr error) {
	if c.northboundMgr == nil || command == nil {
		return
	}

	result := buildNorthboundCommandResult(command, execErr)
	c.northboundMgr.ReportCommandResult(result)
}

func buildNorthboundCommandResult(command *models.NorthboundCommand, execErr error) *models.NorthboundCommandResult {
	if command == nil {
		return nil
	}

	result := &models.NorthboundCommandResult{
		RequestID:  command.RequestID,
		ProductKey: command.ProductKey,
		DeviceKey:  command.DeviceKey,
		FieldName:  command.FieldName,
		Value:      command.Value,
		Source:     command.Source,
		Success:    execErr == nil,
		Code:       200,
	}
	if execErr != nil {
		result.Code = 500
		result.Message = execErr.Error()
	}

	return result
}

func (c *Collector) executeNorthboundCommand(command *models.NorthboundCommand) error {
	if c.driverExecutor == nil {
		return fmt.Errorf("driver executor is nil")
	}

	normalizedCommand, err := normalizeNorthboundCommand(command)
	if err != nil {
		return err
	}

	device, err := database.GetDeviceByIdentity(normalizedCommand.ProductKey, normalizedCommand.DeviceKey)
	if err != nil || device == nil {
		return fmt.Errorf("device not found by identity")
	}
	if device.DriverID == nil {
		return fmt.Errorf("device has no driver")
	}

	config := buildNorthboundCommandConfig(normalizedCommand, device)

	result, err := c.driverExecutor.ExecuteCommand(device, "handle", config)
	if err != nil {
		return err
	}
	if result != nil && !result.Success {
		if strings.TrimSpace(result.Error) != "" {
			return fmt.Errorf("%s", result.Error)
		}
		return fmt.Errorf("driver write returned success=false")
	}

	log.Printf("northbound command executed: source=%s request_id=%s device_id=%d field=%s value=%s",
		normalizedCommand.Source, normalizedCommand.RequestID, device.ID, normalizedCommand.FieldName, normalizedCommand.Value)
	return nil
}

func normalizeNorthboundCommand(command *models.NorthboundCommand) (*models.NorthboundCommand, error) {
	if command == nil {
		return nil, fmt.Errorf("northbound command is nil")
	}

	normalizedCommand := &models.NorthboundCommand{
		RequestID:  strings.TrimSpace(command.RequestID),
		ProductKey: strings.TrimSpace(command.ProductKey),
		DeviceKey:  strings.TrimSpace(command.DeviceKey),
		FieldName:  strings.TrimSpace(command.FieldName),
		Value:      strings.TrimSpace(command.Value),
		Source:     strings.TrimSpace(command.Source),
	}

	if normalizedCommand.ProductKey == "" || normalizedCommand.DeviceKey == "" {
		return nil, fmt.Errorf("missing product_key/device_key")
	}
	if normalizedCommand.FieldName == "" {
		return nil, fmt.Errorf("missing field_name")
	}

	return normalizedCommand, nil
}

func buildNorthboundCommandConfig(command *models.NorthboundCommand, device *models.Device) map[string]string {
	if command == nil || device == nil {
		return nil
	}

	return map[string]string{
		"func_name":      "write",
		"field_name":     command.FieldName,
		"value":          command.Value,
		"product_key":    command.ProductKey,
		"productKey":     command.ProductKey,
		"device_key":     command.DeviceKey,
		"deviceKey":      command.DeviceKey,
		"device_address": device.DeviceAddress,
	}
}

// ThresholdChecker 阈值检查器
type ThresholdChecker struct {
	mu sync.RWMutex
}

// NewThresholdChecker 创建阈值检查器
func NewThresholdChecker() *ThresholdChecker {
	return &ThresholdChecker{}
}

// Check 检查值是否触发阈值
func (t *ThresholdChecker) Check(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}
