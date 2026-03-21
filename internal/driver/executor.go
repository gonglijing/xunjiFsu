package driver

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const pooledModbusFrameSize = 512

// DriverExecutor 驱动执行器
type DriverExecutor struct {
	manager                   *DriverManager
	serialPorts               map[int64]SerialPort // 资源ID到串口的映射
	tcpConns                  map[int64]net.Conn   // 资源ID到TCP连接
	resourcePaths             map[int64]string     // 资源ID到路径的映射 (用于TCP懒连接)
	resourceMux               sync.Map             // key:int64 -> *sync.Mutex, 同一资源串口的互斥锁
	mu                        sync.RWMutex
	executing                 map[int64]bool
	serialTimeout             time.Duration
	serialOpenRetries         int
	serialOpenBackoffOverride time.Duration
	tcpDialTimeoutOverride    time.Duration
	tcpDialRetries            int
	tcpDialBackoffOverride    time.Duration
	tcpReadTimeoutOverride    time.Duration
	tcpDialFn                 func(network, address string, timeout time.Duration) (net.Conn, error)
}

// NewDriverExecutor 创建驱动执行器
func NewDriverExecutor(manager *DriverManager) *DriverExecutor {
	executor := &DriverExecutor{
		manager:       manager,
		serialPorts:   make(map[int64]SerialPort),
		tcpConns:      make(map[int64]net.Conn),
		resourcePaths: make(map[int64]string),
		executing:     make(map[int64]bool),
	}

	// 双向绑定
	if manager != nil {
		manager.SetExecutor(executor)
	}

	return executor
}

func (e *DriverExecutor) tcpDial(network, address string, timeout time.Duration) (net.Conn, error) {
	if e != nil && e.tcpDialFn != nil {
		return e.tcpDialFn(network, address, timeout)
	}
	return net.DialTimeout(network, address, timeout)
}

var modbusFrameBufferPool = sync.Pool{
	New: func() any {
		return make([]byte, pooledModbusFrameSize)
	},
}

func getModbusFrameBuffer(size int) []byte {
	if size <= 0 {
		return nil
	}
	if size > pooledModbusFrameSize {
		return make([]byte, size)
	}
	buf, _ := modbusFrameBufferPool.Get().([]byte)
	if cap(buf) < pooledModbusFrameSize {
		buf = make([]byte, pooledModbusFrameSize)
	}
	return buf[:size]
}

func putModbusFrameBuffer(buf []byte) {
	if cap(buf) < pooledModbusFrameSize || cap(buf) > pooledModbusFrameSize {
		return
	}
	modbusFrameBufferPool.Put(buf[:pooledModbusFrameSize])
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
	res, err := database.LoadResource(resourceID)
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
	slog.Info("Serial port opened", "resource_id", resourceID, "path", res.Path, "baud", baud, "data_bits", dataBits, "stop_bits", stopBits, "parity", parity)
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

	unlock := e.lockResource(resourceID)
	if unlock != nil {
		defer unlock()
	}

	e.mu.RLock()
	conn, exists = e.tcpConns[resourceID]
	e.mu.RUnlock()
	if exists && conn != nil {
		return conn
	}

	dialTimeout := e.tcpDialTimeout()
	dialAttempts := e.tcpDialAttempts()
	dialBackoff := e.tcpDialBackoff()

	// 建立新连接
	var dialConn net.Conn
	var err error
	for i := 0; i < dialAttempts; i++ {
		dialConn, err = e.tcpDial("tcp", path, dialTimeout)
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
	path = strings.TrimSpace(path)
	e.mu.Lock()
	if prev, ok := e.resourcePaths[resourceID]; ok && prev != "" && prev != path {
		if conn, ok := e.tcpConns[resourceID]; ok {
			_ = conn.Close()
			delete(e.tcpConns, resourceID)
		}
	}
	if path == "" {
		delete(e.resourcePaths, resourceID)
		e.mu.Unlock()
		return
	}
	e.resourcePaths[resourceID] = path
	e.mu.Unlock()
}

// RefreshResource syncs executor-side resource cache after resource CRUD changes.
func (e *DriverExecutor) RefreshResource(resource *models.Resource) {
	if e == nil || resource == nil || resource.ID <= 0 {
		return
	}

	if resource.Enabled != 1 {
		e.CloseResource(resource.ID)
		return
	}

	resourceType := strings.ToLower(strings.TrimSpace(resource.Type))
	if resourceType != "net" {
		e.CloseResource(resource.ID)
		return
	}

	e.SetResourcePath(resource.ID, resource.Path)
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
	e.resourceMux.Delete(resourceID)
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
	e.resourceMux.Range(func(key, _ any) bool {
		e.resourceMux.Delete(key)
		return true
	})
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
	if lock, ok := e.resourceMux.Load(resourceID); ok {
		return lock.(*sync.Mutex)
	}
	lock := &sync.Mutex{}
	actual, _ := e.resourceMux.LoadOrStore(resourceID, lock)
	return actual.(*sync.Mutex)
}

