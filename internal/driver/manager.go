package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/gonglijing/xunjiFsu/internal/models"
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

	readWithTimeout := func(port SerialPort, buf []byte, expect int, timeout time.Duration) (int, error) {
		deadline := time.Now().Add(timeout)
		read := 0
		tmp := make([]byte, expect)
		for read < expect && time.Now().Before(deadline) {
			n, err := port.Read(tmp)
			if n > 0 {
				copy(buf[read:], tmp[:n])
				read += n
			}
			if err != nil {
				return read, err
			}
			if read >= expect {
				break
			}
			time.Sleep(2 * time.Millisecond)
		}
		if read < expect {
			return read, fmt.Errorf("timeout")
		}
		return read, nil
	}

	// serial_read: 从串口读取数据
	serialRead := extism.NewHostFunctionWithStack(
		"serial_read",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			size := int32(stack[1]) // 读取请求的大小
			buf := make([]byte, size)

			port := executor.GetSerialPort(resourceID)
			if port == nil {
				stack[0] = 0 // 返回 0 表示失败
				return
			}

			n, err := port.Read(buf)
			if err != nil {
				stack[0] = 0
				return
			}

			// 将数据写入插件内存
			p.Memory().Write(uint32(stack[0]), buf[:n])
			stack[0] = uint64(n) // 返回实际读取的字节数
		},
		[]extism.ValueType{extism.ValueTypeI32, extism.ValueTypeI32},
		[]extism.ValueType{extism.ValueTypeI32},
	)

	// serial_write: 向串口写入数据
	serialWrite := extism.NewHostFunctionWithStack(
		"serial_write",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ptr := int32(stack[0])
			size := int32(stack[1])

			// 从插件内存读取数据
			data, _ := p.Memory().Read(uint32(ptr), uint32(size))

			port := executor.GetSerialPort(resourceID)
			if port == nil {
				stack[0] = 0
				return
			}

			n, err := port.Write(data)
			if err != nil {
				stack[0] = 0
				return
			}

			stack[0] = uint64(n) // 返回实际写入的字节数
		},
		[]extism.ValueType{extism.ValueTypeI32, extism.ValueTypeI32},
		[]extism.ValueType{extism.ValueTypeI32},
	)

	// serial_transceive: 先写再读（用于半双工协议，如自定义 RTU）
	serialTransceive := extism.NewHostFunctionWithStack(
		"serial_transceive",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			writePtr := uint32(stack[0])
			writeSize := int(stack[1])
			readPtr := uint32(stack[2])
			readCap := int(stack[3])
			timeoutMs := int(stack[4])

			port := executor.GetSerialPort(resourceID)
			if port == nil || writeSize <= 0 || readCap <= 0 {
				stack[0] = 0
				return
			}

			req, _ := p.Memory().Read(writePtr, uint32(writeSize))
			if _, err := port.Write(req); err != nil {
				stack[0] = 0
				return
			}

			buf := make([]byte, readCap)
			tout := time.Duration(timeoutMs)
			if tout <= 0 {
				tout = 500 * time.Millisecond
			}
			n, err := readWithTimeout(port, buf, readCap, tout)
			if err != nil || n == 0 {
				stack[0] = 0
				return
			}
			p.Memory().Write(readPtr, buf[:n])
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI32, // write ptr
			extism.ValueTypeI32, // write size
			extism.ValueTypeI32, // read ptr
			extism.ValueTypeI32, // read cap
			extism.ValueTypeI32, // timeout ms
		},
		[]extism.ValueType{extism.ValueTypeI32}, // bytes read
	)

	// sleep_ms: 毫秒延时
	sleepMs := extism.NewHostFunctionWithStack(
		"sleep_ms",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ms := int(stack[0])
			time.Sleep(time.Duration(ms) * time.Millisecond)
		},
		[]extism.ValueType{extism.ValueTypeI32},
		[]extism.ValueType{},
	)

	// output: 输出日志
	outputLog := extism.NewHostFunctionWithStack(
		"output",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ptr := int32(stack[0])
			size := int32(stack[1])

			// 从插件内存读取数据
			data, _ := p.Memory().Read(uint32(ptr), uint32(size))
			fmt.Printf("[Driver] %s\n", string(data))
		},
		[]extism.ValueType{extism.ValueTypeI32, extism.ValueTypeI32},
		[]extism.ValueType{},
	)

	// tcp_read: 低层 TCP 读取（基于已注册的 resourceID 连接）
	tcpRead := extism.NewHostFunctionWithStack(
		"tcp_read",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			bufPtr := uint32(stack[0])
			bufCap := int(stack[1])
			timeoutMs := int(stack[2])

			conn := executor.GetTCPConn(resourceID)
			if conn == nil || bufCap <= 0 {
				stack[0] = 0
				return
			}
			tout := time.Duration(timeoutMs)
			if tout <= 0 {
				tout = 500 * time.Millisecond
			}
			_ = conn.SetReadDeadline(time.Now().Add(tout))
			buf := make([]byte, bufCap)
			n, err := conn.Read(buf)
			if err != nil || n <= 0 {
				stack[0] = 0
				return
			}
			p.Memory().Write(bufPtr, buf[:n])
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI32, // buf ptr
			extism.ValueTypeI32, // buf cap
			extism.ValueTypeI32, // timeout ms
		},
		[]extism.ValueType{extism.ValueTypeI32},
	)

	// tcp_write: 低层 TCP 写
	tcpWrite := extism.NewHostFunctionWithStack(
		"tcp_write",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ptr := uint32(stack[0])
			size := int(stack[1])
			conn := executor.GetTCPConn(resourceID)
			if conn == nil || size <= 0 {
				stack[0] = 0
				return
			}
			data, _ := p.Memory().Read(ptr, uint32(size))
			n, err := conn.Write(data)
			if err != nil {
				stack[0] = 0
				return
			}
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI32, // ptr
			extism.ValueTypeI32, // size
		},
		[]extism.ValueType{extism.ValueTypeI32},
	)

	return []extism.HostFunction{serialRead, serialWrite, serialTransceive, sleepMs, outputLog, tcpRead, tcpWrite}
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

	// 创建 Extism plugin (包含 Host Functions)
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			&extism.WasmData{
				Name: driver.Name,
				Data: wasmData,
			},
		},
	}

	// 创建设置配置 (传递给插件)
	config := map[string]string{
		"resource_id": fmt.Sprintf("%d", resourceID),
	}

	ctx := context.Background()
	hostFuncs := m.createHostFunctions(resourceID)

	plugin, err := extism.NewPlugin(ctx, manifest, extism.PluginConfig{
		EnableWasi: true,
	}, hostFuncs)
	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	// 设置配置
	plugin.Config = config

	wasmDriver := &WasmDriver{
		ID:         driver.ID,
		Name:       driver.Name,
		plugin:     plugin,
		lastActive: time.Now(),
		config:     driver.ConfigSchema,
		resourceID: resourceID,
	}

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
	if driver.plugin != nil {
		driver.plugin.Close()
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
	_, output, err := driver.plugin.Call(function, inputJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to call plugin: %w", err)
	}

	// 解析输出
	var result DriverResult
	if err := json.Unmarshal(output, &result); err != nil {
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
	manager     *DriverManager
	serialPorts map[int64]SerialPort // 资源ID到串口的映射
	tcpConns    map[int64]net.Conn   // 资源ID到TCP连接
	mu          sync.RWMutex
	executing   map[int64]bool
	resourceMux map[int64]*sync.Mutex // 同一资源串口的互斥锁，避免并发读写
}

// NewDriverExecutor 创建驱动执行器
func NewDriverExecutor(manager *DriverManager) *DriverExecutor {
	executor := &DriverExecutor{
		manager:     manager,
		serialPorts: make(map[int64]SerialPort),
		tcpConns:    make(map[int64]net.Conn),
		executing:   make(map[int64]bool),
		resourceMux: make(map[int64]*sync.Mutex),
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
	delete(e.serialPorts, resourceID)
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
		c.Close()
		delete(e.tcpConns, resourceID)
	}
}

// GetSerialPort 获取串口
func (e *DriverExecutor) GetSerialPort(resourceID int64) SerialPort {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.serialPorts[resourceID]
}

// GetTCPConn 获取 TCP 连接
func (e *DriverExecutor) GetTCPConn(resourceID int64) net.Conn {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.tcpConns[resourceID]
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

	// 构建设备配置
	resourceID := int64(0)
	if device.ResourceID != nil {
		resourceID = *device.ResourceID
	}
	resourceType := device.ResourceType
	if resourceType == "" {
		resourceType = device.DriverType
		if resourceType == "" {
			resourceType = "modbus_rtu"
		}
	}

	// 构建 DeviceConfig 从接口字段
	deviceConfig := make(map[string]string)
	if device.DriverType == "modbus_rtu" {
		deviceConfig["serial_port"] = device.SerialPort
		deviceConfig["baud_rate"] = fmt.Sprintf("%d", device.BaudRate)
		deviceConfig["data_bits"] = fmt.Sprintf("%d", device.DataBits)
		deviceConfig["stop_bits"] = fmt.Sprintf("%d", device.StopBits)
		deviceConfig["parity"] = device.Parity
	} else {
		deviceConfig["ip_address"] = device.IPAddress
		deviceConfig["port_num"] = fmt.Sprintf("%d", device.PortNum)
	}
	if device.DeviceAddress != "" {
		deviceConfig["device_address"] = device.DeviceAddress
	}
	deviceConfig["func_name"] = "read"

	ctx := &DriverContext{
		DeviceID:     device.ID,
		DeviceName:   device.Name,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Config:       deviceConfig,
		DeviceConfig: "",
	}

	// 对同一资源做互斥，避免并发串口访问
	var unlock func()
	if resourceID > 0 {
		lock := e.getResourceLock(resourceID)
		lock.Lock()
		unlock = lock.Unlock
	}
	if unlock != nil {
		defer unlock()
	}

	return e.manager.ExecuteDriver(*device.DriverID, "collect", ctx)
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
