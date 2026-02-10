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

const (
	defaultIThingsUpPropertyTopicTemplate = "$thing/up/property/{productID}/{deviceName}"
	defaultIThingsUpEventTopicTemplate    = "$thing/up/event/{productID}/{deviceName}"
	defaultIThingsUpActionTopicTemplate   = "$thing/up/action/{productID}/{deviceName}"
	defaultIThingsDownPropertyTopic       = "$thing/down/property/+/+"
	defaultIThingsDownActionTopic         = "$thing/down/action/+/+"
	defaultIThingsAlarmEventID            = "alarm"
	defaultIThingsAlarmEventType          = "alert"
)

// IThingsConfig iThings 北向配置
type IThingsConfig struct {
	ServerURL string `json:"serverUrl"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ClientID  string `json:"clientId"`
	QOS       int    `json:"qos"`
	Retain    bool   `json:"retain"`
	KeepAlive int    `json:"keepAlive"`
	Timeout   int    `json:"connectTimeout"`

	UploadIntervalMs     int `json:"uploadIntervalMs"`
	AlarmFlushIntervalMs int `json:"alarmFlushIntervalMs"`
	AlarmBatchSize       int `json:"alarmBatchSize"`
	AlarmQueueSize       int `json:"alarmQueueSize"`
	RealtimeQueueSize    int `json:"realtimeQueueSize"`
	CommandQueueSize     int `json:"commandQueueSize"`

	GatewayMode       bool   `json:"gatewayMode"`
	ProductKey        string `json:"productKey"`
	DeviceKey         string `json:"deviceKey"`
	DeviceNameMode    string `json:"deviceNameMode"`
	SubDeviceNameMode string `json:"subDeviceNameMode"`

	UpPropertyTopicTemplate string `json:"upPropertyTopicTemplate"`
	UpEventTopicTemplate    string `json:"upEventTopicTemplate"`
	UpActionTopicTemplate   string `json:"upActionTopicTemplate"`
	DownPropertyTopic       string `json:"downPropertyTopic"`
	DownActionTopic         string `json:"downActionTopic"`

	AlarmEventID   string `json:"alarmEventID"`
	AlarmEventType string `json:"alarmEventType"`
}

type iThingsRequestState struct {
	RequestID  string
	ProductID  string
	DeviceName string
	Method     string
	TopicType  string
	ActionID   string
	Pending    int
	Success    bool
	Code       int
	Message    string
	FieldName  string
	Value      string
}

// IThingsAdapter iThings 北向适配器
type IThingsAdapter struct {
	name   string
	config *IThingsConfig

	client  mqtt.Client
	qos     byte
	retain  bool
	timeout time.Duration

	reportEvery time.Duration
	alarmEvery  time.Duration
	alarmBatch  int
	alarmCap    int
	realtimeCap int
	commandCap  int

	upPropertyTopicTemplate string
	upEventTopicTemplate    string
	upActionTopicTemplate   string
	downPropertyTopic       string
	downActionTopic         string
	alarmEventID            string
	alarmEventType          string
	deviceNameMode          string
	subDeviceNameMode       string

	realtimeQueue []*models.CollectData
	dataMu        sync.RWMutex

	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

	requestStates map[string]*iThingsRequestState
	requestMu     sync.Mutex

	flushNow chan struct{}
	stopChan chan struct{}
	wg       sync.WaitGroup

	mu          sync.RWMutex
	initialized bool
	enabled     bool
	connected   bool
	lastSend    time.Time
	seq         uint64
}

// NewIThingsAdapter 创建 iThings 适配器
func NewIThingsAdapter(name string) *IThingsAdapter {
	return &IThingsAdapter{
		name:          name,
		flushNow:      make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		realtimeQueue: make([]*models.CollectData, 0),
		alarmQueue:    make([]*models.AlarmPayload, 0),
		commandQueue:  make([]*models.NorthboundCommand, 0),
		requestStates: make(map[string]*iThingsRequestState),
	}
}

func (a *IThingsAdapter) Name() string {
	return a.name
}

func (a *IThingsAdapter) Type() string {
	return "ithings"
}

func (a *IThingsAdapter) Initialize(configStr string) error {
	cfg, err := parseIThingsConfig(configStr)
	if err != nil {
		return err
	}

	_ = a.Close()

	broker := normalizeBroker(cfg.ServerURL)
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("ithings-%s-%d", a.name, time.Now().UnixNano())
	}

	client, err := connectMQTT(broker, clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.mu.Lock()
	a.config = cfg
	a.client = client
	a.qos = clampQOS(cfg.QOS)
	a.retain = cfg.Retain
	a.timeout = time.Duration(resolvePositive(cfg.Timeout, 10)) * time.Second
	a.reportEvery = resolveInterval(cfg.UploadIntervalMs, defaultReportInterval)
	a.alarmEvery = resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval)
	a.alarmBatch = resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch)
	a.alarmCap = resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue)
	a.realtimeCap = resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue)
	a.commandCap = resolvePositive(cfg.CommandQueueSize, defaultRealtimeQueue)
	a.upPropertyTopicTemplate = pickFirstNonEmpty(cfg.UpPropertyTopicTemplate, defaultIThingsUpPropertyTopicTemplate)
	a.upEventTopicTemplate = pickFirstNonEmpty(cfg.UpEventTopicTemplate, defaultIThingsUpEventTopicTemplate)
	a.upActionTopicTemplate = pickFirstNonEmpty(cfg.UpActionTopicTemplate, defaultIThingsUpActionTopicTemplate)
	a.downPropertyTopic = pickFirstNonEmpty(cfg.DownPropertyTopic, defaultIThingsDownPropertyTopic)
	a.downActionTopic = pickFirstNonEmpty(cfg.DownActionTopic, defaultIThingsDownActionTopic)
	a.alarmEventID = pickFirstNonEmpty(cfg.AlarmEventID, defaultIThingsAlarmEventID)
	a.alarmEventType = pickFirstNonEmpty(cfg.AlarmEventType, defaultIThingsAlarmEventType)
	a.deviceNameMode = pickFirstNonEmpty(cfg.DeviceNameMode, "deviceKey")
	a.subDeviceNameMode = pickFirstNonEmpty(cfg.SubDeviceNameMode, a.deviceNameMode)
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.requestStates = make(map[string]*iThingsRequestState)
	a.initialized = true
	a.connected = true
	a.enabled = false
	a.mu.Unlock()

	a.subscribeDownTopics(client)

	log.Printf("iThings adapter initialized: %s (broker=%s)", a.name, broker)
	return nil
}

func (a *IThingsAdapter) Start() {
	a.mu.Lock()
	if a.initialized && !a.enabled {
		a.enabled = true
		if a.stopChan == nil {
			a.stopChan = make(chan struct{})
		}
		a.wg.Add(2)
		go a.reportLoop()
		go a.alarmLoop()
		log.Printf("iThings adapter started: %s", a.name)
	}
	a.mu.Unlock()
}

func (a *IThingsAdapter) Stop() {
	a.mu.Lock()
	if a.enabled {
		a.enabled = false
		if a.stopChan != nil {
			close(a.stopChan)
			a.stopChan = nil
		}
	}
	a.mu.Unlock()
	a.wg.Wait()
	log.Printf("iThings adapter stopped: %s", a.name)
}

func (a *IThingsAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = resolveInterval(int(interval.Milliseconds()), defaultReportInterval)
	a.mu.Unlock()
}

func (a *IThingsAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *IThingsAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

func (a *IThingsAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(cloneCollectData(data))
	a.dataMu.Unlock()

	return nil
}

func (a *IThingsAdapter) SendAlarm(alarm *models.AlarmPayload) error {
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

func (a *IThingsAdapter) Close() error {
	a.Stop()

	a.mu.Lock()
	client := a.client
	a.initialized = false
	a.connected = false
	a.enabled = false
	a.mu.Unlock()

	_ = a.flushRealtime()
	_ = a.flushAlarmBatch()

	if client != nil && client.IsConnected() {
		client.Disconnect(250)
	}

	a.mu.Lock()
	a.client = nil
	a.config = nil
	a.flushNow = nil
	a.stopChan = nil
	a.realtimeQueue = nil
	a.alarmQueue = nil
	a.commandQueue = nil
	a.requestStates = nil
	a.mu.Unlock()

	return nil
}

func (a *IThingsAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
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

	items := make([]*models.NorthboundCommand, limit)
	copy(items, a.commandQueue[:limit])
	clear(a.commandQueue[:limit])
	a.commandQueue = a.commandQueue[limit:]

	return items, nil
}

func (a *IThingsAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	if result == nil || strings.TrimSpace(result.RequestID) == "" {
		return nil
	}

	a.mu.RLock()
	initialized := a.initialized
	a.mu.RUnlock()
	if !initialized {
		return nil
	}

	state, ready := a.applyCommandResult(result)
	if !ready {
		return nil
	}

	code := state.Code
	if code == 0 {
		if state.Success {
			code = 0
		} else {
			code = 500
		}
	}
	message := strings.TrimSpace(state.Message)
	if message == "" {
		if state.Success {
			message = "success"
		} else {
			message = "failed"
		}
	}

	payload := map[string]interface{}{
		"msgToken":  state.RequestID,
		"code":      code,
		"msg":       message,
		"timestamp": time.Now().UnixMilli(),
	}

	if state.TopicType == "action" {
		payload["method"] = "actionReply"
		if strings.TrimSpace(state.ActionID) != "" {
			payload["actionID"] = state.ActionID
		} else if strings.TrimSpace(state.FieldName) != "" {
			payload["actionID"] = state.FieldName
		}
		if strings.TrimSpace(state.FieldName) != "" {
			payload["data"] = map[string]interface{}{
				state.FieldName: convertFieldValue(state.Value),
			}
		}
		topic := renderIThingsTopic(a.upActionTopicTemplate, state.ProductID, state.DeviceName)
		body, _ := json.Marshal(payload)
		return a.publish(topic, body)
	}

	payload["method"] = "controlReply"
	topic := renderIThingsTopic(a.upPropertyTopicTemplate, state.ProductID, state.DeviceName)
	body, _ := json.Marshal(payload)
	return a.publish(topic, body)
}

func (a *IThingsAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.dataMu.RLock()
	pendingData := len(a.realtimeQueue)
	a.dataMu.RUnlock()

	a.alarmMu.RLock()
	pendingAlarm := len(a.alarmQueue)
	a.alarmMu.RUnlock()

	a.commandMu.RLock()
	pendingCmd := len(a.commandQueue)
	a.commandMu.RUnlock()

	return map[string]interface{}{
		"name":                       a.name,
		"type":                       "ithings",
		"enabled":                    a.enabled,
		"initialized":                a.initialized,
		"connected":                  a.connected && a.client != nil && a.client.IsConnected(),
		"interval_ms":                a.reportEvery.Milliseconds(),
		"pending_data":               pendingData,
		"pending_alarm":              pendingAlarm,
		"pending_cmd":                pendingCmd,
		"up_property_topic_template": a.upPropertyTopicTemplate,
		"down_property_topic":        a.downPropertyTopic,
		"down_action_topic":          a.downActionTopic,
	}
}

func (a *IThingsAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

func (a *IThingsAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}

func (a *IThingsAdapter) reportLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.reportEvery
	stopChan := a.stopChan
	a.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			if err := a.flushRealtime(); err != nil {
				log.Printf("iThings realtime flush failed: %v", err)
			}
		}
	}
}

func (a *IThingsAdapter) alarmLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.alarmEvery
	flushNow := a.flushNow
	stopChan := a.stopChan
	a.mu.RUnlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			for {
				if err := a.flushAlarmBatch(); err != nil {
					log.Printf("iThings alarm flush failed on close: %v", err)
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
				log.Printf("iThings alarm flush failed: %v", err)
			}
		case <-flushNow:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("iThings alarm flush failed: %v", err)
			}
		}
	}
}

func (a *IThingsAdapter) flushRealtime() error {
	a.dataMu.Lock()
	if len(a.realtimeQueue) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := make([]*models.CollectData, len(a.realtimeQueue))
	copy(batch, a.realtimeQueue)
	clear(a.realtimeQueue)
	a.realtimeQueue = a.realtimeQueue[:0]
	a.dataMu.Unlock()

	for _, item := range batch {
		topic, body, err := a.buildRealtimePublish(item)
		if err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			return err
		}
		if err := a.publish(topic, body); err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			return err
		}
	}

	return nil
}

func (a *IThingsAdapter) flushAlarmBatch() error {
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
	a.alarmMu.Unlock()

	for _, item := range batch {
		topic, body, err := a.buildAlarmPublish(item)
		if err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
		if err := a.publish(topic, body); err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
	}

	return nil
}

func (a *IThingsAdapter) buildRealtimePublish(data *models.CollectData) (string, []byte, error) {
	a.mu.RLock()
	cfg := a.config
	upPropertyTpl := a.upPropertyTopicTemplate
	deviceNameMode := a.deviceNameMode
	subDeviceNameMode := a.subDeviceNameMode
	a.mu.RUnlock()

	if data == nil {
		return upPropertyTpl, []byte("{}"), nil
	}

	values := make(map[string]interface{}, len(data.Fields))
	for key, value := range data.Fields {
		values[key] = convertFieldValue(value)
	}

	ts := data.Timestamp.UnixMilli()
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}

	if cfg == nil {
		return "", nil, fmt.Errorf("ithings config is nil")
	}

	gatewayProductID := strings.TrimSpace(cfg.ProductKey)
	gatewayDeviceName := strings.TrimSpace(cfg.DeviceKey)
	if gatewayProductID == "" || gatewayDeviceName == "" {
		return "", nil, fmt.Errorf("productKey and deviceKey are required for iThings gateway mode")
	}
	topic := renderIThingsTopic(upPropertyTpl, gatewayProductID, gatewayDeviceName)

	subProductID := pickFirstNonEmpty2(strings.TrimSpace(data.ProductKey), gatewayProductID)
	subDeviceName := pickFirstNonEmpty2(a.resolveCollectDeviceName(data, subDeviceNameMode), a.resolveCollectDeviceName(data, deviceNameMode))
	if subDeviceName == "" {
		subDeviceName = virtualDeviceName(data.DeviceID)
	}

	payload := map[string]interface{}{
		"method":     "packReport",
		"msgToken":   a.nextID("pack"),
		"timestamp":  ts,
		"properties": []interface{}{},
		"events":     []interface{}{},
		"subDevices": []interface{}{
			map[string]interface{}{
				"productID":  subProductID,
				"deviceName": subDeviceName,
				"properties": []interface{}{
					map[string]interface{}{
						"timestamp": ts,
						"params":    values,
					},
				},
				"events": []interface{}{},
			},
		},
	}
	body, _ := json.Marshal(payload)
	return topic, body, nil
}

func (a *IThingsAdapter) buildAlarmPublish(alarm *models.AlarmPayload) (string, []byte, error) {
	a.mu.RLock()
	cfg := a.config
	upEventTpl := a.upEventTopicTemplate
	alarmEventID := a.alarmEventID
	alarmEventType := a.alarmEventType
	deviceNameMode := a.deviceNameMode
	a.mu.RUnlock()

	if alarm == nil {
		return upEventTpl, []byte("{}"), nil
	}

	if cfg == nil {
		return "", nil, fmt.Errorf("ithings config is nil")
	}
	gatewayProductID := strings.TrimSpace(cfg.ProductKey)
	gatewayDeviceName := strings.TrimSpace(cfg.DeviceKey)
	if gatewayProductID == "" || gatewayDeviceName == "" {
		return "", nil, fmt.Errorf("productKey and deviceKey are required for iThings gateway mode")
	}
	topic := renderIThingsTopic(upEventTpl, gatewayProductID, gatewayDeviceName)

	subProductID := pickFirstNonEmpty2(strings.TrimSpace(alarm.ProductKey), gatewayProductID)
	subDeviceName := strings.TrimSpace(a.resolveAlarmDeviceName(alarm, deviceNameMode))
	if subDeviceName == "" {
		subDeviceName = virtualDeviceName(alarm.DeviceID)
	}

	params := map[string]interface{}{
		"device_name":  alarm.DeviceName,
		"product_key":  subProductID,
		"device_key":   subDeviceName,
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"severity":     alarm.Severity,
		"message":      alarm.Message,
	}
	payload := map[string]interface{}{
		"method":    "eventPost",
		"msgToken":  a.nextID("alarm"),
		"timestamp": time.Now().UnixMilli(),
		"eventID":   alarmEventID,
		"type":      alarmEventType,
		"params":    params,
	}
	body, _ := json.Marshal(payload)
	return topic, body, nil
}

func (a *IThingsAdapter) publish(topic string, payload []byte) error {
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
			a.mu.Lock()
			a.connected = false
			a.mu.Unlock()
			return err
		}
		a.subscribeDownTopics(client)
	}

	token := client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		a.mu.Lock()
		a.connected = false
		a.mu.Unlock()
		return err
	}

	a.mu.Lock()
	a.lastSend = time.Now()
	a.connected = true
	a.mu.Unlock()

	return nil
}

func virtualDeviceName(deviceID int64) string {
	return "device_" + strconv.FormatInt(deviceID, 10)
}

func (a *IThingsAdapter) subscribeDownTopics(client mqtt.Client) {
	a.mu.RLock()
	downPropertyTopic := strings.TrimSpace(a.downPropertyTopic)
	downActionTopic := strings.TrimSpace(a.downActionTopic)
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	topics := make(map[string]struct{})
	if downPropertyTopic != "" {
		topics[downPropertyTopic] = struct{}{}
	}
	if downActionTopic != "" {
		topics[downActionTopic] = struct{}{}
	}

	for topic := range topics {
		token := client.Subscribe(topic, qos, a.handleDownlink)
		if !token.WaitTimeout(timeout) {
			continue
		}
		if err := token.Error(); err != nil {
			log.Printf("iThings subscribe failed topic=%s: %v", topic, err)
		}
	}
}

func (a *IThingsAdapter) handleDownlink(_ mqtt.Client, message mqtt.Message) {
	topicType, productID, deviceName := parseIThingsDownTopic(message.Topic())
	if topicType == "" {
		return
	}

	var req struct {
		Method   string      `json:"method"`
		MsgToken string      `json:"msgToken"`
		ActionID string      `json:"actionID"`
		Params   interface{} `json:"params"`
		Data     interface{} `json:"data"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	requestID := strings.TrimSpace(req.MsgToken)
	if requestID == "" {
		requestID = a.nextID("req")
	}

	commands, method, actionID := buildIThingsCommands(requestID, topicType, req.Method, req.ActionID, req.Params, productID, deviceName)
	if len(commands) == 0 {
		return
	}

	a.commandMu.Lock()
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
	a.commandMu.Unlock()

	a.requestMu.Lock()
	a.requestStates[requestID] = &iThingsRequestState{
		RequestID:  requestID,
		ProductID:  productID,
		DeviceName: deviceName,
		Method:     method,
		TopicType:  topicType,
		ActionID:   actionID,
		Pending:    len(commands),
		Success:    true,
	}
	a.requestMu.Unlock()
}

func buildIThingsCommands(requestID, topicType, method, actionID string, params interface{}, productID, deviceName string) ([]*models.NorthboundCommand, string, string) {
	topicType = strings.ToLower(strings.TrimSpace(topicType))
	method = strings.TrimSpace(method)
	actionID = strings.TrimSpace(actionID)

	out := make([]*models.NorthboundCommand, 0)

	appendPropertyCommands := func(values map[string]interface{}) {
		if len(values) == 0 {
			return
		}
		keys := make([]string, 0, len(values))
		for key := range values {
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
		for _, key := range keys {
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: productID,
				DeviceKey:  deviceName,
				FieldName:  key,
				Value:      stringifyAny(values[key]),
				Source:     "ithings.down.property",
			})
		}
	}

	if topicType == "property" {
		if strings.EqualFold(method, "control") {
			if obj, ok := mapFromAny(params); ok {
				appendPropertyCommands(obj)
			}
		}
	} else if topicType == "action" {
		if strings.EqualFold(method, "action") && actionID != "" {
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: productID,
				DeviceKey:  deviceName,
				FieldName:  actionID,
				Value:      stringifyAny(params),
				Source:     "ithings.down.action",
			})
		}
	}

	return out, method, actionID
}

func (a *IThingsAdapter) applyCommandResult(result *models.NorthboundCommandResult) (*iThingsRequestState, bool) {
	a.requestMu.Lock()
	defer a.requestMu.Unlock()

	state, ok := a.requestStates[result.RequestID]
	if !ok {
		fallback := &iThingsRequestState{
			RequestID:  result.RequestID,
			ProductID:  strings.TrimSpace(result.ProductKey),
			DeviceName: strings.TrimSpace(result.DeviceKey),
			TopicType:  "property",
			Method:     "control",
			Pending:    1,
			Success:    result.Success,
			Code:       result.Code,
			Message:    strings.TrimSpace(result.Message),
			FieldName:  strings.TrimSpace(result.FieldName),
			Value:      result.Value,
		}
		return fallback, true
	}

	state.Pending--
	if state.Pending < 0 {
		state.Pending = 0
	}
	if !result.Success {
		state.Success = false
	}
	if result.Code != 0 {
		state.Code = result.Code
	}
	if text := strings.TrimSpace(result.Message); text != "" {
		state.Message = text
	}
	if field := strings.TrimSpace(result.FieldName); field != "" {
		state.FieldName = field
	}
	if strings.TrimSpace(result.Value) != "" {
		state.Value = result.Value
	}
	if strings.TrimSpace(result.ProductKey) != "" {
		state.ProductID = strings.TrimSpace(result.ProductKey)
	}
	if strings.TrimSpace(result.DeviceKey) != "" {
		state.DeviceName = strings.TrimSpace(result.DeviceKey)
	}

	if state.Pending > 0 {
		return nil, false
	}

	delete(a.requestStates, result.RequestID)
	return state, true
}

func (a *IThingsAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.realtimeQueue = appendQueueItemWithCap(a.realtimeQueue, item, a.realtimeCap)
}

func (a *IThingsAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.realtimeQueue = prependQueueWithCap(a.realtimeQueue, items, a.realtimeCap)
}

func (a *IThingsAdapter) enqueueAlarmLocked(item *models.AlarmPayload) {
	if item == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = appendQueueItemWithCap(a.alarmQueue, item, a.alarmCap)
}

func (a *IThingsAdapter) prependAlarms(items []*models.AlarmPayload) {
	if len(items) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = prependQueueWithCap(a.alarmQueue, items, a.alarmCap)
}

func (a *IThingsAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func (a *IThingsAdapter) resolveCollectDeviceName(data *models.CollectData, mode string) string {
	if data == nil {
		return ""
	}
	deviceName := strings.TrimSpace(data.DeviceName)
	deviceKey := strings.TrimSpace(data.DeviceKey)
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "devicename", "device_name":
		return pickFirstNonEmpty2(deviceName, deviceKey)
	case "devicekey", "device_key", "":
		return pickFirstNonEmpty2(deviceKey, deviceName)
	default:
		return pickFirstNonEmpty2(deviceKey, deviceName)
	}
}

func (a *IThingsAdapter) resolveAlarmDeviceName(alarm *models.AlarmPayload, mode string) string {
	if alarm == nil {
		return ""
	}
	deviceName := strings.TrimSpace(alarm.DeviceName)
	deviceKey := strings.TrimSpace(alarm.DeviceKey)
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "devicename", "device_name":
		return pickFirstNonEmpty2(deviceName, deviceKey)
	case "devicekey", "device_key", "":
		return pickFirstNonEmpty2(deviceKey, deviceName)
	default:
		return pickFirstNonEmpty2(deviceKey, deviceName)
	}
}

func parseIThingsConfig(configStr string) (*IThingsConfig, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &IThingsConfig{
		ServerURL:               strings.TrimSpace(pickConfigString(raw, "serverUrl", "broker", "server_url")),
		Username:                strings.TrimSpace(pickConfigString(raw, "username")),
		Password:                strings.TrimSpace(pickConfigString(raw, "password")),
		ClientID:                strings.TrimSpace(pickConfigString(raw, "clientId", "client_id")),
		QOS:                     pickConfigInt(raw, 0, "qos"),
		Retain:                  pickConfigBool(raw, false, "retain"),
		KeepAlive:               pickConfigInt(raw, 60, "keepAlive", "keep_alive"),
		Timeout:                 pickConfigInt(raw, 10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:        pickConfigInt(raw, int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		AlarmFlushIntervalMs:    pickConfigInt(raw, int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs"),
		AlarmBatchSize:          pickConfigInt(raw, defaultAlarmBatch, "alarmBatchSize"),
		AlarmQueueSize:          pickConfigInt(raw, defaultAlarmQueue, "alarmQueueSize"),
		RealtimeQueueSize:       pickConfigInt(raw, defaultRealtimeQueue, "realtimeQueueSize"),
		GatewayMode:             pickConfigBool(raw, true, "gatewayMode"),
		ProductKey:              strings.TrimSpace(pickConfigString(raw, "productKey", "productID", "product_id")),
		DeviceKey:               strings.TrimSpace(pickConfigString(raw, "deviceKey", "deviceName", "device_name")),
		DeviceNameMode:          strings.TrimSpace(pickConfigString(raw, "deviceNameMode")),
		SubDeviceNameMode:       strings.TrimSpace(pickConfigString(raw, "subDeviceNameMode")),
		UpPropertyTopicTemplate: strings.TrimSpace(pickConfigString(raw, "upPropertyTopicTemplate")),
		UpEventTopicTemplate:    strings.TrimSpace(pickConfigString(raw, "upEventTopicTemplate")),
		UpActionTopicTemplate:   strings.TrimSpace(pickConfigString(raw, "upActionTopicTemplate")),
		DownPropertyTopic:       strings.TrimSpace(pickConfigString(raw, "downPropertyTopic")),
		DownActionTopic:         strings.TrimSpace(pickConfigString(raw, "downActionTopic")),
		AlarmEventID:            strings.TrimSpace(pickConfigString(raw, "alarmEventID")),
		AlarmEventType:          strings.TrimSpace(pickConfigString(raw, "alarmEventType")),
	}

	cfg.CommandQueueSize = pickConfigInt(raw, cfg.RealtimeQueueSize, "commandQueueSize")

	if err := normalizeIThingsConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizeIThingsConfig(cfg *IThingsConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}
	if cfg.Username == "" {
		return fmt.Errorf("username is required")
	}
	if cfg.ProductKey == "" {
		return fmt.Errorf("productKey is required")
	}
	if cfg.DeviceKey == "" {
		return fmt.Errorf("deviceKey is required")
	}
	if !cfg.GatewayMode {
		return fmt.Errorf("iThings adapter only supports gatewayMode=true")
	}
	if cfg.QOS < 0 || cfg.QOS > 2 {
		return fmt.Errorf("qos must be between 0 and 2")
	}
	if cfg.UploadIntervalMs <= 0 {
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
	if cfg.CommandQueueSize <= 0 {
		cfg.CommandQueueSize = cfg.RealtimeQueueSize
	}

	if cfg.UpPropertyTopicTemplate == "" {
		cfg.UpPropertyTopicTemplate = defaultIThingsUpPropertyTopicTemplate
	}
	if cfg.UpEventTopicTemplate == "" {
		cfg.UpEventTopicTemplate = defaultIThingsUpEventTopicTemplate
	}
	if cfg.UpActionTopicTemplate == "" {
		cfg.UpActionTopicTemplate = defaultIThingsUpActionTopicTemplate
	}
	if cfg.DownPropertyTopic == "" {
		cfg.DownPropertyTopic = defaultIThingsDownPropertyTopic
	}
	if cfg.DownActionTopic == "" {
		cfg.DownActionTopic = defaultIThingsDownActionTopic
	}
	if cfg.AlarmEventID == "" {
		cfg.AlarmEventID = defaultIThingsAlarmEventID
	}
	if cfg.AlarmEventType == "" {
		cfg.AlarmEventType = defaultIThingsAlarmEventType
	}
	if cfg.DeviceNameMode == "" {
		cfg.DeviceNameMode = "deviceKey"
	}
	if cfg.SubDeviceNameMode == "" {
		cfg.SubDeviceNameMode = cfg.DeviceNameMode
	}

	return nil
}

func parseIThingsDownTopic(topic string) (topicType, productID, deviceName string) {
	parts := splitTopic(topic)
	if len(parts) < 5 {
		return "", "", ""
	}
	if parts[0] != "$thing" || parts[1] != "down" {
		return "", "", ""
	}

	topicType = strings.TrimSpace(parts[2])
	if topicType == "" {
		return "", "", ""
	}

	if len(parts) >= 6 && parts[3] == "custom" {
		productID = strings.TrimSpace(parts[4])
		deviceName = strings.TrimSpace(parts[5])
	} else {
		productID = strings.TrimSpace(parts[3])
		deviceName = strings.TrimSpace(parts[4])
	}

	if productID == "" || deviceName == "" {
		return "", "", ""
	}

	return topicType, productID, deviceName
}

func renderIThingsTopic(template, productID, deviceName string) string {
	topic := strings.TrimSpace(template)
	if topic == "" {
		return ""
	}
	topic = strings.ReplaceAll(topic, "{productID}", strings.TrimSpace(productID))
	topic = strings.ReplaceAll(topic, "{deviceName}", strings.TrimSpace(deviceName))
	return topic
}

func (a *IThingsAdapter) nextID(prefix string) string {
	return nextPrefixedID(prefix, &a.seq)
}
