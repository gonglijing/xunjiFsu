package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"go.bug.st/serial"
)

// ErrDriverNotFound 驱动未找到
var ErrDriverNotFound = errors.New("driver not found")

// ErrDriverNotLoaded 驱动未加载
var ErrDriverNotLoaded = errors.New("driver not loaded")

// ErrDriverExecutionFailed 驱动执行失败
var ErrDriverExecutionFailed = errors.New("driver execution failed")

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

// DriverPoint 驱动测点数据
type DriverPoint struct {
	FieldName string  `json:"field_name"`
	Value     float64 `json:"value"`
	RW        string  `json:"rw"` // "R" | "W" | "RW"
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
	mu       sync.RWMutex
	drivers  map[int64]*WasmDriver
	executor *DriverExecutor // 引用执行器以访问串口
}

// NewDriverManager 创建驱动管理器
func NewDriverManager() *DriverManager {
	extism.SetLogLevel(extism.LogLevelDebug)
	return &DriverManager{
		drivers: make(map[int64]*WasmDriver),
	}
}

// SetExecutor 设置驱动执行器（用于访问串口资源）
func (m *DriverManager) SetExecutor(executor *DriverExecutor) {
	m.executor = executor
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
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.drivers[driver.ID]; exists {
		return fmt.Errorf("driver %d already loaded", driver.ID)
	}

	// 从 driver config 中解析 resource_id (如果没有传入)
	if resourceID == 0 && driver.ConfigSchema != "" {
		var cfg struct {
			ResourceID int64 `json:"resource_id"`
		}
		if err := json.Unmarshal([]byte(driver.ConfigSchema), &cfg); err == nil {
			resourceID = cfg.ResourceID
		}
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
	m.mu.RLock()
	driver, exists := m.drivers[id]
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

	// 调用插件函数
	rc, output, err := driver.plugin.Call(function, inputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to call plugin: %w", err)
	}

	if rc != 0 {
		logger.Warn("Plugin returned non-zero rc", "driver_id", id, "function", function, "rc", rc)
	}

	// 若直接返回为空，尝试从 runtime 读取输出寄存
	if len(output) == 0 {
		if alt, err2 := driver.plugin.GetOutput(); err2 == nil && len(alt) > 0 {
			output = alt
		} else {
			errMsg := driver.plugin.GetError()
			if errMsg != "" {
				logger.Warn("Plugin error set", "driver_id", id, "function", function, "error", errMsg)
			}
			logger.Warn("Plugin returned empty output", "driver_id", id, "function", function, "rc", rc, "input_len", len(inputJSON))
			return nil, fmt.Errorf("plugin returned empty output (function=%s)", function)
		}
	}

	// 解析输出
	var result DriverResult
	if err := json.Unmarshal(output, &result); err != nil {
		max := 512
		if len(output) < max {
			max = len(output)
		}
		logger.Warn("Failed to parse plugin output", "driver_id", id, "function", function, "output_preview", string(output[:max]))
		return nil, fmt.Errorf("failed to parse output: %w", err)
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

// DriverExecutor 驱动执行器
type DriverExecutor struct {
	manager       *DriverManager
	serialPorts   map[int64]SerialPort // 资源ID到串口的映射
	tcpConns      map[int64]net.Conn   // 资源ID到TCP连接
	resourcePaths map[int64]string     // 资源ID到路径的映射 (用于TCP懒连接)
	mu            sync.RWMutex
	executing     map[int64]bool
	resourceMux   map[int64]*sync.Mutex // 同一资源串口的互斥锁，避免并发读写
}

// NewDriverExecutor 创建驱动执行器
func NewDriverExecutor(manager *DriverManager) *DriverExecutor {
	executor := &DriverExecutor{
		manager:       manager,
		serialPorts:   make(map[int64]SerialPort),
		tcpConns:      make(map[int64]net.Conn),
		resourcePaths: make(map[int64]string),
		executing:     make(map[int64]bool),
		resourceMux:   make(map[int64]*sync.Mutex),
	}

	// 双向绑定
	if manager != nil {
		manager.SetExecutor(executor)
	}

	return executor
}

// RegisterSerialPort 注册串口
func (e *DriverExecutor) RegisterSerialPort(resourceID int64, port SerialPort) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.serialPorts[resourceID] = port
}

// UnregisterSerialPort 取消注册串口
func (e *DriverExecutor) UnregisterSerialPort(resourceID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if port, ok := e.serialPorts[resourceID]; ok {
		_ = port.Close()
		delete(e.serialPorts, resourceID)
	}
}

func (e *DriverExecutor) ensureSerialPort(resourceID int64, device *models.Device) error {
	if resourceID <= 0 {
		return fmt.Errorf("invalid resource id")
	}
	if e.GetSerialPort(resourceID) != nil {
		return nil
	}
	res, err := database.GetResourceByID(resourceID)
	if err != nil {
		return fmt.Errorf("get resource %d failed: %w", resourceID, err)
	}
	if res.Type != "serial" {
		return fmt.Errorf("resource %d type %s is not serial", resourceID, res.Type)
	}
	if res.Path == "" {
		return fmt.Errorf("resource %d path is empty", resourceID)
	}

	baud := device.BaudRate
	if baud == 0 {
		baud = 9600
	}
	dataBits := device.DataBits
	if dataBits < 5 || dataBits > 8 {
		dataBits = 8
	}

	parity := serial.NoParity
	switch strings.ToUpper(strings.TrimSpace(device.Parity)) {
	case "E", "EVEN":
		parity = serial.EvenParity
	case "O", "ODD":
		parity = serial.OddParity
	}

	stopBits := serial.OneStopBit
	if device.StopBits == 2 {
		stopBits = serial.TwoStopBits
	}

	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: dataBits,
		Parity:   parity,
		StopBits: stopBits,
	}
	port, err := serial.Open(res.Path, mode)
	if err != nil {
		return fmt.Errorf("open serial %s failed: %w", res.Path, err)
	}
	if setter, ok := port.(interface{ SetReadTimeout(time.Duration) error }); ok {
		_ = setter.SetReadTimeout(200 * time.Millisecond)
	}
	e.RegisterSerialPort(resourceID, port)
	logger.Info("Serial port opened", "resource_id", resourceID, "path", res.Path, "baud", baud, "data_bits", dataBits, "stop_bits", device.StopBits, "parity", device.Parity)
	return nil
}

// RegisterTCP 注册 TCP 连接
func (e *DriverExecutor) RegisterTCP(resourceID int64, conn net.Conn) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tcpConns[resourceID] = conn
}

// UnregisterTCP 取消注册 TCP 连接
func (e *DriverExecutor) UnregisterTCP(resourceID int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if c, ok := e.tcpConns[resourceID]; ok {
		_ = c.Close()
		delete(e.tcpConns, resourceID)
	}
}

// GetSerialPort 获取串口
func (e *DriverExecutor) GetSerialPort(resourceID int64) SerialPort {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.serialPorts[resourceID]
}

// GetTCPConn 获取 TCP 连接 (懒加载)
func (e *DriverExecutor) GetTCPConn(resourceID int64) net.Conn {
	// 先尝试获取现有连接
	e.mu.RLock()
	conn, exists := e.tcpConns[resourceID]
	e.mu.RUnlock()

	if exists && conn != nil {
		// 检查连接是否仍然有效
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			if err := tcpConn.SetReadDeadline(time.Now()); err == nil {
				// 测试连接是否存活
				one := []byte{0}
				_, err := tcpConn.Write(one)
				if err == nil {
					return conn
				}
			}
		}
		// 连接无效，关闭并删除
		e.mu.Lock()
		delete(e.tcpConns, resourceID)
		e.mu.Unlock()
	}

	// 获取路径信息
	e.mu.RLock()
	path, hasPath := e.resourcePaths[resourceID]
	e.mu.RUnlock()

	if !hasPath || path == "" {
		return nil
	}

	// 建立新连接
	e.mu.Lock()
	// 双重检查
	if existingConn, ok := e.tcpConns[resourceID]; ok && existingConn != nil {
		e.mu.Unlock()
		return existingConn
	}

	conn, err := net.DialTimeout("tcp", path, 5*time.Second)
	if err != nil {
		e.mu.Unlock()
		return nil
	}

	// 设置连接参数
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	e.tcpConns[resourceID] = conn
	e.mu.Unlock()

	return conn
}

// SetResourcePath 设置资源路径 (用于TCP懒连接)
func (e *DriverExecutor) SetResourcePath(resourceID int64, path string) {
	e.mu.Lock()
	if prev, ok := e.resourcePaths[resourceID]; ok && prev != "" && prev != path {
		if conn, ok := e.tcpConns[resourceID]; ok {
			_ = conn.Close()
			delete(e.tcpConns, resourceID)
		}
	}
	e.resourcePaths[resourceID] = path
	e.mu.Unlock()
}

// CloseResource 关闭指定资源相关的连接和缓存
func (e *DriverExecutor) CloseResource(resourceID int64) {
	e.mu.Lock()
	if port, ok := e.serialPorts[resourceID]; ok {
		_ = port.Close()
		delete(e.serialPorts, resourceID)
	}
	if conn, ok := e.tcpConns[resourceID]; ok {
		_ = conn.Close()
		delete(e.tcpConns, resourceID)
	}
	delete(e.resourcePaths, resourceID)
	delete(e.resourceMux, resourceID)
	e.mu.Unlock()
}

// CloseAllResources 关闭所有资源连接
func (e *DriverExecutor) CloseAllResources() {
	e.mu.Lock()
	for id, port := range e.serialPorts {
		_ = port.Close()
		delete(e.serialPorts, id)
	}
	for id, conn := range e.tcpConns {
		_ = conn.Close()
		delete(e.tcpConns, id)
	}
	e.resourcePaths = make(map[int64]string)
	e.resourceMux = make(map[int64]*sync.Mutex)
	e.mu.Unlock()
}

// GetResourcePath 获取资源路径
func (e *DriverExecutor) GetResourcePath(resourceID int64) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.resourcePaths[resourceID]
}

// getResourceLock 返回资源级互斥锁（懒创建）
func (e *DriverExecutor) getResourceLock(resourceID int64) *sync.Mutex {
	e.mu.Lock()
	defer e.mu.Unlock()
	if l, ok := e.resourceMux[resourceID]; ok {
		return l
	}
	l := &sync.Mutex{}
	e.resourceMux[resourceID] = l
	return l
}

// Execute 执行驱动读取（带资源访问锁）
func (e *DriverExecutor) Execute(device *models.Device) (*DriverResult, error) {
	if device.DriverID == nil {
		return nil, fmt.Errorf("device %s has no driver", device.Name)
	}
	done, err := e.startExecution(device)
	if err != nil {
		return nil, err
	}
	defer done()

	resourceID, resourceType := resolveResource(device)
	e.ensureResourcePath(resourceID, resourceType, device)

	deviceConfig := buildDeviceConfig(device)
	ctx := buildDriverContext(device, resourceID, resourceType, deviceConfig)

	unlock := e.lockResource(resourceID)
	if unlock != nil {
		defer unlock()
	}

	if err := e.ensureSerialResource(resourceID, resourceType, device); err != nil {
		return nil, err
	}

	if err := e.ensureDriverLoaded(device, resourceID); err != nil {
		return nil, err
	}

	// 默认入口函数名（TinyGo 驱动导出为 handle）
	return e.manager.ExecuteDriver(*device.DriverID, defaultDriverFunction, ctx)
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

	// 解析返回数据
	fields := make(map[string]string)

	// 优先使用新格式 (points)
	if len(result.Points) > 0 {
		for _, point := range result.Points {
			fields[point.FieldName] = fmt.Sprintf("%.6f", point.Value)
		}
	} else {
		// 兼容旧格式
		for k, v := range result.Data {
			fields[k] = v
		}
	}

	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		Timestamp:  result.Timestamp,
		Fields:     fields,
	}, nil
}
