package northbound

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
func NewNorthboundManager() *NorthboundManager {
	return &NorthboundManager{
		adapters:    make(map[string]Northbound),
		uploadTimes: make(map[string]time.Time),
		intervals:   make(map[string]time.Duration),
		enabled:     make(map[string]bool),
		stopChan:    make(chan struct{}),
		breakers:    make(map[string]*circuit.CircuitBreaker),
		pending:     make(map[string]*models.CollectData),
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
		adapter.Close()
		delete(m.adapters, name)
		delete(m.uploadTimes, name)
		delete(m.intervals, name)
		delete(m.enabled, name)
		// 清理熔断器
		if breaker, exists := m.breakers[name]; exists {
			breaker.Reset()
			delete(m.breakers, name)
		}
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

// XunJiAdapter 循迹适配器
type XunJiAdapter struct {
	config      *models.XunJiConfig
	client      interface{} // MQTT客户端
	lastUpload  time.Time
	mu          sync.RWMutex
	initialized bool
}

// NewXunJiAdapter 创建循迹适配器
func NewXunJiAdapter() *XunJiAdapter {
	return &XunJiAdapter{
		lastUpload: time.Time{},
	}
}

// Name 获取名称
func (a *XunJiAdapter) Name() string {
	return "xunji"
}

// Initialize 初始化
func (a *XunJiAdapter) Initialize(configStr string) error {
	config := &models.XunJiConfig{}
	if err := json.Unmarshal([]byte(configStr), config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	a.config = config

	// 初始化MQTT客户端
	// 注意：需要使用纯Go的MQTT库，如 paho.mqtt.golang

	a.initialized = true
	return nil
}

// Send 发送数据
func (a *XunJiAdapter) Send(data *models.CollectData) error {
	if !a.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	// 构建循迹消息
	message := a.buildMessage(data)
	log.Printf("XunJi message: %s", message)

	// 发送到MQTT服务器
	// 实际实现需要使用MQTT客户端

	return nil
}

// SendAlarm 发送报警
func (a *XunJiAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if !a.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	// 构建报警消息
	message := a.buildAlarmMessage(alarm)
	log.Printf("XunJi alarm message: %s", message)

	// 发送到MQTT服务器

	return nil
}

// buildMessage 构建循迹消息
func (a *XunJiAdapter) buildMessage(data *models.CollectData) string {
	properties := make(map[string]interface{})
	for key, value := range data.Fields {
		properties[key] = value
	}

	msg := map[string]interface{}{
		"id":      fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		"version": "1.0",
		"sys": map[string]interface{}{
			"ack": 1,
		},
		"params": map[string]interface{}{
			"properties": properties,
			"events":     map[string]interface{}{},
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]string{
						"productKey": data.ProductKey,
						"deviceKey":  data.DeviceKey,
					},
					"properties": properties,
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(msg)
	return string(jsonBytes)
}

// buildAlarmMessage 构建报警消息
func (a *XunJiAdapter) buildAlarmMessage(alarm *models.AlarmPayload) string {
	eventValue := map[string]interface{}{
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"message":      alarm.Message,
	}

	event := map[string]interface{}{
		"value": eventValue,
		"time":  time.Now().UnixMilli(),
	}

	events := map[string]interface{}{
		"alarm": event,
	}

	msg := map[string]interface{}{
		"id":      fmt.Sprintf("alarm_%d", time.Now().UnixNano()),
		"version": "1.0",
		"sys": map[string]interface{}{
			"ack": 1,
		},
		"params": map[string]interface{}{
			"properties": map[string]interface{}{},
			"events":     events,
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]string{
						"productKey": alarm.ProductKey,
						"deviceKey":  alarm.DeviceKey,
					},
					"properties": map[string]interface{}{},
					"events":     events,
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(msg)
	return string(jsonBytes)
}

// Close 关闭
func (a *XunJiAdapter) Close() error {
	a.initialized = false
	a.config = nil
	return nil
}

// HTTPAdapter HTTP适配器
type HTTPAdapter struct {
	config      string
	url         string
	headers     map[string]string
	lastUpload  time.Time
	timeout     time.Duration
	mu          sync.RWMutex
	initialized bool
}

// HTTPConfig HTTP配置
type HTTPConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout"` // 秒
}

// NewHTTPAdapter 创建HTTP适配器
func NewHTTPAdapter() *HTTPAdapter {
	return &HTTPAdapter{
		lastUpload: time.Time{},
		timeout:    30 * time.Second,
	}
}

// Name 获取名称
func (a *HTTPAdapter) Name() string {
	return "http"
}

// Initialize 初始化
func (a *HTTPAdapter) Initialize(configStr string) error {
	config := &HTTPConfig{}
	if err := json.Unmarshal([]byte(configStr), config); err != nil {
		return fmt.Errorf("failed to parse HTTP config: %w", err)
	}

	a.config = configStr
	a.url = config.URL
	a.headers = config.Headers
	if config.Timeout > 0 {
		a.timeout = time.Duration(config.Timeout) * time.Second
	}
	a.initialized = true

	log.Printf("HTTP adapter initialized: %s", a.url)
	return nil
}

// Send 发送数据
func (a *HTTPAdapter) Send(data *models.CollectData) error {
	if !a.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	// 构建消息
	msg := map[string]interface{}{
		"device_name": data.DeviceName,
		"timestamp":   data.Timestamp,
		"fields":      data.Fields,
	}

	body, _ := json.Marshal(msg)
	return a.sendRequest(a.url, body, "data")
}

// SendAlarm 发送报警
func (a *HTTPAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if !a.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	body, _ := json.Marshal(alarm)
	return a.sendRequest(a.url, body, "alarm")
}

// sendRequest 发送HTTP请求
func (a *HTTPAdapter) sendRequest(url string, body []byte, msgType string) error {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range a.headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: a.timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP %s request failed: %v", msgType, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	log.Printf("HTTP %s sent successfully to %s", msgType, url)
	return nil
}

// Close 关闭
func (a *HTTPAdapter) Close() error {
	a.initialized = false
	return nil
}

// MQTTAdapter MQTT适配器
type MQTTAdapter struct {
	config      string
	broker      string
	topic       string
	clientID    string
	lastUpload  time.Time
	mu          sync.RWMutex
	initialized bool
}

// NewMQTTAdapter 创建MQTT适配器
func NewMQTTAdapter() *MQTTAdapter {
	return &MQTTAdapter{
		lastUpload: time.Time{},
	}
}

// Name 获取名称
func (a *MQTTAdapter) Name() string {
	return "mqtt"
}

// Initialize 初始化
func (a *MQTTAdapter) Initialize(configStr string) error {
	// 解析MQTT配置
	a.config = configStr
	a.initialized = true
	return nil
}

// Send 发送数据
func (a *MQTTAdapter) Send(data *models.CollectData) error {
	if !a.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	// 发布MQTT消息
	log.Printf("MQTT data to %s: %v", a.topic, data)
	return nil
}

// SendAlarm 发送报警
func (a *MQTTAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if !a.initialized {
		return fmt.Errorf("adapter not initialized")
	}

	// 发布MQTT消息
	log.Printf("MQTT alarm to %s: %v", a.topic, alarm)
	return nil
}

// Close 关闭
func (a *MQTTAdapter) Close() error {
	a.initialized = false
	return nil
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
