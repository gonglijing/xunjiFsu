package collector

import (
	"container/heap"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

// Collector 采集器
type Collector struct {
	driverExecutor *driver.DriverExecutor
	northboundMgr  *northbound.NorthboundManager
	resourceLock   map[int64]chan struct{} // 每个资源一个串行锁
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
	wg             sync.WaitGroup
	// 设备采集任务
	tasks    map[int64]*collectTask
	taskHeap *taskHeap // 优先队列，按下次采集时间排序
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
	h := &taskHeap{}
	heap.Init(h)
	return &Collector{
		driverExecutor: driverExecutor,
		northboundMgr:  northboundMgr,
		running:        false,
		stopChan:       make(chan struct{}),
		tasks:          make(map[int64]*collectTask),
		taskHeap:       h,
		resourceLock:   make(map[int64]chan struct{}),
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
	c.mu.Unlock()

	// 开机时加载所有 enable 的设备
	if err := c.loadEnabledDevices(); err != nil {
		log.Printf("Failed to load enabled devices: %v", err)
	}

	// 每 10 秒同步设备状态
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.stopChan:
				return
			case <-ticker.C:
				c.SyncDeviceStatus()
			}
		}
	}()

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
		if device.Enabled == 1 {
			interval := time.Duration(device.CollectInterval) * time.Millisecond
			storageInterval := resolveStorageInterval(device.StorageInterval)
			task := &collectTask{
				device:          device,
				interval:        interval,
				storageInterval: storageInterval,
				nextRun:         time.Now().Add(interval),
				lastRun:         time.Time{},
			}
			c.tasks[device.ID] = task
			heap.Push(c.taskHeap, task)
			loadedCount++
		}
	}

	log.Printf("Loaded %d enabled devices for collection", loadedCount)
	return nil
}

// runLoop 主采集循环（资源串行）
func (c *Collector) runLoop() {
	defer c.wg.Done()
	for {
		c.mu.Lock()
		if len(*c.taskHeap) == 0 {
			c.mu.Unlock()
			time.Sleep(500 * time.Millisecond)
			select {
			case <-c.stopChan:
				return
			default:
				continue
			}
		}
		task := heap.Pop(c.taskHeap).(*collectTask)
		now := time.Now()
		delay := task.nextRun.Sub(now)
		if delay > 0 {
			c.mu.Unlock()
			select {
			case <-time.After(delay):
			case <-c.stopChan:
				return
			}
			c.mu.Lock()
		}
		// 重新计算下一次时间并放回堆\n\t\t// 在采集后更新 nextRun\n\t\t// 防止 stop 时漏 unlock\n
		task.nextRun = time.Now().Add(task.interval)
		heap.Push(c.taskHeap, task)
		c.mu.Unlock()

		// 实际采集
		c.collectOnce(task)
		select {
		case <-c.stopChan:
			return
		default:
		}
	}
}

// collectOnce 执行单次采集
func (c *Collector) collectOnce(task *collectTask) {
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

	storeHistory := shouldStoreHistory(task, collect.Timestamp)
	if err := database.InsertCollectDataWithOptions(collect, storeHistory); err != nil {
		log.Printf("Failed to insert data points: %v", err)
	}
	if storeHistory {
		task.lastStored = collect.Timestamp
	}

	// 阈值计算 & 报警、北向上传
	c.handleThresholdAndNorthbound(collect)
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

	// 获取当前正在采集的设备 ID
	activeDeviceIDs := make(map[int64]bool)
	for id := range c.tasks {
		activeDeviceIDs[id] = true
	}

	for _, device := range devices {
		task, exists := c.tasks[device.ID]

		if device.Enabled == 1 {
			// 设备启用
			if !exists {
				// 新启用的设备，加入采集
				interval := time.Duration(device.CollectInterval) * time.Millisecond
				storageInterval := resolveStorageInterval(device.StorageInterval)
				newTask := &collectTask{
					device:          device,
					interval:        interval,
					storageInterval: storageInterval,
					nextRun:         time.Now().Add(interval),
					lastRun:         time.Time{},
				}
				c.tasks[device.ID] = newTask
				heap.Push(c.taskHeap, newTask)
				log.Printf("Device %s (ID:%d) enabled, added to collection", device.Name, device.ID)
			} else {
				// 已存在的设备，更新配置
				task.device = device
				task.interval = time.Duration(device.CollectInterval) * time.Millisecond
				task.storageInterval = resolveStorageInterval(device.StorageInterval)
				log.Printf("Device %s (ID:%d) config updated", device.Name, device.ID)
			}
		} else {
			// 设备禁用
			if exists {
				// 从采集任务中移除
				delete(c.tasks, device.ID)
				log.Printf("Device %s (ID:%d) disabled, removed from collection", device.Name, device.ID)
			}
		}
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

// AddDevice 添加设备到采集任务
func (c *Collector) AddDevice(device *models.Device) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.tasks[device.ID]; exists {
		return fmt.Errorf("device %d already in collector", device.ID)
	}

	interval := time.Duration(device.CollectInterval) * time.Millisecond
	storageInterval := resolveStorageInterval(device.StorageInterval)
	task := &collectTask{
		device:          device,
		interval:        interval,
		storageInterval: storageInterval,
		nextRun:         time.Now().Add(interval),
		lastRun:         time.Time{},
	}

	c.tasks[device.ID] = task
	heap.Push(c.taskHeap, task)
	return nil
}

// RemoveDevice 从采集任务移除设备
func (c *Collector) RemoveDevice(deviceID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.tasks, deviceID)
	return nil
}

// UpdateDevice 更新设备采集配置
func (c *Collector) UpdateDevice(device *models.Device) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	task, exists := c.tasks[device.ID]
	if !exists {
		return fmt.Errorf("device %d not in collector", device.ID)
	}

	task.device = device
	task.interval = time.Duration(device.CollectInterval) * time.Millisecond
	task.storageInterval = resolveStorageInterval(device.StorageInterval)
	task.nextRun = time.Now().Add(task.interval)
	return nil
}

func resolveStorageInterval(seconds int) time.Duration {
	if seconds <= 0 {
		seconds = database.DefaultStorageIntervalSeconds
	}
	return time.Duration(seconds) * time.Second
}

func shouldStoreHistory(task *collectTask, collectedAt time.Time) bool {
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

// checkThresholds 检查阈值
func (c *Collector) checkThresholds(device *models.Device, data *models.CollectData) error {
	thresholds, err := GetDeviceThresholds(device.ID)
	if err != nil {
		return err
	}

	for _, threshold := range thresholds {
		valueStr, ok := data.Fields[threshold.FieldName]
		if !ok {
			continue
		}

		var value float64
		if _, err := fmt.Sscanf(valueStr, "%f", &value); err != nil {
			continue
		}

		triggered := false
		switch threshold.Operator {
		case ">":
			triggered = value > threshold.Value
		case "<":
			triggered = value < threshold.Value
		case ">=":
			triggered = value >= threshold.Value
		case "<=":
			triggered = value <= threshold.Value
		case "==":
			triggered = value == threshold.Value
		case "!=":
			triggered = value != threshold.Value
		}

		if triggered {
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

// uploadToNorthbound 上传到北向
func (c *Collector) uploadToNorthbound(data *models.CollectData) {
	c.northboundMgr.SendData(data)
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
	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		ProductKey: device.ProductKey,
		DeviceKey:  device.DeviceKey,
		Timestamp:  ts,
		Fields:     fields,
	}
}

// handleThresholdAndNorthbound 阈值 + 北向上传
func (c *Collector) handleThresholdAndNorthbound(data *models.CollectData) {
	device, err := database.GetDeviceByID(data.DeviceID)
	if err == nil && device != nil {
		if err := c.checkThresholds(device, data); err != nil {
			log.Printf("check thresholds error: %v", err)
		}
	}
	// 当前策略：采集后即上传；后续可按设备/北向上传周期优化
	c.uploadToNorthbound(data)
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
