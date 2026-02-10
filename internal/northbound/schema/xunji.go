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

const NorthboundTypeSagoo = "sagoo"
const NorthboundTypeXunJi = "xunji"

var SupportedNorthboundSchemaTypes = []string{NorthboundTypeSagoo}

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
// 只保留关键连接参数
var XunJiConfigSchema = []Field{
	{Key: "productKey", Label: "ProductKey", Type: FieldTypeString, Required: true, Default: "", Description: "网关 ProductKey（必填）"},
	{Key: "deviceKey", Label: "DeviceKey", Type: FieldTypeString, Required: true, Default: "", Description: "网关 DeviceKey（必填）"},
	{Key: "serverUrl", Label: "MQTT 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://192.168.1.100:1883"},
	{Key: "username", Label: "用户名", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 用户名（可选）"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 密码（可选）"},
}

func FieldsByType(nbType string) ([]Field, bool) {
	switch strings.ToLower(strings.TrimSpace(nbType)) {
	case "", NorthboundTypeSagoo, NorthboundTypeXunJi:
		return cloneFields(XunJiConfigSchema), true
	case "mqtt":
		return cloneFields(MQTTConfigSchema), true
	case "pandax":
		return cloneFields(PandaXConfigSchema), true
	case "ithings":
		return cloneFields(IThingsConfigSchema), true
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
