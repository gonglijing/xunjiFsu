package adapters

import (
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
	raw, err := parseAdapterRawConfig(configStr)
	if err != nil {
		return nil, err
	}

	cfg := &SagooConfig{
		ProductKey:           raw.string("productKey", "product_key", "productID", "product_id"),
		DeviceKey:            raw.string("deviceKey", "device_key", "deviceName", "device_name"),
		Username:             raw.string("username"),
		Password:             raw.string("password"),
		Topic:                raw.string("topic"),
		AlarmTopic:           raw.string("alarmTopic", "alarm_topic"),
		ClientID:             raw.string("clientId", "client_id"),
		QOS:                  raw.int(0, "qos"),
		Retain:               raw.bool(false, "retain"),
		KeepAlive:            raw.int(60, "keepAlive", "keep_alive"),
		Timeout:              raw.int(10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:     raw.int(int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms"),
		ReportIntervalMs:     raw.int(0, "reportIntervalMs", "report_interval_ms"),
		AlarmFlushIntervalMs: raw.int(int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs", "alarm_flush_interval_ms"),
		AlarmBatchSize:       raw.int(defaultAlarmBatch, "alarmBatchSize", "alarm_batch_size"),
		AlarmQueueSize:       raw.int(defaultAlarmQueue, "alarmQueueSize", "alarm_queue_size"),
		RealtimeQueueSize:    raw.int(defaultRealtimeQueue, "realtimeQueueSize", "realtime_queue_size"),
	}

	cfg.ServerURL = raw.normalizedServerURL("serverUrl", "server_url", "broker")

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
	if err := validateConfigQOS(cfg.QOS); err != nil {
		return err
	}
	applyDefaultPositiveInt(&cfg.KeepAlive, 60)
	applyDefaultPositiveInt(&cfg.Timeout, 10)
	applyFallbackOrDefaultPositiveInt(&cfg.UploadIntervalMs, cfg.ReportIntervalMs, int(defaultReportInterval.Milliseconds()))
	applyDefaultPositiveInt(&cfg.AlarmFlushIntervalMs, int(defaultAlarmInterval.Milliseconds()))
	applyDefaultPositiveInt(&cfg.AlarmBatchSize, defaultAlarmBatch)
	applyDefaultPositiveInt(&cfg.AlarmQueueSize, defaultAlarmQueue)
	applyDefaultPositiveInt(&cfg.RealtimeQueueSize, defaultRealtimeQueue)

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
