package schema

// PandaXConfigSchema is schema source for PandaX northbound config.
var PandaXConfigSchema = []Field{
	{Key: "serverUrl", Label: "MQTT 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://127.0.0.1:1883"},
	{Key: "username", Label: "设备 Token", Type: FieldTypeString, Required: true, Default: "", Description: "PandaX 使用 MQTT Username 认证"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "默认可留空"},
	{Key: "clientId", Label: "Client ID", Type: FieldTypeString, Optional: true, Default: "", Description: "可为空，自动生成"},
	{Key: "qos", Label: "QOS", Type: FieldTypeInt, Optional: true, Default: 0, Description: "范围 0~2"},
	{Key: "retain", Label: "Retain", Type: FieldTypeBool, Optional: true, Default: false, Description: "MQTT retain 标记"},
	{Key: "keepAlive", Label: "KeepAlive(秒)", Type: FieldTypeInt, Optional: true, Default: 60, Description: "MQTT keep alive"},
	{Key: "connectTimeout", Label: "连接超时(秒)", Type: FieldTypeInt, Optional: true, Default: 10, Description: "MQTT 连接超时"},
	{Key: "gatewayMode", Label: "网关模式", Type: FieldTypeBool, Optional: true, Default: true, Description: "开启后按 v1/gateway/* 上报"},
	{Key: "subDeviceTokenMode", Label: "子设备 Token 规则", Type: FieldTypeString, Optional: true, Default: "deviceName", Description: "deviceName/deviceKey/product_deviceKey"},
	{Key: "gatewayTelemetryTopic", Label: "网关遥测 Topic", Type: FieldTypeString, Optional: true, Default: "v1/gateway/telemetry", Description: "默认 PandaX 网关遥测主题"},
	{Key: "gatewayAttributesTopic", Label: "网关属性 Topic", Type: FieldTypeString, Optional: true, Default: "v1/gateway/attributes", Description: "默认 PandaX 网关属性主题"},
	{Key: "telemetryTopic", Label: "直连遥测 Topic", Type: FieldTypeString, Optional: true, Default: "v1/devices/me/telemetry", Description: "gatewayMode=false 时使用"},
	{Key: "attributesTopic", Label: "直连属性 Topic", Type: FieldTypeString, Optional: true, Default: "v1/devices/me/attributes", Description: "gatewayMode=false 时使用"},
	{Key: "eventTopicPrefix", Label: "事件 Topic 前缀", Type: FieldTypeString, Optional: true, Default: "v1/devices/event", Description: "事件上报前缀"},
	{Key: "alarmIdentifier", Label: "报警标识", Type: FieldTypeString, Optional: true, Default: "alarm", Description: "事件标识，例如 alarm"},
	{Key: "rpcRequestTopic", Label: "RPC 下行 Topic", Type: FieldTypeString, Optional: true, Default: "v1/devices/me/rpc/request", Description: "默认订阅 request/+"},
	{Key: "rpcResponseTopic", Label: "RPC 回执 Topic", Type: FieldTypeString, Optional: true, Default: "v1/devices/me/rpc/response", Description: "命令执行结果回传"},
	{Key: "uploadIntervalMs", Label: "上传周期(ms)", Type: FieldTypeInt, Optional: true, Default: 5000, Description: "插件内部上传周期"},
	{Key: "alarmFlushIntervalMs", Label: "报警刷新(ms)", Type: FieldTypeInt, Optional: true, Default: 2000, Description: "报警批量发送周期"},
	{Key: "alarmBatchSize", Label: "报警批量条数", Type: FieldTypeInt, Optional: true, Default: 20, Description: "每次发送报警条数"},
	{Key: "alarmQueueSize", Label: "报警队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
	{Key: "realtimeQueueSize", Label: "实时队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
	{Key: "commandQueueSize", Label: "命令队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
	{Key: "productKey", Label: "ProductKey", Type: FieldTypeString, Optional: true, Default: "", Description: "用于命令路由（可选）"},
	{Key: "deviceKey", Label: "DeviceKey", Type: FieldTypeString, Optional: true, Default: "", Description: "用于命令路由（可选）"},
}

func init() {
	SupportedNorthboundSchemaTypes = append(SupportedNorthboundSchemaTypes, "pandax")
}
