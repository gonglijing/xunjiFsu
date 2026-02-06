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
	ID          int64  `json:"id" db:"id"`
	ProductKey  string `json:"product_key" db:"product_key"`
	DeviceKey   string `json:"device_key" db:"device_key"`
	GatewayName string `json:"gateway_name" db:"gateway_name"`
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
	ID             int64     `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Type           string    `json:"type" db:"type"` // xunji, mqtt, http, etc.
	Enabled        int       `json:"enabled" db:"enabled"`
	Config         string    `json:"config" db:"config"`
	UploadInterval int       `json:"upload_interval" db:"upload_interval"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
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
}

// NorthboundPayload 北向数据载荷
type NorthboundPayload struct {
	DeviceID   int64             `json:"device_id"`
	DeviceName string            `json:"device_name"`
	Properties map[string]string `json:"properties"`
	Events     map[string]Event  `json:"events"`
}

// Event 事件数据
type Event struct {
	Value map[string]interface{} `json:"value"`
	Time  int64                  `json:"time"`
}
