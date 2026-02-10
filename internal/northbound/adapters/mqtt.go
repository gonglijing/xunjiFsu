package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Broker         string `json:"broker"`
	Topic          string `json:"topic"`
	AlarmTopic     string `json:"alarm_topic"`
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	QOS            int    `json:"qos"`
	Retain         bool   `json:"retain"`
	CleanSession   bool   `json:"clean_session"`
	KeepAlive      int    `json:"keep_alive"`
	ConnectTimeout int    `json:"connect_timeout"`
	UploadInterval int    `json:"upload_interval"`
}

// MQTTAdapter MQTT北向适配器
// 每个 MQTTAdapter 自己管理自己的状态和发送线程
type MQTTAdapter struct {
	name         string
	config       *MQTTConfig
	broker       string
	topic        string
	alarmTopic   string
	clientID     string
	username     string
	password     string
	qos          byte
	retain       bool
	cleanSession bool
	timeout      time.Duration
	keepAlive    time.Duration
	interval     time.Duration
	lastSend     time.Time

	// MQTT客户端
	client mqtt.Client

	// 数据缓冲
	pendingData []*models.CollectData
	pendingMu   sync.RWMutex

	// 报警缓冲
	pendingAlarms []*models.AlarmPayload
	alarmMu       sync.RWMutex

	// 控制通道
	stopChan     chan struct{}
	dataChan     chan struct{}
	reconnectNow chan struct{}
	wg           sync.WaitGroup

	// 状态
	mu                sync.RWMutex
	initialized       bool
	enabled           bool
	connected         bool
	reconnectInterval time.Duration
}

// NewMQTTAdapter 创建MQTT适配器
func NewMQTTAdapter(name string) *MQTTAdapter {
	return &MQTTAdapter{
		name:              name,
		lastSend:          time.Time{},
		interval:          5 * time.Second, // 默认5秒
		reconnectInterval: 5 * time.Second,
		stopChan:          make(chan struct{}),
		dataChan:          make(chan struct{}, 1),
		reconnectNow:      make(chan struct{}, 1),
		pendingData:       make([]*models.CollectData, 0),
		pendingAlarms:     make([]*models.AlarmPayload, 0),
	}
}

func (a *MQTTAdapter) SetReconnectInterval(interval time.Duration) {
	a.mu.Lock()
	if interval <= 0 {
		interval = 5 * time.Second
	}
	a.reconnectInterval = interval
	a.mu.Unlock()
}

// Name 获取名称
func (a *MQTTAdapter) Name() string {
	return a.name
}

// Type 获取类型
func (a *MQTTAdapter) Type() string {
	return "mqtt"
}

// Initialize 初始化
func (a *MQTTAdapter) Initialize(configStr string) error {
	cfg := &MQTTConfig{}
	if err := json.Unmarshal([]byte(configStr), cfg); err != nil {
		return fmt.Errorf("failed to parse MQTT config: %w", err)
	}

	if cfg.Broker == "" {
		return fmt.Errorf("broker is required")
	}
	if cfg.Topic == "" {
		return fmt.Errorf("topic is required")
	}

	a.mu.Lock()
	a.config = cfg
	a.broker = normalizeBroker(cfg.Broker)
	a.topic = cfg.Topic
	a.alarmTopic = cfg.AlarmTopic
	if a.alarmTopic == "" {
		a.alarmTopic = a.topic + "/alarm"
	}
	a.clientID = cfg.ClientID
	if a.clientID == "" {
		a.clientID = fmt.Sprintf("fsu-mqtt-%d", time.Now().UnixNano())
	}
	a.username = cfg.Username
	a.password = cfg.Password
	a.qos = clampQOS(cfg.QOS)
	a.retain = cfg.Retain
	a.cleanSession = cfg.CleanSession
	if cfg.ConnectTimeout > 0 {
		a.timeout = time.Duration(cfg.ConnectTimeout) * time.Second
	} else {
		a.timeout = 10 * time.Second
	}
	if cfg.KeepAlive > 0 {
		a.keepAlive = time.Duration(cfg.KeepAlive) * time.Second
	}
	if cfg.UploadInterval > 0 {
		a.interval = time.Duration(cfg.UploadInterval) * time.Millisecond
	}
	a.mu.Unlock()

	// 连接MQTT
	client, err := a.connectMQTT()
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.mu.Lock()
	a.client = client
	a.initialized = true
	a.connected = true
	a.mu.Unlock()

	log.Printf("MQTT adapter initialized: %s (broker=%s, topic=%s)", a.name, a.broker, a.topic)
	return nil
}

// connectMQTT 创建并连接MQTT客户端
func (a *MQTTAdapter) connectMQTT() (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(a.broker).
		SetClientID(a.clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second)

	if a.username != "" {
		opts.SetUsername(a.username)
	}
	if a.password != "" {
		opts.SetPassword(a.password)
	}
	if a.cleanSession {
		opts.SetCleanSession(true)
	} else {
		opts.SetCleanSession(false)
	}
	if a.keepAlive > 0 {
		opts.SetKeepAlive(a.keepAlive)
	}
	opts.SetConnectTimeout(a.timeout)

	// 连接状态回调
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		if err != nil {
			log.Printf("MQTT [%s] connection lost: %v", a.name, err)
		}
		a.markDisconnected()
	}
	opts.OnConnect = func(_ mqtt.Client) {
		log.Printf("MQTT [%s] connected: %s", a.name, a.broker)
		a.mu.Lock()
		a.connected = true
		a.mu.Unlock()
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()

	if !token.WaitTimeout(a.timeout) {
		return nil, fmt.Errorf("MQTT [%s] connect timeout", a.name)
	}
	if err := token.Error(); err != nil {
		return nil, err
	}

	return client, nil
}

func (a *MQTTAdapter) signalReconnect() {
	a.mu.RLock()
	reconnectNow := a.reconnectNow
	a.mu.RUnlock()
	if reconnectNow == nil {
		return
	}
	select {
	case reconnectNow <- struct{}{}:
	default:
	}
}

func (a *MQTTAdapter) markDisconnected() {
	a.mu.Lock()
	a.connected = false
	enabled := a.enabled
	a.mu.Unlock()
	if enabled {
		a.signalReconnect()
	}
}

func (a *MQTTAdapter) reconnectOnce() error {
	log.Printf("MQTT [%s] attempting to reconnect...", a.name)
	client, err := a.connectMQTT()
	if err != nil {
		return err
	}

	a.mu.Lock()
	oldClient := a.client
	a.client = client
	a.connected = true
	a.mu.Unlock()

	if oldClient != nil && oldClient != client && oldClient.IsConnected() {
		oldClient.Disconnect(250)
	}

	log.Printf("MQTT [%s] reconnected successfully", a.name)
	return nil
}

// Start 启动适配器的后台线程
func (a *MQTTAdapter) Start() {
	needReconnect := false
	a.mu.Lock()
	if a.initialized && !a.enabled {
		if a.stopChan == nil {
			a.stopChan = make(chan struct{})
		}
		if a.dataChan == nil {
			a.dataChan = make(chan struct{}, 1)
		}
		if a.reconnectNow == nil {
			a.reconnectNow = make(chan struct{}, 1)
		}
		a.enabled = true
		needReconnect = !a.connected
		a.wg.Add(1)
		go a.runLoop()
		log.Printf("MQTT adapter started: %s", a.name)
	}
	a.mu.Unlock()

	if needReconnect {
		a.signalReconnect()
	}
}

// Stop 停止适配器的后台线程
func (a *MQTTAdapter) Stop() {
	a.mu.Lock()
	stopChan := a.stopChan
	if a.enabled {
		a.enabled = false
		if stopChan != nil {
			close(stopChan)
		}
	}
	a.mu.Unlock()
	a.wg.Wait()
	if stopChan != nil {
		a.mu.Lock()
		if a.stopChan == stopChan {
			a.stopChan = nil
		}
		a.mu.Unlock()
	}
	log.Printf("MQTT adapter stopped: %s", a.name)
}

// SetInterval 设置发送周期
func (a *MQTTAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	if interval < 500*time.Millisecond {
		interval = 500 * time.Millisecond
	}
	a.interval = interval
	a.lastSend = time.Time{}
	a.mu.Unlock()
}

// IsEnabled 检查是否启用
func (a *MQTTAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// IsConnected 检查连接状态
func (a *MQTTAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

// Send 发送数据（加入缓冲队列）
func (a *MQTTAdapter) Send(data *models.CollectData) error {
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

// SendAlarm 发送报警
func (a *MQTTAdapter) SendAlarm(alarm *models.AlarmPayload) error {
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

	select {
	case a.dataChan <- struct{}{}:
	default:
	}

	return nil
}

// Close 关闭
func (a *MQTTAdapter) Close() error {
	a.Stop()

	a.flushPendingData()
	_ = a.flushAlarms()

	a.mu.Lock()
	client := a.client
	a.initialized = false
	a.connected = false
	a.enabled = false
	a.client = nil
	a.stopChan = nil
	a.dataChan = nil
	a.reconnectNow = nil
	a.mu.Unlock()

	if client != nil && client.IsConnected() {
		client.Disconnect(250)
	}

	return nil
}

// runLoop 单协程事件循环（数据/报警/重连）
func (a *MQTTAdapter) runLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.interval
	stopChan := a.stopChan
	dataChan := a.dataChan
	reconnectNow := a.reconnectNow
	a.mu.RUnlock()

	if interval < 500*time.Millisecond {
		interval = 500 * time.Millisecond
	}

	dataTicker := time.NewTicker(interval)
	alarmTicker := time.NewTicker(2 * time.Second)
	defer dataTicker.Stop()
	defer alarmTicker.Stop()

	var reconnectTimer *time.Timer
	var reconnectTimerCh <-chan time.Time

	scheduleReconnect := func(delay time.Duration) {
		if reconnectTimer == nil {
			reconnectTimer = time.NewTimer(delay)
			reconnectTimerCh = reconnectTimer.C
			return
		}
		if !reconnectTimer.Stop() {
			select {
			case <-reconnectTimer.C:
			default:
			}
		}
		reconnectTimer.Reset(delay)
		reconnectTimerCh = reconnectTimer.C
	}

	stopReconnect := func() {
		if reconnectTimer == nil {
			reconnectTimerCh = nil
			return
		}
		if !reconnectTimer.Stop() {
			select {
			case <-reconnectTimer.C:
			default:
			}
		}
		reconnectTimerCh = nil
	}

	defer func() {
		if reconnectTimer != nil {
			if !reconnectTimer.Stop() {
				select {
				case <-reconnectTimer.C:
				default:
				}
			}
		}
	}()

	for {
		select {
		case <-stopChan:
			a.flushPendingData()
			_ = a.flushAlarms()
			stopReconnect()
			return
		case <-dataChan:
			a.flushPendingData()
			_ = a.flushAlarms()
		case <-dataTicker.C:
			a.flushPendingData()
		case <-alarmTicker.C:
			_ = a.flushAlarms()
		case <-reconnectNow:
			scheduleReconnect(0)
		case <-reconnectTimerCh:
			if !a.shouldReconnect() {
				stopReconnect()
				continue
			}
			if err := a.reconnectOnce(); err != nil {
				delay := a.currentReconnectInterval()
				log.Printf("MQTT [%s] reconnect failed: %v, retry in %v", a.name, err, delay)
				scheduleReconnect(delay)
				continue
			}
			stopReconnect()
		}
	}
}

// flushPendingData 发送待处理数据
func (a *MQTTAdapter) flushPendingData() {
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
		if err := a.publish(a.topic, data); err != nil {
			log.Printf("MQTT [%s] send data failed: %v", a.name, err)
			// 断线重连
			if !a.IsConnected() {
				a.signalReconnect()
			}
		}
	}
}

// flushAlarms 发送报警
func (a *MQTTAdapter) flushAlarms() error {
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
		if err := a.publish(a.alarmTopic, alarm); err != nil {
			log.Printf("MQTT [%s] send alarm failed: %v", a.name, err)
			// 断线重连
			if !a.IsConnected() {
				a.signalReconnect()
			}
		}
	}

	return nil
}

// publish 发布消息
func (a *MQTTAdapter) publish(topic string, payload interface{}) error {
	a.mu.RLock()
	if !a.initialized || !a.enabled {
		a.mu.RUnlock()
		return fmt.Errorf("adapter not initialized or disabled")
	}
	client := a.client
	qos := a.qos
	retain := a.retain
	timeout := a.timeout
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	var body []byte
	if data, ok := payload.(*models.CollectData); ok {
		msg := map[string]interface{}{
			"device_name": data.DeviceName,
			"device_id":   data.DeviceID,
			"timestamp":   data.Timestamp.Unix(),
			"fields":      data.Fields,
		}
		body, _ = json.Marshal(msg)
	} else if alarm, ok := payload.(*models.AlarmPayload); ok {
		msg := map[string]interface{}{
			"device_id":    alarm.DeviceID,
			"device_name":  alarm.DeviceName,
			"field_name":   alarm.FieldName,
			"actual_value": alarm.ActualValue,
			"threshold":    alarm.Threshold,
			"operator":     alarm.Operator,
			"severity":     alarm.Severity,
			"message":      alarm.Message,
			"timestamp":    time.Now().Unix(),
		}
		body, _ = json.Marshal(msg)
	} else {
		return fmt.Errorf("unknown payload type")
	}

	token := client.Publish(topic, qos, retain, body)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		a.markDisconnected()
		return err
	}

	a.mu.Lock()
	a.lastSend = time.Now()
	a.connected = true
	a.mu.Unlock()

	return nil
}

func (a *MQTTAdapter) currentReconnectInterval() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.reconnectInterval <= 0 {
		return 5 * time.Second
	}
	return a.reconnectInterval
}

func (a *MQTTAdapter) shouldReconnect() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if !a.initialized || !a.enabled || a.client == nil {
		return false
	}
	return !a.client.IsConnected()
}

// GetStats 获取适配器统计信息
func (a *MQTTAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	enabled := a.enabled
	initialized := a.initialized
	connected := a.connected
	interval := a.interval
	a.mu.RUnlock()

	a.pendingMu.RLock()
	pendingCount := len(a.pendingData)
	a.pendingMu.RUnlock()

	a.alarmMu.RLock()
	alarmCount := len(a.pendingAlarms)
	a.alarmMu.RUnlock()

	return map[string]interface{}{
		"name":          a.name,
		"type":          "mqtt",
		"enabled":       enabled,
		"initialized":   initialized,
		"connected":     connected,
		"interval_ms":   interval.Milliseconds(),
		"pending_data":  pendingCount,
		"pending_alarm": alarmCount,
		"broker":        a.broker,
		"topic":         a.topic,
		"alarm_topic":   a.alarmTopic,
		"client_id":     a.clientID,
		"qos":           a.qos,
		"retain":        a.retain,
	}
}

// GetLastSendTime 获取最后发送时间
func (a *MQTTAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

// PendingCommandCount 获取待处理命令数量
func (a *MQTTAdapter) PendingCommandCount() int {
	return 0
}

// 连接MQTT（复制自 plugin_north/adapter/helpers.go）
func connectMQTT(broker, clientID, username, password string, keepAliveSec, timeoutSec int) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	if keepAliveSec > 0 {
		opts.SetKeepAlive(time.Duration(keepAliveSec) * time.Second)
	}
	if timeoutSec > 0 {
		opts.SetConnectTimeout(time.Duration(timeoutSec) * time.Second)
	} else {
		opts.SetConnectTimeout(10 * time.Second)
	}
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		if err != nil {
			log.Printf("MQTT connection lost: %v", err)
		}
	}
	opts.OnConnect = func(_ mqtt.Client) {
		log.Printf("MQTT connected: %s", broker)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	timeout := 10 * time.Second
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}
	if !token.WaitTimeout(timeout) {
		return nil, fmt.Errorf("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return client, nil
}

func normalizeBroker(broker string) string {
	broker = strings.TrimSpace(broker)
	if broker == "" {
		return ""
	}
	// 如果已经有协议前缀，直接返回
	if strings.Contains(broker, "://") {
		return broker
	}
	// 默认添加 tcp:// 前缀
	return "tcp://" + broker
}

func clampQOS(qos int) byte {
	if qos < 0 {
		return 0
	}
	if qos > 2 {
		return 2
	}
	return byte(qos)
}
