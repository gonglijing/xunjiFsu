package schema

// PandaXConfigSchema is schema source for PandaX northbound config.
var PandaXConfigSchema = []Field{
	{Key: "serverUrl", Label: "MQTT 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://127.0.0.1:1883"},
	{Key: "username", Label: "用户名", Type: FieldTypeString, Required: true, Default: "", Description: "PandaX 使用 MQTT Username 认证"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "PandaX 使用 MQTT Password 认证（可选）"},
	{Key: "qos", Label: "QOS", Type: FieldTypeInt, Optional: true, Default: 0, Description: "范围 0~2"},
}

func init() {
	SupportedNorthboundSchemaTypes = append(SupportedNorthboundSchemaTypes, "pandax")
}
