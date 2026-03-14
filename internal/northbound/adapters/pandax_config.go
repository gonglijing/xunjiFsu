package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type pandaXInitSettings struct {
	broker                 string
	clientID               string
	qos                    byte
	retain                 bool
	timeout                time.Duration
	reportEvery            time.Duration
	alarmEvery             time.Duration
	alarmBatch             int
	alarmCap               int
	realtimeCap            int
	commandCap             int
	telemetryTopic         string
	attributesTopic        string
	rowTopic               string
	gatewayTelemetryTopic  string
	gatewayRegisterTopic   string
	gatewayAttributesTopic string
	eventTopicPrefix       string
	alarmTopic             string
	alarmIdentifier        string
	rpcRequestTopic        string
	rpcResponseTopic       string
}

func parsePandaXConfig(configStr string) (*PandaXConfig, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &PandaXConfig{
		ServerURL: normalizePandaXServerURL(
			pickConfigString(raw, "serverUrl", "broker", "server_url"),
			pickConfigString(raw, "protocol"),
			pickConfigInt(raw, 0, "port"),
		),
		Username:               strings.TrimSpace(pickConfigString(raw, "username", "token", "deviceToken")),
		Password:               strings.TrimSpace(pickConfigString(raw, "password")),
		ClientID:               strings.TrimSpace(pickConfigString(raw, "clientId", "client_id")),
		QOS:                    pickConfigInt(raw, 0, "qos"),
		Retain:                 pickConfigBool(raw, false, "retain"),
		KeepAlive:              pickConfigInt(raw, 60, "keepAlive", "keep_alive"),
		Timeout:                pickConfigInt(raw, 10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:       pickConfigInt(raw, int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		AlarmFlushIntervalMs:   pickConfigInt(raw, int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs"),
		AlarmBatchSize:         pickConfigInt(raw, defaultAlarmBatch, "alarmBatchSize"),
		AlarmQueueSize:         pickConfigInt(raw, defaultAlarmQueue, "alarmQueueSize"),
		RealtimeQueueSize:      pickConfigInt(raw, defaultRealtimeQueue, "realtimeQueueSize"),
		GatewayMode:            pickConfigBool(raw, true, "gatewayMode"),
		SubDeviceTokenMode:     strings.TrimSpace(pickConfigString(raw, "subDeviceTokenMode")),
		TelemetryTopic:         strings.TrimSpace(pickConfigString(raw, "telemetryTopic", "topic")),
		AttributesTopic:        strings.TrimSpace(pickConfigString(raw, "attributesTopic")),
		RowTopic:               strings.TrimSpace(pickConfigString(raw, "rowTopic")),
		GatewayTelemetryTopic:  strings.TrimSpace(pickConfigString(raw, "gatewayTelemetryTopic")),
		GatewayRegisterTopic:   strings.TrimSpace(pickConfigString(raw, "gatewayRegisterTopic", "registerTopic")),
		GatewayAttributesTopic: strings.TrimSpace(pickConfigString(raw, "gatewayAttributesTopic")),
		EventTopicPrefix:       strings.TrimSpace(pickConfigString(raw, "eventTopicPrefix")),
		AlarmTopic:             strings.TrimSpace(pickConfigString(raw, "alarmTopic")),
		AlarmIdentifier:        strings.TrimSpace(pickConfigString(raw, "alarmIdentifier")),
		RPCRequestTopic:        strings.TrimSpace(pickConfigString(raw, "rpcRequestTopic")),
		RPCResponseTopic:       strings.TrimSpace(pickConfigString(raw, "rpcResponseTopic")),
		ProductKey:             strings.TrimSpace(pickConfigString(raw, "productKey", "product_key")),
		DeviceKey:              strings.TrimSpace(pickConfigString(raw, "deviceKey", "device_key")),
	}

	cfg.CommandQueueSize = pickConfigInt(raw, cfg.RealtimeQueueSize, "commandQueueSize")

	if err := normalizePandaXConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizePandaXConfig(cfg *PandaXConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}
	if cfg.Username == "" {
		return fmt.Errorf("username is required")
	}
	if !cfg.GatewayMode {
		return fmt.Errorf("PandaX adapter only supports gatewayMode=true")
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
	if cfg.GatewayRegisterTopic == "" {
		cfg.GatewayRegisterTopic = defaultPandaXGatewayRegisterTopic
	}

	return nil
}

func normalizePandaXServerURL(serverURL, protocol string, port int) string {
	return normalizeServerURLWithPort(serverURL, protocol, port)
}

func buildPandaXInitSettings(adapterName string, cfg *PandaXConfig) pandaXInitSettings {
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("pandax-%s-%d", adapterName, time.Now().UnixNano())
	}

	topicTelemetry := pickFirstNonEmpty(cfg.TelemetryTopic, defaultPandaXTelemetryTopic)
	topicAttributes := pickFirstNonEmpty(cfg.AttributesTopic, defaultPandaXAttributesTopic)
	topicRow := pickFirstNonEmpty(cfg.RowTopic, defaultPandaXRowTopic)
	topicGatewayTelemetry := pickFirstNonEmpty(cfg.GatewayTelemetryTopic, defaultPandaXGatewayTelemetryTopic)
	topicGatewayRegister := pickFirstNonEmpty(cfg.GatewayRegisterTopic, defaultPandaXGatewayRegisterTopic)
	topicGatewayAttributes := pickFirstNonEmpty(cfg.GatewayAttributesTopic, defaultPandaXGatewayAttributesTopic)
	eventTopicPrefix := pickFirstNonEmpty(cfg.EventTopicPrefix, defaultPandaXEventPrefix)
	alarmIdentifier := pickFirstNonEmpty(cfg.AlarmIdentifier, defaultPandaXAlarmIdentifier)
	alarmTopic := strings.TrimSpace(cfg.AlarmTopic)
	if alarmTopic == "" {
		alarmTopic = strings.TrimRight(eventTopicPrefix, "/") + "/" + alarmIdentifier
	}

	return pandaXInitSettings{
		broker:                 normalizeBroker(cfg.ServerURL),
		clientID:               clientID,
		qos:                    clampQOS(cfg.QOS),
		retain:                 cfg.Retain,
		timeout:                time.Duration(resolvePositive(cfg.Timeout, 10)) * time.Second,
		reportEvery:            resolveInterval(cfg.UploadIntervalMs, defaultReportInterval),
		alarmEvery:             resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval),
		alarmBatch:             resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch),
		alarmCap:               resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue),
		realtimeCap:            resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue),
		commandCap:             resolvePositive(cfg.CommandQueueSize, defaultRealtimeQueue),
		telemetryTopic:         topicTelemetry,
		attributesTopic:        topicAttributes,
		rowTopic:               topicRow,
		gatewayTelemetryTopic:  topicGatewayTelemetry,
		gatewayRegisterTopic:   topicGatewayRegister,
		gatewayAttributesTopic: topicGatewayAttributes,
		eventTopicPrefix:       eventTopicPrefix,
		alarmTopic:             alarmTopic,
		alarmIdentifier:        alarmIdentifier,
		rpcRequestTopic:        pickFirstNonEmpty(cfg.RPCRequestTopic, defaultPandaXRPCRequestTopic),
		rpcResponseTopic:       pickFirstNonEmpty(cfg.RPCResponseTopic, defaultPandaXRPCResponseTopic),
	}
}

func (a *PandaXAdapter) applyConfig(cfg *PandaXConfig, client mqtt.Client, settings pandaXInitSettings) {
	a.mu.Lock()
	a.config = cfg
	a.client = client
	a.qos = settings.qos
	a.retain = settings.retain
	a.timeout = settings.timeout
	a.reportEvery = settings.reportEvery
	a.alarmEvery = settings.alarmEvery
	a.alarmBatch = settings.alarmBatch
	a.alarmCap = settings.alarmCap
	a.realtimeCap = settings.realtimeCap
	a.commandCap = settings.commandCap
	a.telemetryTopic = settings.telemetryTopic
	a.attributesTopic = settings.attributesTopic
	a.rowTopic = settings.rowTopic
	a.gatewayTelemetryTopic = settings.gatewayTelemetryTopic
	a.gatewayRegisterTopic = settings.gatewayRegisterTopic
	a.gatewayAttributesTopic = settings.gatewayAttributesTopic
	a.eventTopicPrefix = settings.eventTopicPrefix
	a.alarmTopic = settings.alarmTopic
	a.alarmIdentifier = settings.alarmIdentifier
	a.rpcRequestTopic = settings.rpcRequestTopic
	a.rpcResponseTopic = settings.rpcResponseTopic
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.reconnectNow = make(chan struct{}, 1)
	a.initialized = true
	a.connected = true
	a.enabled = false
	a.loopState = adapterLoopStopped
	a.mu.Unlock()
}
