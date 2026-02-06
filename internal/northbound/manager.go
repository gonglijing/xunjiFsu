package northbound

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/circuit"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// Northbound 北向接口接口
type Northbound interface {
	// Initialize 初始化
	Initialize(config string) error
	// Send 发送数据
	Send(data *models.CollectData) error
	// SendAlarm 发送报警
	SendAlarm(alarm *models.AlarmPayload) error
	// Close 关闭
	Close() error
	// Name 获取名称
	Name() string
}

// NorthboundManager 北向管理器
type NorthboundManager struct {
	mu          sync.RWMutex
	adapters    map[string]Northbound
	uploadTimes map[string]time.Time
	intervals   map[string]time.Duration
	enabled     map[string]bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	running     bool
	breakers    map[string]*circuit.CircuitBreaker
	pending     map[string]*models.CollectData
	pluginDir   string
}

// DefaultBreakerConfig 默认熔断器配置
var DefaultBreakerConfig = circuit.Config{
	FailureThreshold: 5,                // 5次失败后打开熔断
	FailureWindow:    time.Minute,      // 1分钟窗口
	SuccessThreshold: 3,                // 半开状态下需要3次成功
	RecoveryTimeout:  30 * time.Second, // 30秒后尝试恢复
	RequestTimeout:   10 * time.Second, // 单次请求超时
}

// NewNorthboundManager 创建北向管理器
func NewNorthboundManager(pluginDir string) *NorthboundManager {
	return &NorthboundManager{
		adapters:    make(map[string]Northbound),
		uploadTimes: make(map[string]time.Time),
		intervals:   make(map[string]time.Duration),
		enabled:     make(map[string]bool),
		stopChan:    make(chan struct{}),
		breakers:    make(map[string]*circuit.CircuitBreaker),
		pending:     make(map[string]*models.CollectData),
		pluginDir:   pluginDir,
	}
}

// PluginDir returns the configured northbound plugin directory.
func (m *NorthboundManager) PluginDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.pluginDir
}

// Start 启动管理器
func (m *NorthboundManager) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stopChan = make(chan struct{})
	m.mu.Unlock()

	// 启动上传循环
	m.wg.Add(1)
	go m.flushLoop()

	log.Println("Northbound manager started")
}

// Stop 停止管理器
func (m *NorthboundManager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	close(m.stopChan)
	m.mu.Unlock()
	m.wg.Wait()
	log.Println("Northbound manager stopped")
}

// RegisterAdapter 注册适配器
func (m *NorthboundManager) RegisterAdapter(name string, adapter Northbound) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, exists := m.adapters[name]; exists {
		_ = old.Close()
	}
	if breaker, exists := m.breakers[name]; exists {
		breaker.Reset()
	}
	m.adapters[name] = adapter
	m.uploadTimes[name] = time.Time{}
	m.intervals[name] = 0
	m.enabled[name] = true
	// 为每个适配器创建熔断器
	cfg := DefaultBreakerConfig
	m.breakers[name] = circuit.NewCircuitBreaker(&cfg)
	log.Printf("Northbound adapter registered: %s (with circuit breaker)", name)
}

// UnregisterAdapter 注销适配器
func (m *NorthboundManager) UnregisterAdapter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if adapter, exists := m.adapters[name]; exists {
		_ = adapter.Close()
	}
	delete(m.adapters, name)
	delete(m.uploadTimes, name)
	delete(m.intervals, name)
	delete(m.enabled, name)
	delete(m.pending, name)
	// 清理熔断器
	if breaker, exists := m.breakers[name]; exists {
		breaker.Reset()
		delete(m.breakers, name)
	}
}

// GetAdapter 获取适配器
func (m *NorthboundManager) GetAdapter(name string) (Northbound, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	adapter, exists := m.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}
	return adapter, nil
}

// GetAdapterCount 获取适配器数量

// SetInterval 设置上传周期
func (m *NorthboundManager) SetInterval(name string, interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if interval < 500*time.Millisecond {
		interval = 500 * time.Millisecond
	}
	m.intervals[name] = interval
	// 重置上次发送时间，确保新周期立即生效
	delete(m.uploadTimes, name)
}

// SetEnabled 设置北向启停
func (m *NorthboundManager) SetEnabled(name string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled[name] = enabled
	if !enabled {
		delete(m.uploadTimes, name)
		delete(m.pending, name)
	}
}

func (m *NorthboundManager) GetAdapterCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.adapters)
}

// HasAdapter 检查指定适配器是否已注册
func (m *NorthboundManager) HasAdapter(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.adapters[name]
	return ok
}

// IsEnabled 返回北向运行使能状态
func (m *NorthboundManager) IsEnabled(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled[name]
}

// GetInterval 返回北向上传周期
func (m *NorthboundManager) GetInterval(name string) time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.intervals[name]
}

// GetLastUploadTime 返回最后发送时间
func (m *NorthboundManager) GetLastUploadTime(name string) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.uploadTimes[name]
}

// HasPending 返回是否存在待发送数据
func (m *NorthboundManager) HasPending(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.pending[name]
	return ok
}

// ListRuntimeNames 返回当前北向运行时所有名称（配置、适配器、缓冲、熔断器）
func (m *NorthboundManager) ListRuntimeNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	seen := make(map[string]struct{})
	for name := range m.adapters {
		seen[name] = struct{}{}
	}
	for name := range m.intervals {
		seen[name] = struct{}{}
	}
	for name := range m.enabled {
		seen[name] = struct{}{}
	}
	for name := range m.pending {
		seen[name] = struct{}{}
	}
	for name := range m.breakers {
		seen[name] = struct{}{}
	}
	for name := range m.uploadTimes {
		seen[name] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names
}

// RemoveAdapter 移除适配器
func (m *NorthboundManager) RemoveAdapter(name string) {
	m.UnregisterAdapter(name)
}

// SendData 发送数据到所有启用的北向
func (m *NorthboundManager) SendData(data *models.CollectData) {
	m.mu.Lock()
	for name := range m.adapters {
		m.pending[name] = data
	}
	m.mu.Unlock()
}

// SendAlarm 发送报警到所有启用的北向
func (m *NorthboundManager) SendAlarm(alarm *models.AlarmPayload) {
	m.mu.RLock()
	adapters := make([]Northbound, 0, len(m.adapters))
	names := make([]string, 0, len(m.adapters))
	for name, adapter := range m.adapters {
		adapters = append(adapters, adapter)
		names = append(names, name)
	}
	m.mu.RUnlock()

	for i, adapter := range adapters {
		name := names[i]
		if !m.enabled[name] {
			continue
		}
		err := m.executeWithBreaker(name, func() error {
			return adapter.SendAlarm(alarm)
		})
		if err != nil {
			if _, ok := err.(*circuit.CircuitOpenError); ok {
				log.Printf("Circuit breaker OPEN for %s, skipping alarm send", name)
			} else {
				log.Printf("Failed to send alarm to %s: %v", name, err)
			}
		}
	}
}

// executeWithBreaker 使用熔断器执行函数
func (m *NorthboundManager) executeWithBreaker(name string, fn func() error) error {
	m.mu.RLock()
	breaker, exists := m.breakers[name]
	m.mu.RUnlock()

	if !exists {
		return fn()
	}

	return breaker.Execute(fn)
}

// GetBreakerState 获取熔断器状态
func (m *NorthboundManager) GetBreakerState(name string) circuit.CircuitState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if breaker, exists := m.breakers[name]; exists {
		return breaker.State() // State现在是方法调用
	}
	return circuit.Closed
}

func (m *NorthboundManager) shouldSend(name string) bool {
	interval := m.intervals[name]
	if interval <= 0 {
		return true
	}
	last := m.uploadTimes[name]
	if last.IsZero() {
		return true
	}
	return time.Since(last) >= interval
}

func (m *NorthboundManager) markSent(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploadTimes[name] = time.Now()
}

func (m *NorthboundManager) uploadToAdapters(data *models.CollectData) {
	m.mu.RLock()
	adapters := make([]Northbound, 0, len(m.adapters))
	names := make([]string, 0, len(m.adapters))
	for name, adapter := range m.adapters {
		adapters = append(adapters, adapter)
		names = append(names, name)
	}
	m.mu.RUnlock()

	for i, adapter := range adapters {
		name := names[i]
		if !m.enabled[name] {
			continue
		}
		if !m.shouldSend(name) {
			continue
		}
		err := m.executeWithBreaker(name, func() error { return adapter.Send(data) })
		if err != nil {
			if _, ok := err.(*circuit.CircuitOpenError); ok {
				log.Printf("Circuit breaker OPEN for %s, skipping data send", name)
			} else {
				log.Printf("Failed to send data to %s: %v", name, err)
			}
		} else {
			m.markSent(name)
		}
	}
}

// flushLoop 周期性检查待发送数据，按各自上传周期发送
func (m *NorthboundManager) flushLoop() {
	defer m.wg.Done()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.flushPending()
		}
	}
}

// flushPending 将待发数据推送到各适配器（符合发送周期才发送）
func (m *NorthboundManager) flushPending() {
	m.mu.RLock()
	// 复制一份，避免长时间持锁
	pendingCopy := make(map[string]*models.CollectData, len(m.pending))
	for name, data := range m.pending {
		pendingCopy[name] = data
	}
	adapters := make(map[string]Northbound, len(m.adapters))
	for name, ad := range m.adapters {
		adapters[name] = ad
	}
	m.mu.RUnlock()

	for name, data := range pendingCopy {
		if data == nil {
			continue
		}
		if !m.enabled[name] {
			continue
		}
		if !m.shouldSend(name) {
			continue
		}
		adapter, ok := adapters[name]
		if !ok {
			continue
		}
		err := m.executeWithBreaker(name, func() error { return adapter.Send(data) })
		if err != nil {
			if _, ok := err.(*circuit.CircuitOpenError); ok {
				log.Printf("Circuit breaker OPEN for %s, skipping data send", name)
			} else {
				log.Printf("Failed to send data to %s: %v", name, err)
			}
			continue
		}
		m.markSent(name)
		// 清空已发送的待发数据
		m.mu.Lock()
		delete(m.pending, name)
		m.mu.Unlock()
	}
}
