package resource

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gogw/internal/models"
)

// ErrResourceNotFound 资源未找到
var ErrResourceNotFound = errors.New("resource not found")

// ErrResourceBusy 资源繁忙（超时）
var ErrResourceBusy = errors.New("resource is busy, try again later")

// ErrResourceLocked 资源已被锁定
var ErrResourceLocked = errors.New("resource is locked by another operation")

// ResourceLocker 资源访问锁管理器
// 确保对同一个串口/网口资源的串行访问，防止并发读取导致数据乱码
type ResourceLocker struct {
	mu       sync.RWMutex
	locks    map[int64]*resourceLock
	maxWait  time.Duration // 获取锁的最大等待时间
}

type resourceLock struct {
	mu         sync.Mutex
	refCount   int
	lastAccess time.Time
}

// NewResourceLocker 创建资源访问锁管理器
func NewResourceLocker() *ResourceLocker {
	return &ResourceLocker{
		locks:   make(map[int64]*resourceLock),
		maxWait: 30 * time.Second, // 默认最大等待30秒
	}
}

// SetMaxWait 设置获取锁的最大等待时间
func (l *ResourceLocker) SetMaxWait(duration time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxWait = duration
}

// Lock 获取资源的独占访问锁
// 阻塞等待直到获得锁或超时
func (l *ResourceLocker) Lock(resourceID int64) error {
	return l.lockWithTimeout(resourceID, 0) // 0表示无限等待
}

// TryLock 尝试获取资源的独占访问锁（非阻塞）
func (l *ResourceLocker) TryLock(resourceID int64) error {
	return l.lockWithTimeout(resourceID, time.Microsecond)
}

// LockWithTimeout 在指定时间内尝试获取资源的独占访问锁
func (l *ResourceLocker) LockWithTimeout(resourceID int64, timeout time.Duration) error {
	return l.lockWithTimeout(resourceID, timeout)
}

func (l *ResourceLocker) lockWithTimeout(resourceID int64, timeout time.Duration) error {
	// 获取或创建资源锁
	lock, created := l.getOrCreateLock(resourceID)
	if !created {
		// 资源锁已存在，尝试获取
		deadline := time.Now().Add(timeout)
		acquired := false
		
		for {
			if timeout > 0 && time.Now().After(deadline) {
				return ErrResourceBusy
			}
			
			if lock.mu.TryLock() {
				acquired = true
				break
			}
			
			// 短暂休眠后重试，避免CPU busy wait
			time.Sleep(time.Millisecond)
		}
		
		if !acquired {
			return ErrResourceBusy
		}
	} else {
		// 新创建的资源锁，直接获取
		lock.mu.Lock()
	}
	
	// 更新最后访问时间
	lock.lastAccess = time.Now()
	return nil
}

func (l *ResourceLocker) getOrCreateLock(resourceID int64) (*resourceLock, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if lock, exists := l.locks[resourceID]; exists {
		lock.refCount++
		return lock, false
	}
	
	lock := &resourceLock{
		refCount:   1,
		lastAccess: time.Now(),
	}
	l.locks[resourceID] = lock
	return lock, true
}

// Unlock 释放资源的独占访问锁
func (l *ResourceLocker) Unlock(resourceID int64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	lock, exists := l.locks[resourceID]
	if !exists {
		return ErrResourceNotFound
	}
	
	lock.refCount--
	if lock.refCount <= 0 {
		delete(l.locks, resourceID)
	}
	
	lock.mu.Unlock()
	return nil
}

// IsLocked 检查资源是否被锁定
func (l *ResourceLocker) IsLocked(resourceID int64) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if lock, exists := l.locks[resourceID]; exists {
		// 检查锁是否被持有（通过尝试非阻塞获取）
		if lock.mu.TryLock() {
			lock.mu.Unlock()
			return false // 锁空闲
		}
		return true // 锁被持有
	}
	return false // 锁不存在，视为空闲
}

// GetLockCount 获取资源的锁引用计数
func (l *ResourceLocker) GetLockCount(resourceID int64) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if lock, exists := l.locks[resourceID]; exists {
		return lock.refCount
	}
	return 0
}

// Cleanup 清理过期的资源锁（可选，用于定期清理孤儿锁）
func (l *ResourceLocker) Cleanup(maxAge time.Duration) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	cleaned := 0
	
	for id, lock := range l.locks {
		// 检查锁是否空闲且超过最大年龄
		if lock.refCount <= 0 && lock.lastAccess.Before(cutoff) {
			delete(l.locks, id)
			cleaned++
		}
	}
	
	return cleaned
}

// ResourceManager 资源管理器接口
type ResourceManager interface {
	// Open 打开资源
	Open(resource *models.Resource) error
	// Close 关闭资源
	Close(id int64) error
	// IsOpen 检查资源是否打开
	IsOpen(id int64) bool
	// Read 从资源读取数据
	Read(id int64, buffer []byte) (int, error)
	// Write 向资源写入数据
	Write(id int64, data []byte) (int, error)
	// GetLastError 获取最后错误
	GetLastError(id int64) error
}

// SerialManager 串口管理器
type SerialManager struct {
	mu       sync.RWMutex
	ports    map[int64]*serialPort
	portChan chan int64
}

type serialPort struct {
	resource *models.Resource
	conn     interface{} // 实际连接对象
	open     bool
	lastErr  error
	mu       sync.RWMutex
}

// NewSerialManager 创建串口管理器
func NewSerialManager() *SerialManager {
	return &SerialManager{
		ports:    make(map[int64]*serialPort),
		portChan: make(chan int64, 100),
	}
}

// Open 打开串口
func (m *SerialManager) Open(resource *models.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.ports[resource.ID]; exists {
		if m.ports[resource.ID].open {
			return ErrResourceBusy
		}
	}

	// 创建串口实例
	port := &serialPort{
		resource: resource,
		open:     true,
	}

	// 初始化串口连接
	// 注意：这里需要使用纯Go的串口库，如 go-serial
	// 由于没有纯Go的串口库，我们使用基于文件的模拟方式
	// 在实际部署时，可以使用 gobindata 或者其他方式

	m.ports[resource.ID] = port
	return nil
}

// Close 关闭串口
func (m *SerialManager) Close(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	port, exists := m.ports[id]
	if !exists {
		return ErrResourceNotFound
	}

	port.mu.Lock()
	port.open = false
	port.conn = nil
	port.mu.Unlock()

	delete(m.ports, id)
	return nil
}

// IsOpen 检查串口是否打开
func (m *SerialManager) IsOpen(id int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	port, exists := m.ports[id]
	if !exists {
		return false
	}

	port.mu.RLock()
	defer port.mu.RUnlock()
	return port.open
}

// Read 从串口读取数据
func (m *SerialManager) Read(id int64, buffer []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	port, exists := m.ports[id]
	if !exists {
		return 0, ErrResourceNotFound
	}

	port.mu.RLock()
	defer port.mu.RUnlock()

	if !port.open {
		return 0, fmt.Errorf("port is closed")
	}

	// 模拟读取
	// 实际实现需要使用串口库
	return 0, nil
}

// Write 向串口写入数据
func (m *SerialManager) Write(id int64, data []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	port, exists := m.ports[id]
	if !exists {
		return 0, ErrResourceNotFound
	}

	port.mu.RLock()
	defer port.mu.RUnlock()

	if !port.open {
		return 0, fmt.Errorf("port is closed")
	}

	// 模拟写入
	return len(data), nil
}

// GetLastError 获取最后错误
func (m *SerialManager) GetLastError(id int64) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	port, exists := m.ports[id]
	if !exists {
		return ErrResourceNotFound
	}

	port.mu.RLock()
	defer port.mu.RUnlock()
	return port.lastErr
}

// NetworkManager 网口管理器
type NetworkManager struct {
	mu      sync.RWMutex
	clients map[int64]*networkClient
}

type networkClient struct {
	resource   *models.Resource
	conn       interface{} // 实际连接对象
	connected  bool
	lastErr    error
	mu         sync.RWMutex
	lastActive time.Time
}

// NewNetworkManager 创建网口管理器
func NewNetworkManager() *NetworkManager {
	return &NetworkManager{
		clients: make(map[int64]*networkClient),
	}
}

// Open 连接到网络资源
func (m *NetworkManager) Open(resource *models.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[resource.ID]; exists {
		if m.clients[resource.ID].connected {
			return ErrResourceBusy
		}
	}

	client := &networkClient{
		resource:   resource,
		connected:  true,
		lastActive: time.Now(),
	}

	// 建立TCP/UDP连接
	// 实际实现需要使用net包

	m.clients[resource.ID] = client
	return nil
}

// Close 关闭连接
func (m *NetworkManager) Close(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[id]
	if !exists {
		return ErrResourceNotFound
	}

	client.mu.Lock()
	client.connected = false
	client.conn = nil
	client.mu.Unlock()

	delete(m.clients, id)
	return nil
}

// IsOpen 检查是否已连接
func (m *NetworkManager) IsOpen(id int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[id]
	if !exists {
		return false
	}

	client.mu.RLock()
	defer client.mu.RUnlock()
	return client.connected
}

// Read 从网络读取数据
func (m *NetworkManager) Read(id int64, buffer []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[id]
	if !exists {
		return 0, ErrResourceNotFound
	}

	client.mu.RLock()
	defer client.mu.RUnlock()

	if !client.connected {
		return 0, fmt.Errorf("connection is closed")
	}

	client.lastActive = time.Now()

	// 模拟读取
	return 0, nil
}

// Write 向网络写入数据
func (m *NetworkManager) Write(id int64, data []byte) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[id]
	if !exists {
		return 0, ErrResourceNotFound
	}

	client.mu.RLock()
	defer client.mu.RUnlock()

	if !client.connected {
		return 0, fmt.Errorf("connection is closed")
	}

	client.lastActive = time.Now()

	// 模拟写入
	return len(data), nil
}

// GetLastError 获取最后错误
func (m *NetworkManager) GetLastError(id int64) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[id]
	if !exists {
		return ErrResourceNotFound
	}

	client.mu.RLock()
	defer client.mu.RUnlock()
	return client.lastErr
}

// ResourceManagerImpl 资源管理器实现
type ResourceManagerImpl struct {
	serialManager  *SerialManager
	networkManager *NetworkManager
	locker         *ResourceLocker // 资源访问锁
	mu             sync.RWMutex
}

// NewResourceManagerImpl 创建资源管理器
func NewResourceManagerImpl() *ResourceManagerImpl {
	return &ResourceManagerImpl{
		serialManager:  NewSerialManager(),
		networkManager: NewNetworkManager(),
		locker:         NewResourceLocker(),
	}
}

// GetLocker 获取资源访问锁管理器
func (m *ResourceManagerImpl) GetLocker() *ResourceLocker {
	return m.locker
}

// OpenResource 打开资源
func (m *ResourceManagerImpl) OpenResource(resource *models.Resource) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch resource.Type {
	case "serial":
		return m.serialManager.Open(resource)
	case "network":
		return m.networkManager.Open(resource)
	default:
		return fmt.Errorf("unknown resource type: %s", resource.Type)
	}
}

// CloseResource 关闭资源
func (m *ResourceManagerImpl) CloseResource(id int64, resourceType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch resourceType {
	case "serial":
		return m.serialManager.Close(id)
	case "network":
		return m.networkManager.Close(id)
	default:
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// IsResourceOpen 检查资源是否打开
func (m *ResourceManagerImpl) IsResourceOpen(id int64, resourceType string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch resourceType {
	case "serial":
		return m.serialManager.IsOpen(id)
	case "network":
		return m.networkManager.IsOpen(id)
	default:
		return false
	}
}

// ReadResource 从资源读取（带访问锁）
func (m *ResourceManagerImpl) ReadResource(id int64, resourceType string, buffer []byte) (int, error) {
	// 先获取资源访问锁
	if err := m.locker.Lock(id); err != nil {
		return 0, err
	}
	defer m.locker.Unlock(id)

	switch resourceType {
	case "serial":
		return m.serialManager.Read(id, buffer)
	case "network":
		return m.networkManager.Read(id, buffer)
	default:
		return 0, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// WriteResource 向资源写入（带访问锁）
func (m *ResourceManagerImpl) WriteResource(id int64, resourceType string, data []byte) (int, error) {
	// 先获取资源访问锁
	if err := m.locker.Lock(id); err != nil {
		return 0, err
	}
	defer m.locker.Unlock(id)

	switch resourceType {
	case "serial":
		return m.serialManager.Write(id, data)
	case "network":
		return m.networkManager.Write(id, data)
	default:
		return 0, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

// ExecuteWithLock 带锁执行操作
// 这是一个辅助方法，确保对资源的操作是串行的
func (m *ResourceManagerImpl) ExecuteWithLock(resourceID int64, resourceType string, operation func() error) error {
	// 获取资源访问锁
	if err := m.locker.Lock(resourceID); err != nil {
		return fmt.Errorf("failed to lock resource %d: %w", resourceID, err)
	}
	defer m.locker.Unlock(resourceID)

	// 执行操作
	return operation()
}
