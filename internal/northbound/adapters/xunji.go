//go:build !no_paho_mqtt

package adapters

import (
	"fmt"
	"log"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	defaultXunjiTopicTemplate = "v1/gateway/{gatewayname}"
	xunjiPendingDataCap       = 1000
	xunjiPendingAlarmCap      = 100
)

type XunjiConfig struct {
	ServerURL          string `json:"serverUrl"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	ClientID           string `json:"clientId"`
	QOS                int    `json:"qos"`
	Retain             bool   `json:"retain"`
	KeepAlive          int    `json:"keepAlive"`
	Timeout            int    `json:"connectTimeout"`
	UploadIntervalMs   int    `json:"uploadIntervalMs"`
	Topic              string `json:"topic"`
	AlarmTopic         string `json:"alarmTopic"`
	GatewayName        string `json:"gatewayName"`
	SubDeviceTokenMode string `json:"subDeviceTokenMode"`
}

type XunjiAdapter struct {
	name               string
	config             *XunjiConfig
	broker             string
	topic              string
	alarmTopic         string
	gatewayName        string
	subDeviceTokenMode string
	clientID           string
	username           string
	password           string
	qos                byte
	retain             bool
	timeout            time.Duration
	keepAlive          time.Duration
	interval           time.Duration
	lastSend           time.Time

	client mqtt.Client

	pendingData []*models.CollectData
	pendingMu   sync.RWMutex

	pendingAlarms []*models.AlarmPayload
	alarmMu       sync.RWMutex

	stopChan     chan struct{}
	dataChan     chan struct{}
	reconnectNow chan struct{}
	wg           sync.WaitGroup

	mu                sync.RWMutex
	initialized       bool
	enabled           bool
	connected         bool
	loopState         adapterLoopState
	reconnectInterval time.Duration
}

func NewXunjiAdapter(name string) *XunjiAdapter {
	return &XunjiAdapter{
		name:              name,
		lastSend:          time.Time{},
		interval:          defaultReportInterval,
		reconnectInterval: defaultReconnectInterval,
		stopChan:          make(chan struct{}),
		dataChan:          make(chan struct{}, 1),
		reconnectNow:      make(chan struct{}, 1),
		pendingData:       make([]*models.CollectData, 0),
		pendingAlarms:     make([]*models.AlarmPayload, 0),
		loopState:         adapterLoopStopped,
	}
}

func (a *XunjiAdapter) Name() string { return a.name }

func (a *XunjiAdapter) Type() string { return "xunji" }

func (a *XunjiAdapter) Initialize(configStr string) error {
	cfg, err := parseXunjiConfig(configStr)
	if err != nil {
		return err
	}

	settings := buildXunjiInitSettings(a.name, cfg)

	client, err := connectMQTT(settings.broker, settings.clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.applyConfig(cfg, client, settings)

	log.Printf("Xunji adapter initialized: %s (broker=%s, topic=%s)", a.name, settings.broker, settings.topic)
	return nil
}
