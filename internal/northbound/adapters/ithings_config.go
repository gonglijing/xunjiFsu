package adapters

import (
	"encoding/json"
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
