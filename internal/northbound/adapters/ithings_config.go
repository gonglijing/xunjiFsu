package adapters

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type iThingsInitSettings struct {
	broker                  string
	clientID                string
	qos                     byte
	retain                  bool
	timeout                 time.Duration
	reportEvery             time.Duration
	alarmEvery              time.Duration
	alarmBatch              int
	alarmCap                int
	realtimeCap             int
	commandCap              int
	upPropertyTopicTemplate string
	upEventTopicTemplate    string
	upActionTopicTemplate   string
	downPropertyTopic       string
	downActionTopic         string
	alarmEventID            string
	alarmEventType          string
	deviceNameMode          string
	subDeviceNameMode       string
}

func parseIThingsConfig(configStr string) (*IThingsConfig, error) {
	raw, err := parseAdapterRawConfig(configStr)
	if err != nil {
		return nil, err
	}

	cfg := &IThingsConfig{
		ServerURL:               raw.string("serverUrl", "broker", "server_url"),
		Username:                raw.string("username"),
		Password:                raw.string("password"),
		ClientID:                raw.string("clientId", "client_id"),
		QOS:                     raw.int(0, "qos"),
		Retain:                  raw.bool(false, "retain"),
		KeepAlive:               raw.int(60, "keepAlive", "keep_alive"),
		Timeout:                 raw.int(10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:        raw.int(int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		AlarmFlushIntervalMs:    raw.int(int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs"),
		AlarmBatchSize:          raw.int(defaultAlarmBatch, "alarmBatchSize"),
		AlarmQueueSize:          raw.int(defaultAlarmQueue, "alarmQueueSize"),
		RealtimeQueueSize:       raw.int(defaultRealtimeQueue, "realtimeQueueSize"),
		GatewayMode:             raw.bool(true, "gatewayMode"),
		ProductKey:              raw.string("productKey", "productID", "product_id"),
		DeviceKey:               raw.string("deviceKey", "deviceName", "device_name"),
		DeviceNameMode:          raw.string("deviceNameMode"),
		SubDeviceNameMode:       raw.string("subDeviceNameMode"),
		UpPropertyTopicTemplate: raw.string("upPropertyTopicTemplate"),
		UpEventTopicTemplate:    raw.string("upEventTopicTemplate"),
		UpActionTopicTemplate:   raw.string("upActionTopicTemplate"),
		DownPropertyTopic:       raw.string("downPropertyTopic"),
		DownActionTopic:         raw.string("downActionTopic"),
		AlarmEventID:            raw.string("alarmEventID"),
		AlarmEventType:          raw.string("alarmEventType"),
	}

	cfg.CommandQueueSize = raw.int(cfg.RealtimeQueueSize, "commandQueueSize")

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
	if err := validateConfigQOS(cfg.QOS); err != nil {
		return err
	}
	applyDefaultPositiveInt(&cfg.UploadIntervalMs, int(defaultReportInterval.Milliseconds()))
	applyDefaultPositiveInt(&cfg.AlarmFlushIntervalMs, int(defaultAlarmInterval.Milliseconds()))
	applyDefaultPositiveInt(&cfg.AlarmBatchSize, defaultAlarmBatch)
	applyDefaultPositiveInt(&cfg.AlarmQueueSize, defaultAlarmQueue)
	applyDefaultPositiveInt(&cfg.RealtimeQueueSize, defaultRealtimeQueue)
	applyFallbackPositiveInt(&cfg.CommandQueueSize, cfg.RealtimeQueueSize)

	applyDefaultString(&cfg.UpPropertyTopicTemplate, defaultIThingsUpPropertyTopicTemplate)
	applyDefaultString(&cfg.UpEventTopicTemplate, defaultIThingsUpEventTopicTemplate)
	applyDefaultString(&cfg.UpActionTopicTemplate, defaultIThingsUpActionTopicTemplate)
	applyDefaultString(&cfg.DownPropertyTopic, defaultIThingsDownPropertyTopic)
	applyDefaultString(&cfg.DownActionTopic, defaultIThingsDownActionTopic)
	applyDefaultString(&cfg.AlarmEventID, defaultIThingsAlarmEventID)
	applyDefaultString(&cfg.AlarmEventType, defaultIThingsAlarmEventType)
	applyDefaultString(&cfg.DeviceNameMode, "deviceKey")
	applyDefaultString(&cfg.SubDeviceNameMode, cfg.DeviceNameMode)

	return nil
}

func buildIThingsInitSettings(adapterName string, cfg *IThingsConfig) iThingsInitSettings {
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("ithings-%s-%d", adapterName, time.Now().UnixNano())
	}

	deviceNameMode := pickFirstNonEmpty(cfg.DeviceNameMode, "deviceKey")

	return iThingsInitSettings{
		broker:                  normalizeBroker(cfg.ServerURL),
		clientID:                clientID,
		qos:                     clampQOS(cfg.QOS),
		retain:                  cfg.Retain,
		timeout:                 time.Duration(resolvePositive(cfg.Timeout, 10)) * time.Second,
		reportEvery:             resolveInterval(cfg.UploadIntervalMs, defaultReportInterval),
		alarmEvery:              resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval),
		alarmBatch:              resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch),
		alarmCap:                resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue),
		realtimeCap:             resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue),
		commandCap:              resolvePositive(cfg.CommandQueueSize, defaultRealtimeQueue),
		upPropertyTopicTemplate: pickFirstNonEmpty(cfg.UpPropertyTopicTemplate, defaultIThingsUpPropertyTopicTemplate),
		upEventTopicTemplate:    pickFirstNonEmpty(cfg.UpEventTopicTemplate, defaultIThingsUpEventTopicTemplate),
		upActionTopicTemplate:   pickFirstNonEmpty(cfg.UpActionTopicTemplate, defaultIThingsUpActionTopicTemplate),
		downPropertyTopic:       pickFirstNonEmpty(cfg.DownPropertyTopic, defaultIThingsDownPropertyTopic),
		downActionTopic:         pickFirstNonEmpty(cfg.DownActionTopic, defaultIThingsDownActionTopic),
		alarmEventID:            pickFirstNonEmpty(cfg.AlarmEventID, defaultIThingsAlarmEventID),
		alarmEventType:          pickFirstNonEmpty(cfg.AlarmEventType, defaultIThingsAlarmEventType),
		deviceNameMode:          deviceNameMode,
		subDeviceNameMode:       pickFirstNonEmpty(cfg.SubDeviceNameMode, deviceNameMode),
	}
}

func (a *IThingsAdapter) applyConfig(cfg *IThingsConfig, client mqtt.Client, settings iThingsInitSettings) {
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
	a.upPropertyTopicTemplate = settings.upPropertyTopicTemplate
	a.upEventTopicTemplate = settings.upEventTopicTemplate
	a.upActionTopicTemplate = settings.upActionTopicTemplate
	a.downPropertyTopic = settings.downPropertyTopic
	a.downActionTopic = settings.downActionTopic
	a.alarmEventID = settings.alarmEventID
	a.alarmEventType = settings.alarmEventType
	a.deviceNameMode = settings.deviceNameMode
	a.subDeviceNameMode = settings.subDeviceNameMode
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.requestStates = make(map[string]*iThingsRequestState)
	a.initialized = true
	a.connected = true
	a.enabled = false
	a.loopState = adapterLoopStopped
	a.mu.Unlock()
}
