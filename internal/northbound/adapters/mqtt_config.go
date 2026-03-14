//go:build !no_paho_mqtt

package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttInitSettings struct {
	broker       string
	topic        string
	alarmTopic   string
	clientID     string
	qos          byte
	retain       bool
	cleanSession bool
	timeout      time.Duration
	keepAlive    time.Duration
	interval     time.Duration
}

func parseMQTTConfig(configStr string) (*MQTTConfig, error) {
	cfg := &MQTTConfig{}
	if err := json.Unmarshal([]byte(configStr), cfg); err != nil {
		return nil, fmt.Errorf("failed to parse MQTT config: %w", err)
	}
	if err := normalizeMQTTConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func normalizeMQTTConfig(cfg *MQTTConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if strings.TrimSpace(cfg.Broker) == "" {
		return fmt.Errorf("broker is required")
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		return fmt.Errorf("topic is required")
	}
	if err := validateConfigQOS(cfg.QOS); err != nil {
		return err
	}
	applyDefaultPositiveInt(&cfg.ConnectTimeout, 10)
	applyDefaultPositiveInt(&cfg.KeepAlive, 60)
	applyMinimumPositiveInt(&cfg.UploadInterval, 500)
	return nil
}

func buildMQTTInitSettings(cfg *MQTTConfig) mqttInitSettings {
	topic := cfg.Topic
	alarmTopic := cfg.AlarmTopic
	if alarmTopic == "" {
		alarmTopic = topic + "/alarm"
	}

	clientID := cfg.ClientID
	if clientID == "" {
		clientID = fmt.Sprintf("fsu-mqtt-%d", time.Now().UnixNano())
	}

	interval := defaultReportInterval
	if cfg.UploadInterval > 0 {
		interval = time.Duration(cfg.UploadInterval) * time.Millisecond
	}

	return mqttInitSettings{
		broker:       normalizeBroker(cfg.Broker),
		topic:        topic,
		alarmTopic:   alarmTopic,
		clientID:     clientID,
		qos:          clampQOS(cfg.QOS),
		retain:       cfg.Retain,
		cleanSession: cfg.CleanSession,
		timeout:      time.Duration(cfg.ConnectTimeout) * time.Second,
		keepAlive:    time.Duration(cfg.KeepAlive) * time.Second,
		interval:     interval,
	}
}

func (a *MQTTAdapter) applyConfig(cfg *MQTTConfig, client mqtt.Client, settings mqttInitSettings) {
	a.mu.Lock()
	a.config = cfg
	a.broker = settings.broker
	a.topic = settings.topic
	a.alarmTopic = settings.alarmTopic
	a.clientID = settings.clientID
	a.username = cfg.Username
	a.password = cfg.Password
	a.qos = settings.qos
	a.retain = settings.retain
	a.cleanSession = settings.cleanSession
	a.timeout = settings.timeout
	a.keepAlive = settings.keepAlive
	a.interval = settings.interval
	a.client = client
	a.initialized = true
	a.connected = true
	a.loopState = adapterLoopStopped
	a.mu.Unlock()
}
