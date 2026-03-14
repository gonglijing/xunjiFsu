package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type sagooInitSettings struct {
	broker      string
	clientID    string
	qos         byte
	retain      bool
	topic       string
	alarmTopic  string
	timeout     time.Duration
	reportEvery time.Duration
	alarmEvery  time.Duration
	alarmBatch  int
	alarmCap    int
	realtimeCap int
	commandCap  int
}

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

func buildSagooInitSettings(adapterName string, cfg *SagooConfig) sagooInitSettings {
	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("sagoo-%s-%s-%d", cfg.ProductKey, cfg.DeviceKey, time.Now().UnixNano())
	}

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

	uploadMs := cfg.UploadIntervalMs
	if uploadMs <= 0 {
		uploadMs = cfg.ReportIntervalMs
	}

	return sagooInitSettings{
		broker:      normalizeBroker(cfg.ServerURL),
		clientID:    clientID,
		qos:         clampQOS(cfg.QOS),
		retain:      cfg.Retain,
		topic:       topic,
		alarmTopic:  alarmTopic,
		timeout:     timeout,
		reportEvery: resolveInterval(uploadMs, defaultReportInterval),
		alarmEvery:  resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval),
		alarmBatch:  resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch),
		alarmCap:    resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue),
		realtimeCap: resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue),
		commandCap:  resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue),
	}
}

func (a *SagooAdapter) applyConfig(cfg *SagooConfig, client mqtt.Client, settings sagooInitSettings) {
	a.mu.Lock()
	a.config = cfg
	a.client = client
	a.qos = settings.qos
	a.retain = settings.retain
	a.topic = settings.topic
	a.alarmTopic = settings.alarmTopic
	a.timeout = settings.timeout
	a.reportEvery = settings.reportEvery
	a.alarmEvery = settings.alarmEvery
	a.alarmBatch = settings.alarmBatch
	a.alarmCap = settings.alarmCap
	a.realtimeCap = settings.realtimeCap
	a.commandCap = settings.commandCap
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.initialized = true
	a.connected = true
	a.loopState = adapterLoopStopped
	a.mu.Unlock()
}
