package adapters

import (
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
	raw, err := parseAdapterRawConfig(configStr)
	if err != nil {
		return nil, err
	}

	cfg := &PandaXConfig{
		ServerURL:              normalizePandaXServerURL(raw.string("serverUrl", "broker", "server_url"), raw.string("protocol"), raw.int(0, "port")),
		Username:               raw.string("username", "token", "deviceToken"),
		Password:               raw.string("password"),
		ClientID:               raw.string("clientId", "client_id"),
		QOS:                    raw.int(0, "qos"),
		Retain:                 raw.bool(false, "retain"),
		KeepAlive:              raw.int(60, "keepAlive", "keep_alive"),
		Timeout:                raw.int(10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:       raw.int(int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		AlarmFlushIntervalMs:   raw.int(int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs"),
		AlarmBatchSize:         raw.int(defaultAlarmBatch, "alarmBatchSize"),
		AlarmQueueSize:         raw.int(defaultAlarmQueue, "alarmQueueSize"),
		RealtimeQueueSize:      raw.int(defaultRealtimeQueue, "realtimeQueueSize"),
		GatewayMode:            raw.bool(true, "gatewayMode"),
		SubDeviceTokenMode:     raw.string("subDeviceTokenMode"),
		TelemetryTopic:         raw.string("telemetryTopic", "topic"),
		AttributesTopic:        raw.string("attributesTopic"),
		RowTopic:               raw.string("rowTopic"),
		GatewayTelemetryTopic:  raw.string("gatewayTelemetryTopic"),
		GatewayRegisterTopic:   raw.string("gatewayRegisterTopic", "registerTopic"),
		GatewayAttributesTopic: raw.string("gatewayAttributesTopic"),
		EventTopicPrefix:       raw.string("eventTopicPrefix"),
		AlarmTopic:             raw.string("alarmTopic"),
		AlarmIdentifier:        raw.string("alarmIdentifier"),
		RPCRequestTopic:        raw.string("rpcRequestTopic"),
		RPCResponseTopic:       raw.string("rpcResponseTopic"),
		ProductKey:             raw.string("productKey", "product_key"),
		DeviceKey:              raw.string("deviceKey", "device_key"),
	}

	cfg.CommandQueueSize = raw.int(cfg.RealtimeQueueSize, "commandQueueSize")

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
	if err := validateConfigQOS(cfg.QOS); err != nil {
		return err
	}
	applyDefaultPositiveInt(&cfg.UploadIntervalMs, int(defaultReportInterval.Milliseconds()))
	applyDefaultPositiveInt(&cfg.AlarmFlushIntervalMs, int(defaultAlarmInterval.Milliseconds()))
	applyDefaultPositiveInt(&cfg.AlarmBatchSize, defaultAlarmBatch)
	applyDefaultPositiveInt(&cfg.AlarmQueueSize, defaultAlarmQueue)
	applyDefaultPositiveInt(&cfg.RealtimeQueueSize, defaultRealtimeQueue)
	applyFallbackPositiveInt(&cfg.CommandQueueSize, cfg.RealtimeQueueSize)
	applyDefaultString(&cfg.GatewayRegisterTopic, defaultPandaXGatewayRegisterTopic)

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
