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
		ServerURL:               raw.pickString("serverUrl", "broker", "server_url"),
		Username:                raw.pickString("username"),
		Password:                raw.pickString("password"),
		ClientID:                raw.pickString("clientId", "client_id"),
		QOS:                     raw.pickInt(0, "qos"),
		Retain:                  raw.pickBool(false, "retain"),
		KeepAlive:               raw.pickInt(60, "keepAlive", "keep_alive"),
		Timeout:                 raw.pickInt(10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:        raw.pickInt(int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		AlarmFlushIntervalMs:    raw.pickInt(int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs"),
		AlarmBatchSize:          raw.pickInt(defaultAlarmBatch, "alarmBatchSize"),
		AlarmQueueSize:          raw.pickInt(defaultAlarmQueue, "alarmQueueSize"),
		RealtimeQueueSize:       raw.pickInt(defaultRealtimeQueue, "realtimeQueueSize"),
		GatewayMode:             raw.pickBool(true, "gatewayMode"),
		ProductKey:              raw.pickString("productKey", "productID", "product_id"),
		DeviceKey:               raw.pickString("deviceKey", "deviceName", "device_name"),
		DeviceNameMode:          raw.pickString("deviceNameMode"),
		SubDeviceNameMode:       raw.pickString("subDeviceNameMode"),
		UpPropertyTopicTemplate: raw.pickString("upPropertyTopicTemplate"),
		UpEventTopicTemplate:    raw.pickString("upEventTopicTemplate"),
		UpActionTopicTemplate:   raw.pickString("upActionTopicTemplate"),
		DownPropertyTopic:       raw.pickString("downPropertyTopic"),
		DownActionTopic:         raw.pickString("downActionTopic"),
		AlarmEventID:            raw.pickString("alarmEventID"),
		AlarmEventType:          raw.pickString("alarmEventType"),
	}

	cfg.CommandQueueSize = raw.pickInt(cfg.RealtimeQueueSize, "commandQueueSize")

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
