//go:build !no_paho_mqtt

package adapters

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type xunjiInitSettings struct {
	broker             string
	gatewayName        string
	topic              string
	alarmTopic         string
	clientID           string
	subDeviceTokenMode string
	qos                byte
	retain             bool
	timeout            time.Duration
	keepAlive          time.Duration
	interval           time.Duration
}

func parseXunjiConfig(configStr string) (*XunjiConfig, error) {
	raw, err := parseAdapterRawConfig(configStr)
	if err != nil {
		return nil, err
	}

	cfg := &XunjiConfig{
		ServerURL:          raw.normalizedServerURL("serverUrl", "broker", "server_url"),
		Username:           raw.string("username"),
		Password:           raw.string("password"),
		ClientID:           raw.string("clientId", "client_id"),
		QOS:                raw.int(0, "qos"),
		Retain:             raw.bool(false, "retain"),
		KeepAlive:          raw.int(60, "keepAlive", "keep_alive"),
		Timeout:            raw.int(10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:   raw.int(int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		Topic:              raw.string("topic", "gatewayTelemetryTopic", "gatewayTopic"),
		AlarmTopic:         raw.string("alarmTopic", "alarm_topic"),
		GatewayName:        raw.string("gatewayName", "gateway_name"),
		SubDeviceTokenMode: raw.string("subDeviceTokenMode"),
	}

	if err := normalizeXunjiConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizeXunjiConfig(cfg *XunjiConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		return fmt.Errorf("serverUrl is required")
	}
	if err := validateConfigQOS(cfg.QOS); err != nil {
		return err
	}
	applyDefaultPositiveInt(&cfg.KeepAlive, 60)
	applyDefaultPositiveInt(&cfg.Timeout, 10)
	applyDefaultPositiveInt(&cfg.UploadIntervalMs, int(defaultReportInterval.Milliseconds()))
	applyDefaultString(&cfg.Topic, defaultXunjiTopicTemplate)
	return nil
}

func buildXunjiInitSettings(adapterName string, cfg *XunjiConfig) xunjiInitSettings {
	broker := normalizeBroker(cfg.ServerURL)
	gatewayName := resolveXunjiGatewayName(cfg.GatewayName, adapterName)
	topic := renderXunjiTopic(cfg.Topic, gatewayName)
	alarmTopic := strings.TrimSpace(cfg.AlarmTopic)
	if alarmTopic == "" {
		alarmTopic = topic + "/alarm"
	} else {
		alarmTopic = renderXunjiTopic(alarmTopic, gatewayName)
	}

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("xunji-%s-%d", adapterName, time.Now().UnixNano())
	}

	timeout := 10 * time.Second
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	keepAlive := 60 * time.Second
	if cfg.KeepAlive > 0 {
		keepAlive = time.Duration(cfg.KeepAlive) * time.Second
	}

	interval := defaultReportInterval
	if cfg.UploadIntervalMs > 0 {
		interval = time.Duration(cfg.UploadIntervalMs) * time.Millisecond
	}

	return xunjiInitSettings{
		broker:             broker,
		gatewayName:        gatewayName,
		topic:              topic,
		alarmTopic:         alarmTopic,
		clientID:           clientID,
		subDeviceTokenMode: strings.TrimSpace(cfg.SubDeviceTokenMode),
		qos:                clampQOS(cfg.QOS),
		retain:             cfg.Retain,
		timeout:            timeout,
		keepAlive:          keepAlive,
		interval:           interval,
	}
}

func (a *XunjiAdapter) applyConfig(cfg *XunjiConfig, client mqtt.Client, settings xunjiInitSettings) {
	a.mu.Lock()
	a.config = cfg
	a.broker = settings.broker
	a.topic = settings.topic
	a.alarmTopic = settings.alarmTopic
	a.gatewayName = settings.gatewayName
	a.subDeviceTokenMode = settings.subDeviceTokenMode
	a.clientID = settings.clientID
	a.username = cfg.Username
	a.password = cfg.Password
	a.qos = settings.qos
	a.retain = settings.retain
	a.timeout = settings.timeout
	a.keepAlive = settings.keepAlive
	a.interval = settings.interval
	a.client = client
	a.initialized = true
	a.connected = true
	a.loopState = adapterLoopStopped
	a.mu.Unlock()
}
