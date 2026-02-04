package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
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

// readWasmFile 尝试读取 wasm 文件，支持相对路径的多种基准
func readWasmFile(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty wasm path")
	}
	// 1) 绝对路径或直接相对当前工作目录
	if data, err := os.ReadFile(path); err == nil {
		return data, nil
	}
	// 2) 以当前工作目录拼接
	if cwd, err := os.Getwd(); err == nil {
		if data, err := os.ReadFile(filepath.Join(cwd, path)); err == nil {
			return data, nil
		}
	}
	// 3) 以可执行文件目录为基准
	if exePath, err := os.Executable(); err == nil {
		base := filepath.Dir(exePath)
		if data, err := os.ReadFile(filepath.Join(base, path)); err == nil {
			return data, nil
		}
		// 4) 上一级（常见于 go run 临时目录）
		if data, err := os.ReadFile(filepath.Join(base, "..", path)); err == nil {
			return data, nil
		}
	}
	return nil, fmt.Errorf("cannot read wasm file: %s", path)
}

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
			ptr := stack[0]
			size := int(stack[1]) // 读取请求的大小
			if ptr == 0 || size <= 0 || ptr > uint64(^uint32(0)) {
				stack[0] = 0 // 返回 0 表示失败
				return
			}
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
			p.Memory().Write(uint32(ptr), buf[:n])
			stack[0] = uint64(n) // 返回实际读取的字节数
		},
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)

	// serial_write: 向串口写入数据
	serialWrite := extism.NewHostFunctionWithStack(
		"serial_write",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ptr := stack[0]
			size := int(stack[1])
			if ptr == 0 || size <= 0 || ptr > uint64(^uint32(0)) {
				stack[0] = 0
				return
			}

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
		[]extism.ValueType{extism.ValueTypeI64, extism.ValueTypeI64},
		[]extism.ValueType{extism.ValueTypeI64},
	)

	// serial_transceive: 先写再读（用于半双工协议，如自定义 RTU）
	serialTransceive := extism.NewHostFunctionWithStack(
		"serial_transceive",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			writePtr := stack[0]
			writeSize := int(stack[1])
			readPtr := stack[2]
			readCap := int(stack[3])
			timeoutMs := int(stack[4])

			port := executor.GetSerialPort(resourceID)
			if port == nil || writeSize <= 0 || readCap <= 0 {
				if port == nil {
					logger.Warn("Serial port not found", "resource_id", resourceID)
				}
				stack[0] = 0
				return
			}
			if writePtr == 0 || readPtr == 0 || writePtr > uint64(^uint32(0)) || readPtr > uint64(^uint32(0)) {
				stack[0] = 0
				return
			}

			if resetIn, ok := port.(interface{ ResetInputBuffer() error }); ok {
				_ = resetIn.ResetInputBuffer()
			}
			if resetOut, ok := port.(interface{ ResetOutputBuffer() error }); ok {
				_ = resetOut.ResetOutputBuffer()
			}

			if rts, ok := port.(interface{ SetRTS(bool) error }); ok {
				_ = rts.SetRTS(true)
			}
			if dtr, ok := port.(interface{ SetDTR(bool) error }); ok {
				_ = dtr.SetDTR(true)
			}

			req, _ := p.Memory().Read(uint32(writePtr), uint32(writeSize))
			if n, err := port.Write(req); err != nil {
				logger.Warn("Serial write failed", "resource_id", resourceID, "error", err)
				stack[0] = 0
				return
			} else {
				logger.Info("Serial write", "resource_id", resourceID, "written", n, "len", len(req), "req", hexPreview(req, 32))
			}

			if rts, ok := port.(interface{ SetRTS(bool) error }); ok {
				_ = rts.SetRTS(false)
			}
			if dtr, ok := port.(interface{ SetDTR(bool) error }); ok {
				_ = dtr.SetDTR(false)
			}
			time.Sleep(3 * time.Millisecond)

			buf := make([]byte, readCap)
			tout := time.Duration(timeoutMs)
			if tout <= 0 {
				tout = 500 * time.Millisecond
			}
			n, err := readWithTimeout(port, buf, readCap, tout)
			if n == 0 {
				if err != nil {
					logger.Warn("Serial read failed", "resource_id", resourceID, "error", err.Error(), "req", hexPreview(req, 32))
				} else {
					logger.Warn("Serial read timeout", "resource_id", resourceID, "req", hexPreview(req, 32))
				}
				stack[0] = 0
				return
			}
			if err != nil {
				logger.Warn("Serial read partial", "resource_id", resourceID, "read", n, "error", err.Error())
			}
			p.Memory().Write(uint32(readPtr), buf[:n])
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI64, // write ptr
			extism.ValueTypeI64, // write size
			extism.ValueTypeI64, // read ptr
			extism.ValueTypeI64, // read cap
			extism.ValueTypeI64, // timeout ms
		},
		[]extism.ValueType{extism.ValueTypeI64}, // bytes read
	)

	// sleep_ms: 毫秒延时
	sleepMs := extism.NewHostFunctionWithStack(
		"sleep_ms",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			ms := int(stack[0])
			time.Sleep(time.Duration(ms) * time.Millisecond)
		},
		[]extism.ValueType{extism.ValueTypeI64},
		[]extism.ValueType{},
	)

	// tcp_transceive: 写后读（与 serial_transceive 对齐）
	tcpTransceive := extism.NewHostFunctionWithStack(
		"tcp_transceive",
		func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
			wPtr := stack[0]
			wSize := int(stack[1])
			rPtr := stack[2]
			rCap := int(stack[3])
			timeoutMs := int(stack[4])

			conn := executor.GetTCPConn(resourceID)
			if conn == nil || wSize <= 0 || rCap <= 0 {
				stack[0] = 0
				return
			}
			if wPtr == 0 || rPtr == 0 || wPtr > uint64(^uint32(0)) || rPtr > uint64(^uint32(0)) {
				stack[0] = 0
				return
			}

			req, _ := p.Memory().Read(uint32(wPtr), uint32(wSize))
			if _, err := conn.Write(req); err != nil {
				stack[0] = 0
				return
			}

			tout := time.Duration(timeoutMs)
			if tout <= 0 {
				tout = 500 * time.Millisecond
			}
			_ = conn.SetReadDeadline(time.Now().Add(tout))
			buf := make([]byte, rCap)
			n, err := conn.Read(buf)
			if err != nil || n <= 0 {
				stack[0] = 0
				return
			}
			p.Memory().Write(uint32(rPtr), buf[:n])
			stack[0] = uint64(n)
		},
		[]extism.ValueType{
			extism.ValueTypeI64, // wPtr
			extism.ValueTypeI64, // wSize
			extism.ValueTypeI64, // rPtr
			extism.ValueTypeI64, // rCap
			extism.ValueTypeI64, // timeout
		},
		[]extism.ValueType{extism.ValueTypeI64},
	)

	return []extism.HostFunction{serialRead, serialWrite, serialTransceive, sleepMs, tcpTransceive}
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

	plugin.SetLogger(func(level extism.LogLevel, message string) {
		switch level {
		case extism.LogLevelError:
			logger.Error("Driver log", fmt.Errorf(message), "driver", driver.Name)
		case extism.LogLevelWarn:
			logger.Warn("Driver log", "driver", driver.Name, "message", message)
		default:
			logger.Info("Driver log", "driver", driver.Name, "level", level.String(), "message", message)
		}
	})

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
	delete(e.serialPorts, resourceID)
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
	e.resourcePaths[resourceID] = path
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

	// 对于网络类型设备，设置资源路径（用于TCP懒连接）
	if resourceType == "net" && resourceID > 0 {
		// 如果还没有缓存路径，尝试从设备配置或资源获取
		if e.GetResourcePath(resourceID) == "" && device.IPAddress != "" {
			path := fmt.Sprintf("%s:%d", device.IPAddress, device.PortNum)
			e.SetResourcePath(resourceID, path)
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

	// 串口资源自动打开
	if resourceID > 0 {
		if err := e.ensureSerialPort(resourceID, device); err != nil && resourceType == "serial" {
			return nil, fmt.Errorf("open serial resource %d failed: %w", resourceID, err)
		}
	}

	// 若驱动未加载，尝试按需加载
	if e.manager.IsLoaded(*device.DriverID) {
		if loaded, err := e.manager.GetDriver(*device.DriverID); err == nil && loaded != nil {
			if resourceID > 0 && loaded.resourceID != resourceID {
				logger.Warn("Reloading driver with new resource", "driver_id", loaded.ID, "old_resource_id", loaded.resourceID, "new_resource_id", resourceID)
				_ = e.manager.UnloadDriver(loaded.ID)
			}
		}
	}
	if !e.manager.IsLoaded(*device.DriverID) {
		if drv, err := database.GetDriverByID(*device.DriverID); err == nil && drv != nil && drv.FilePath != "" {
			if wasmData, err := readWasmFile(drv.FilePath); err == nil {
				if err := e.manager.LoadDriver(drv, wasmData, resourceID); err != nil {
					return nil, fmt.Errorf("load driver %d failed: %w", drv.ID, err)
				}
			} else {
				return nil, fmt.Errorf("read driver wasm failed: %w", err)
			}
		} else {
			return nil, ErrDriverNotFound
		}
	}

	// 默认入口函数名（TinyGo 驱动导出为 handle）
	return e.manager.ExecuteDriver(*device.DriverID, "handle", ctx)
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
