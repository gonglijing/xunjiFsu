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
	mu             sync.RWMutex
	running        bool
	stopChan       chan struct{}
	wg             sync.WaitGroup
	// 设备采集任务
	tasks      map[int64]*collectTask
	taskHeap   *taskHeap // 优先队列，按下次采集时间排序
}

// collectTask 采集任务
type collectTask struct {
	device     *models.Device
	interval   time.Duration
	nextRun    time.Time
	lastRun    time.Time
	lastUpload time.Time
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

	log.Println("Collector started")
	return nil
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
	task := &collectTask{
		device:   device,
		interval: interval,
		nextRun:  time.Now().Add(interval),
		lastRun:  time.Time{},
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
	task.nextRun = time.Now().Add(task.interval)
	return nil
}

// Run 运行采集任务（使用时间轮调度）
func (c *Collector) Run() {
	c.wg.Add(1)
	defer c.wg.Done()

	for {
		c.mu.RLock()
		taskCount := len(c.tasks)
		c.mu.RUnlock()

		if taskCount == 0 {
			select {
			case <-c.stopChan:
				return
			case <-time.After(time.Second):
				continue
			}
		}

		// 获取下一个要执行的任务
		c.mu.Lock()
		if c.taskHeap.Len() == 0 {
			c.mu.Unlock()
			select {
			case <-c.stopChan:
				return
			case <-time.After(time.Second):
				continue
			}
		}

		task := heap.Pop(c.taskHeap).(*collectTask)
		now := time.Now()

		// 如果还没到执行时间，等待
		waitTime := task.nextRun.Sub(now)
		c.mu.Unlock()

		if waitTime > 0 {
			select {
			case <-c.stopChan:
				return
			case <-time.After(waitTime):
				// 时间到，执行采集
			}
		}

		// 执行采集
		c.collectDevice(task)
	}
}

// collectDevice 采集单个设备数据
func (c *Collector) collectDevice(task *collectTask) {
	device := task.device
	if device.Enabled != 1 {
		return
	}

	data, err := c.driverExecutor.CollectData(device)
	if err := err; err != nil {
		log.Printf("Failed to collect data from device %s: %v", device.Name, err)
		// 重新加入队列，稍后重试
		c.requeueTask(task)
		return
	}

	now := time.Now()
	task.lastRun = now
	task.nextRun = now.Add(task.interval)

	// 收集数据点用于批量保存
	dataPoints := make([]database.DataPointEntry, 0, len(data.Fields))
	for fieldName, value := range data.Fields {
		// 更新缓存
		if err := database.SaveDataCache(device.ID, device.Name, fieldName, value, "float64"); err != nil {
			log.Printf("Failed to save data cache: %v", err)
		}
		// 收集历史数据点
		dataPoints = append(dataPoints, database.DataPointEntry{
			DeviceID:    device.ID,
			DeviceName:  device.Name,
			FieldName:   fieldName,
			Value:       value,
			ValueType:   "float64",
			CollectedAt: now,
		})
	}

	// 批量保存历史数据点
	if len(dataPoints) > 0 {
		if err := database.BatchSaveDataPoints(dataPoints); err != nil {
			log.Printf("Failed to batch save data points: %v", err)
		}
	}

	// 检查阈值
	if err := c.checkThresholds(device, data); err != nil {
		log.Printf("Failed to check thresholds: %v", err)
	}

	// 检查是否需要上传到北向
	if now.Sub(task.lastUpload) >= time.Duration(device.UploadInterval)*time.Millisecond {
		c.uploadToNorthbound(data)
		task.lastUpload = now
	}

	// 重新加入队列
	c.mu.Lock()
	c.tasks[device.ID] = task
	heap.Push(c.taskHeap, task)
	c.mu.Unlock()
}

// requeueTask 重新加入任务队列
func (c *Collector) requeueTask(task *collectTask) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 30秒后重试
	task.nextRun = time.Now().Add(30 * time.Second)
	c.tasks[task.device.ID] = task
	heap.Push(c.taskHeap, task)
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
