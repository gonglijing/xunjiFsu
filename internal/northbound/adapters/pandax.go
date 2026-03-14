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
	defaultPandaXGatewayTelemetryTopic  = "v1/gateway/telemetry"
	defaultPandaXGatewayRegisterTopic   = "v1/gateway/register/telemetry"
	defaultPandaXGatewayAttributesTopic = "v1/gateway/attributes"
	defaultPandaXTelemetryTopic         = "v1/devices/me/telemetry"
	defaultPandaXAttributesTopic        = "v1/devices/me/attributes"
	defaultPandaXRowTopic               = "v1/devices/me/row"
	defaultPandaXRPCRequestTopic        = "v1/devices/me/rpc/request"
	defaultPandaXRPCResponseTopic       = "v1/devices/me/rpc/response"
	defaultPandaXEventPrefix            = "v1/devices/event"
	defaultPandaXAlarmIdentifier        = "alarm"
	defaultPandaXReconnectInterval      = 5 * time.Second
	maxPandaXReconnectInterval          = 5 * time.Minute
)

// PandaXConfig PandaX 北向配置
type PandaXConfig struct {
	ServerURL string `json:"serverUrl"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ClientID  string `json:"clientId"`
	QOS       int    `json:"qos"`
	Retain    bool   `json:"retain"`
	KeepAlive int    `json:"keepAlive"`
	Timeout   int    `json:"connectTimeout"`

	UploadIntervalMs     int `json:"uploadIntervalMs"`
	AlarmFlushIntervalMs int `json:"alarmFlushIntervalMs"`
	AlarmBatchSize       int `json:"alarmBatchSize"`
	AlarmQueueSize       int `json:"alarmQueueSize"`
	RealtimeQueueSize    int `json:"realtimeQueueSize"`
	CommandQueueSize     int `json:"commandQueueSize"`

	GatewayMode            bool   `json:"gatewayMode"`
	SubDeviceTokenMode     string `json:"subDeviceTokenMode"`
	TelemetryTopic         string `json:"telemetryTopic"`
	AttributesTopic        string `json:"attributesTopic"`
	RowTopic               string `json:"rowTopic"`
	GatewayTelemetryTopic  string `json:"gatewayTelemetryTopic"`
	GatewayRegisterTopic   string `json:"gatewayRegisterTopic"`
	GatewayAttributesTopic string `json:"gatewayAttributesTopic"`
	EventTopicPrefix       string `json:"eventTopicPrefix"`
	AlarmTopic             string `json:"alarmTopic"`
	AlarmIdentifier        string `json:"alarmIdentifier"`
	RPCRequestTopic        string `json:"rpcRequestTopic"`
	RPCResponseTopic       string `json:"rpcResponseTopic"`

	ProductKey string `json:"productKey"`
	DeviceKey  string `json:"deviceKey"`
}

// SystemStatsProvider 系统属性提供者接口
type SystemStatsProvider interface {
	CollectSystemStatsOnce() *models.SystemStats
}

// PandaXAdapter PandaX 北向适配器
type PandaXAdapter struct {
	name   string
	config *PandaXConfig

	client  mqtt.Client
	qos     byte
	retain  bool
	timeout time.Duration

	reportEvery time.Duration
	alarmEvery  time.Duration
	alarmBatch  int
	alarmCap    int
	realtimeCap int
	commandCap  int

	telemetryTopic         string
	attributesTopic        string
	rowTopic               string
	gatewayTelemetryTopic  string
	gatewayRegisterTopic   string
	gatewayAttributesTopic string
	eventTopicPrefix       string
	alarmTopic             string
	alarmIdentifier        string
	rpcRequestTopic        string
	rpcResponseTopic       string

	realtimeQueue []*models.CollectData
	dataMu        sync.RWMutex

	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

	flushNow     chan struct{}
	stopChan     chan struct{}
	reconnectNow chan struct{}
	wg           sync.WaitGroup

	mu                sync.RWMutex
	initialized       bool
	enabled           bool
	connected         bool
	loopState         adapterLoopState
	reconnectInterval time.Duration
	lastSend          time.Time
	seq               uint64

	// 系统属性提供者
	systemStatsProvider SystemStatsProvider
}

// NewPandaXAdapter 创建 PandaX 适配器
func NewPandaXAdapter(name string) *PandaXAdapter {
	return &PandaXAdapter{
		name:              name,
		flushNow:          make(chan struct{}, 1),
		stopChan:          make(chan struct{}),
		reconnectNow:      make(chan struct{}, 1),
		realtimeQueue:     make([]*models.CollectData, 0),
		alarmQueue:        make([]*models.AlarmPayload, 0),
		commandQueue:      make([]*models.NorthboundCommand, 0),
		reconnectInterval: defaultPandaXReconnectInterval,
		loopState:         adapterLoopStopped,
	}
}

func (a *PandaXAdapter) Name() string {
	return a.name
}

func (a *PandaXAdapter) Type() string {
	return "pandax"
}

// SetSystemStatsProvider 设置系统属性提供者
func (a *PandaXAdapter) SetSystemStatsProvider(provider SystemStatsProvider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemStatsProvider = provider
	log.Printf("[PandaX-%s] SetSystemStatsProvider: 系统属性提供者已设置", a.name)
}

func (a *PandaXAdapter) Initialize(configStr string) error {
	log.Printf("[PandaX-%s] Initialize: 开始初始化", a.name)

	cfg, err := parsePandaXConfig(configStr)
	if err != nil {
		log.Printf("[PandaX-%s] Initialize: 配置解析失败: %v", a.name, err)
		return err
	}
	log.Printf("[PandaX-%s] Initialize: 配置解析成功, serverUrl=%s, username=%s, gatewayMode=%v",
		a.name, cfg.ServerURL, cfg.Username, cfg.GatewayMode)

	_ = a.Close()

	settings := buildPandaXInitSettings(a.name, cfg)
	log.Printf("[PandaX-%s] Initialize: broker=%s", a.name, settings.broker)
	log.Printf("[PandaX-%s] Initialize: clientId=%s", a.name, settings.clientID)

	client, err := a.connectPandaXMQTT(settings.broker, settings.clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		log.Printf("[PandaX-%s] Initialize: MQTT 连接失败: %v", a.name, err)
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}
	log.Printf("[PandaX-%s] Initialize: MQTT 连接成功", a.name)

	a.applyConfig(cfg, client, settings)

	a.subscribeRPCTopics(client)

	log.Printf("PandaX adapter initialized: %s (broker=%s)", a.name, settings.broker)
	return nil
}
