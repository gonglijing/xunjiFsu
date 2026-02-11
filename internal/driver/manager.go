//go:build !no_extism

package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ErrDriverNotFound 驱动未找到
var ErrDriverNotFound = errors.New("driver not found")

// ErrDriverNotLoaded 驱动未加载
var ErrDriverNotLoaded = errors.New("driver not loaded")

// ErrDriverExecutionFailed 驱动执行失败
var ErrDriverExecutionFailed = errors.New("driver execution failed")

// ErrDriverTimeout 驱动执行超时
var ErrDriverTimeout = errors.New("driver execution timeout")

// ErrDriverCanceled 驱动执行被取消
var ErrDriverCanceled = errors.New("driver execution canceled")

// ErrDriverBadOutput 驱动输出无效
var ErrDriverBadOutput = errors.New("driver output invalid")

func hexPreview(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}
	if max <= 0 || len(b) <= max {
		return fmt.Sprintf("% X", b)
	}
	return fmt.Sprintf("% X", b[:max])
}

// DriverResult 驱动执行结果
type DriverResult struct {
	Success   bool              `json:"success"`
	Data      map[string]string `json:"data"`   // 旧格式: {"temperature": "25.3"}
	Points    []DriverPoint     `json:"points"` // 新格式: [{"field_name":"temperature","value":25.3,"rw":"R"}]
	Error     string            `json:"error"`
	Timestamp time.Time         `json:"timestamp"`
}

func isReadFunction(function string, driverCtx *DriverContext) bool {
	f := strings.ToLower(strings.TrimSpace(function))
	if f == "read" || f == "collect" {
		return true
	}

	if driverCtx == nil || driverCtx.Config == nil {
		return false
	}

	funcName := strings.ToLower(strings.TrimSpace(driverCtx.Config["func_name"]))
	return funcName == "read" || funcName == "collect"
}

// DriverPoint 驱动测点数据
type DriverPoint struct {
	FieldName string      `json:"field_name"`
	Value     interface{} `json:"value"` // 支持 string 或 float64
	RW        string      `json:"rw"`    // "R" | "W" | "RW"
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

// SerialPort 串口接口
type SerialPort interface {
	Write([]byte) (int, error)
	Read([]byte) (int, error)
	Close() error
}

// WasmDriver WASM驱动实现
type WasmDriver struct {
	ID         int64
	Name       string
	plugin     *extism.Plugin
	mu         sync.RWMutex
	lastActive time.Time
	config     string
	resourceID int64 // 关联的串口资源ID
}

// DriverManager 驱动管理器
type DriverManager struct {
	mu          sync.RWMutex
	drivers     map[int64]*WasmDriver
	executor    *DriverExecutor // 引用执行器以访问串口
	callTimeout time.Duration
}

// NewDriverManager 创建驱动管理器
func NewDriverManager() *DriverManager {
	extism.SetLogLevel(extism.LogLevelError)
	return &DriverManager{
		drivers: make(map[int64]*WasmDriver),
	}
}

// SetExecutor 设置驱动执行器（用于访问串口资源）
func (m *DriverManager) SetExecutor(executor *DriverExecutor) {
	m.executor = executor
}

// SetCallTimeout 设置驱动执行超时时间（为0表示不设置超时）
func (m *DriverManager) SetCallTimeout(timeout time.Duration) {
	m.mu.Lock()
	m.callTimeout = timeout
	m.mu.Unlock()
}

// createHostFunctions 创建 Host Functions
func (m *DriverManager) createHostFunctions(resourceID int64) []extism.HostFunction {
	executor := m.executor
	if executor == nil {
		return nil
	}
	funcs := make([]extism.HostFunction, 0, 5)
	funcs = append(funcs, m.createSerialHostFunctions(resourceID)...)
	funcs = append(funcs, m.createTCPHostFunctions(resourceID)...)
	return funcs
}

// LoadDriver 加载驱动
func (m *DriverManager) LoadDriver(driver *models.Driver, wasmData []byte, resourceID int64) error {
	if driver == nil {
		return fmt.Errorf("driver is nil")
	}
	if len(wasmData) == 0 {
		return fmt.Errorf("driver wasm is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.drivers[driver.ID]; exists {
		return fmt.Errorf("driver %d already loaded", driver.ID)
	}

	// 从 driver config 中解析 resource_id (如果没有传入)
	if resourceID == 0 {
		resourceID = parseDriverResourceID(driver.ConfigSchema)
	}

	// 创建设置配置 (传递给插件)
	config := map[string]string{
		"resource_id": fmt.Sprintf("%d", resourceID),
	}

	hostFuncs := m.createHostFunctions(resourceID)
	plugin, err := newWasmPlugin(driver.Name, wasmData, hostFuncs, config)
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	wasmDriver := &WasmDriver{
		ID:         driver.ID,
		Name:       driver.Name,
		plugin:     plugin,
		lastActive: time.Now(),
		config:     driver.ConfigSchema,
		resourceID: resourceID,
	}

	m.drivers[driver.ID] = wasmDriver

	if mod := plugin.Module(); mod != nil {
		exports := mod.ExportedFunctions()
		if len(exports) > 0 {
			names := make([]string, 0, len(exports))
			for name := range exports {
				names = append(names, name)
			}
			logger.Debug("Driver exports", "driver", driver.Name, "count", len(names), "functions", names)
		}
	}
	return nil
}

// ReloadDriver 重载驱动
func (m *DriverManager) ReloadDriver(driver *models.Driver, wasmData []byte, resourceID int64) error {
	if driver == nil {
		return fmt.Errorf("driver is nil")
	}
	if m.IsLoaded(driver.ID) {
		if err := m.UnloadDriver(driver.ID); err != nil && !errors.Is(err, ErrDriverNotFound) {
			return err
		}
	}
	return m.LoadDriver(driver, wasmData, resourceID)
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
	if driver.plugin != nil {
		_ = driver.plugin.Close(context.Background())
	}

	delete(m.drivers, id)
	return nil
}

// ExecuteDriver 执行驱动
func (m *DriverManager) ExecuteDriver(id int64, function string, driverCtx *DriverContext) (*DriverResult, error) {
	return m.ExecuteDriverWithContext(context.Background(), id, function, driverCtx)
}

// ExecuteDriverWithContext 执行驱动（支持超时/取消）
func (m *DriverManager) ExecuteDriverWithContext(ctx context.Context, id int64, function string, driverCtx *DriverContext) (*DriverResult, error) {
	m.mu.RLock()
	driver, exists := m.drivers[id]
	timeout := m.callTimeout
	m.mu.RUnlock()

	if !exists {
		return nil, ErrDriverNotFound
	}

	driver.mu.Lock()
	driver.lastActive = time.Now()
	driver.mu.Unlock()

	if !driver.plugin.FunctionExists(function) {
		return nil, fmt.Errorf("plugin function not found: %s", function)
	}

	// 准备输入数据
	input := map[string]interface{}{
		"device_id":     driverCtx.DeviceID,
		"device_name":   driverCtx.DeviceName,
		"resource_id":   driverCtx.ResourceID,
		"resource_type": driverCtx.ResourceType,
		"config":        driverCtx.Config,
		"device_config": driverCtx.DeviceConfig,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// 调用插件函数
	rc, output, err := callPlugin(ctx, driver, function, inputJSON)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			return nil, fmt.Errorf("%w: %v", ErrDriverTimeout, err)
		case errors.Is(err, context.Canceled):
			return nil, fmt.Errorf("%w: %v", ErrDriverCanceled, err)
		case errors.Is(err, ErrPluginEmptyOutput):
			return nil, fmt.Errorf("%w: %v", ErrDriverBadOutput, err)
		}
		logger.Warn("Plugin call failed", "driver_id", id, "function", function, "error", err, "rc", rc, "input_len", len(inputJSON))
		return nil, fmt.Errorf("%w: %v", ErrDriverExecutionFailed, err)
	}

	if rc != 0 {
		logger.Warn("Plugin returned non-zero rc", "driver_id", id, "function", function, "rc", rc)
	}

	if isReadFunction(function, driverCtx) {
		logger.Info(
			"Driver read full output",
			"driver_id", id,
			"device_id", driverCtx.DeviceID,
			"device_name", driverCtx.DeviceName,
			"resource_id", driverCtx.ResourceID,
			"function", function,
			"output", string(output),
		)
	}

	// 解析输出
	var result DriverResult
	if err := json.Unmarshal(output, &result); err != nil {
		max := 512
		if len(output) < max {
			max = len(output)
		}
		logger.Warn("Failed to parse plugin output", "driver_id", id, "function", function, "output_preview", string(output[:max]))
		return nil, fmt.Errorf("%w: %v", ErrDriverBadOutput, err)
	}

	result.Timestamp = time.Now()
	return &result, nil
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

// GetDriverResourceID 获取驱动关联的资源ID
func (m *DriverManager) GetDriverResourceID(id int64) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	driver, exists := m.drivers[id]
	if !exists {
		return 0, ErrDriverNotFound
	}

	return driver.resourceID, nil
}

// LoadDriverFromModel 从驱动模型加载或重载驱动
func (m *DriverManager) LoadDriverFromModel(driver *models.Driver, resourceID int64) error {
	if driver == nil {
		return fmt.Errorf("driver is nil")
	}
	if resourceID == 0 {
		resourceID = parseDriverResourceID(driver.ConfigSchema)
	}
	if driver.FilePath == "" {
		return fmt.Errorf("driver file path is empty")
	}
	wasmData, err := readWasmFile(driver.FilePath)
	if err != nil {
		return fmt.Errorf("read driver wasm failed: %w", err)
	}
	if err := m.ReloadDriver(driver, wasmData, resourceID); err != nil {
		return fmt.Errorf("reload driver failed: %w", err)
	}
	return nil
}

// DriverRuntime 驱动运行时信息
type DriverRuntime struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	Loaded            bool      `json:"loaded"`
	ResourceID        int64     `json:"resource_id"`
	LastActive        time.Time `json:"last_active"`
	ExportedFunctions []string  `json:"exported_functions,omitempty"`
}

func runtimeFromDriver(driver *WasmDriver) *DriverRuntime {
	if driver == nil {
		return nil
	}

	driver.mu.RLock()
	runtime := &DriverRuntime{
		ID:         driver.ID,
		Name:       driver.Name,
		Loaded:     true,
		ResourceID: driver.resourceID,
		LastActive: driver.lastActive,
	}
	if driver.plugin != nil {
		if mod := driver.plugin.Module(); mod != nil {
			exports := mod.ExportedFunctions()
			if len(exports) > 0 {
				names := make([]string, 0, len(exports))
				for name := range exports {
					names = append(names, name)
				}
				sort.Strings(names)
				runtime.ExportedFunctions = names
			}
		}
	}
	driver.mu.RUnlock()

	return runtime
}

// GetRuntime 获取单个驱动运行时信息
func (m *DriverManager) GetRuntime(id int64) (*DriverRuntime, error) {
	m.mu.RLock()
	driver, exists := m.drivers[id]
	m.mu.RUnlock()
	if !exists {
		return nil, ErrDriverNotLoaded
	}
	return runtimeFromDriver(driver), nil
}

// ListRuntimes 获取所有已加载驱动运行态
func (m *DriverManager) ListRuntimes() []*DriverRuntime {
	m.mu.RLock()
	drivers := make([]*WasmDriver, 0, len(m.drivers))
	for _, d := range m.drivers {
		drivers = append(drivers, d)
	}
	m.mu.RUnlock()

	runtimes := make([]*DriverRuntime, 0, len(drivers))
	for _, d := range drivers {
		if rt := runtimeFromDriver(d); rt != nil {
			runtimes = append(runtimes, rt)
		}
	}
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].ID < runtimes[j].ID })

	return runtimes
}
