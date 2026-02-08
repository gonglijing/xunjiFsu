package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	defaultPandaXGatewayTelemetryTopic  = "v1/gateway/telemetry"
	defaultPandaXGatewayAttributesTopic = "v1/gateway/attributes"
	defaultPandaXTelemetryTopic         = "v1/devices/me/telemetry"
	defaultPandaXAttributesTopic        = "v1/devices/me/attributes"
	defaultPandaXRowTopic               = "v1/devices/me/row"
	defaultPandaXRPCRequestTopic        = "v1/devices/me/rpc/request"
	defaultPandaXRPCResponseTopic       = "v1/devices/me/rpc/response"
	defaultPandaXEventPrefix            = "v1/devices/event"
	defaultPandaXAlarmIdentifier        = "alarm"
)

// PandaXConfig PandaX 北向配置
type PandaXConfig struct {
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

	GatewayMode            bool   `json:"gatewayMode"`
	SubDeviceTokenMode     string `json:"subDeviceTokenMode"`
	TelemetryTopic         string `json:"telemetryTopic"`
	AttributesTopic        string `json:"attributesTopic"`
	RowTopic               string `json:"rowTopic"`
	GatewayTelemetryTopic  string `json:"gatewayTelemetryTopic"`
	GatewayAttributesTopic string `json:"gatewayAttributesTopic"`
	EventTopicPrefix       string `json:"eventTopicPrefix"`
	AlarmTopic             string `json:"alarmTopic"`
	AlarmIdentifier        string `json:"alarmIdentifier"`
	RPCRequestTopic        string `json:"rpcRequestTopic"`
	RPCResponseTopic       string `json:"rpcResponseTopic"`

	ProductKey string `json:"productKey"`
	DeviceKey  string `json:"deviceKey"`
}

// PandaXAdapter PandaX 北向适配器
type PandaXAdapter struct {
	name   string
	config *PandaXConfig

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

	telemetryTopic         string
	attributesTopic        string
	rowTopic               string
	gatewayTelemetryTopic  string
	gatewayAttributesTopic string
	eventTopicPrefix       string
	alarmTopic             string
	alarmIdentifier        string
	rpcRequestTopic        string
	rpcResponseTopic       string

	realtimeQueue []*models.CollectData
	dataMu        sync.RWMutex

	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

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

// NewPandaXAdapter 创建 PandaX 适配器
func NewPandaXAdapter(name string) *PandaXAdapter {
	return &PandaXAdapter{
		name:          name,
		flushNow:      make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		realtimeQueue: make([]*models.CollectData, 0),
		alarmQueue:    make([]*models.AlarmPayload, 0),
		commandQueue:  make([]*models.NorthboundCommand, 0),
	}
}

func (a *PandaXAdapter) Name() string {
	return a.name
}

func (a *PandaXAdapter) Type() string {
	return "pandax"
}

func (a *PandaXAdapter) Initialize(configStr string) error {
	cfg, err := parsePandaXConfig(configStr)
	if err != nil {
		return err
	}

	_ = a.Close()

	broker := normalizeBroker(cfg.ServerURL)
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("pandax-%s-%d", a.name, time.Now().UnixNano())
	}

	client, err := connectMQTT(broker, clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	topicTelemetry := pickFirstNonEmpty(cfg.TelemetryTopic, defaultPandaXTelemetryTopic)
	topicAttributes := pickFirstNonEmpty(cfg.AttributesTopic, defaultPandaXAttributesTopic)
	topicRow := pickFirstNonEmpty(cfg.RowTopic, defaultPandaXRowTopic)
	topicGatewayTelemetry := pickFirstNonEmpty(cfg.GatewayTelemetryTopic, defaultPandaXGatewayTelemetryTopic)
	topicGatewayAttributes := pickFirstNonEmpty(cfg.GatewayAttributesTopic, defaultPandaXGatewayAttributesTopic)
	eventTopicPrefix := pickFirstNonEmpty(cfg.EventTopicPrefix, defaultPandaXEventPrefix)
	alarmIdentifier := pickFirstNonEmpty(cfg.AlarmIdentifier, defaultPandaXAlarmIdentifier)
	alarmTopic := strings.TrimSpace(cfg.AlarmTopic)
	if alarmTopic == "" {
		alarmTopic = strings.TrimRight(eventTopicPrefix, "/") + "/" + alarmIdentifier
	}

	a.mu.Lock()
	a.config = cfg
	a.client = client
	a.qos = clampQOS(cfg.QOS)
	a.retain = cfg.Retain
	a.timeout = time.Duration(maxInt(cfg.Timeout, 10)) * time.Second
	a.reportEvery = resolveInterval(cfg.UploadIntervalMs, defaultReportInterval)
	a.alarmEvery = resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval)
	a.alarmBatch = resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch)
	a.alarmCap = resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue)
	a.realtimeCap = resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue)
	a.commandCap = resolvePositive(cfg.CommandQueueSize, defaultRealtimeQueue)
	a.telemetryTopic = topicTelemetry
	a.attributesTopic = topicAttributes
	a.rowTopic = topicRow
	a.gatewayTelemetryTopic = topicGatewayTelemetry
	a.gatewayAttributesTopic = topicGatewayAttributes
	a.eventTopicPrefix = eventTopicPrefix
	a.alarmTopic = alarmTopic
	a.alarmIdentifier = alarmIdentifier
	a.rpcRequestTopic = pickFirstNonEmpty(cfg.RPCRequestTopic, defaultPandaXRPCRequestTopic)
	a.rpcResponseTopic = pickFirstNonEmpty(cfg.RPCResponseTopic, defaultPandaXRPCResponseTopic)
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.initialized = true
	a.connected = true
	a.enabled = false
	a.mu.Unlock()

	a.subscribeRPCTopics(client)

	log.Printf("PandaX adapter initialized: %s (broker=%s)", a.name, broker)
	return nil
}

func (a *PandaXAdapter) Start() {
	a.mu.Lock()
	if a.initialized && !a.enabled {
		a.enabled = true
		if a.stopChan == nil {
			a.stopChan = make(chan struct{})
		}
		a.wg.Add(2)
		go a.reportLoop()
		go a.alarmLoop()
		log.Printf("PandaX adapter started: %s", a.name)
	}
	a.mu.Unlock()
}

func (a *PandaXAdapter) Stop() {
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
	log.Printf("PandaX adapter stopped: %s", a.name)
}

func (a *PandaXAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = resolveInterval(int(interval.Milliseconds()), defaultReportInterval)
	a.mu.Unlock()
}

func (a *PandaXAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *PandaXAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

func (a *PandaXAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(cloneCollectData(data))
	a.dataMu.Unlock()

	return nil
}

func (a *PandaXAdapter) SendAlarm(alarm *models.AlarmPayload) error {
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

func (a *PandaXAdapter) Close() error {
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
	a.mu.Unlock()

	return nil
}

func (a *PandaXAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
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

	items := make([]*models.NorthboundCommand, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, a.commandQueue[0])
		a.commandQueue = a.commandQueue[1:]
	}

	return items, nil
}

func (a *PandaXAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	if result == nil || strings.TrimSpace(result.RequestID) == "" {
		return nil
	}

	a.mu.RLock()
	initialized := a.initialized
	topic := a.rpcResponseTopic
	a.mu.RUnlock()
	if !initialized {
		return nil
	}

	code := result.Code
	if code == 0 {
		if result.Success {
			code = 200
		} else {
			code = 500
		}
	}
	message := strings.TrimSpace(result.Message)
	if message == "" {
		if result.Success {
			message = "success"
		} else {
			message = "failed"
		}
	}

	resp := map[string]interface{}{
		"requestId": result.RequestID,
		"method":    "write",
		"params": map[string]interface{}{
			"success":    result.Success,
			"code":       code,
			"message":    message,
			"productKey": result.ProductKey,
			"deviceKey":  result.DeviceKey,
			"fieldName":  result.FieldName,
			"value":      convertFieldValue(result.Value),
		},
	}
	body, _ := json.Marshal(resp)

	return a.publish(topic, body)
}

func (a *PandaXAdapter) GetStats() map[string]interface{} {
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
		"name":                    a.name,
		"type":                    "pandax",
		"enabled":                 a.enabled,
		"initialized":             a.initialized,
		"connected":               a.connected && a.client != nil && a.client.IsConnected(),
		"interval_ms":             a.reportEvery.Milliseconds(),
		"pending_data":            pendingData,
		"pending_alarm":           pendingAlarm,
		"pending_cmd":             pendingCmd,
		"telemetry_topic":         a.telemetryTopic,
		"gateway_telemetry_topic": a.gatewayTelemetryTopic,
		"rpc_request_topic":       a.rpcRequestTopic,
		"rpc_response_topic":      a.rpcResponseTopic,
	}
}

func (a *PandaXAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

func (a *PandaXAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}

func (a *PandaXAdapter) reportLoop() {
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
				log.Printf("PandaX realtime flush failed: %v", err)
			}
		}
	}
}

func (a *PandaXAdapter) alarmLoop() {
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
					log.Printf("PandaX alarm flush failed on close: %v", err)
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
				log.Printf("PandaX alarm flush failed: %v", err)
			}
		case <-flushNow:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("PandaX alarm flush failed: %v", err)
			}
		}
	}
}

func (a *PandaXAdapter) flushRealtime() error {
	a.dataMu.Lock()
	if len(a.realtimeQueue) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := make([]*models.CollectData, len(a.realtimeQueue))
	for i := 0; i < len(a.realtimeQueue); i++ {
		batch[i] = cloneCollectData(a.realtimeQueue[i])
	}
	a.realtimeQueue = a.realtimeQueue[:0]
	a.dataMu.Unlock()

	for _, item := range batch {
		topic, body := a.buildRealtimePublish(item)
		if err := a.publish(topic, body); err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			return err
		}
	}

	return nil
}

func (a *PandaXAdapter) flushAlarmBatch() error {
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
	a.alarmMu.Unlock()

	for _, item := range batch {
		topic, body := a.buildAlarmPublish(item)
		if err := a.publish(topic, body); err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
	}

	return nil
}

func (a *PandaXAdapter) buildRealtimePublish(data *models.CollectData) (string, []byte) {
	if data == nil {
		return a.gatewayTelemetryTopic, []byte("{}")
	}

	a.mu.RLock()
	topic := a.gatewayTelemetryTopic
	a.mu.RUnlock()

	values := make(map[string]interface{}, len(data.Fields))
	for key, value := range data.Fields {
		values[key] = convertFieldValue(value)
	}

	ts := data.Timestamp.UnixMilli()
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}

	subToken := a.resolveSubDeviceToken(data)
	payload := map[string]interface{}{
		subToken: map[string]interface{}{
			"ts":     ts,
			"values": values,
		},
	}
	body, _ := json.Marshal(payload)
	return topic, body
}

func (a *PandaXAdapter) buildAlarmPublish(alarm *models.AlarmPayload) (string, []byte) {
	a.mu.RLock()
	topic := a.alarmTopic
	a.mu.RUnlock()

	if alarm == nil {
		return topic, []byte("{}")
	}

	payload := map[string]interface{}{
		"device_name":  alarm.DeviceName,
		"product_key":  alarm.ProductKey,
		"device_key":   alarm.DeviceKey,
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"severity":     alarm.Severity,
		"message":      alarm.Message,
		"ts":           time.Now().UnixMilli(),
	}
	body, _ := json.Marshal(payload)
	return topic, body
}

func (a *PandaXAdapter) publish(topic string, payload []byte) error {
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
		a.subscribeRPCTopics(client)
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

func (a *PandaXAdapter) subscribeRPCTopics(client mqtt.Client) {
	a.mu.RLock()
	rpcReqTopic := a.rpcRequestTopic
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	topics := make(map[string]struct{})
	if strings.TrimSpace(rpcReqTopic) != "" {
		topics[strings.TrimSpace(rpcReqTopic)] = struct{}{}
		if !strings.HasSuffix(strings.TrimSpace(rpcReqTopic), "/+") {
			topics[strings.TrimRight(strings.TrimSpace(rpcReqTopic), "/")+"/+"] = struct{}{}
		}
	}

	for topic := range topics {
		token := client.Subscribe(topic, qos, a.handleRPCRequest)
		if !token.WaitTimeout(timeout) {
			continue
		}
		if err := token.Error(); err != nil {
			log.Printf("PandaX subscribe failed topic=%s: %v", topic, err)
		}
	}
}

func (a *PandaXAdapter) handleRPCRequest(_ mqtt.Client, message mqtt.Message) {
	var req struct {
		RequestID string      `json:"requestId"`
		Method    string      `json:"method"`
		Params    interface{} `json:"params"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		return
	}

	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = requestIDFromPandaXRPCTopic(message.Topic())
	}

	commands := a.buildCommandsFromRPC(req.RequestID, req.Method, req.Params)
	if len(commands) == 0 {
		return
	}

	a.commandMu.Lock()
	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		if len(a.commandQueue) >= a.commandCap && len(a.commandQueue) > 0 {
			a.commandQueue = a.commandQueue[1:]
		}
		a.commandQueue = append(a.commandQueue, cmd)
	}
	a.commandMu.Unlock()
}

func (a *PandaXAdapter) buildCommandsFromRPC(requestID, method string, params interface{}) []*models.NorthboundCommand {
	defaultPK, defaultDK := a.defaultIdentity()
	commands := buildPandaXCommands(requestID, method, params, defaultPK, defaultDK)
	if len(commands) == 0 {
		return nil
	}

	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		cmd.Source = "pandax.rpc.request"
	}

	return commands
}

func buildPandaXCommands(requestID, method string, params interface{}, defaultPK, defaultDK string) []*models.NorthboundCommand {
	out := make([]*models.NorthboundCommand, 0)
	appendProperties := func(pk, dk string, props map[string]interface{}) {
		if len(props) == 0 {
			return
		}
		keys := make([]string, 0, len(props))
		for key := range props {
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
			item := &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: pk,
				DeviceKey:  dk,
				FieldName:  key,
				Value:      stringifyAny(props[key]),
			}
			if strings.TrimSpace(item.ProductKey) == "" || strings.TrimSpace(item.DeviceKey) == "" {
				continue
			}
			out = append(out, item)
		}
	}

	obj, ok := params.(map[string]interface{})
	if ok {
		pk := pickFirstNonEmpty(pandaXPickString(obj, "productKey", "product_key"), defaultPK)
		dk := pickFirstNonEmpty(pandaXPickString(obj, "deviceKey", "device_key"), defaultDK)

		if props, ok := mapFromAny(obj["properties"]); ok {
			appendProperties(pk, dk, props)
		}

		for _, key := range []string{"sub_device", "subDevice"} {
			sub, ok := mapFromAny(obj[key])
			if !ok {
				continue
			}
			subPK := pk
			subDK := dk
			if identity, ok := mapFromAny(sub["identity"]); ok {
				subPK = pickFirstNonEmpty(pandaXPickString(identity, "productKey", "product_key"), subPK)
				subDK = pickFirstNonEmpty(pandaXPickString(identity, "deviceKey", "device_key"), subDK)
			}
			if props, ok := mapFromAny(sub["properties"]); ok {
				appendProperties(subPK, subDK, props)
			}
		}

		for _, key := range []string{"sub_devices", "subDevices"} {
			list, ok := obj[key].([]interface{})
			if !ok || len(list) == 0 {
				continue
			}
			for _, item := range list {
				row, ok := mapFromAny(item)
				if !ok {
					continue
				}
				subPK := pk
				subDK := dk
				if identity, ok := mapFromAny(row["identity"]); ok {
					subPK = pickFirstNonEmpty(pandaXPickString(identity, "productKey", "product_key"), subPK)
					subDK = pickFirstNonEmpty(pandaXPickString(identity, "deviceKey", "device_key"), subDK)
				}
				if props, ok := mapFromAny(row["properties"]); ok {
					appendProperties(subPK, subDK, props)
				}
			}
		}

		if fieldName := strings.TrimSpace(pandaXPickString(obj, "fieldName", "field_name")); fieldName != "" {
			if rawValue, exists := obj["value"]; exists {
				appendProperties(pk, dk, map[string]interface{}{fieldName: rawValue})
			}
		}

		if len(out) == 0 {
			generic := make(map[string]interface{})
			reserved := map[string]struct{}{
				"productKey": {}, "product_key": {}, "deviceKey": {}, "device_key": {},
				"properties": {}, "sub_device": {}, "subDevice": {}, "sub_devices": {}, "subDevices": {},
				"fieldName": {}, "field_name": {}, "value": {},
			}
			for key, value := range obj {
				if _, exists := reserved[key]; exists {
					continue
				}
				generic[key] = value
			}
			appendProperties(pk, dk, generic)
		}
	}

	if len(out) == 0 && strings.TrimSpace(method) != "" {
		if strings.TrimSpace(defaultPK) != "" && strings.TrimSpace(defaultDK) != "" {
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: defaultPK,
				DeviceKey:  defaultDK,
				FieldName:  strings.TrimSpace(method),
				Value:      stringifyAny(params),
			})
		}
	}

	return out
}

func (a *PandaXAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	if len(a.realtimeQueue) >= a.realtimeCap {
		a.realtimeQueue = a.realtimeQueue[1:]
	}
	a.realtimeQueue = append(a.realtimeQueue, item)
}

func (a *PandaXAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	queue := make([]*models.CollectData, 0, len(items)+len(a.realtimeQueue))
	queue = append(queue, items...)
	queue = append(queue, a.realtimeQueue...)
	if len(queue) > a.realtimeCap {
		queue = queue[:a.realtimeCap]
	}
	a.realtimeQueue = queue
}

func (a *PandaXAdapter) enqueueAlarmLocked(item *models.AlarmPayload) {
	if item == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	if len(a.alarmQueue) >= a.alarmCap {
		a.alarmQueue = a.alarmQueue[1:]
	}
	a.alarmQueue = append(a.alarmQueue, item)
}

func (a *PandaXAdapter) prependAlarms(items []*models.AlarmPayload) {
	if len(items) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	queue := make([]*models.AlarmPayload, 0, len(items)+len(a.alarmQueue))
	queue = append(queue, items...)
	queue = append(queue, a.alarmQueue...)
	if len(queue) > a.alarmCap {
		queue = queue[:a.alarmCap]
	}
	a.alarmQueue = queue
}

func (a *PandaXAdapter) defaultIdentity() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config == nil {
		return "", ""
	}
	return strings.TrimSpace(a.config.ProductKey), strings.TrimSpace(a.config.DeviceKey)
}

func (a *PandaXAdapter) resolveSubDeviceToken(data *models.CollectData) string {
	a.mu.RLock()
	cfg := a.config
	a.mu.RUnlock()

	if cfg == nil {
		if strings.TrimSpace(data.DeviceName) != "" {
			return strings.TrimSpace(data.DeviceName)
		}
		if strings.TrimSpace(data.DeviceKey) != "" {
			return strings.TrimSpace(data.DeviceKey)
		}
		return fmt.Sprintf("device_%d", data.DeviceID)
	}

	pk := pickFirstNonEmpty(data.ProductKey, cfg.ProductKey)
	name := pickFirstNonEmpty(data.DeviceName, data.DeviceKey)
	dk := pickFirstNonEmpty(data.DeviceKey, data.DeviceName)

	switch strings.ToLower(strings.TrimSpace(cfg.SubDeviceTokenMode)) {
	case "devicekey", "device_key":
		if dk != "" {
			return dk
		}
	case "product_devicekey", "product_device_key":
		if pk != "" && dk != "" {
			return pk + "_" + dk
		}
	case "product_devicename", "product_device_name":
		if pk != "" && name != "" {
			return pk + "_" + name
		}
	default:
		if name != "" {
			return name
		}
	}

	if dk != "" {
		return dk
	}
	if name != "" {
		return name
	}
	return fmt.Sprintf("device_%d", data.DeviceID)
}

func (a *PandaXAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func parsePandaXConfig(configStr string) (*PandaXConfig, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &PandaXConfig{}
	cfg.ServerURL = normalizePandaXServerURL(
		pandaXPickString(raw, "serverUrl", "broker", "server_url"),
		pandaXPickString(raw, "protocol"),
		pandaXPickInt(raw, 0, "port"),
	)

	cfg.Username = strings.TrimSpace(pandaXPickString(raw, "username", "token", "deviceToken"))
	cfg.Password = strings.TrimSpace(pandaXPickString(raw, "password"))
	cfg.ClientID = strings.TrimSpace(pandaXPickString(raw, "clientId", "client_id"))
	cfg.QOS = pandaXPickInt(raw, 0, "qos")
	cfg.Retain = pandaXPickBool(raw, false, "retain")
	cfg.KeepAlive = pandaXPickInt(raw, 60, "keepAlive", "keep_alive")
	cfg.Timeout = pandaXPickInt(raw, 10, "connectTimeout", "connect_timeout", "timeout")

	cfg.UploadIntervalMs = pandaXPickInt(raw, 5000, "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs")
	cfg.AlarmFlushIntervalMs = pandaXPickInt(raw, 2000, "alarmFlushIntervalMs")
	cfg.AlarmBatchSize = pandaXPickInt(raw, 20, "alarmBatchSize")
	cfg.AlarmQueueSize = pandaXPickInt(raw, 1000, "alarmQueueSize")
	cfg.RealtimeQueueSize = pandaXPickInt(raw, 1000, "realtimeQueueSize")
	cfg.CommandQueueSize = pandaXPickInt(raw, cfg.RealtimeQueueSize, "commandQueueSize")

	cfg.GatewayMode = pandaXPickBool(raw, true, "gatewayMode")
	cfg.SubDeviceTokenMode = strings.TrimSpace(pandaXPickString(raw, "subDeviceTokenMode"))
	cfg.TelemetryTopic = strings.TrimSpace(pandaXPickString(raw, "telemetryTopic", "topic"))
	cfg.AttributesTopic = strings.TrimSpace(pandaXPickString(raw, "attributesTopic"))
	cfg.RowTopic = strings.TrimSpace(pandaXPickString(raw, "rowTopic"))
	cfg.GatewayTelemetryTopic = strings.TrimSpace(pandaXPickString(raw, "gatewayTelemetryTopic"))
	cfg.GatewayAttributesTopic = strings.TrimSpace(pandaXPickString(raw, "gatewayAttributesTopic"))
	cfg.EventTopicPrefix = strings.TrimSpace(pandaXPickString(raw, "eventTopicPrefix"))
	cfg.AlarmTopic = strings.TrimSpace(pandaXPickString(raw, "alarmTopic"))
	cfg.AlarmIdentifier = strings.TrimSpace(pandaXPickString(raw, "alarmIdentifier"))
	cfg.RPCRequestTopic = strings.TrimSpace(pandaXPickString(raw, "rpcRequestTopic"))
	cfg.RPCResponseTopic = strings.TrimSpace(pandaXPickString(raw, "rpcResponseTopic"))

	cfg.ProductKey = strings.TrimSpace(pandaXPickString(raw, "productKey", "product_key"))
	cfg.DeviceKey = strings.TrimSpace(pandaXPickString(raw, "deviceKey", "device_key"))

	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("serverUrl is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if !cfg.GatewayMode {
		return nil, fmt.Errorf("PandaX adapter only supports gatewayMode=true")
	}
	if cfg.QOS < 0 || cfg.QOS > 2 {
		return nil, fmt.Errorf("qos must be between 0 and 2")
	}
	if cfg.UploadIntervalMs <= 0 {
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
	if cfg.CommandQueueSize <= 0 {
		cfg.CommandQueueSize = cfg.RealtimeQueueSize
	}

	return cfg, nil
}

func normalizePandaXServerURL(serverURL, protocol string, port int) string {
	serverURL = strings.TrimSpace(serverURL)
	if serverURL == "" {
		return ""
	}

	if !strings.Contains(serverURL, "://") {
		transport := strings.TrimSpace(protocol)
		if transport == "" {
			transport = "tcp"
		}
		serverURL = transport + "://" + serverURL
	}

	if port <= 0 {
		return serverURL
	}

	parsed, err := url.Parse(serverURL)
	if err != nil {
		return serverURL
	}
	if parsed.Port() != "" {
		return serverURL
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return serverURL
	}
	parsed.Host = net.JoinHostPort(hostname, strconv.Itoa(port))
	return parsed.String()
}

func pandaXPickString(data map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		if text, ok := value.(string); ok {
			trimmed := strings.TrimSpace(text)
			if trimmed != "" {
				return trimmed
			}
			continue
		}
		str := strings.TrimSpace(fmt.Sprintf("%v", value))
		if str != "" {
			return str
		}
	}
	return ""
}

func pandaXPickInt(data map[string]interface{}, fallback int, keys ...string) int {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int8:
			return int(v)
		case int16:
			return int(v)
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float32:
			return int(v)
		case float64:
			return int(v)
		case string:
			trimmed := strings.TrimSpace(v)
			if trimmed == "" {
				continue
			}
			number, err := strconv.Atoi(trimmed)
			if err == nil {
				return number
			}
		}
	}
	return fallback
}

func pandaXPickBool(data map[string]interface{}, fallback bool, keys ...string) bool {
	for _, key := range keys {
		value, ok := data[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v
		case int:
			return v != 0
		case int64:
			return v != 0
		case float64:
			return v != 0
		case string:
			trimmed := strings.TrimSpace(strings.ToLower(v))
			if trimmed == "" {
				continue
			}
			if trimmed == "true" || trimmed == "1" || trimmed == "yes" {
				return true
			}
			if trimmed == "false" || trimmed == "0" || trimmed == "no" {
				return false
			}
		}
	}
	return fallback
}

func requestIDFromPandaXRPCTopic(topic string) string {
	parts := splitTopic(topic)
	if len(parts) < 6 {
		return ""
	}
	if parts[0] != "v1" || parts[1] != "devices" || parts[2] != "me" || parts[3] != "rpc" || parts[4] != "request" {
		return ""
	}
	return strings.TrimSpace(parts[5])
}

func maxInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for i := 1; i < len(values); i++ {
		if values[i] > max {
			max = values[i]
		}
	}
	return max
}

func (a *PandaXAdapter) nextID(prefix string) string {
	n := atomic.AddUint64(&a.seq, 1)
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixMilli(), n)
}
