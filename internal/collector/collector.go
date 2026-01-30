package collector

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gogw/internal/database"
	"gogw/internal/driver"
	"gogw/internal/models"
	"gogw/internal/northbound"
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
	tasks map[int64]*collectTask
}

type collectTask struct {
	device     *models.Device
	interval   time.Duration
	lastRun    time.Time
	lastUpload time.Time
}

// NewCollector 创建采集器
func NewCollector(driverExecutor *driver.DriverExecutor, northboundMgr *northbound.NorthboundManager) *Collector {
	return &Collector{
		driverExecutor: driverExecutor,
		northboundMgr:  northboundMgr,
		running:        false,
		stopChan:       make(chan struct{}),
		tasks:          make(map[int64]*collectTask),
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

	task := &collectTask{
		device:   device,
		interval: time.Duration(device.CollectInterval) * time.Millisecond,
		lastRun:  time.Time{},
	}

	c.tasks[device.ID] = task
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
	return nil
}

// Run 运行采集任务
func (c *Collector) Run() {
	c.wg.Add(1)
	defer c.wg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collectAll()
		}
	}
}

// collectAll 采集所有设备数据
func (c *Collector) collectAll() {
	c.mu.RLock()
	tasks := make([]*collectTask, 0, len(c.tasks))
	for _, task := range c.tasks {
		tasks = append(tasks, task)
	}
	c.mu.RUnlock()

	now := time.Now()
	for _, task := range tasks {
		// 检查是否需要采集
		if now.Sub(task.lastRun) >= task.interval {
			c.collectDevice(task, now)
		}
	}
}

// collectDevice 采集单个设备数据
func (c *Collector) collectDevice(task *collectTask, now time.Time) {
	device := task.device
	if device.Enabled != 1 {
		return
	}

	data, err := c.driverExecutor.CollectData(device)
	if err != nil {
		log.Printf("Failed to collect data from device %s: %v", device.Name, err)
		return
	}

	task.lastRun = now

	// 保存数据到缓存和历史数据
	for fieldName, value := range data.Fields {
		if err := database.SaveDataCache(device.ID, device.Name, fieldName, value, "float64"); err != nil {
			log.Printf("Failed to save data cache: %v", err)
		}
		// 保存历史数据点
		if err := database.SaveDataPoint(device.ID, device.Name, fieldName, value, "float64"); err != nil {
			log.Printf("Failed to save data point: %v", err)
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
}

// checkThresholds 检查阈值
func (c *Collector) checkThresholds(device *models.Device, data *models.CollectData) error {
	thresholds, err := database.GetEnabledThresholdsByDeviceID(device.ID)
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
