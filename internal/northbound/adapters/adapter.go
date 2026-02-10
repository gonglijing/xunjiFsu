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

// NorthboundAdapterWithCommands 支持命令下发的适配器接口
type NorthboundAdapterWithCommands interface {
	NorthboundAdapter
	// PullCommands 拉取待执行命令
	PullCommands(limit int) ([]*models.NorthboundCommand, error)
	// ReportCommandResult 上报命令执行结果
	ReportCommandResult(result *models.NorthboundCommandResult) error
}

// NewAdapter 创建指定类型的适配器
func NewAdapter(northboundType, name string) NorthboundAdapter {
	switch nbtype.Normalize(northboundType) {
	case nbtype.TypeMQTT:
		return NewMQTTAdapter(name)
	case nbtype.TypePandaX:
		return NewPandaXAdapter(name)
	case nbtype.TypeIThings:
		return NewIThingsAdapter(name)
	case nbtype.TypeSagoo:
		return NewXunJiAdapter(name)
	default:
		return nil
	}
}

// SupportedTypes 返回支持的北向类型
func SupportedTypes() []string {
	return nbtype.SupportedTypes()
}
