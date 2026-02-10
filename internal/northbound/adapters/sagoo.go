package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// SagooConfig 循迹北向配置（与 models.SagooConfig 保持一致）
type SagooConfig struct {
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

// SagooAdapter 循迹北向适配器
// 每个 SagooAdapter 自己管理自己的状态和发送线程
type SagooAdapter struct {
	name        string
	config      *SagooConfig
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
	loopState   adapterLoopState
	seq         uint64
}

// NewSagooAdapter 创建循迹适配器
func NewSagooAdapter(name string) *SagooAdapter {
	return &SagooAdapter{
		name:         name,
		stopChan:     make(chan struct{}),
		flushNow:     make(chan struct{}, 1),
		latestData:   make([]*models.CollectData, 0),
		alarmQueue:   make([]*models.AlarmPayload, 0),
		commandQueue: make([]*models.NorthboundCommand, 0),
		loopState:    adapterLoopStopped,
	}
}

// Name 获取名称
func (a *SagooAdapter) Name() string {
	return a.name
}

// Type 获取类型
func (a *SagooAdapter) Type() string {
	return "sagoo"
}

// Initialize 初始化
func (a *SagooAdapter) Initialize(configStr string) error {
	cfg, err := parseSagooConfig(configStr)
	if err != nil {
		return err
	}

	// 清理旧连接
	_ = a.Close()

	broker := normalizeBroker(cfg.ServerURL)
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("sagoo-%s-%s-%d", cfg.ProductKey, cfg.DeviceKey, time.Now().UnixNano())
	}

	qos := clampQOS(cfg.QOS)
	retain := cfg.Retain
	topic := strings.TrimSpace(cfg.Topic)
	if !strings.HasPrefix(topic, "/sys/") {
		topic = sagooSysTopic(cfg.ProductKey, cfg.DeviceKey, "thing/event/property/pack/post")
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
	a.loopState = adapterLoopStopped
	a.mu.Unlock()

	// 订阅命令主题
	a.subscribeCommandTopics(client)

	log.Printf("Sagoo adapter initialized: %s (broker=%s, topic=%s)",
		a.name, broker, topic)
	return nil
}

// Start 启动适配器的后台线程
func (a *SagooAdapter) Start() {
	transition := loopStateTransition{}
	a.mu.Lock()
	if a.initialized && !a.enabled && a.loopState == adapterLoopStopped {
		a.enabled = true
		transition = updateLoopState(&a.loopState, adapterLoopRunning)
		if a.stopChan == nil {
			a.stopChan = make(chan struct{})
		}
		if a.flushNow == nil {
			a.flushNow = make(chan struct{}, 1)
		}
		a.wg.Add(1)
		go a.runLoop()
		log.Printf("Sagoo adapter started: %s", a.name)
	}
	a.mu.Unlock()
	logLoopStateTransition("sagoo", a.name, transition)
}

// Stop 停止适配器的后台线程
func (a *SagooAdapter) Stop() {
	transitionStopping := loopStateTransition{}
	transitionStopped := loopStateTransition{}

	a.mu.Lock()
	stopChan := a.stopChan
	if a.enabled {
		a.enabled = false
		transitionStopping = updateLoopState(&a.loopState, adapterLoopStopping)
		if stopChan != nil {
			close(stopChan)
		}
	}
	a.mu.Unlock()
	logLoopStateTransition("sagoo", a.name, transitionStopping)

	a.wg.Wait()
	if stopChan != nil {
		a.mu.Lock()
		if a.stopChan == stopChan {
			a.stopChan = nil
		}
		transitionStopped = updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
	}
	logLoopStateTransition("sagoo", a.name, transitionStopped)
	log.Printf("Sagoo adapter stopped: %s", a.name)
}

// SetInterval 设置发送周期
func (a *SagooAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = interval
	a.mu.Unlock()
}

// IsEnabled 检查是否启用
func (a *SagooAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// IsConnected 检查连接状态
func (a *SagooAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

// Send 发送数据（加入缓冲队列）
func (a *SagooAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(cloneCollectData(data))
	a.dataMu.Unlock()

	return nil
}

// SendAlarm 发送报警
func (a *SagooAdapter) SendAlarm(alarm *models.AlarmPayload) error {
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
func (a *SagooAdapter) Close() error {
	a.Stop()

	transitionStopped := loopStateTransition{}
	a.mu.Lock()
	client := a.client
	a.initialized = false
	a.connected = false
	a.enabled = false
	transitionStopped = updateLoopState(&a.loopState, adapterLoopStopped)
	a.mu.Unlock()
	logLoopStateTransition("sagoo", a.name, transitionStopped)

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
func (a *SagooAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
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

	out := make([]*models.NorthboundCommand, limit)
	copy(out, a.commandQueue[:limit])
	clear(a.commandQueue[:limit])
	a.commandQueue = a.commandQueue[limit:]

	return out, nil
}

// ReportCommandResult 上报命令执行结果
func (a *SagooAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
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

	pk := pickFirstNonEmpty2(strings.TrimSpace(cfg.ProductKey), result.ProductKey)
	dk := pickFirstNonEmpty2(strings.TrimSpace(cfg.DeviceKey), result.DeviceKey)
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

	topic := sagooSysTopic(pk, dk, "thing/service/property/set_reply")
	return a.publish(topic, body)
}

// runLoop 单协程事件循环（实时与报警）
func (a *SagooAdapter) runLoop() {
	defer func() {
		a.mu.Lock()
		transition := updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
		logLoopStateTransition("sagoo", a.name, transition)
		a.wg.Done()
	}()

	a.mu.RLock()
	reportInterval := a.reportEvery
	alarmInterval := a.alarmEvery
	flushNow := a.flushNow
	stopChan := a.stopChan
	a.mu.RUnlock()

	if reportInterval <= 0 {
		reportInterval = defaultReportInterval
	}
	if alarmInterval <= 0 {
		alarmInterval = defaultAlarmInterval
	}

	reportTicker := time.NewTicker(reportInterval)
	alarmTicker := time.NewTicker(alarmInterval)
	defer reportTicker.Stop()
	defer alarmTicker.Stop()

	for {
		select {
		case <-stopChan:
			for {
				if err := a.flushAlarmBatch(); err != nil {
					log.Printf("Sagoo alarm flush failed on close: %v", err)
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
		case <-reportTicker.C:
			if err := a.flushLatestData(); err != nil {
				log.Printf("Sagoo report flush failed: %v", err)
			}
		case <-alarmTicker.C:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("Sagoo alarm flush failed: %v", err)
			}
		case <-flushNow:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("Sagoo alarm flush failed: %v", err)
			}
		}
	}
}

// flushLatestData 发送实时数据
func (a *SagooAdapter) flushLatestData() error {
	a.dataMu.Lock()
	if len(a.latestData) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := make([]*models.CollectData, len(a.latestData))
	copy(batch, a.latestData)
	clear(a.latestData)
	a.latestData = a.latestData[:0]
	topic := a.topic
	a.dataMu.Unlock()

	for _, data := range batch {
		message := a.buildMessage(data)
		if err := a.publish(topic, message); err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			return err
		}
	}

	return nil
}

// flushAlarmBatch 发送报警批次
func (a *SagooAdapter) flushAlarmBatch() error {
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
	clear(a.alarmQueue[:count])
	a.alarmQueue = a.alarmQueue[count:]
	topic := a.alarmTopic
	a.alarmMu.Unlock()

	for _, alarm := range batch {
		message := a.buildAlarmMessage(alarm)
		if err := a.publish(topic, message); err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
	}

	return nil
}

// buildMessage 构建循迹消息
func (a *SagooAdapter) buildMessage(data *models.CollectData) []byte {
	if data == nil {
		return []byte("{}")
	}

	properties := make(map[string]interface{}, len(data.Fields))
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
	return jsonBytes
}

// buildAlarmMessage 构建报警消息
func (a *SagooAdapter) buildAlarmMessage(alarm *models.AlarmPayload) []byte {
	if alarm == nil {
		return []byte("{}")
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
						"productKey": pickFirstNonEmpty2(alarm.ProductKey, defaultPK),
						"deviceKey":  pickFirstNonEmpty2(alarm.DeviceKey, defaultDK),
					},
					"properties": map[string]interface{}{},
					"events":     events,
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(msg)
	return jsonBytes
}

// publish 发布MQTT消息
func (a *SagooAdapter) publish(topic string, payload []byte) error {
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
func (a *SagooAdapter) subscribeCommandTopics(client mqtt.Client) {
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

	a.subscribe(client, sagooSysTopic(pk, dk, "thing/service/property/set"), a.handlePropertySet)
	a.subscribe(client, sagooSysTopic(pk, dk, "thing/service/+"), a.handleServiceCall)
	a.subscribe(client, sagooSysTopic(pk, dk, "thing/config/push"), a.handleConfigPush)
}

func (a *SagooAdapter) subscribe(client mqtt.Client, topic string, handler mqtt.MessageHandler) {
	a.mu.RLock()
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	token := client.Subscribe(topic, qos, handler)
	if !token.WaitTimeout(timeout) {
		return
	}
}

func (a *SagooAdapter) handlePropertySet(_ mqtt.Client, message mqtt.Message) {
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
	_ = a.publish(sagooSysTopic(pk, dk, "thing/service/property/set_reply"), respBody)
}

func (a *SagooAdapter) handleServiceCall(_ mqtt.Client, message mqtt.Message) {
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
	_ = a.publish(sagooSysTopic(pk, dk, "thing/service/"+svc+"_reply"), respBody)
}

func (a *SagooAdapter) handleConfigPush(_ mqtt.Client, message mqtt.Message) {
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
	_ = a.publish(sagooSysTopic(pk, dk, "thing/config/push/reply"), respBody)
}

func (a *SagooAdapter) enqueueCommandFromPropertySet(defaultPK, defaultDK, requestID string, params map[string]interface{}, rootIdentityPK, rootIdentityDK string) {
	properties, identityPK, identityDK := extractCommandProperties(params)
	if len(properties) == 0 {
		return
	}

	pk := pickFirstNonEmpty3(rootIdentityPK, identityPK, defaultPK)
	dk := pickFirstNonEmpty3(rootIdentityDK, identityDK, defaultDK)
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

	commands := make([]*models.NorthboundCommand, 0, len(keys))
	for _, key := range keys {
		raw := properties[key]
		commands = append(commands, &models.NorthboundCommand{
			RequestID:  requestID,
			ProductKey: pk,
			DeviceKey:  dk,
			FieldName:  key,
			Value:      stringifyAny(raw),
			Source:     "sagoo.property.set",
		})
	}
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
}

// 辅助函数
func (a *SagooAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func (a *SagooAdapter) defaultIdentity() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config == nil {
		return "", ""
	}
	return strings.TrimSpace(a.config.ProductKey), strings.TrimSpace(a.config.DeviceKey)
}

func (a *SagooAdapter) nextID(prefix string) string {
	return nextPrefixedID(prefix, &a.seq)
}

func (a *SagooAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.latestData = appendQueueItemWithCap(a.latestData, item, a.realtimeCap)
}

func (a *SagooAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.latestData = prependQueueWithCap(a.latestData, items, a.realtimeCap)
}

func (a *SagooAdapter) enqueueAlarmLocked(alarm *models.AlarmPayload) {
	if alarm == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = appendQueueItemWithCap(a.alarmQueue, alarm, a.alarmCap)
}

func (a *SagooAdapter) prependAlarms(alarms []*models.AlarmPayload) {
	if len(alarms) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = prependQueueWithCap(a.alarmQueue, alarms, a.alarmCap)
}

// GetStats 获取适配器统计信息
func (a *SagooAdapter) GetStats() map[string]interface{} {
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
		"type":          "sagoo",
		"enabled":       a.enabled,
		"initialized":   a.initialized,
		"connected":     a.connected && a.client != nil && a.client.IsConnected(),
		"loop_state":    a.loopState.String(),
		"interval_ms":   a.reportEvery.Milliseconds(),
		"pending_data":  dataCount,
		"pending_alarm": alarmCount,
		"pending_cmd":   commandCount,
		"product_key":   productKey,
		"device_key":    deviceKey,
	}
}

// GetLastSendTime 获取最后发送时间（返回零值，因为是内部管理）
func (a *SagooAdapter) GetLastSendTime() time.Time {
	return time.Time{}
}

// PendingCommandCount 获取待处理命令数量
func (a *SagooAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}

// 辅助函数
func parseSagooConfig(configStr string) (*SagooConfig, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &SagooConfig{
		ProductKey:           strings.TrimSpace(pickConfigString(raw, "productKey", "product_key", "productID", "product_id")),
		DeviceKey:            strings.TrimSpace(pickConfigString(raw, "deviceKey", "device_key", "deviceName", "device_name")),
		Username:             strings.TrimSpace(pickConfigString(raw, "username")),
		Password:             strings.TrimSpace(pickConfigString(raw, "password")),
		Topic:                strings.TrimSpace(pickConfigString(raw, "topic")),
		AlarmTopic:           strings.TrimSpace(pickConfigString(raw, "alarmTopic", "alarm_topic")),
		ClientID:             strings.TrimSpace(pickConfigString(raw, "clientId", "client_id")),
		QOS:                  pickConfigInt(raw, 0, "qos"),
		Retain:               pickConfigBool(raw, false, "retain"),
		KeepAlive:            pickConfigInt(raw, 60, "keepAlive", "keep_alive"),
		Timeout:              pickConfigInt(raw, 10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:     pickConfigInt(raw, int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms"),
		ReportIntervalMs:     pickConfigInt(raw, 0, "reportIntervalMs", "report_interval_ms"),
		AlarmFlushIntervalMs: pickConfigInt(raw, int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs", "alarm_flush_interval_ms"),
		AlarmBatchSize:       pickConfigInt(raw, defaultAlarmBatch, "alarmBatchSize", "alarm_batch_size"),
		AlarmQueueSize:       pickConfigInt(raw, defaultAlarmQueue, "alarmQueueSize", "alarm_queue_size"),
		RealtimeQueueSize:    pickConfigInt(raw, defaultRealtimeQueue, "realtimeQueueSize", "realtime_queue_size"),
	}

	cfg.ServerURL = normalizeServerURLWithPort(
		pickConfigString(raw, "serverUrl", "server_url", "broker"),
		pickConfigString(raw, "protocol"),
		pickConfigInt(raw, 0, "port"),
	)

	if err := normalizeSagooConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizeSagooConfig(cfg *SagooConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.ProductKey == "" {
		return fmt.Errorf("productKey is required")
	}
	if cfg.DeviceKey == "" {
		return fmt.Errorf("deviceKey is required")
	}
	if cfg.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}
	if cfg.QOS < 0 || cfg.QOS > 2 {
		return fmt.Errorf("qos must be between 0 and 2")
	}

	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10
	}
	if cfg.UploadIntervalMs <= 0 && cfg.ReportIntervalMs <= 0 {
		cfg.UploadIntervalMs = int(defaultReportInterval.Milliseconds())
	}
	if cfg.AlarmFlushIntervalMs <= 0 {
		cfg.AlarmFlushIntervalMs = int(defaultAlarmInterval.Milliseconds())
	}
	if cfg.AlarmBatchSize <= 0 {
		cfg.AlarmBatchSize = defaultAlarmBatch
	}
	if cfg.AlarmQueueSize <= 0 {
		cfg.AlarmQueueSize = defaultAlarmQueue
	}
	if cfg.RealtimeQueueSize <= 0 {
		cfg.RealtimeQueueSize = defaultRealtimeQueue
	}

	return nil
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
		out.Fields = nil
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

func sagooSysTopic(productKey, deviceKey, suffix string) string {
	if suffix == "" {
		return "/sys/" + productKey + "/" + deviceKey
	}
	return "/sys/" + productKey + "/" + deviceKey + "/" + suffix
}

func splitTopic(topic string) []string {
	trimmed := strings.TrimSpace(topic)
	if trimmed == "" {
		return make([]string, 0)
	}

	parts := make([]string, 0, 1+strings.Count(trimmed, "/"))
	segmentStart := -1
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == '/' {
			if segmentStart >= 0 {
				parts = append(parts, trimmed[segmentStart:i])
				segmentStart = -1
			}
			continue
		}
		if segmentStart < 0 {
			segmentStart = i
		}
	}
	if segmentStart >= 0 {
		parts = append(parts, trimmed[segmentStart:])
	}

	return parts
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
		identityPK, identityDK = parseIdentityMap(identity)
	}

	if props, ok := mapFromAny(params["properties"]); ok {
		return props, identityPK, identityDK
	}

	if sub, ok := mapFromAnyByKey2(params, "sub_device", "subDevice"); ok {
		if identity, ok := mapFromAny(sub["identity"]); ok {
			identityPK, identityDK = parseIdentityMap(identity)
		}
		if props, ok := mapFromAny(sub["properties"]); ok {
			return props, identityPK, identityDK
		}
	}

	if list, ok := interfaceSliceByKey2(params, "sub_devices", "subDevices"); ok && len(list) > 0 {
		if item, ok := mapFromAny(list[0]); ok {
			if identity, ok := mapFromAny(item["identity"]); ok {
				identityPK, identityDK = parseIdentityMap(identity)
			}
			if props, ok := mapFromAny(item["properties"]); ok {
				return props, identityPK, identityDK
			}
		}
	}

	// sagoo 南向下发常见格式：params 直接是属性键值
	directProperties := make(map[string]interface{}, len(params))
	for key, raw := range params {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || isReservedCommandKeyNormalized(strings.ToLower(trimmedKey)) {
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
	normalized := strings.ToLower(strings.TrimSpace(key))
	return isReservedCommandKeyNormalized(normalized)
}

func isReservedCommandKeyNormalized(normalized string) bool {
	switch normalized {
	case "id", "method", "version", "params",
		"identity", "properties", "events",
		"sub_device", "subdevice", "sub_devices", "subdevices":
		return true
	default:
		return false
	}
}

func mapFromAnyByKey2(values map[string]interface{}, key1, key2 string) (map[string]interface{}, bool) {
	if out, ok := mapFromAny(values[key1]); ok {
		return out, true
	}
	return mapFromAny(values[key2])
}

func interfaceSliceByKey2(values map[string]interface{}, key1, key2 string) ([]interface{}, bool) {
	if list, ok := values[key1].([]interface{}); ok {
		return list, true
	}
	list, ok := values[key2].([]interface{})
	return list, ok
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

func pickFirstNonEmpty2(left, right string) string {
	if v := strings.TrimSpace(left); v != "" {
		return v
	}
	return strings.TrimSpace(right)
}

func pickFirstNonEmpty3(first, second, third string) string {
	if v := pickFirstNonEmpty2(first, second); v != "" {
		return v
	}
	return strings.TrimSpace(third)
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
