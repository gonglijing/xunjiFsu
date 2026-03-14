package schema

// XunjiConfigSchema is the schema source for XunJi northbound config.
var XunjiConfigSchema = []Field{
	{Key: "serverUrl", Label: "MQTT 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://127.0.0.1:1883"},
	{Key: "topic", Label: "上报 Topic", Type: FieldTypeString, Optional: true, Default: "v1/gateway/{gatewayname}", Description: "支持占位符 {gatewayname}"},
	{Key: "gatewayName", Label: "网关名称", Type: FieldTypeString, Optional: true, Default: "", Description: "为空时自动取系统网关名/北向名称"},
	{Key: "username", Label: "用户名", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 用户名（可选）"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "MQTT 密码（可选）"},
	{Key: "qos", Label: "QOS", Type: FieldTypeInt, Optional: true, Default: 0, Description: "范围 0~2"},
	{Key: "subDeviceTokenMode", Label: "子设备标识模式", Type: FieldTypeString, Optional: true, Default: "", Description: "可选 device_key / product_devicekey / product_devicename"},
}
