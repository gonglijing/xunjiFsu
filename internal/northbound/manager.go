package northbound

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/circuit"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

// Northbound 北向接口接口（用于外部集成）
type Northbound interface {
	Name() string
	Send(data *models.CollectData) error
	SendAlarm(alarm *models.AlarmPayload) error
	Close() error
}

// NorthboundManager 北向管理器
// 简化版：不再使用插件，直接使用内置适配器
// 每个适配器自己管理自己的状态和发送线程
type NorthboundManager struct {
	mu       sync.RWMutex
	adapters map[string]adapters.NorthboundAdapter

	// 以下字段保留用于兼容和状态查询
	selfManaged map[string]bool
	enabled     map[string]bool
	intervals   map[string]time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
	running     bool

	// 熔断器（可选，由适配器自己管理）
	breakers map[string]*circuit.CircuitBreaker
}

// DefaultBreakerConfig 默认熔断器配置
var DefaultBreakerConfig = circuit.Config{
	FailureThreshold:  5,                // 5次失败后打开熔断
	FailureWindow:     time.Minute,       // 1分钟窗口
	SuccessThreshold:  3,                  // 半开状态下需要3次成功
	RecoveryTimeout:   30 * time.Second,  // 30秒后尝试恢复
	RequestTimeout:   10 * time.Second,  // 单次请求超时
}

// NewNorthboundManager 创建北向管理器（简化版，不再需要 pluginDir）
func NewNorthboundManager() *NorthboundManager {
	return &NorthboundManager{
		adapters:    make(map[string]adapters.NorthboundAdapter),
		selfManaged: make(map[string]bool),
		enabled:     make(map[string]bool),
		intervals:   make(map[string]time.Duration),
		stopChan:    make(chan struct{}),
		breakers:    make(map[string]*circuit.CircuitBreaker),
	}
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

	log.Println("Northbound manager started (built-in adapters)")
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

	// 停止所有适配器
	m.mu.Lock()
	for _, adapter := range m.adapters {
		adapter.Stop()
	}
	m.mu.Unlock()

	m.wg.Wait()
	log.Println("Northbound manager stopped")
}

// RegisterAdapter 注册适配器
func (m *NorthboundManager) RegisterAdapter(name string, adapter adapters.NorthboundAdapter) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, exists := m.adapters[name]; exists {
		old.Stop()
		old.Close()
	}

	m.adapters[name] = adapter
	m.selfManaged[name] = true // 内置适配器都是 self-managed
	m.enabled[name] = true

	log.Printf("Northbound adapter registered: %s (built-in)", name)
}

// UnregisterAdapter 注销适配器
func (m *NorthboundManager) UnregisterAdapter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if adapter, exists := m.adapters[name]; exists {
		adapter.Stop()
		adapter.Close()
	}
	delete(m.adapters, name)
	delete(m.selfManaged, name)
	delete(m.enabled, name)
	delete(m.intervals, name)
	delete(m.breakers, name)

	log.Printf("Northbound adapter unregistered: %s", name)
}

// GetAdapter 获取适配器
func (m *NorthboundManager) GetAdapter(name string) (adapters.NorthboundAdapter, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapter, exists := m.adapters[name]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", name)
	}
	return adapter, nil
}

// SetInterval 设置上传周期（委托给适配器）
func (m *NorthboundManager) SetInterval(name string, interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if adapter, exists := m.adapters[name]; exists {
		adapter.SetInterval(interval)
	}
	m.intervals[name] = interval
}

// SetEnabled 设置北向启停
func (m *NorthboundManager) SetEnabled(name string, enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if adapter, exists := m.adapters[name]; exists {
		if enabled {
			adapter.Start()
		} else {
			adapter.Stop()
		}
	}
	m.enabled[name] = enabled
}

// GetAdapterCount 获取适配器数量
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

// IsConnected 返回北向连接状态
func (m *NorthboundManager) IsConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	adapter, exists := m.adapters[name]
	if !exists || !m.enabled[name] {
		return false
	}

	return adapter.IsConnected()
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

	if adapter, exists := m.adapters[name]; exists {
		return adapter.GetLastSendTime()
	}
	return time.Time{}
}

// HasPending 返回是否存在待发送数据（检查适配器状态）
func (m *NorthboundManager) HasPending(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if adapter, exists := m.adapters[name]; exists {
		stats := adapter.GetStats()
		if pending, ok := stats["pending_data"].(int); ok && pending > 0 {
			return true
		}
		if pending, ok := stats["pending_alarm"].(int); ok && pending > 0 {
			return true
		}
	}
	return false
}

// ListRuntimeNames 返回当前北向运行时所有名称
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
	for name := range m.breakers {
		seen[name] = struct{}{}
	}
	for name := range m.selfManaged {
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
	m.mu.RLock()
	adapters := make([]adapters.NorthboundAdapter, 0, len(m.adapters))
	names := make([]string, 0, len(m.adapters))
	enabled := make(map[string]bool, len(m.adapters))

	for name, adapter := range m.adapters {
		adapters = append(adapters, adapter)
		names = append(names, name)
		enabled[name] = m.enabled[name]
	}
	m.mu.RUnlock()

	for i, adapter := range adapters {
		name := names[i]
		if !enabled[name] {
			continue
		}
		// 内置适配器自己管理发送，不需要通过熔断器
		if err := adapter.Send(data); err != nil {
			log.Printf("Failed to send data to %s: %v", name, err)
		}
	}
}

// SendAlarm 发送报警到所有启用的北向
func (m *NorthboundManager) SendAlarm(alarm *models.AlarmPayload) {
	m.mu.RLock()
	adapters := make([]adapters.NorthboundAdapter, 0, len(m.adapters))
	names := make([]string, 0, len(m.adapters))
	enabled := make(map[string]bool, len(m.adapters))

	for name, adapter := range m.adapters {
		adapters = append(adapters, adapter)
		names = append(names, name)
		enabled[name] = m.enabled[name]
	}
	m.mu.RUnlock()

	for i, adapter := range adapters {
		name := names[i]
		if !enabled[name] {
			continue
		}
		if err := adapter.SendAlarm(alarm); err != nil {
			log.Printf("Failed to send alarm to %s: %v", name, err)
		}
	}
}

// PullCommands 从所有启用北向拉取待执行命令
func (m *NorthboundManager) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	if limit <= 0 {
		limit = 50
	}

	m.mu.RLock()
	adapters := make([]adapters.NorthboundAdapter, 0, len(m.adapters))
	names := make([]string, 0, len(m.adapters))
	enabled := make(map[string]bool, len(m.adapters))

	for name, adapter := range m.adapters {
		adapters = append(adapters, adapter)
		names = append(names, name)
		enabled[name] = m.enabled[name]
	}
	m.mu.RUnlock()

	commands := make([]*models.NorthboundCommand, 0, limit)

	for idx, adapter := range adapters {
		name := names[idx]
		if !enabled[name] {
			continue
		}

		// 检查适配器是否支持命令下发
		cmdAdapter, ok := adapter.(interface{ PullCommands(int) ([]*models.NorthboundCommand, error) })
		if !ok {
			continue
		}

		remaining := limit - len(commands)
		if remaining <= 0 {
			break
		}

		pulled, err := cmdAdapter.PullCommands(remaining)
		if err != nil {
			log.Printf("Failed to pull commands from %s: %v", name, err)
			continue
		}
		if len(pulled) == 0 {
			continue
		}
		commands = append(commands, pulled...)
	}

	return commands, nil
}

// ReportCommandResult 将执行结果回传给各启用北向
func (m *NorthboundManager) ReportCommandResult(result *models.NorthboundCommandResult) {
	if result == nil {
		return
	}

	m.mu.RLock()
	adapters := make([]adapters.NorthboundAdapter, 0, len(m.adapters))
	names := make([]string, 0, len(m.adapters))
	enabled := make(map[string]bool, len(m.adapters))

	for name, adapter := range m.adapters {
		adapters = append(adapters, adapter)
		names = append(names, name)
		enabled[name] = m.enabled[name]
	}
	m.mu.RUnlock()

	for idx, adapter := range adapters {
		name := names[idx]
		if !enabled[name] {
			continue
		}

		// 检查适配器是否支持命令下发
		cmdAdapter, ok := adapter.(interface{ ReportCommandResult(*models.NorthboundCommandResult) error })
		if !ok {
			continue
		}

		if err := cmdAdapter.ReportCommandResult(result); err != nil {
			log.Printf("Failed to report command result to %s: %v", name, err)
		}
	}
}

// GetBreakerState 获取熔断器状态（兼容接口）
func (m *NorthboundManager) GetBreakerState(name string) circuit.CircuitState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if breaker, exists := m.breakers[name]; exists {
		return breaker.State()
	}
	return circuit.Closed
}

// GetStats 获取所有适配器的状态统计
func (m *NorthboundManager) GetStats() map[string]map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]map[string]interface{})
	for name, adapter := range m.adapters {
		stats := adapter.GetStats()
		stats["manager_enabled"] = m.enabled[name]
		stats["manager_interval_ms"] = m.intervals[name].Milliseconds()
		result[name] = stats
	}
	return result
}
