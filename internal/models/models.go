package models

import "time"

// User 用户模型
type User struct {
	ID        int64     `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Resource 资源模型（串口/网口/DI/DO）
type Resource struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"` // serial, net, di, do
	Path      string    `json:"path" db:"path"` // 资源路径: /dev/ttyUSB0 或 eth0 等
	Enabled   int       `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// GatewayConfig 网关配置模型
type GatewayConfig struct {
	ID                int64  `json:"id" db:"id"`
	GatewayName       string `json:"gateway_name" db:"gateway_name"`
	DataRetentionDays int    `json:"data_retention_days" db:"data_retention_days"`
}

// Driver 驱动模型
type Driver struct {
	ID           int64     `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	FilePath     string    `json:"file_path" db:"file_path"`
	Description  string    `json:"description" db:"description"`
	Version      string    `json:"version" db:"version"`
	ConfigSchema string    `json:"config_schema" db:"config_schema"`
	Filename     string    `json:"filename,omitempty"` // 文件名（不存数据库）
	Size         int64     `json:"size,omitempty"`     // 文件大小（不存数据库）
	Loaded       bool      `json:"loaded,omitempty"`
	ResourceID   int64     `json:"resource_id,omitempty"`
	LastActive   time.Time `json:"last_active,omitempty"`
	Exports      []string  `json:"exports,omitempty"`
	Enabled      int       `json:"enabled" db:"enabled"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Device 设备模型
type Device struct {
	ID          int64  `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	ProductKey  string `json:"product_key" db:"product_key"`
	DeviceKey   string `json:"device_key" db:"device_key"`
	// 驱动类型: modbus_rtu, modbus_tcp
	DriverType string `json:"driver_type" db:"driver_type"`
	// 串口参数
	SerialPort string `json:"serial_port" db:"serial_port"`
	BaudRate   int    `json:"baud_rate" db:"baud_rate"`
	DataBits   int    `json:"data_bits" db:"data_bits"`
	StopBits   int    `json:"stop_bits" db:"stop_bits"`
	Parity     string `json:"parity" db:"parity"` // N, O, E
	// 网口参数
	IPAddress string `json:"ip_address" db:"ip_address"`
	PortNum   int    `json:"port_num" db:"port_num"`
	// 设备地址和周期
	DeviceAddress   string `json:"device_address" db:"device_address"`
	CollectInterval int    `json:"collect_interval" db:"collect_interval"` // 采集周期(ms)
	StorageInterval int    `json:"storage_interval" db:"storage_interval"` // 存储周期(s)
	Timeout         int    `json:"timeout" db:"timeout"`                   // 响应超时(ms)
	// 驱动（保留用于未来扩展）
	DriverID     *int64 `json:"driver_id" db:"driver_id"`
	DriverName   string `json:"driver_name,omitempty"`
	ResourceID   *int64 `json:"resource_id" db:"resource_id"`
	ResourceName string `json:"resource_name,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourcePath string `json:"resource_path,omitempty"`
	// 状态
	Enabled   int       `json:"enabled" db:"enabled"` // 1=采集, 0=停止
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DeviceDriverMapping 设备驱动映射
type DeviceDriverMapping struct {
	ID        int64     `json:"id" db:"id"`
	DeviceID  int64     `json:"device_id" db:"device_id"`
	DriverID  int64     `json:"driver_id" db:"driver_id"`
	Config    string    `json:"config" db:"config"`
	Priority  int       `json:"priority" db:"priority"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// NorthboundConfig 北向配置模型
type NorthboundConfig struct {
	ID      int64  `json:"id" db:"id"`
	Name    string `json:"name" db:"name"`
	Type    string `json:"type" db:"type"` // xunji, pandax, ithings, mqtt, http
	Enabled int    `json:"enabled" db:"enabled"`

	// 基础配置
	UploadInterval int `json:"upload_interval" db:"upload_interval"`

	// 连接配置（数据库字段）
	ServerURL string `json:"server_url" db:"server_url"` // 服务器地址
	Port      int    `json:"port" db:"port"`             // 端口
	Path      string `json:"path" db:"path"`             // 路径（HTTP）
	Username  string `json:"username" db:"username"`     // 认证用户名
	Password  string `json:"-" db:"password"`            // 认证密码（不返回给前端）
	ClientID  string `json:"client_id" db:"client_id"`   // MQTT ClientID

	// 主题配置
	Topic      string `json:"topic" db:"topic"`             // 数据主题
	AlarmTopic string `json:"alarm_topic" db:"alarm_topic"` // 报警主题

	// 协议配置
	QOS       int  `json:"qos" db:"qos"`               // MQTT QOS (0-2)
	Retain    bool `json:"retain" db:"retain"`         // MQTT Retain
	KeepAlive int  `json:"keep_alive" db:"keep_alive"` // 心跳周期(秒)
	Timeout   int  `json:"timeout" db:"timeout"`       // 连接超时(秒)

	// XunJi 特定配置
	ProductKey string `json:"product_key" db:"product_key"` // 产品密钥
	DeviceKey  string `json:"device_key" db:"device_key"`   // 设备密钥

	// 高级配置（JSON格式）
	ExtConfig string `json:"ext_config" db:"ext_config"` // 扩展配置（JSON）
	Config    string `json:"config" db:"config"`         // Schema 配置（JSON）

	// 状态字段
	Connected       bool       `json:"connected" db:"connected"` // 是否已连接
	LastConnectedAt *time.Time `json:"last_connected_at" db:"last_connected_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Threshold 阈值配置模型
type Threshold struct {
	ID        int64     `json:"id" db:"id"`
	DeviceID  int64     `json:"device_id" db:"device_id"`
	FieldName string    `json:"field_name" db:"field_name"`
	Operator  string    `json:"operator" db:"operator"`
	Value     float64   `json:"value" db:"value"`
	Severity  string    `json:"severity" db:"severity"`
	Enabled   int       `json:"enabled" db:"enabled"`
	Message   string    `json:"message" db:"message"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AlarmLog 报警日志模型
type AlarmLog struct {
	ID             int64      `json:"id" db:"id"`
	DeviceID       int64      `json:"device_id" db:"device_id"`
	ThresholdID    *int64     `json:"threshold_id" db:"threshold_id"`
	FieldName      string     `json:"field_name" db:"field_name"`
	ActualValue    float64    `json:"actual_value" db:"actual_value"`
	ThresholdValue float64    `json:"threshold_value" db:"threshold_value"`
	Operator       string     `json:"operator" db:"operator"`
	Severity       string     `json:"severity" db:"severity"`
	Message        string     `json:"message" db:"message"`
	TriggeredAt    time.Time  `json:"triggered_at" db:"triggered_at"`
	Acknowledged   int        `json:"acknowledged" db:"acknowledged"`
	AcknowledgedBy string     `json:"acknowledged_by" db:"acknowledged_by"`
	AcknowledgedAt *time.Time `json:"acknowledged_at" db:"acknowledged_at"`
}

// DataCache 采集数据缓存
type DataCache struct {
	ID          int64     `json:"id" db:"id"`
	DeviceID    int64     `json:"device_id" db:"device_id"`
	FieldName   string    `json:"field_name" db:"field_name"`
	Value       string    `json:"value" db:"value"`
	ValueType   string    `json:"value_type" db:"value_type"`
	CollectedAt time.Time `json:"collected_at" db:"collected_at"`
}

// CollectData 采集数据结构
type CollectData struct {
	DeviceID   int64             `json:"device_id"`
	DeviceName string            `json:"device_name"`
	ProductKey string            `json:"product_key"`
	DeviceKey  string            `json:"device_key"`
	Timestamp  time.Time         `json:"timestamp"`
	Fields     map[string]string `json:"fields"`
}

// AlarmPayload 报警载荷
type AlarmPayload struct {
	DeviceID    int64   `json:"device_id"`
	DeviceName  string  `json:"device_name"`
	ProductKey  string  `json:"product_key"`
	DeviceKey   string  `json:"device_key"`
	FieldName   string  `json:"field_name"`
	ActualValue float64 `json:"actual_value"`
	Threshold   float64 `json:"threshold"`
	Operator    string  `json:"operator"`
	Severity    string  `json:"severity"`
	Message     string  `json:"message"`
}

// XunJiConfig 循迹北向配置
type XunJiConfig struct {
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

// NorthboundPayload 北向数据载荷
type NorthboundPayload struct {
	DeviceID   int64             `json:"device_id"`
	DeviceName string            `json:"device_name"`
	Properties map[string]string `json:"properties"`
	Events     map[string]Event  `json:"events"`
}

// NorthboundCommand 北向下发命令（用于写入子设备）
type NorthboundCommand struct {
	RequestID  string `json:"request_id"`
	ProductKey string `json:"product_key"`
	DeviceKey  string `json:"device_key"`
	FieldName  string `json:"field_name"`
	Value      string `json:"value"`
	Source     string `json:"source"`
}

// NorthboundCommandResult 北向下发命令执行结果
type NorthboundCommandResult struct {
	RequestID  string `json:"request_id"`
	ProductKey string `json:"product_key"`
	DeviceKey  string `json:"device_key"`
	FieldName  string `json:"field_name"`
	Value      string `json:"value"`
	Source     string `json:"source"`
	Success    bool   `json:"success"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

// Event 事件数据
type Event struct {
	Value map[string]interface{} `json:"value"`
	Time  int64                  `json:"time"`
}

// SystemStats FSU 本身系统属性（CPU、内存、硬盘等）
type SystemStats struct {
	// CPU
	CpuUsage float64 `json:"cpu_usage"` // CPU 使用率 (%)
	// 内存
	MemTotal     float64 `json:"mem_total"`     // 内存总量 (MB)
	MemUsed      float64 `json:"mem_used"`      // 已使用内存 (MB)
	MemUsage     float64 `json:"mem_usage"`     // 内存使用率 (%)
	MemAvailable float64 `json:"mem_available"` // 可用内存 (MB)
	// 硬盘
	DiskTotal float64 `json:"disk_total"` // 硬盘总量 (GB)
	DiskUsed  float64 `json:"disk_used"`  // 已使用硬盘 (GB)
	DiskUsage float64 `json:"disk_usage"` // 硬盘使用率 (%)
	DiskFree  float64 `json:"disk_free"`  // 可用硬盘 (GB)
	// 其他
	Uptime    int64   `json:"uptime"`    // 运行时间 (秒)
	Load1     float64 `json:"load_1"`    // 1分钟负载
	Load5     float64 `json:"load_5"`    // 5分钟负载
	Load15    float64 `json:"load_15"`   // 15分钟负载
	Timestamp int64   `json:"timestamp"` // 采集时间戳
}

// SystemStatsDeviceID FSU 系统属性的固定设备 ID
const SystemStatsDeviceID int64 = -1

// SystemStatsDeviceName FSU 系统属性的设备名称
const SystemStatsDeviceName string = "__system__"
