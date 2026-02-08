package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// XunJiConfig 循迹北向配置（与 models.XunJiConfig 保持一致）
type XunJiConfig struct {
	ProductKey string `json:"productKey"`
	DeviceKey  string `json:"deviceKey"`
	ServerURL  string `json:"serverUrl"` // MQTT服务器地址
	Username   string `json:"username"`
	Password   string `json:"password"`
	Topic      string `json:"topic"`
	AlarmTopic string `json:"alarmTopic"`
	ClientID   string `json:"clientId"`
	QOS        int    `json:"qos"`
	Retain     bool   `json:"retain"`
	KeepAlive  int    `json:"keepAlive"`      // 秒
	Timeout    int    `json:"connectTimeout"` // 秒
	// 插件上报周期（毫秒）。<=0 使用默认值。
	UploadIntervalMs int `json:"uploadIntervalMs"`
	// 兼容旧字段：插件内管理上传节奏（毫秒）。<=0 使用默认值。
	ReportIntervalMs int `json:"reportIntervalMs"`
	// 报警批量发送周期（毫秒）。<=0 使用默认值。
	AlarmFlushIntervalMs int `json:"alarmFlushIntervalMs"`
	// 单次批量发送报警条数。<=0 使用默认值。
	AlarmBatchSize int `json:"alarmBatchSize"`
	// 内部报警队列大小（超出后丢弃最旧数据）。<=0 使用默认值。
	AlarmQueueSize int `json:"alarmQueueSize"`
	// 内部实时队列大小（超出后丢弃最旧数据）。<=0 使用默认值。
	RealtimeQueueSize int `json:"realtimeQueueSize"`
}

const (
	defaultReportInterval = 5 * time.Second
	defaultAlarmInterval  = 2 * time.Second
	defaultAlarmBatch     = 20
	defaultAlarmQueue     = 1000
	defaultRealtimeQueue  = 1000
)

// XunJiAdapter 循迹北向适配器
// 每个 XunJiAdapter 自己管理自己的状态和发送线程
type XunJiAdapter struct {
	name        string
	config      *XunJiConfig
	client      mqtt.Client
	topic       string
	alarmTopic  string
	qos         byte
	retain      bool
	timeout     time.Duration
	enabled     bool
	reportEvery time.Duration
	alarmEvery  time.Duration
	alarmBatch  int
	alarmCap    int
	realtimeCap int
	commandCap  int

	// 数据缓冲
	latestData []*models.CollectData
	dataMu     sync.RWMutex

	// 报警缓冲
	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	// 命令队列
	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

	// 控制通道
	flushNow chan struct{}
	stopChan chan struct{}
	wg       sync.WaitGroup

	// 状态
	mu          sync.RWMutex
	initialized bool
	connected   bool
	seq         uint64
}

// NewXunJiAdapter 创建循迹适配器
func NewXunJiAdapter(name string) *XunJiAdapter {
	return &XunJiAdapter{
		name:         name,
		stopChan:     make(chan struct{}),
		flushNow:     make(chan struct{}, 1),
		latestData:   make([]*models.CollectData, 0),
		alarmQueue:   make([]*models.AlarmPayload, 0),
		commandQueue: make([]*models.NorthboundCommand, 0),
	}
}

// Name 获取名称
func (a *XunJiAdapter) Name() string {
	return a.name
}

// Type 获取类型
func (a *XunJiAdapter) Type() string {
	return "xunji"
}

// Initialize 初始化
func (a *XunJiAdapter) Initialize(configStr string) error {
	cfg, err := parseXunJiConfig(configStr)
	if err != nil {
		return err
	}

	// 清理旧连接
	_ = a.Close()

	broker := normalizeBroker(cfg.ServerURL)
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("xunji-%s-%s-%d", cfg.ProductKey, cfg.DeviceKey, time.Now().UnixNano())
	}

	qos := clampQOS(cfg.QOS)
	retain := cfg.Retain
	topic := strings.TrimSpace(cfg.Topic)
	if !strings.HasPrefix(topic, "/sys/") {
		topic = fmt.Sprintf("/sys/%s/%s/thing/event/property/pack/post", cfg.ProductKey, cfg.DeviceKey)
	}
	alarmTopic := strings.TrimSpace(cfg.AlarmTopic)
	if !strings.HasPrefix(alarmTopic, "/sys/") {
		alarmTopic = topic
	}

	timeout := 10 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	// 连接 MQTT
	client, err := connectMQTT(broker, clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	uploadMs := cfg.UploadIntervalMs
	if uploadMs <= 0 {
		uploadMs = cfg.ReportIntervalMs
	}

	a.mu.Lock()
	a.config = cfg
	a.client = client
	a.qos = qos
	a.retain = retain
	a.topic = topic
	a.alarmTopic = alarmTopic
	a.timeout = timeout
	a.reportEvery = resolveInterval(uploadMs, defaultReportInterval)
	a.alarmEvery = resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval)
	a.alarmBatch = resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch)
	a.alarmCap = resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue)
	a.realtimeCap = resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue)
	a.commandCap = resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue)
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.initialized = true
	a.connected = true
	a.mu.Unlock()

	// 订阅命令主题
	a.subscribeCommandTopics(client)

	log.Printf("XunJi adapter initialized: %s (broker=%s, topic=%s)",
		a.name, broker, topic)
	return nil
}

// Start 启动适配器的后台线程
func (a *XunJiAdapter) Start() {
	a.mu.Lock()
	if a.initialized && !a.enabled {
		a.enabled = true
		a.wg.Add(2)
		go a.reportLoop()
		go a.alarmLoop()
		log.Printf("XunJi adapter started: %s", a.name)
	}
	a.mu.Unlock()
}

// Stop 停止适配器的后台线程
func (a *XunJiAdapter) Stop() {
	a.mu.Lock()
	if a.enabled {
		a.enabled = false
		close(a.stopChan)
	}
	a.mu.Unlock()
	a.wg.Wait()
	log.Printf("XunJi adapter stopped: %s", a.name)
}

// SetInterval 设置发送周期
func (a *XunJiAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = interval
	a.mu.Unlock()
}

// IsEnabled 检查是否启用
func (a *XunJiAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// IsConnected 检查连接状态
func (a *XunJiAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

// Send 发送数据（加入缓冲队列）
func (a *XunJiAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(cloneCollectData(data))
	a.dataMu.Unlock()

	return nil
}

// SendAlarm 发送报警
func (a *XunJiAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.enqueueAlarmLocked(cloneAlarmPayload(alarm))
	needFlush := len(a.alarmQueue) >= a.alarmBatch
	flushNow := a.flushNow
	a.alarmMu.Unlock()

	if needFlush {
		select {
		case flushNow <- struct{}{}:
		default:
		}
	}

	return nil
}

// Close 关闭
func (a *XunJiAdapter) Close() error {
	a.Stop()

	a.mu.Lock()
	stopChan := a.stopChan
	client := a.client
	a.initialized = false
	a.connected = false
	a.mu.Unlock()

	if stopChan != nil {
		close(stopChan)
		a.wg.Wait()
	}

	// 刷新剩余数据
	_ = a.flushLatestData()
	_ = a.flushAlarmBatch()

	if client != nil && client.IsConnected() {
		client.Disconnect(250)
	}

	a.mu.Lock()
	a.stopChan = nil
	a.flushNow = nil
	a.client = nil
	a.config = nil
	a.latestData = nil
	a.alarmQueue = nil
	a.commandQueue = nil
	a.mu.Unlock()

	return nil
}

// PullCommands 拉取待执行命令
func (a *XunJiAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	if limit <= 0 {
		limit = 20
	}

	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	if !a.isInitialized() {
		return nil, fmt.Errorf("adapter not initialized")
	}

	if len(a.commandQueue) == 0 {
		return nil, nil
	}

	if limit > len(a.commandQueue) {
		limit = len(a.commandQueue)
	}

	out := make([]*models.NorthboundCommand, 0, limit)
	for i := 0; i < limit; i++ {
		cmd := a.commandQueue[0]
		a.commandQueue = a.commandQueue[1:]
		out = append(out, cmd)
	}

	return out, nil
}

// ReportCommandResult 上报命令执行结果
func (a *XunJiAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	if result == nil {
		return nil
	}

	a.mu.RLock()
	initialized := a.initialized
	cfg := a.config
	a.mu.RUnlock()

	if !initialized || cfg == nil {
		return nil
	}

	pk := pickFirstNonEmpty(strings.TrimSpace(cfg.ProductKey), result.ProductKey)
	dk := pickFirstNonEmpty(strings.TrimSpace(cfg.DeviceKey), result.DeviceKey)
	if pk == "" || dk == "" {
		return nil
	}

	code := result.Code
	if code == 0 && result.Success {
		code = 200
	}
	msg := result.Message
	if msg == "" && result.Success {
		msg = "success"
	}

	resp := map[string]interface{}{
		"code":    code,
		"id":      result.RequestID,
		"message": msg,
		"version": "1.0.0",
		"data": map[string]interface{}{
			result.FieldName: result.Value,
		},
	}
	body, _ := json.Marshal(resp)

	topic := fmt.Sprintf("/sys/%s/%s/thing/service/property/set_reply", pk, dk)
	return a.publish(topic, body)
}

// reportLoop 数据上报循环（独立线程）
func (a *XunJiAdapter) reportLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.reportEvery
	a.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			return
		case <-ticker.C:
			if err := a.flushLatestData(); err != nil {
				log.Printf("XunJi report flush failed: %v", err)
			}
		}
	}
}

// alarmLoop 报警发送循环（独立线程）
func (a *XunJiAdapter) alarmLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.alarmEvery
	flushNow := a.flushNow
	a.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopChan:
			// 关闭前发送剩余报警
			for {
				if err := a.flushAlarmBatch(); err != nil {
					log.Printf("XunJi alarm flush failed on close: %v", err)
					return
				}
				a.alarmMu.RLock()
				empty := len(a.alarmQueue) == 0
				a.alarmMu.RUnlock()
				if empty {
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		case <-ticker.C:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("XunJi alarm flush failed: %v", err)
			}
		case <-flushNow:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("XunJi alarm flush failed: %v", err)
			}
		}
	}
}

// flushLatestData 发送实时数据
func (a *XunJiAdapter) flushLatestData() error {
	a.dataMu.Lock()
	if len(a.latestData) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := make([]*models.CollectData, len(a.latestData))
	for i := 0; i < len(a.latestData); i++ {
		batch[i] = cloneCollectData(a.latestData[i])
	}
	a.latestData = a.latestData[:0]
	topic := a.topic
	a.dataMu.Unlock()

	for _, data := range batch {
		message := a.buildMessage(data)
		if err := a.publish(topic, []byte(message)); err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			return err
		}
	}

	return nil
}

// flushAlarmBatch 发送报警批次
func (a *XunJiAdapter) flushAlarmBatch() error {
	a.alarmMu.Lock()
	if len(a.alarmQueue) == 0 {
		a.alarmMu.Unlock()
		return nil
	}
	count := a.alarmBatch
	if count > len(a.alarmQueue) {
		count = len(a.alarmQueue)
	}
	batch := make([]*models.AlarmPayload, count)
	copy(batch, a.alarmQueue[:count])
	a.alarmQueue = a.alarmQueue[count:]
	topic := a.alarmTopic
	a.alarmMu.Unlock()

	for _, alarm := range batch {
		message := a.buildAlarmMessage(alarm)
		if err := a.publish(topic, []byte(message)); err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
	}

	return nil
}

// buildMessage 构建循迹消息
func (a *XunJiAdapter) buildMessage(data *models.CollectData) string {
	if data == nil {
		return "{}"
	}

	properties := make(map[string]interface{})
	for key, value := range data.Fields {
		properties[key] = convertFieldValue(value)
	}

	defaultPK, defaultDK := a.defaultIdentity()
	subPK := strings.TrimSpace(data.ProductKey)
	subDK := strings.TrimSpace(data.DeviceKey)
	if subPK == "" {
		subPK = defaultPK
	}
	if subDK == "" {
		subDK = defaultDK
	}

	msg := map[string]interface{}{
		"id":      a.nextID("msg"),
		"version": "1.0",
		"sys": map[string]interface{}{
			"ack": 0,
		},
		"method": "thing.event.property.pack.post",
		"params": map[string]interface{}{
			"properties": map[string]interface{}{},
			"events":     map[string]interface{}{},
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]string{
						"productKey": subPK,
						"deviceKey":  subDK,
					},
					"properties": properties,
					"events":     map[string]interface{}{},
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(msg)
	return string(jsonBytes)
}

// buildAlarmMessage 构建报警消息
func (a *XunJiAdapter) buildAlarmMessage(alarm *models.AlarmPayload) string {
	if alarm == nil {
		return "{}"
	}

	defaultPK, defaultDK := a.defaultIdentity()

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
		"id":      a.nextID("alarm"),
		"version": "1.0",
		"sys": map[string]interface{}{
			"ack": 0,
		},
		"method": "thing.event.property.pack.post",
		"params": map[string]interface{}{
			"properties": map[string]interface{}{},
			"events":     map[string]interface{}{},
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]string{
						"productKey": pickFirstNonEmpty(alarm.ProductKey, defaultPK),
						"deviceKey":  pickFirstNonEmpty(alarm.DeviceKey, defaultDK),
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

// publish 发布MQTT消息
func (a *XunJiAdapter) publish(topic string, payload []byte) error {
	a.mu.RLock()
	client := a.client
	timeout := a.timeout
	qos := a.qos
	retain := a.retain
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("mqtt client is nil")
	}

	if !client.IsConnected() {
		token := client.Connect()
		if !token.WaitTimeout(timeout) {
			return fmt.Errorf("mqtt connect timeout")
		}
		if err := token.Error(); err != nil {
			return err
		}
		a.subscribeCommandTopics(client)
	}

	token := client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		return err
	}

	return nil
}

// subscribeCommandTopics 订阅命令主题
func (a *XunJiAdapter) subscribeCommandTopics(client mqtt.Client) {
	a.mu.RLock()
	cfg := a.config
	a.mu.RUnlock()

	if cfg == nil {
		return
	}

	pk := strings.TrimSpace(cfg.ProductKey)
	dk := strings.TrimSpace(cfg.DeviceKey)
	if pk == "" || dk == "" {
		return
	}

	a.subscribe(client, fmt.Sprintf("/sys/%s/%s/thing/service/property/set", pk, dk), a.handlePropertySet)
	a.subscribe(client, fmt.Sprintf("/sys/%s/%s/thing/service/+", pk, dk), a.handleServiceCall)
	a.subscribe(client, fmt.Sprintf("/sys/%s/%s/thing/config/push", pk, dk), a.handleConfigPush)
}

func (a *XunJiAdapter) subscribe(client mqtt.Client, topic string, handler mqtt.MessageHandler) {
	a.mu.RLock()
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	token := client.Subscribe(topic, qos, handler)
	if !token.WaitTimeout(timeout) {
		return
	}
}

func (a *XunJiAdapter) handlePropertySet(_ mqtt.Client, message mqtt.Message) {
	pk, dk, ok := extractIdentity(message.Topic())
	if !ok {
		return
	}

	var req struct {
		Id       string                 `json:"id"`
		Params   map[string]interface{} `json:"params"`
		Identity map[string]interface{} `json:"identity"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	identityPK, identityDK := parseIdentityMap(req.Identity)
	a.enqueueCommandFromPropertySet(pk, dk, req.Id, req.Params, identityPK, identityDK)

	resp := map[string]interface{}{
		"code":    200,
		"data":    req.Params,
		"id":      req.Id,
		"message": "success",
		"version": "1.0.0",
	}
	respBody, _ := json.Marshal(resp)
	_ = a.publish(fmt.Sprintf("/sys/%s/%s/thing/service/property/set_reply", pk, dk), respBody)
}

func (a *XunJiAdapter) handleServiceCall(_ mqtt.Client, message mqtt.Message) {
	parts := splitTopic(message.Topic())
	if len(parts) != 7 {
		return
	}

	pk, dk, svc := parts[1], parts[2], parts[6]
	if strings.HasSuffix(svc, "reply") || svc == "property" {
		return
	}

	var req struct {
		Id       string                 `json:"id"`
		Params   map[string]interface{} `json:"params"`
		Identity map[string]interface{} `json:"identity"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	identityPK, identityDK := parseIdentityMap(req.Identity)
	if len(req.Params) > 0 {
		a.enqueueCommandFromPropertySet(pk, dk, req.Id, req.Params, identityPK, identityDK)
	}

	resp := map[string]interface{}{
		"code":    200,
		"data":    req.Params,
		"id":      req.Id,
		"message": "success",
		"version": "1.0.0",
	}
	respBody, _ := json.Marshal(resp)
	_ = a.publish(fmt.Sprintf("/sys/%s/%s/thing/service/%s_reply", pk, dk, svc), respBody)
}

func (a *XunJiAdapter) handleConfigPush(_ mqtt.Client, message mqtt.Message) {
	pk, dk, ok := extractIdentity(message.Topic())
	if !ok {
		return
	}

	var req struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	resp := map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{},
		"id":   req.Id,
	}
	respBody, _ := json.Marshal(resp)
	_ = a.publish(fmt.Sprintf("/sys/%s/%s/thing/config/push/reply", pk, dk), respBody)
}

func (a *XunJiAdapter) enqueueCommandFromPropertySet(defaultPK, defaultDK, requestID string, params map[string]interface{}, rootIdentityPK, rootIdentityDK string) {
	properties, identityPK, identityDK := extractCommandProperties(params)
	if len(properties) == 0 {
		return
	}

	pk := pickFirstNonEmpty(rootIdentityPK, identityPK, defaultPK)
	dk := pickFirstNonEmpty(rootIdentityDK, identityDK, defaultDK)
	if pk == "" || dk == "" {
		return
	}

	keys := make([]string, 0, len(properties))
	for key := range properties {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	if len(keys) == 0 {
		return
	}
	sort.Strings(keys)

	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	for _, key := range keys {
		raw := properties[key]
		command := &models.NorthboundCommand{
			RequestID:  requestID,
			ProductKey: pk,
			DeviceKey:  dk,
			FieldName:  key,
			Value:      stringifyAny(raw),
			Source:     "xunji.property.set",
		}
		if len(a.commandQueue) >= a.commandCap && len(a.commandQueue) > 0 {
			a.commandQueue = a.commandQueue[1:]
		}
		a.commandQueue = append(a.commandQueue, command)
	}
}

// 辅助函数
func (a *XunJiAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func (a *XunJiAdapter) defaultIdentity() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config == nil {
		return "", ""
	}
	return strings.TrimSpace(a.config.ProductKey), strings.TrimSpace(a.config.DeviceKey)
}

func (a *XunJiAdapter) nextID(prefix string) string {
	n := atomic.AddUint64(&a.seq, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), n)
}

func (a *XunJiAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	if len(a.latestData) >= a.realtimeCap {
		a.latestData = a.latestData[1:]
	}
	a.latestData = append(a.latestData, item)
}

func (a *XunJiAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	queue := make([]*models.CollectData, 0, len(items)+len(a.latestData))
	queue = append(queue, items...)
	queue = append(queue, a.latestData...)
	if len(queue) > a.realtimeCap {
		queue = queue[:a.realtimeCap]
	}
	a.latestData = queue
}

func (a *XunJiAdapter) enqueueAlarmLocked(alarm *models.AlarmPayload) {
	if alarm == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	if len(a.alarmQueue) >= a.alarmCap {
		a.alarmQueue = a.alarmQueue[1:]
	}
	a.alarmQueue = append(a.alarmQueue, alarm)
}

func (a *XunJiAdapter) prependAlarms(alarms []*models.AlarmPayload) {
	if len(alarms) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	queue := make([]*models.AlarmPayload, 0, len(alarms)+len(a.alarmQueue))
	queue = append(queue, alarms...)
	queue = append(queue, a.alarmQueue...)
	if len(queue) > a.alarmCap {
		queue = queue[:a.alarmCap]
	}
	a.alarmQueue = queue
}

// GetStats 获取适配器统计信息
func (a *XunJiAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	productKey := ""
	deviceKey := ""
	if a.config != nil {
		productKey = strings.TrimSpace(a.config.ProductKey)
		deviceKey = strings.TrimSpace(a.config.DeviceKey)
	}
	defer a.mu.RUnlock()

	a.dataMu.RLock()
	dataCount := len(a.latestData)
	a.dataMu.RUnlock()

	a.alarmMu.RLock()
	alarmCount := len(a.alarmQueue)
	a.alarmMu.RUnlock()

	a.commandMu.RLock()
	commandCount := len(a.commandQueue)
	a.commandMu.RUnlock()

	return map[string]interface{}{
		"name":          a.name,
		"type":          "xunji",
		"enabled":       a.enabled,
		"initialized":   a.initialized,
		"connected":     a.connected && a.client != nil && a.client.IsConnected(),
		"interval_ms":   a.reportEvery.Milliseconds(),
		"pending_data":  dataCount,
		"pending_alarm": alarmCount,
		"pending_cmd":   commandCount,
		"product_key":   productKey,
		"device_key":    deviceKey,
	}
}

// GetLastSendTime 获取最后发送时间（返回零值，因为是内部管理）
func (a *XunJiAdapter) GetLastSendTime() time.Time {
	return time.Time{}
}

// PendingCommandCount 获取待处理命令数量
func (a *XunJiAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}

// 辅助函数
func parseXunJiConfig(configStr string) (*XunJiConfig, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &XunJiConfig{}
	cfg.ProductKey = strings.TrimSpace(xunjiPickString(raw, "productKey", "product_key", "productID", "product_id"))
	cfg.DeviceKey = strings.TrimSpace(xunjiPickString(raw, "deviceKey", "device_key", "deviceName", "device_name"))
	cfg.ServerURL = normalizeServerURLWithPort(
		xunjiPickString(raw, "serverUrl", "server_url", "broker"),
		xunjiPickString(raw, "protocol"),
		xunjiPickInt(raw, 0, "port"),
	)
	cfg.Username = strings.TrimSpace(xunjiPickString(raw, "username"))
	cfg.Password = strings.TrimSpace(xunjiPickString(raw, "password"))
	cfg.Topic = strings.TrimSpace(xunjiPickString(raw, "topic"))
	cfg.AlarmTopic = strings.TrimSpace(xunjiPickString(raw, "alarmTopic", "alarm_topic"))
	cfg.ClientID = strings.TrimSpace(xunjiPickString(raw, "clientId", "client_id"))
	cfg.QOS = xunjiPickInt(raw, 0, "qos")
	cfg.Retain = xunjiPickBool(raw, false, "retain")
	cfg.KeepAlive = xunjiPickInt(raw, 60, "keepAlive", "keep_alive")
	cfg.Timeout = xunjiPickInt(raw, 10, "connectTimeout", "connect_timeout", "timeout")
	cfg.UploadIntervalMs = xunjiPickInt(raw, 5000, "uploadIntervalMs", "upload_interval_ms")
	cfg.ReportIntervalMs = xunjiPickInt(raw, 0, "reportIntervalMs", "report_interval_ms")
	cfg.AlarmFlushIntervalMs = xunjiPickInt(raw, 2000, "alarmFlushIntervalMs", "alarm_flush_interval_ms")
	cfg.AlarmBatchSize = xunjiPickInt(raw, 20, "alarmBatchSize", "alarm_batch_size")
	cfg.AlarmQueueSize = xunjiPickInt(raw, 1000, "alarmQueueSize", "alarm_queue_size")
	cfg.RealtimeQueueSize = xunjiPickInt(raw, 1000, "realtimeQueueSize", "realtime_queue_size")

	if cfg.ProductKey == "" {
		return nil, fmt.Errorf("productKey is required")
	}
	if cfg.DeviceKey == "" {
		return nil, fmt.Errorf("deviceKey is required")
	}
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("serverUrl is required")
	}
	if cfg.QOS < 0 || cfg.QOS > 2 {
		return nil, fmt.Errorf("qos must be between 0 and 2")
	}
	// 设置默认值
	if cfg.QOS <= 0 {
		cfg.QOS = 0
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10
	}
	if cfg.UploadIntervalMs <= 0 && cfg.ReportIntervalMs <= 0 {
		cfg.UploadIntervalMs = 5000
	}
	if cfg.AlarmFlushIntervalMs <= 0 {
		cfg.AlarmFlushIntervalMs = 2000
	}
	if cfg.AlarmBatchSize <= 0 {
		cfg.AlarmBatchSize = 20
	}
	if cfg.AlarmQueueSize <= 0 {
		cfg.AlarmQueueSize = 1000
	}
	if cfg.RealtimeQueueSize <= 0 {
		cfg.RealtimeQueueSize = 1000
	}

	return cfg, nil
}

func xunjiPickString(data map[string]interface{}, keys ...string) string {
	return pickConfigString(data, keys...)
}

func xunjiPickInt(data map[string]interface{}, fallback int, keys ...string) int {
	return pickConfigInt(data, fallback, keys...)
}

func xunjiPickBool(data map[string]interface{}, fallback bool, keys ...string) bool {
	return pickConfigBool(data, fallback, keys...)
}

func cloneCollectData(data *models.CollectData) *models.CollectData {
	if data == nil {
		return nil
	}
	out := *data
	if len(data.Fields) > 0 {
		out.Fields = make(map[string]string, len(data.Fields))
		for key, value := range data.Fields {
			out.Fields[key] = value
		}
	} else {
		out.Fields = map[string]string{}
	}
	return &out
}

func cloneAlarmPayload(alarm *models.AlarmPayload) *models.AlarmPayload {
	if alarm == nil {
		return nil
	}
	out := *alarm
	return &out
}

func splitTopic(topic string) []string {
	raw := strings.Split(strings.TrimSpace(topic), "/")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func extractIdentity(topic string) (string, string, bool) {
	parts := splitTopic(topic)
	if len(parts) < 4 || parts[0] != "sys" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func extractCommandProperties(params map[string]interface{}) (map[string]interface{}, string, string) {
	identityPK, identityDK := "", ""
	if identity, ok := mapFromAny(params["identity"]); ok {
		if v, ok := identity["productKey"].(string); ok {
			identityPK = strings.TrimSpace(v)
		}
		if v, ok := identity["deviceKey"].(string); ok {
			identityDK = strings.TrimSpace(v)
		}
	}

	if props, ok := mapFromAny(params["properties"]); ok {
		return props, identityPK, identityDK
	}

	for _, key := range []string{"sub_device", "subDevice"} {
		sub, ok := mapFromAny(params[key])
		if !ok {
			continue
		}
		if identity, ok := mapFromAny(sub["identity"]); ok {
			if v, ok := identity["productKey"].(string); ok {
				identityPK = strings.TrimSpace(v)
			}
			if v, ok := identity["deviceKey"].(string); ok {
				identityDK = strings.TrimSpace(v)
			}
		}
		if props, ok := mapFromAny(sub["properties"]); ok {
			return props, identityPK, identityDK
		}
	}

	for _, key := range []string{"sub_devices", "subDevices"} {
		list, ok := params[key].([]interface{})
		if !ok || len(list) == 0 {
			continue
		}
		item, ok := mapFromAny(list[0])
		if !ok {
			continue
		}
		if identity, ok := mapFromAny(item["identity"]); ok {
			if v, ok := identity["productKey"].(string); ok {
				identityPK = strings.TrimSpace(v)
			}
			if v, ok := identity["deviceKey"].(string); ok {
				identityDK = strings.TrimSpace(v)
			}
		}
		if props, ok := mapFromAny(item["properties"]); ok {
			return props, identityPK, identityDK
		}
	}

	// xunji 南向下发常见格式：params 直接是属性键值
	directProperties := make(map[string]interface{})
	for key, raw := range params {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || isReservedCommandKey(trimmedKey) {
			continue
		}
		switch raw.(type) {
		case map[string]interface{}, []interface{}:
			continue
		}
		directProperties[trimmedKey] = raw
	}
	if len(directProperties) > 0 {
		return directProperties, identityPK, identityDK
	}

	return nil, identityPK, identityDK
}

func parseIdentityMap(identity map[string]interface{}) (string, string) {
	if identity == nil {
		return "", ""
	}
	productKey, deviceKey := "", ""
	if value, ok := identity["productKey"].(string); ok {
		productKey = strings.TrimSpace(value)
	}
	if value, ok := identity["deviceKey"].(string); ok {
		deviceKey = strings.TrimSpace(value)
	}
	return productKey, deviceKey
}

func isReservedCommandKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "id", "method", "version", "params",
		"identity", "properties", "events",
		"sub_device", "subdevice", "sub_devices", "subdevices":
		return true
	default:
		return false
	}
}

func mapFromAny(value interface{}) (map[string]interface{}, bool) {
	out, ok := value.(map[string]interface{})
	if !ok || out == nil {
		return nil, false
	}
	return out, true
}

func stringifyAny(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return strconv.FormatBool(v)
	case int:
		return strconv.Itoa(v)
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
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func convertFieldValue(value string) interface{} {
	v := strings.TrimSpace(value)
	if v == "" {
		return ""
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	if i, err := strconv.ParseInt(v, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		return f
	}
	return value
}

func pickFirstNonEmpty(values ...string) string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" {
			return v
		}
	}
	return ""
}

func resolveInterval(ms int, fallback time.Duration) time.Duration {
	if ms <= 0 {
		return fallback
	}
	interval := time.Duration(ms) * time.Millisecond
	if interval < 200*time.Millisecond {
		return 200 * time.Millisecond
	}
	return interval
}

func resolvePositive(v int, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
