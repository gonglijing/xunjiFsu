package schema

// IThingsConfigSchema is schema source for iThings northbound config.
var IThingsConfigSchema = []Field{
	{Key: "serverUrl", Label: "MQTT 地址", Type: FieldTypeString, Required: true, Default: "", Description: "例如 tcp://127.0.0.1:1883"},
	{Key: "username", Label: "用户名", Type: FieldTypeString, Required: true, Default: "", Description: "iThings MQTT 用户名"},
	{Key: "password", Label: "密码", Type: FieldTypeString, Optional: true, Default: "", Description: "iThings MQTT 密码"},
	{Key: "productKey", Label: "网关 ProductID", Type: FieldTypeString, Required: true, Default: "", Description: "网关产品ID（上行/下行路由）"},
	{Key: "deviceKey", Label: "网关 DeviceName", Type: FieldTypeString, Required: true, Default: "", Description: "网关设备名（上行/下行路由）"},
	{Key: "clientId", Label: "Client ID", Type: FieldTypeString, Optional: true, Default: "", Description: "可为空，自动生成"},
	{Key: "qos", Label: "QOS", Type: FieldTypeInt, Optional: true, Default: 0, Description: "范围 0~2"},
	{Key: "retain", Label: "Retain", Type: FieldTypeBool, Optional: true, Default: false, Description: "MQTT retain 标记"},
	{Key: "keepAlive", Label: "KeepAlive(秒)", Type: FieldTypeInt, Optional: true, Default: 60, Description: "MQTT keep alive"},
	{Key: "connectTimeout", Label: "连接超时(秒)", Type: FieldTypeInt, Optional: true, Default: 10, Description: "MQTT 连接超时"},
	{Key: "gatewayMode", Label: "网关模式", Type: FieldTypeBool, Optional: true, Default: true, Description: "仅支持 true（网关+子设备）"},
	{Key: "deviceNameMode", Label: "设备名映射", Type: FieldTypeString, Optional: true, Default: "deviceKey", Description: "deviceKey 或 deviceName"},
	{Key: "subDeviceNameMode", Label: "子设备名映射", Type: FieldTypeString, Optional: true, Default: "deviceKey", Description: "deviceKey 或 deviceName"},
	{Key: "upPropertyTopicTemplate", Label: "属性上行 Topic", Type: FieldTypeString, Optional: true, Default: "$thing/up/property/{productID}/{deviceName}", Description: "属性/packReport 上报 Topic 模板"},
	{Key: "upEventTopicTemplate", Label: "事件上行 Topic", Type: FieldTypeString, Optional: true, Default: "$thing/up/event/{productID}/{deviceName}", Description: "事件/eventPost 上报 Topic 模板"},
	{Key: "upActionTopicTemplate", Label: "行为上行 Topic", Type: FieldTypeString, Optional: true, Default: "$thing/up/action/{productID}/{deviceName}", Description: "行为/actionReply 回执 Topic 模板"},
	{Key: "downPropertyTopic", Label: "属性下行订阅", Type: FieldTypeString, Optional: true, Default: "$thing/down/property/+/+", Description: "属性控制下发订阅 Topic"},
	{Key: "downActionTopic", Label: "行为下行订阅", Type: FieldTypeString, Optional: true, Default: "$thing/down/action/+/+", Description: "行为调用下发订阅 Topic"},
	{Key: "alarmEventID", Label: "报警事件ID", Type: FieldTypeString, Optional: true, Default: "alarm", Description: "报警上报 eventID"},
	{Key: "alarmEventType", Label: "报警事件类型", Type: FieldTypeString, Optional: true, Default: "alert", Description: "报警上报 type"},
	{Key: "uploadIntervalMs", Label: "上传周期(ms)", Type: FieldTypeInt, Optional: true, Default: 5000, Description: "插件内部上传周期"},
	{Key: "alarmFlushIntervalMs", Label: "报警刷新(ms)", Type: FieldTypeInt, Optional: true, Default: 2000, Description: "报警批量发送周期"},
	{Key: "alarmBatchSize", Label: "报警批量条数", Type: FieldTypeInt, Optional: true, Default: 20, Description: "每次发送报警条数"},
	{Key: "alarmQueueSize", Label: "报警队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
	{Key: "realtimeQueueSize", Label: "实时队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
	{Key: "commandQueueSize", Label: "命令队列长度", Type: FieldTypeInt, Optional: true, Default: 1000, Description: "超过后丢弃最旧"},
}

func init() {
	SupportedNorthboundSchemaTypes = append(SupportedNorthboundSchemaTypes, "ithings")
}
