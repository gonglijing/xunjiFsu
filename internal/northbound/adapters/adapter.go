package adapters

import (
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
)

// NorthboundAdapter 北向适配器接口
// 所有内置适配器都实现这个接口
type NorthboundAdapter interface {
	// Name 获取名称
	Name() string
	// Type 获取类型
	Type() string
	// Initialize 初始化配置
	Initialize(config string) error
	// Start 启动适配器（启动后台线程）
	Start()
	// Stop 停止适配器（停止后台线程）
	Stop()
	// Close 关闭并释放资源
	Close() error
	// Send 发送采集数据
	Send(data *models.CollectData) error
	// SendAlarm 发送报警
	SendAlarm(alarm *models.AlarmPayload) error
	// SetInterval 设置发送周期
	SetInterval(interval time.Duration)
	// IsEnabled 检查是否启用
	IsEnabled() bool
	// IsConnected 检查连接状态
	IsConnected() bool
	// GetStats 获取统计信息
	GetStats() map[string]interface{}
	// GetLastSendTime 获取最后发送时间
	GetLastSendTime() time.Time
	// PendingCommandCount 获取待处理命令数量
	PendingCommandCount() int
}

type RuntimeStatsSnapshot struct {
	Name                    string
	Type                    string
	Enabled                 bool
	Initialized             bool
	Connected               bool
	LoopState               string
	IntervalMS              int64
	PendingData             int
	PendingAlarm            int
	PendingCmd              int
	Broker                  string
	Topic                   string
	AlarmTopic              string
	ClientID                string
	QOS                     byte
	Retain                  bool
	GatewayName             string
	TelemetryTopic          string
	GatewayTelemetryTopic   string
	GatewayRegisterTopic    string
	RPCRequestTopic         string
	RPCResponseTopic        string
	UpPropertyTopicTemplate string
	DownPropertyTopic       string
	DownActionTopic         string
	ProductKey              string
	DeviceKey               string
	Error                   string
}

func (s RuntimeStatsSnapshot) HasPending() bool {
	return s.PendingData > 0 || s.PendingAlarm > 0 || s.PendingCmd > 0
}

func (s RuntimeStatsSnapshot) ToMap() map[string]interface{} {
	out := make(map[string]interface{}, 18)
	out["name"] = s.Name
	out["type"] = s.Type
	out["enabled"] = s.Enabled
	out["initialized"] = s.Initialized
	out["connected"] = s.Connected
	if s.LoopState != "" {
		out["loop_state"] = s.LoopState
	}
	out["interval_ms"] = s.IntervalMS
	out["pending_data"] = s.PendingData
	out["pending_alarm"] = s.PendingAlarm
	if s.PendingCmd > 0 {
		out["pending_cmd"] = s.PendingCmd
	}
	if s.Broker != "" {
		out["broker"] = s.Broker
	}
	if s.Topic != "" {
		out["topic"] = s.Topic
	}
	if s.AlarmTopic != "" {
		out["alarm_topic"] = s.AlarmTopic
	}
	if s.ClientID != "" {
		out["client_id"] = s.ClientID
	}
	if s.QOS > 0 {
		out["qos"] = s.QOS
	}
	if s.Retain {
		out["retain"] = s.Retain
	}
	if s.GatewayName != "" {
		out["gateway_name"] = s.GatewayName
	}
	if s.TelemetryTopic != "" {
		out["telemetry_topic"] = s.TelemetryTopic
	}
	if s.GatewayTelemetryTopic != "" {
		out["gateway_telemetry_topic"] = s.GatewayTelemetryTopic
	}
	if s.GatewayRegisterTopic != "" {
		out["gateway_register_topic"] = s.GatewayRegisterTopic
	}
	if s.RPCRequestTopic != "" {
		out["rpc_request_topic"] = s.RPCRequestTopic
	}
	if s.RPCResponseTopic != "" {
		out["rpc_response_topic"] = s.RPCResponseTopic
	}
	if s.UpPropertyTopicTemplate != "" {
		out["up_property_topic_template"] = s.UpPropertyTopicTemplate
	}
	if s.DownPropertyTopic != "" {
		out["down_property_topic"] = s.DownPropertyTopic
	}
	if s.DownActionTopic != "" {
		out["down_action_topic"] = s.DownActionTopic
	}
	if s.ProductKey != "" {
		out["product_key"] = s.ProductKey
	}
	if s.DeviceKey != "" {
		out["device_key"] = s.DeviceKey
	}
	if s.Error != "" {
		out["error"] = s.Error
	}
	return out
}

type NorthboundAdapterWithRuntimeStats interface {
	RuntimeStatsSnapshot() RuntimeStatsSnapshot
}

// NorthboundAdapterWithCommands 支持命令下发的适配器接口
type NorthboundAdapterWithCommands interface {
	NorthboundAdapter
	// PullCommands 拉取待执行命令
	PullCommands(limit int) ([]*models.NorthboundCommand, error)
	// ReportCommandResult 上报命令执行结果
	ReportCommandResult(result *models.NorthboundCommandResult) error
}

// NorthboundAdapterWithDeviceSync 支持设备同步能力的适配器接口
type NorthboundAdapterWithDeviceSync interface {
	NorthboundAdapter
	// SyncDevices 触发设备/物模型同步
	SyncDevices() error
}

// NewAdapter 创建指定类型的适配器
func NewAdapter(northboundType, name string) NorthboundAdapter {
	switch nbtype.Normalize(northboundType) {
	case nbtype.TypeMQTT:
		return NewMQTTAdapter(name)
	case nbtype.TypeXunji:
		return NewXunjiAdapter(name)
	case nbtype.TypePandaX:
		return NewPandaXAdapter(name)
	case nbtype.TypeIThings:
		return NewIThingsAdapter(name)
	case nbtype.TypeSagoo:
		return NewSagooAdapter(name)
	default:
		return nil
	}
}

// SupportedTypes 返回支持的北向类型
func SupportedTypes() []string {
	return nbtype.SupportedTypes()
}
