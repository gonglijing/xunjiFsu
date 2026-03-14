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
	defaultIThingsUpPropertyTopicTemplate = "$thing/up/property/{productID}/{deviceName}"
	defaultIThingsUpEventTopicTemplate    = "$thing/up/event/{productID}/{deviceName}"
	defaultIThingsUpActionTopicTemplate   = "$thing/up/action/{productID}/{deviceName}"
	defaultIThingsDownPropertyTopic       = "$thing/down/property/+/+"
	defaultIThingsDownActionTopic         = "$thing/down/action/+/+"
	defaultIThingsAlarmEventID            = "alarm"
	defaultIThingsAlarmEventType          = "alert"
)

// IThingsConfig iThings 北向配置
type IThingsConfig struct {
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

	GatewayMode       bool   `json:"gatewayMode"`
	ProductKey        string `json:"productKey"`
	DeviceKey         string `json:"deviceKey"`
	DeviceNameMode    string `json:"deviceNameMode"`
	SubDeviceNameMode string `json:"subDeviceNameMode"`

	UpPropertyTopicTemplate string `json:"upPropertyTopicTemplate"`
	UpEventTopicTemplate    string `json:"upEventTopicTemplate"`
	UpActionTopicTemplate   string `json:"upActionTopicTemplate"`
	DownPropertyTopic       string `json:"downPropertyTopic"`
	DownActionTopic         string `json:"downActionTopic"`

	AlarmEventID   string `json:"alarmEventID"`
	AlarmEventType string `json:"alarmEventType"`
}

type iThingsRequestState struct {
	RequestID  string
	ProductID  string
	DeviceName string
	Method     string
	TopicType  string
	ActionID   string
	Pending    int
	Success    bool
	Code       int
	Message    string
	FieldName  string
	Value      string
}

// IThingsAdapter iThings 北向适配器
type IThingsAdapter struct {
	name   string
	config *IThingsConfig

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

	upPropertyTopicTemplate string
	upEventTopicTemplate    string
	upActionTopicTemplate   string
	downPropertyTopic       string
	downActionTopic         string
	alarmEventID            string
	alarmEventType          string
	deviceNameMode          string
	subDeviceNameMode       string

	realtimeQueue []*models.CollectData
	dataMu        sync.RWMutex

	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

	requestStates map[string]*iThingsRequestState
	requestMu     sync.Mutex

	flushNow chan struct{}
	stopChan chan struct{}
	wg       sync.WaitGroup

	mu          sync.RWMutex
	initialized bool
	enabled     bool
	connected   bool
	loopState   adapterLoopState
	lastSend    time.Time
	seq         uint64
}

// NewIThingsAdapter 创建 iThings 适配器
func NewIThingsAdapter(name string) *IThingsAdapter {
	return &IThingsAdapter{
		name:          name,
		flushNow:      make(chan struct{}, 1),
		stopChan:      make(chan struct{}),
		realtimeQueue: make([]*models.CollectData, 0),
		alarmQueue:    make([]*models.AlarmPayload, 0),
		commandQueue:  make([]*models.NorthboundCommand, 0),
		requestStates: make(map[string]*iThingsRequestState),
		loopState:     adapterLoopStopped,
	}
}

func (a *IThingsAdapter) Name() string {
	return a.name
}

func (a *IThingsAdapter) Type() string {
	return "ithings"
}

func (a *IThingsAdapter) Initialize(configStr string) error {
	cfg, err := parseIThingsConfig(configStr)
	if err != nil {
		return err
	}

	_ = a.Close()

	settings := buildIThingsInitSettings(a.name, cfg)

	client, err := connectMQTT(settings.broker, settings.clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.applyConfig(cfg, client, settings)

	a.subscribeDownTopics(client)

	log.Printf("iThings adapter initialized: %s (broker=%s)", a.name, settings.broker)
	return nil
}
