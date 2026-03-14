package adapters

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// SagooConfig 循迹北向配置（与 models.SagooConfig 保持一致）
type SagooConfig struct {
	ProductKey string `json:"productKey"`
	DeviceKey  string `json:"deviceKey"`
	ServerURL  string `json:"serverUrl"` // MQTT服务器地址
	Username   string `json:"username"`
	Password   string `json:"password"`
	Topic      string `json:"topic"`
	AlarmTopic string `json:"alarmTopic"`
	ClientID   string `json:"clientId"`
	QOS        int    `json:"qos"`
	Retain     bool   `json:"retain"`
	KeepAlive  int    `json:"keepAlive"`      // 秒
	Timeout    int    `json:"connectTimeout"` // 秒
	// 插件上报周期（毫秒）。<=0 使用默认值。
	UploadIntervalMs int `json:"uploadIntervalMs"`
	// 兼容旧字段：插件内管理上传节奏（毫秒）。<=0 使用默认值。
	ReportIntervalMs int `json:"reportIntervalMs"`
	// 报警批量发送周期（毫秒）。<=0 使用默认值。
	AlarmFlushIntervalMs int `json:"alarmFlushIntervalMs"`
	// 单次批量发送报警条数。<=0 使用默认值。
	AlarmBatchSize int `json:"alarmBatchSize"`
	// 内部报警队列大小（超出后丢弃最旧数据）。<=0 使用默认值。
	AlarmQueueSize int `json:"alarmQueueSize"`
	// 内部实时队列大小（超出后丢弃最旧数据）。<=0 使用默认值。
	RealtimeQueueSize int `json:"realtimeQueueSize"`
}

// SagooAdapter 循迹北向适配器
// 每个 SagooAdapter 自己管理自己的状态和发送线程
type SagooAdapter struct {
	name        string
	config      *SagooConfig
	client      mqtt.Client
	topic       string
	alarmTopic  string
	qos         byte
	retain      bool
	timeout     time.Duration
	enabled     bool
	reportEvery time.Duration
	alarmEvery  time.Duration
	alarmBatch  int
	alarmCap    int
	realtimeCap int
	commandCap  int

	// 数据缓冲
	latestData []*models.CollectData
	dataMu     sync.RWMutex

	// 报警缓冲
	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	// 命令队列
	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

	// 控制通道
	flushNow chan struct{}
	stopChan chan struct{}
	wg       sync.WaitGroup

	// 状态
	mu          sync.RWMutex
	initialized bool
	connected   bool
	loopState   adapterLoopState
	seq         uint64
}

// NewSagooAdapter 创建循迹适配器
func NewSagooAdapter(name string) *SagooAdapter {
	return &SagooAdapter{
		name:         name,
		stopChan:     make(chan struct{}),
		flushNow:     make(chan struct{}, 1),
		latestData:   make([]*models.CollectData, 0),
		alarmQueue:   make([]*models.AlarmPayload, 0),
		commandQueue: make([]*models.NorthboundCommand, 0),
		loopState:    adapterLoopStopped,
	}
}

// Name 获取名称
func (a *SagooAdapter) Name() string {
	return a.name
}

// Type 获取类型
func (a *SagooAdapter) Type() string {
	return "sagoo"
}

// Initialize 初始化
func (a *SagooAdapter) Initialize(configStr string) error {
	cfg, err := parseSagooConfig(configStr)
	if err != nil {
		return err
	}

	// 清理旧连接
	_ = a.Close()

	settings := buildSagooInitSettings(a.name, cfg)

	// 连接 MQTT
	client, err := connectMQTT(settings.broker, settings.clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}

	a.applyConfig(cfg, client, settings)

	// 订阅命令主题
	a.subscribeCommandTopics(client)

	log.Printf("Sagoo adapter initialized: %s (broker=%s, topic=%s)",
		a.name, settings.broker, settings.topic)
	return nil
}

// 辅助函数
func (a *SagooAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func (a *SagooAdapter) defaultIdentity() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config == nil {
		return "", ""
	}
	return strings.TrimSpace(a.config.ProductKey), strings.TrimSpace(a.config.DeviceKey)
}

func (a *SagooAdapter) nextID(prefix string) string {
	return nextPrefixedID(prefix, &a.seq)
}
