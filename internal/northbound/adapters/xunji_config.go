//go:build !no_paho_mqtt

package adapters

import (
	"encoding/json"
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
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &XunjiConfig{
		ServerURL: normalizeServerURLWithPort(
			pickConfigString(raw, "serverUrl", "broker", "server_url"),
			pickConfigString(raw, "protocol"),
			pickConfigInt(raw, 0, "port"),
		),
		Username:           strings.TrimSpace(pickConfigString(raw, "username")),
		Password:           strings.TrimSpace(pickConfigString(raw, "password")),
		ClientID:           strings.TrimSpace(pickConfigString(raw, "clientId", "client_id")),
		QOS:                pickConfigInt(raw, 0, "qos"),
		Retain:             pickConfigBool(raw, false, "retain"),
		KeepAlive:          pickConfigInt(raw, 60, "keepAlive", "keep_alive"),
		Timeout:            pickConfigInt(raw, 10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:   pickConfigInt(raw, int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		Topic:              strings.TrimSpace(pickConfigString(raw, "topic", "gatewayTelemetryTopic", "gatewayTopic")),
		AlarmTopic:         strings.TrimSpace(pickConfigString(raw, "alarmTopic", "alarm_topic")),
		GatewayName:        strings.TrimSpace(pickConfigString(raw, "gatewayName", "gateway_name")),
		SubDeviceTokenMode: strings.TrimSpace(pickConfigString(raw, "subDeviceTokenMode")),
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
	if cfg.QOS < 0 || cfg.QOS > 2 {
		return fmt.Errorf("qos must be between 0 and 2")
	}
	if cfg.KeepAlive <= 0 {
		cfg.KeepAlive = 60
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10
	}
	if cfg.UploadIntervalMs <= 0 {
		cfg.UploadIntervalMs = int(defaultReportInterval.Milliseconds())
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		cfg.Topic = defaultXunjiTopicTemplate
	}
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
