package driver

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gogw/internal/models"
	"gogw/internal/resource"
)

// ErrDriverNotFound 驱动未找到
var ErrDriverNotFound = errors.New("driver not found")

// ErrDriverNotLoaded 驱动未加载
var ErrDriverNotLoaded = errors.New("driver not loaded")

// ErrDriverExecutionFailed 驱动执行失败
var ErrDriverExecutionFailed = errors.New("driver execution failed")

// DriverResult 驱动执行结果
type DriverResult struct {
	Success   bool              `json:"success"`
	Data      map[string]string `json:"data"`
	Error     string            `json:"error"`
	Timestamp time.Time         `json:"timestamp"`
}

// DriverContext 驱动上下文
type DriverContext struct {
	DeviceID     int64             `json:"device_id"`
	DeviceName   string            `json:"device_name"`
	ResourceID   int64             `json:"resource_id"`
	ResourceType string            `json:"resource_type"`
	Config       map[string]string `json:"config"`
	DeviceConfig string            `json:"device_config"`
}

// DriverPlugin 驱动插件接口
type DriverPlugin interface {
	// Initialize 初始化驱动
	Initialize(config string) error
	// Read 读取数据
	Read(ctx *DriverContext) (*DriverResult, error)
	// Write 写入数据
	Write(ctx *DriverContext, data map[string]string) error
	// Close 关闭驱动
	Close() error
}

// WasmDriver WASM驱动实现
type WasmDriver struct {
	ID         int64
	Name       string
	plugin     interface{} // Extism plugin
	mu         sync.RWMutex
	lastActive time.Time
	config     string
}

// DriverManager 驱动管理器
type DriverManager struct {
	mu      sync.RWMutex
	drivers map[int64]*WasmDriver
	plugins map[int64]interface{} // Extism plugins
}

// NewDriverManager 创建驱动管理器
func NewDriverManager() *DriverManager {
	return &DriverManager{
		drivers: make(map[int64]*WasmDriver),
		plugins: make(map[int64]interface{}),
	}
}

// LoadDriver 加载驱动
func (m *DriverManager) LoadDriver(driver *models.Driver) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.drivers[driver.ID]; exists {
		return fmt.Errorf("driver %d already loaded", driver.ID)
	}

	wasmDriver := &WasmDriver{
		ID:         driver.ID,
		Name:       driver.Name,
		config:     driver.ConfigSchema,
		lastActive: time.Now(),
	}

	// 加载WASM插件
	// 注意：需要使用Extism Go SDK加载WASM文件
	// 这里使用纯Go实现，不依赖cgo

	m.drivers[driver.ID] = wasmDriver
	return nil
}

// UnloadDriver 卸载驱动
func (m *DriverManager) UnloadDriver(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	driver, exists := m.drivers[id]
	if !exists {
		return ErrDriverNotFound
	}

	// 关闭插件
	if _, exists := m.plugins[id]; exists {
		// 清理插件资源
		delete(m.plugins, id)
	}

	delete(m.drivers, id)
	_ = driver // 使用driver变量

	return nil
}

// ExecuteDriver 执行驱动
func (m *DriverManager) ExecuteDriver(id int64, function string, ctx *DriverContext) (*DriverResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	driver, exists := m.drivers[id]
	if !exists {
		return nil, ErrDriverNotFound
	}

	driver.mu.Lock()
	driver.lastActive = time.Now()
	driver.mu.Unlock()

	// 准备输入数据
	input := map[string]interface{}{
		"device_id":     ctx.DeviceID,
		"device_name":   ctx.DeviceName,
		"resource_id":   ctx.ResourceID,
		"resource_type": ctx.ResourceType,
		"config":        ctx.Config,
		"device_config": ctx.DeviceConfig,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}
	_ = inputJSON // 暂时不使用，后续用于Extism调用

	// 调用Extism插件
	// 注意：需要使用Extism Go SDK调用WASM函数
	// 这里使用纯Go实现

	// 模拟执行结果
	result := &DriverResult{
		Success:   true,
		Data:      make(map[string]string),
		Timestamp: time.Now(),
	}

	// 根据驱动名称返回模拟数据
	switch driver.Name {
	case "th.wasm":
		// 温湿度驱动模拟
		result.Data["temperature"] = "25.5"
		result.Data["humidity"] = "60.2"
	default:
		result.Data["value"] = "0"
	}

	return result, nil
}

// GetDriver 获取驱动
func (m *DriverManager) GetDriver(id int64) (*WasmDriver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	driver, exists := m.drivers[id]
	if !exists {
		return nil, ErrDriverNotFound
	}

	return driver, nil
}

// ListDrivers 列出所有已加载的驱动
func (m *DriverManager) ListDrivers() []*WasmDriver {
	m.mu.RLock()
	defer m.mu.RUnlock()

	drivers := make([]*WasmDriver, 0, len(m.drivers))
	for _, driver := range m.drivers {
		drivers = append(drivers, driver)
	}
	return drivers
}

// IsLoaded 检查驱动是否已加载
func (m *DriverManager) IsLoaded(id int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.drivers[id]
	return exists
}

// DriverExecutor 驱动执行器
type DriverExecutor struct {
	manager     *DriverManager
	resourceMgr *resource.ResourceManagerImpl
	mu          sync.RWMutex
	executing   map[int64]bool
}

// NewDriverExecutor 创建驱动执行器
func NewDriverExecutor(manager *DriverManager) *DriverExecutor {
	return &DriverExecutor{
		manager:   manager,
		executing: make(map[int64]bool),
	}
}

// SetResourceManager 设置资源管理器（用于资源访问锁）
func (e *DriverExecutor) SetResourceManager(rm *resource.ResourceManagerImpl) {
	e.resourceMgr = rm
}

// Execute 执行驱动读取（带资源访问锁）
func (e *DriverExecutor) Execute(device *models.Device) (*DriverResult, error) {
	if device.DriverID == nil {
		return nil, fmt.Errorf("device %s has no driver", device.Name)
	}

	e.mu.Lock()
	if e.executing[device.ID] {
		e.mu.Unlock()
		return nil, fmt.Errorf("device %s is already being read", device.Name)
	}
	e.executing[device.ID] = true
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.executing, device.ID)
		e.mu.Unlock()
	}()

	ctx := &DriverContext{
		DeviceID:     device.ID,
		DeviceName:   device.Name,
		ResourceID:   0, // 需要从数据库获取
		ResourceType: "serial",
		Config:       make(map[string]string),
		DeviceConfig: device.DeviceConfig,
	}

	// 如果配置了资源管理器，使用资源访问锁确保串行访问
	if e.resourceMgr != nil && device.ResourceID != nil && *device.ResourceID > 0 {
		locker := e.resourceMgr.GetLocker()
		if locker != nil {
			// 尝试获取资源锁，设置超时避免永久阻塞
			if err := locker.LockWithTimeout(*device.ResourceID, 10*time.Second); err != nil {
				return nil, fmt.Errorf("failed to lock resource %d: %w", *device.ResourceID, err)
			}
			defer locker.Unlock(*device.ResourceID)
		}
	}

	return e.manager.ExecuteDriver(*device.DriverID, "read", ctx)
}

// CollectData 采集数据
func (e *DriverExecutor) CollectData(device *models.Device) (*models.CollectData, error) {
	result, err := e.Execute(device)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.Error)
	}

	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		Timestamp:  result.Timestamp,
		Fields:     result.Data,
	}, nil
}
