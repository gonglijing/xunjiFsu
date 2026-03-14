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

// MQTTConfig MQTT配置
type MQTTConfig struct {
	Broker         string `json:"broker"`
	Topic          string `json:"topic"`
	AlarmTopic     string `json:"alarm_topic"`
	ClientID       string `json:"client_id"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	QOS            int    `json:"qos"`
	Retain         bool   `json:"retain"`
	CleanSession   bool   `json:"clean_session"`
	KeepAlive      int    `json:"keep_alive"`
	ConnectTimeout int    `json:"connect_timeout"`
	UploadInterval int    `json:"upload_interval"`
}

const (
	mqttPendingDataCap  = 1000
	mqttPendingAlarmCap = 100
)

// MQTTAdapter MQTT北向适配器
// 每个 MQTTAdapter 自己管理自己的状态和发送线程
type MQTTAdapter struct {
	name         string
	config       *MQTTConfig
	broker       string
	topic        string
	alarmTopic   string
	clientID     string
	username     string
	password     string
	qos          byte
	retain       bool
	cleanSession bool
	timeout      time.Duration
	keepAlive    time.Duration
	interval     time.Duration
	lastSend     time.Time

	// MQTT客户端
	client mqtt.Client

	// 数据缓冲
	pendingData []*models.CollectData
	pendingMu   sync.RWMutex

	// 报警缓冲
	pendingAlarms []*models.AlarmPayload
	alarmMu       sync.RWMutex

	// 控制通道
	stopChan     chan struct{}
	dataChan     chan struct{}
	reconnectNow chan struct{}
	wg           sync.WaitGroup

	// 状态
	mu                sync.RWMutex
	initialized       bool
	enabled           bool
	connected         bool
	loopState         adapterLoopState
	reconnectInterval time.Duration
}

// NewMQTTAdapter 创建MQTT适配器
func NewMQTTAdapter(name string) *MQTTAdapter {
	return &MQTTAdapter{
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

// Name 获取名称
func (a *MQTTAdapter) Name() string {
	return a.name
}

// Type 获取类型
func (a *MQTTAdapter) Type() string {
	return "mqtt"
}

// Initialize 初始化
func (a *MQTTAdapter) Initialize(configStr string) error {
	cfg, err := parseMQTTConfig(configStr)
	if err != nil {
		return err
	}

	settings := buildMQTTInitSettings(cfg)

	// 连接MQTT
	client, err := a.connectMQTT(settings, cfg.Username, cfg.Password)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.applyConfig(cfg, client, settings)

	log.Printf("MQTT adapter initialized: %s (broker=%s, topic=%s)", a.name, settings.broker, settings.topic)
	return nil
}
