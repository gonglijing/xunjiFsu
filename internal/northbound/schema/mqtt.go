package schema

// MQTTConfigSchema is the schema source for MQTT northbound config.
var MQTTConfigSchema = []Field{
	{Key: "broker", Label: "Broker 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://127.0.0.1:1883"},
	{Key: "topic", Label: "数据 Topic", Type: FieldTypeString, Required: true, Default: "", Description: "实时数据上报主题"},
	{Key: "alarmTopic", Label: "报警 Topic", Type: FieldTypeString, Optional: true, Default: "", Description: "报警数据上报主题，为空时默认在数据 Topic 后加 /alarm"},
	{Key: "clientId", Label: "Client ID", Type: FieldTypeString, Optional: true, Default: "", Description: "为空时自动生成"},
	{Key: "username", Label: "用户名", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 用户名（可选）"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 密码（可选）"},
	{Key: "qos", Label: "QOS", Type: FieldTypeInt, Optional: true, Default: 0, Description: "范围 0~2"},
	{Key: "retain", Label: "Retain", Type: FieldTypeBool, Optional: true, Default: false, Description: "MQTT retain 标记"},
	{Key: "cleanSession", Label: "Clean Session", Type: FieldTypeBool, Optional: true, Default: true, Description: "断开后是否清除会话"},
	{Key: "keepAlive", Label: "KeepAlive(秒)", Type: FieldTypeInt, Optional: true, Default: 60, Description: "MQTT 心跳间隔"},
	{Key: "connectTimeout", Label: "连接超时(秒)", Type: FieldTypeInt, Optional: true, Default: 10, Description: "MQTT 连接超时"},
	{Key: "uploadIntervalMs", Label: "上报周期(ms)", Type: FieldTypeInt, Optional: true, Default: 5000, Description: "数据上报周期"},
}

func init() {
	// Register MQTT schema
	SupportedNorthboundSchemaTypes = append(SupportedNorthboundSchemaTypes, "mqtt")
}
