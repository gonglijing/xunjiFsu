package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// HTTPConfig HTTP配置
type HTTPConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout"` // 单位秒
}

// HTTPAdapter HTTP北向适配器
// 每个 HTTPAdapter 自己管理自己的状态和发送线程
type HTTPAdapter struct {
	name      string
	config    *HTTPConfig
	url       string
	headers   map[string]string
	timeout   time.Duration
	enabled   bool
	interval  time.Duration
	lastSend  time.Time

	// 数据缓冲
	pendingData []*models.CollectData
	pendingMu  sync.RWMutex

	// 报警缓冲
	pendingAlarms []*models.AlarmPayload
	alarmMu      sync.RWMutex

	// 控制通道
	stopChan chan struct{}
	dataChan chan struct{}
	wg       sync.WaitGroup

	// 状态
	mu          sync.RWMutex
	initialized bool
	connected   bool
}

// NewHTTPAdapter 创建HTTP适配器
func NewHTTPAdapter(name string) *HTTPAdapter {
	return &HTTPAdapter{
		name:       name,
		lastSend:   time.Time{},
		interval:   5 * time.Second, // 默认5秒
		stopChan:   make(chan struct{}),
		dataChan:   make(chan struct{}, 1),
		pendingData: make([]*models.CollectData, 0),
		pendingAlarms: make([]*models.AlarmPayload, 0),
	}
}

// Name 获取名称
func (a *HTTPAdapter) Name() string {
	return a.name
}

// Type 获取类型
func (a *HTTPAdapter) Type() string {
	return "http"
}

// Initialize 初始化
func (a *HTTPAdapter) Initialize(configStr string) error {
	cfg := &HTTPConfig{}
	if err := json.Unmarshal([]byte(configStr), cfg); err != nil {
		return fmt.Errorf("failed to parse HTTP config: %w", err)
	}

	if cfg.URL == "" {
		return fmt.Errorf("url is required")
	}

	a.mu.Lock()
	a.config = cfg
	a.url = cfg.URL
	a.headers = cfg.Headers
	if cfg.Timeout > 0 {
		a.timeout = time.Duration(cfg.Timeout) * time.Second
	} else {
		a.timeout = 30 * time.Second
	}
	a.initialized = true
	a.connected = true
	a.mu.Unlock()

	log.Printf("HTTP adapter initialized: %s (url=%s)", a.name, a.url)
	return nil
}

// Start 启动适配器的后台线程
func (a *HTTPAdapter) Start() {
	a.mu.Lock()
	if a.initialized && !a.enabled {
		a.enabled = true
		a.wg.Add(2)
		go a.sendLoop()
		go a.alarmLoop()
		log.Printf("HTTP adapter started: %s", a.name)
	}
	a.mu.Unlock()
}

// Stop 停止适配器的后台线程
func (a *HTTPAdapter) Stop() {
	a.mu.Lock()
	if a.enabled {
		a.enabled = false
		close(a.stopChan)
	}
	a.mu.Unlock()
	a.wg.Wait()
	log.Printf("HTTP adapter stopped: %s", a.name)
}

// SetInterval 设置发送周期
func (a *HTTPAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	if interval < 500*time.Millisecond {
		interval = 500 * time.Millisecond
	}
	a.interval = interval
	a.lastSend = time.Time{} // 重置上次发送时间，立即生效
	a.mu.Unlock()
}

// IsEnabled 检查是否启用
func (a *HTTPAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// IsConnected 检查连接状态
func (a *HTTPAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected
}

// Send 发送数据（加入缓冲队列）
func (a *HTTPAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.pendingMu.Lock()
	a.pendingData = append(a.pendingData, data)
	// 限制队列大小
	if len(a.pendingData) > 1000 {
		a.pendingData = a.pendingData[1:]
	}
	a.pendingMu.Unlock()

	// 触发发送
	select {
	case a.dataChan <- struct{}{}:
	default:
	}

	return nil
}

// SendAlarm 发送报警（直接发送）
func (a *HTTPAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.pendingAlarms = append(a.pendingAlarms, alarm)
	// 限制队列大小
	if len(a.pendingAlarms) > 100 {
		a.pendingAlarms = a.pendingAlarms[1:]
	}
	a.alarmMu.Unlock()

	return a.flushAlarms()
}

// Close 关闭
func (a *HTTPAdapter) Close() error {
	a.Stop()

	a.mu.Lock()
	a.initialized = false
	a.connected = false
	a.mu.Unlock()

	return nil
}

// sendLoop 数据发送循环（独立线程）
func (a *HTTPAdapter) sendLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			// 关闭前发送剩余数据
			a.flushPendingData()
			return
		case <-a.dataChan:
			a.flushPendingData()
		case <-ticker.C:
			a.flushPendingData()
		}
	}
}

// alarmLoop 报警发送循环（独立线程）
func (a *HTTPAdapter) alarmLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			a.flushAlarms()
		}
	}
}

// flushPendingData 发送待处理数据
func (a *HTTPAdapter) flushPendingData() {
	a.pendingMu.Lock()
	if len(a.pendingData) == 0 {
		a.pendingMu.Unlock()
		return
	}
	// 复制数据并清空队列
	batch := make([]*models.CollectData, len(a.pendingData))
	copy(batch, a.pendingData)
	a.pendingData = a.pendingData[:0]
	a.pendingMu.Unlock()

	// 发送每个数据点
	for _, data := range batch {
		if err := a.sendData(data); err != nil {
			log.Printf("HTTP send data failed: %s, error: %v", a.name, err)
		}
	}
}

// sendData 发送单个数据点
func (a *HTTPAdapter) sendData(data *models.CollectData) error {
	a.mu.RLock()
	if !a.initialized || !a.enabled {
		a.mu.RUnlock()
		return fmt.Errorf("adapter not initialized or disabled")
	}
	url := a.url
	headers := a.headers
	timeout := a.timeout
	a.mu.RUnlock()

	msg := map[string]interface{}{
		"device_name": data.DeviceName,
		"timestamp":   data.Timestamp,
		"fields":      data.Fields,
	}

	body, _ := json.Marshal(msg)
	return a.doRequest(url, body, headers, timeout)
}

// flushAlarms 发送报警
func (a *HTTPAdapter) flushAlarms() error {
	a.alarmMu.Lock()
	if len(a.pendingAlarms) == 0 {
		a.alarmMu.Unlock()
		return nil
	}
	// 复制报警并清空队列
	batch := make([]*models.AlarmPayload, len(a.pendingAlarms))
	copy(batch, a.pendingAlarms)
	a.pendingAlarms = a.pendingAlarms[:0]
	a.alarmMu.Unlock()

	for _, alarm := range batch {
		if err := a.sendAlarm(alarm); err != nil {
			log.Printf("HTTP send alarm failed: %s, error: %v", a.name, err)
		}
	}

	return nil
}

// sendAlarm 发送单个报警
func (a *HTTPAdapter) sendAlarm(alarm *models.AlarmPayload) error {
	a.mu.RLock()
	if !a.initialized || !a.enabled {
		a.mu.RUnlock()
		return fmt.Errorf("adapter not initialized or disabled")
	}
	url := a.url
	headers := a.headers
	timeout := a.timeout
	a.mu.RUnlock()

	body, _ := json.Marshal(alarm)
	return a.doRequest(url, body, headers, timeout)
}

// doRequest 发送HTTP请求
func (a *HTTPAdapter) doRequest(url string, body []byte, headers map[string]string, timeout time.Duration) error {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		a.mu.Lock()
		a.connected = false
		a.mu.Unlock()
		return err
	}
	defer resp.Body.Close()

	a.mu.Lock()
	a.connected = (resp.StatusCode < 400)
	a.mu.Unlock()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetStats 获取适配器统计信息
func (a *HTTPAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.pendingMu.RLock()
	pendingCount := len(a.pendingData)
	a.pendingMu.RUnlock()

	a.alarmMu.RLock()
	alarmCount := len(a.pendingAlarms)
	a.alarmMu.RUnlock()

	return map[string]interface{}{
		"name":          a.name,
		"type":         "http",
		"enabled":      a.enabled,
		"initialized":  a.initialized,
		"connected":    a.connected,
		"interval_ms":  a.interval.Milliseconds(),
		"pending_data": pendingCount,
		"pending_alarm": alarmCount,
	}
}

// GetLastSendTime 获取最后发送时间
func (a *HTTPAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

// PendingCommandCount 获取待处理命令数量
func (a *HTTPAdapter) PendingCommandCount() int {
	return 0
}
