package schema

import "strings"

// FieldType defines the supported config value types in northbound schema.
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeInt    FieldType = "int"
	FieldTypeBool   FieldType = "bool"
)

const XunJiSchemaVersion = "1.0.0"

var SupportedNorthboundSchemaTypes = []string{"xunji"}

// Field describes one config field in Terraform SDK Schema-like style.
type Field struct {
	Key         string      `json:"key"`
	Label       string      `json:"label"`
	Type        FieldType   `json:"type"`
	Required    bool        `json:"required"`
	Optional    bool        `json:"optional"`
	Default     interface{} `json:"default"`
	Description string      `json:"description"`
}

// XunJiConfigSchema is the single schema source for XUNJI northbound config.
var XunJiConfigSchema = []Field{
	{Key: "productKey", Label: "ProductKey", Type: FieldTypeString, Required: true, Default: "", Description: "网关 ProductKey（必填）"},
	{Key: "deviceKey", Label: "DeviceKey", Type: FieldTypeString, Required: true, Default: "", Description: "网关 DeviceKey（必填）"},
	{Key: "serverUrl", Label: "MQTT 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://127.0.0.1:1883"},
	{Key: "username", Label: "用户名", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 用户名"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 密码"},
	{Key: "topic", Label: "实时 Topic", Type: FieldTypeString, Optional: true, Default: "", Description: "可为空，插件自动按 PK/DK 生成"},
	{Key: "alarmTopic", Label: "报警 Topic", Type: FieldTypeString, Optional: true, Default: "", Description: "可为空，默认与实时 Topic 一致"},
	{Key: "clientId", Label: "Client ID", Type: FieldTypeString, Optional: true, Default: "", Description: "可为空，插件自动生成"},
	{Key: "qos", Label: "QOS", Type: FieldTypeInt, Optional: true, Default: 0, Description: "范围 0~2"},
	{Key: "retain", Label: "Retain", Type: FieldTypeBool, Optional: true, Default: false, Description: "MQTT retain 标记"},
	{Key: "keepAlive", Label: "KeepAlive(秒)", Type: FieldTypeInt, Optional: true, Default: 60, Description: "MQTT keep alive"},
	{Key: "connectTimeout", Label: "连接超时(秒)", Type: FieldTypeInt, Optional: true, Default: 10, Description: "MQTT 连接超时"},
	{Key: "uploadIntervalMs", Label: "上传周期(ms)", Type: FieldTypeInt, Optional: true, Default: 5000, Description: "插件内部上传周期"},
	{Key: "grpcAddress", Label: "gRPC 地址", Type: FieldTypeString, Optional: true, Default: "", Description: "主程序 -> 插件地址；为空自动按 PK+DK 生成"},
	{Key: "alarmFlushIntervalMs", Label: "报警刷新(ms)", Type: FieldTypeInt, Optional: true, Default: 2000, Description: "报警批量发送周期"},
	{Key: "alarmBatchSize", Label: "报警批量条数", Type: FieldTypeInt, Optional: true, Default: 20, Description: "每次发送报警条数"},
	{Key: "alarmQueueSize", Label: "报警队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
	{Key: "realtimeQueueSize", Label: "实时队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
}

func FieldsByType(nbType string) ([]Field, bool) {
	switch strings.ToLower(strings.TrimSpace(nbType)) {
	case "", "xunji":
		return cloneFields(XunJiConfigSchema), true
	default:
		return nil, false
	}
}

func cloneFields(fields []Field) []Field {
	if len(fields) == 0 {
		return []Field{}
	}
	out := make([]Field, len(fields))
	copy(out, fields)
	return out
}
