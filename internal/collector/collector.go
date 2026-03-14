package collector

import (
	"container/heap"
	"fmt"
	"log"
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
	driverIdentityMu     sync.RWMutex
	driverProductKeys    map[int64]string // 驱动级固定 productKey 缓存
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
}

// collectTask 采集任务
type collectTask struct {
	device              *models.Device
	preparedRead        *driver.PreparedExecution
	interval            time.Duration
	storageInterval     time.Duration
	nextRun             time.Time
	lastRun             time.Time
	lastStored          time.Time
	lastErrorAt         time.Time
	lastError           string
	lastErrorKind       collectErrorKind
	consecutiveFailures int
	index               int
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
func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *taskHeap) Push(x interface{}) {
	task := x.(*collectTask)
	task.index = len(*h)
	*h = append(*h, task)
}
func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	task := old[n-1]
	old[n-1] = nil
	task.index = -1
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
		driverProductKeys:    make(map[int64]string),
		running:              false,
		stopChan:             make(chan struct{}),
		wakeChan:             make(chan struct{}, 1),
		deviceSyncResetChan:  make(chan time.Duration, 1),
		commandPollResetChan: make(chan time.Duration, 1),
		deviceSyncInterval:   deviceSyncInterval,
		commandPollInterval:  commandPollInterval,
		tasks:                make(map[int64]*collectTask),
		taskHeap:             h,
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
	h := &taskHeap{}
	heap.Init(h)
	c.taskHeap = h
	c.driverIdentityMu.Lock()
	c.driverProductKeys = make(map[int64]string)
	c.driverIdentityMu.Unlock()
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
		if c.upsertTaskLocked(device) == deviceSyncActionAdded {
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
	if !c.canCollectTask(task) {
		return
	}

	device := task.device
	log.Printf("Collecting device %s (ID:%d)...", device.Name, device.ID)

	collect, err := c.collectDataFromDriver(task)
	if err != nil {
		c.handleCollectFailure(task, err)
		return
	}

	c.persistCollectData(task, collect)
	c.handleThresholdForDevice(device, collect)
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
	seenDeviceIDs := make(map[int64]struct{}, len(devices))

	for _, device := range devices {
		if device == nil {
			continue
		}
		seenDeviceIDs[device.ID] = struct{}{}

		action := c.syncDeviceTaskLocked(device)
		if action != deviceSyncActionNone {
			changed = true
		}
		c.logDeviceSyncAction(device, action)
	}

	if removed := c.pruneMissingTasksLocked(seenDeviceIDs); removed > 0 {
		changed = true
		log.Printf("Pruned %d orphaned device tasks", removed)
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

	action := c.upsertTaskLocked(device)
	if action != deviceSyncActionNone {
		c.notifyTaskChangedLocked()
	}
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

	action := c.upsertTaskLocked(device)
	if action != deviceSyncActionNone {
		c.notifyTaskChangedLocked()
	}
	c.mu.Unlock()
	return nil
}

func (c *Collector) upsertTaskLocked(device *models.Device) deviceSyncAction {
	if device == nil {
		return deviceSyncActionNone
	}
	current, exists := c.tasks[device.ID]
	if exists && !shouldRefreshCollectTask(current, device) {
		return deviceSyncActionNone
	}

	if exists {
		c.refreshTaskLocked(current, device)
		return deviceSyncActionUpdated
	}

	task := newCollectTask(device, nil)
	c.tasks[device.ID] = task
	heap.Push(c.taskHeap, task)
	return deviceSyncActionAdded
}

func sameOptionalInt64(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func sameTaskDeviceConfig(current, next *models.Device) bool {
	if current == nil || next == nil {
		return false
	}

	return current.Name == next.Name &&
		current.Description == next.Description &&
		current.ProductKey == next.ProductKey &&
		current.DeviceKey == next.DeviceKey &&
		current.DriverType == next.DriverType &&
		current.SerialPort == next.SerialPort &&
		current.BaudRate == next.BaudRate &&
		current.DataBits == next.DataBits &&
		current.StopBits == next.StopBits &&
		current.Parity == next.Parity &&
		current.IPAddress == next.IPAddress &&
		current.PortNum == next.PortNum &&
		current.DeviceAddress == next.DeviceAddress &&
		current.CollectInterval == next.CollectInterval &&
		current.StorageInterval == next.StorageInterval &&
		current.Timeout == next.Timeout &&
		sameOptionalInt64(current.DriverID, next.DriverID) &&
		sameOptionalInt64(current.ResourceID, next.ResourceID) &&
		current.Enabled == next.Enabled
}

func shouldRefreshCollectTask(current *collectTask, nextDevice *models.Device) bool {
	if current == nil || nextDevice == nil {
		return true
	}
	if current.interval != resolveCollectInterval(nextDevice.CollectInterval) ||
		current.storageInterval != resolveStorageInterval(nextDevice.StorageInterval) {
		return true
	}
	return !sameTaskDeviceConfig(current.device, nextDevice)
}

func (c *Collector) syncDeviceTaskLocked(device *models.Device) deviceSyncAction {
	if device == nil {
		return deviceSyncActionNone
	}

	_, exists := c.tasks[device.ID]
	if device.Enabled == 1 {
		return c.upsertTaskLocked(device)
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
	if task, exists := c.tasks[deviceID]; exists && task != nil && task.index >= 0 {
		heap.Remove(c.taskHeap, task.index)
	}
	delete(c.tasks, deviceID)
	clearAlarmStateForDevice(deviceID)
}

func (c *Collector) pruneMissingTasksLocked(seenDeviceIDs map[int64]struct{}) int {
	if len(c.tasks) == 0 {
		return 0
	}

	removed := 0
	for deviceID := range c.tasks {
		if _, exists := seenDeviceIDs[deviceID]; exists {
			continue
		}
		c.removeTaskLocked(deviceID)
		removed++
	}
	return removed
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
		preparedRead:    driver.NewPreparedExecution(device),
		interval:        interval,
		storageInterval: resolveStorageInterval(device.StorageInterval),
		nextRun:         time.Now().Add(interval),
		lastRun:         time.Time{},
		lastErrorKind:   collectErrorKindNone,
		index:           -1,
	}
	if previous != nil {
		task.lastRun = previous.lastRun
		task.lastStored = previous.lastStored
		task.lastErrorAt = previous.lastErrorAt
		task.lastError = previous.lastError
		task.lastErrorKind = previous.lastErrorKind
		task.consecutiveFailures = previous.consecutiveFailures
	}
	return task
}

func (c *Collector) refreshTaskLocked(task *collectTask, device *models.Device) {
	if task == nil || device == nil {
		return
	}

	task.device = device
	task.preparedRead = driver.NewPreparedExecution(device)
	task.interval = resolveCollectInterval(device.CollectInterval)
	task.storageInterval = resolveStorageInterval(device.StorageInterval)
	task.nextRun = time.Now().Add(task.interval)

	if task.index >= 0 {
		heap.Fix(c.taskHeap, task.index)
	}
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
	task.consecutiveFailures = 0
	task.lastError = ""
	task.lastErrorKind = collectErrorKindNone
	task.lastErrorAt = time.Time{}
}

func (c *Collector) markTaskFailed(task *collectTask, err error, kind collectErrorKind) int {
	if task == nil || err == nil {
		return 0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isTaskCurrentLocked(task) {
		return 0
	}

	task.consecutiveFailures++
	task.lastError = strings.TrimSpace(err.Error())
	task.lastErrorKind = kind
	task.lastErrorAt = time.Now()
	return task.consecutiveFailures
}
