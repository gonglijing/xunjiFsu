package driver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// DriverExecutor 驱动执行器
type DriverExecutor struct {
	manager                   *DriverManager
	serialPorts               map[int64]SerialPort // 资源ID到串口的映射
	tcpConns                  map[int64]net.Conn   // 资源ID到TCP连接
	resourcePaths             map[int64]string     // 资源ID到路径的映射 (用于TCP懒连接)
	mu                        sync.RWMutex
	executing                 map[int64]bool
	resourceMux               map[int64]*sync.Mutex // 同一资源串口的互斥锁，避免并发读写
	serialTimeout             time.Duration
	serialOpenRetries         int
	serialOpenBackoffOverride time.Duration
	tcpDialTimeoutOverride    time.Duration
	tcpDialRetries            int
	tcpDialBackoffOverride    time.Duration
	tcpReadTimeoutOverride    time.Duration
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

	parity := strings.ToUpper(strings.TrimSpace(device.Parity))
	if parity == "" {
		parity = "N"
	}

	stopBits := device.StopBits
	if stopBits != 2 {
		stopBits = 1
	}

	mode := serialOpenMode{
		BaudRate: baud,
		DataBits: dataBits,
		Parity:   parity,
		StopBits: stopBits,
	}
	attempts := e.serialOpenAttempts()
	backoff := e.serialOpenBackoff()
	var port SerialPort
	var openErr error
	for i := 0; i < attempts; i++ {
		port, openErr = openSerialPort(res.Path, mode)
		if openErr == nil {
			break
		}
		if i < attempts-1 {
			time.Sleep(backoff)
		}
	}
	if openErr != nil {
		return fmt.Errorf("open serial %s failed: %w", res.Path, openErr)
	}
	if setter, ok := port.(interface{ SetReadTimeout(time.Duration) error }); ok {
		_ = setter.SetReadTimeout(e.serialReadTimeout())
	}
	e.RegisterSerialPort(resourceID, port)
	logger.Info("Serial port opened", "resource_id", resourceID, "path", res.Path, "baud", baud, "data_bits", dataBits, "stop_bits", stopBits, "parity", parity)
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
		return conn
	}

	// 获取路径信息
	e.mu.RLock()
	path, hasPath := e.resourcePaths[resourceID]
	e.mu.RUnlock()

	if !hasPath || path == "" {
		return nil
	}

	dialTimeout := e.tcpDialTimeout()
	dialAttempts := e.tcpDialAttempts()
	dialBackoff := e.tcpDialBackoff()

	// 建立新连接
	var dialConn net.Conn
	var err error
	for i := 0; i < dialAttempts; i++ {
		dialConn, err = net.DialTimeout("tcp", path, dialTimeout)
		if err == nil {
			break
		}
		if i < dialAttempts-1 {
			time.Sleep(dialBackoff)
		}
	}
	if err != nil {
		return nil
	}

	// 设置连接参数
	if tcpConn, ok := dialConn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// 双重检查
	e.mu.Lock()
	if existingConn, ok := e.tcpConns[resourceID]; ok && existingConn != nil {
		e.mu.Unlock()
		_ = dialConn.Close()
		return existingConn
	}
	e.tcpConns[resourceID] = dialConn
	e.mu.Unlock()

	return dialConn
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

// SetTimeouts overrides default timeouts. Use zero to keep defaults.
func (e *DriverExecutor) SetTimeouts(serialRead, tcpDial, tcpRead time.Duration) {
	e.mu.Lock()
	e.serialTimeout = serialRead
	e.tcpDialTimeoutOverride = tcpDial
	e.tcpReadTimeoutOverride = tcpRead
	e.mu.Unlock()
}

// SetRetries overrides retry counts and backoffs. Use zero values to keep defaults.
func (e *DriverExecutor) SetRetries(serialOpen, tcpDial int, serialBackoff, tcpBackoff time.Duration) {
	if serialOpen < 0 {
		serialOpen = 0
	}
	if tcpDial < 0 {
		tcpDial = 0
	}
	e.mu.Lock()
	e.serialOpenRetries = serialOpen
	e.tcpDialRetries = tcpDial
	e.serialOpenBackoffOverride = serialBackoff
	e.tcpDialBackoffOverride = tcpBackoff
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
	return e.ExecuteWithContext(context.Background(), device)
}

// ExecuteWithContext 执行驱动读取（支持超时/取消）
func (e *DriverExecutor) ExecuteWithContext(ctx context.Context, device *models.Device) (*DriverResult, error) {
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
	driverCtx := buildDriverContext(device, resourceID, resourceType, deviceConfig)

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
	return e.manager.ExecuteDriverWithContext(ctx, *device.DriverID, defaultDriverFunction, driverCtx)
}

// ExecuteCommand 执行指定函数（用于写入等主动命令）
func (e *DriverExecutor) ExecuteCommand(device *models.Device, function string, config map[string]string) (*DriverResult, error) {
	return e.ExecuteCommandWithContext(context.Background(), device, function, config)
}

// ExecuteCommandWithContext 执行指定函数（支持超时/取消）
func (e *DriverExecutor) ExecuteCommandWithContext(ctx context.Context, device *models.Device, function string, config map[string]string) (*DriverResult, error) {
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
	for key, value := range config {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		deviceConfig[trimmedKey] = value
	}

	pluginFunc := strings.TrimSpace(function)
	if pluginFunc == "" {
		pluginFunc = defaultDriverFunction
	}

	driverCtx := buildDriverContext(device, resourceID, resourceType, deviceConfig)

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

	return e.manager.ExecuteDriverWithContext(ctx, *device.DriverID, pluginFunc, driverCtx)
}

// CollectData 采集数据
func (e *DriverExecutor) CollectData(device *models.Device) (*models.CollectData, error) {
	return e.CollectDataWithContext(context.Background(), device)
}

// CollectDataWithContext 采集数据（支持超时/取消）
func (e *DriverExecutor) CollectDataWithContext(ctx context.Context, device *models.Device) (*models.CollectData, error) {
	result, err := e.ExecuteWithContext(ctx, device)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, errors.New(result.Error)
	}

	// 解析返回数据
	fields := mapResultFields(result)

	return &models.CollectData{
		DeviceID:   device.ID,
		DeviceName: device.Name,
		Timestamp:  result.Timestamp,
		Fields:     fields,
	}, nil
}

func mapResultFields(result *DriverResult) map[string]string {
	if result == nil {
		return map[string]string{}
	}

	if len(result.Points) > 0 {
		fields := make(map[string]string, len(result.Points))
		for _, point := range result.Points {
			if point.FieldName == "" {
				continue
			}
			fields[point.FieldName] = formatDriverValue(point.Value)
		}
		return fields
	}

	if len(result.Data) == 0 {
		return map[string]string{}
	}

	fields := make(map[string]string, len(result.Data))
	for key, value := range result.Data {
		fields[key] = value
	}
	return fields
}

func formatDriverValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []byte:
		return string(v)
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 6, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', 6, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}
