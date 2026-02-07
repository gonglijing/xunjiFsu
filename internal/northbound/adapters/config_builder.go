package adapters

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// NorthboundConfigBuilder 用于从数据库字段生成适配器配置JSON
type NorthboundConfigBuilder struct {
	northboundType string
	config        map[string]interface{}
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder(northboundType string) *NorthboundConfigBuilder {
	return &NorthboundConfigBuilder{
		northboundType: northboundType,
		config:        make(map[string]interface{}),
	}
}

// SetServerURL 设置服务器地址
func (b *NorthboundConfigBuilder) SetServerURL(url string) *NorthboundConfigBuilder {
	if url != "" {
		b.config["url"] = url
	}
	return b
}

// SetBrokerURL 设置MQTT Broker地址
func (b *NorthboundConfigBuilder) SetBrokerURL(url string) *NorthboundConfigBuilder {
	if url != "" {
		b.config["broker"] = url
	}
	return b
}

// SetPort 设置端口
func (b *NorthboundConfigBuilder) SetPort(port int) *NorthboundConfigBuilder {
	if port > 0 {
		b.config["port"] = port
	}
	return b
}

// SetPath 设置路径
func (b *NorthboundConfigBuilder) SetPath(path string) *NorthboundConfigBuilder {
	if path != "" {
		b.config["path"] = path
	}
	return b
}

// SetUsername 设置用户名
func (b *NorthboundConfigBuilder) SetUsername(username string) *NorthboundConfigBuilder {
	if username != "" {
		b.config["username"] = username
	}
	return b
}

// SetPassword 设置密码
func (b *NorthboundConfigBuilder) SetPassword(password string) *NorthboundConfigBuilder {
	if password != "" {
		b.config["password"] = password
	}
	return b
}

// SetClientID 设置客户端ID
func (b *NorthboundConfigBuilder) SetClientID(clientID string) *NorthboundConfigBuilder {
	if clientID != "" {
		b.config["client_id"] = clientID
	}
	return b
}

// SetTopic 设置主题
func (b *NorthboundConfigBuilder) SetTopic(topic string) *NorthboundConfigBuilder {
	if topic != "" {
		b.config["topic"] = topic
	}
	return b
}

// SetAlarmTopic 设置报警主题
func (b *NorthboundConfigBuilder) SetAlarmTopic(alarmTopic string) *NorthboundConfigBuilder {
	if alarmTopic != "" {
		b.config["alarm_topic"] = alarmTopic
	}
	return b
}

// SetQOS 设置QoS
func (b *NorthboundConfigBuilder) SetQOS(qos int) *NorthboundConfigBuilder {
	if qos >= 0 && qos <= 2 {
		b.config["qos"] = qos
	}
	return b
}

// SetRetain 设置Retain
func (b *NorthboundConfigBuilder) SetRetain(retain bool) *NorthboundConfigBuilder {
	b.config["retain"] = retain
	return b
}

// SetKeepAlive 设置心跳周期
func (b *NorthboundConfigBuilder) SetKeepAlive(keepAlive int) *NorthboundConfigBuilder {
	if keepAlive > 0 {
		b.config["keepAlive"] = keepAlive
	}
	return b
}

// SetTimeout 设置超时
func (b *NorthboundConfigBuilder) SetTimeout(timeout int) *NorthboundConfigBuilder {
	if timeout > 0 {
		b.config["connectTimeout"] = timeout
	}
	return b
}

// SetProductKey 设置产品密钥
func (b *NorthboundConfigBuilder) SetProductKey(productKey string) *NorthboundConfigBuilder {
	if productKey != "" {
		b.config["productKey"] = productKey
	}
	return b
}

// SetDeviceKey 设置设备密钥
func (b *NorthboundConfigBuilder) SetDeviceKey(deviceKey string) *NorthboundConfigBuilder {
	if deviceKey != "" {
		b.config["deviceKey"] = deviceKey
	}
	return b
}

// SetUploadIntervalMs 设置上传周期
func (b *NorthboundConfigBuilder) SetUploadIntervalMs(interval int) *NorthboundConfigBuilder {
	if interval > 0 {
		b.config["uploadIntervalMs"] = interval
	}
	return b
}

// SetExtConfig 设置扩展配置
func (b *NorthboundConfigBuilder) SetExtConfig(extConfig string) *NorthboundConfigBuilder {
	if extConfig != "" {
		var ext map[string]interface{}
		if err := json.Unmarshal([]byte(extConfig), &ext); err == nil {
			for k, v := range ext {
				b.config[k] = v
			}
		}
	}
	return b
}

// Build 生成配置JSON字符串
func (b *NorthboundConfigBuilder) Build() string {
	// 根据类型设置默认值
	switch b.northboundType {
	case "http":
		if _, ok := b.config["url"]; !ok {
			b.config["url"] = ""
		}
		if _, ok := b.config["timeout"]; !ok {
			b.config["timeout"] = 30
		}
	case "mqtt":
		if _, ok := b.config["broker"]; !ok {
			b.config["broker"] = ""
		}
		if _, ok := b.config["topic"]; !ok {
			b.config["topic"] = ""
		}
		if _, ok := b.config["qos"]; !ok {
			b.config["qos"] = 0
		}
		if _, ok := b.config["keepAlive"]; !ok {
			b.config["keepAlive"] = 60
		}
		if _, ok := b.config["connectTimeout"]; !ok {
			b.config["connectTimeout"] = 30
		}
	case "xunji":
		if _, ok := b.config["serverUrl"]; !ok {
			b.config["serverUrl"] = ""
		}
		if _, ok := b.config["productKey"]; !ok {
			b.config["productKey"] = ""
		}
		if _, ok := b.config["deviceKey"]; !ok {
			b.config["deviceKey"] = ""
		}
		if _, ok := b.config["qos"]; !ok {
			b.config["qos"] = 0
		}
		if _, ok := b.config["keepAlive"]; !ok {
			b.config["keepAlive"] = 60
		}
		if _, ok := b.config["connectTimeout"]; !ok {
			b.config["connectTimeout"] = 30
		}
		if _, ok := b.config["uploadIntervalMs"]; !ok {
			b.config["uploadIntervalMs"] = 5000
		}
	}

	data, _ := json.Marshal(b.config)
	return string(data)
}

// BuildFromModel 从 NorthboundConfig 模型生成配置JSON
func BuildConfigFromModel(cfg *models.NorthboundConfig) string {
	builder := NewConfigBuilder(cfg.Type)

	switch cfg.Type {
	case "http":
		// 构建HTTP配置
		url := buildHTTPURL(cfg.ServerURL, cfg.Port, cfg.Path)
		builder.SetServerURL(url)
		if cfg.Username != "" {
			builder.SetUsername(cfg.Username)
		}
		if cfg.Password != "" {
			builder.SetPassword(cfg.Password)
		}
		builder.SetTimeout(cfg.Timeout)
		builder.SetExtConfig(cfg.ExtConfig)

	case "mqtt":
		// 构建MQTT配置
		broker := buildBrokerURL(cfg.ServerURL, cfg.Port)
		builder.SetBrokerURL(broker)
		builder.SetClientID(cfg.ClientID)
		if cfg.Username != "" {
			builder.SetUsername(cfg.Username)
		}
		if cfg.Password != "" {
			builder.SetPassword(cfg.Password)
		}
		builder.SetTopic(cfg.Topic)
		builder.SetAlarmTopic(cfg.AlarmTopic)
		builder.SetQOS(cfg.QOS)
		builder.SetRetain(cfg.Retain)
		builder.SetKeepAlive(cfg.KeepAlive)
		builder.SetTimeout(cfg.Timeout)
		builder.SetExtConfig(cfg.ExtConfig)

	case "xunji":
		// 构建XunJi配置
		serverURL := buildBrokerURL(cfg.ServerURL, cfg.Port)
		builder.SetServerURL(serverURL)
		builder.SetProductKey(cfg.ProductKey)
		builder.SetDeviceKey(cfg.DeviceKey)
		if cfg.Username != "" {
			builder.SetUsername(cfg.Username)
		}
		if cfg.Password != "" {
			builder.SetPassword(cfg.Password)
		}
		builder.SetTopic(cfg.Topic)
		builder.SetAlarmTopic(cfg.AlarmTopic)
		builder.SetClientID(cfg.ClientID)
		builder.SetQOS(cfg.QOS)
		builder.SetRetain(cfg.Retain)
		builder.SetKeepAlive(cfg.KeepAlive)
		builder.SetTimeout(cfg.Timeout)
		builder.SetUploadIntervalMs(cfg.UploadInterval)
		builder.SetExtConfig(cfg.ExtConfig)
	}

	return builder.Build()
}

// buildHTTPURL 构建完整的HTTP URL
func buildHTTPURL(serverURL string, port int, path string) string {
	if serverURL == "" {
		return ""
	}

	// 如果已经是完整URL，直接返回
	if len(serverURL) >= 7 && (serverURL[:7] == "http://" || serverURL[:8] == "https://") {
		return serverURL
	}

	// 构建URL
	scheme := "http://"
	if port == 443 {
		scheme = "https://"
	}

	result := scheme + serverURL

	// 添加端口
	if port > 0 {
		if (port == 80 && scheme == "http://") || (port == 443 && scheme == "https://") {
			// 默认端口不需要显示
		} else {
			result += ":" + strconv.Itoa(port)
		}
	}

	// 添加路径
	if path != "" && path[0] != '/' {
		path = "/" + path
	}
	result += path

	return result
}

// buildBrokerURL 构建完整的Broker URL
func buildBrokerURL(serverURL string, port int) string {
	if serverURL == "" {
		return ""
	}

	// 如果已经是完整URL，直接返回
	if len(serverURL) >= 6 && serverURL[:6] == "tcp://" {
		return serverURL
	}
	if len(serverURL) >= 5 && serverURL[:5] == "mqtt://" {
		return serverURL
	}
	if len(serverURL) >= 8 && serverURL[:8] == "ssl://" {
		return serverURL
	}

	// 添加协议前缀
	result := "tcp://" + serverURL

	// 添加端口
	if port > 0 {
		result += ":" + strconv.Itoa(port)
	}

	return result
}

// ParseConnectionInfo 解析连接信息（用于显示）
type ConnectionInfo struct {
	Type       string `json:"type"`
	Server     string `json:"server"`
	Port       int    `json:"port"`
	Path       string `json:"path,omitempty"`
	Username   string `json:"username,omitempty"`
	ClientID   string `json:"client_id,omitempty"`
	Topic      string `json:"topic,omitempty"`
	AlarmTopic string `json:"alarm_topic,omitempty"`
	Connected  bool   `json:"connected"`
}

// ParseConnectionInfoFromModel 从模型解析连接信息
func ParseConnectionInfoFromModel(cfg *models.NorthboundConfig) *ConnectionInfo {
	info := &ConnectionInfo{
		Type:      cfg.Type,
		Server:    cfg.ServerURL,
		Port:      cfg.Port,
		Path:      cfg.Path,
		Username:  cfg.Username,
		ClientID:  cfg.ClientID,
		Topic:     cfg.Topic,
		AlarmTopic: cfg.AlarmTopic,
		Connected: cfg.Connected,
	}
	return info
}

// GetSupportedTypes 返回支持的北向类型及其字段描述
func GetSupportedTypes() map[string][]string {
	return map[string][]string{
		"http": {
			"server_url: 服务器地址",
			"port: 端口 (默认80)",
			"path: 路径 (可选)",
			"username: 用户名 (可选)",
			"password: 密码 (可选)",
			"timeout: 超时秒数 (默认30)",
		},
		"mqtt": {
			"server_url: Broker地址",
			"port: 端口 (默认1883)",
			"client_id: 客户端ID (可选)",
			"username: 用户名 (可选)",
			"password: 密码 (可选)",
			"topic: 数据主题",
			"alarm_topic: 报警主题 (可选)",
			"qos: QoS等级 (0-2)",
			"retain: 是否保留消息",
			"keep_alive: 心跳周期秒数",
			"timeout: 连接超时秒数",
		},
		"xunji": {
			"server_url: 服务器地址",
			"port: 端口 (默认1883)",
			"product_key: 产品密钥",
			"device_key: 设备密钥",
			"username: 用户名 (可选)",
			"password: 密码 (可选)",
			"topic: 数据主题 (可选)",
			"alarm_topic: 报警主题 (可选)",
			"client_id: 客户端ID (可选)",
			"qos: QoS等级 (0-2)",
			"retain: 是否保留消息",
			"keep_alive: 心跳周期秒数",
			"timeout: 连接超时秒数",
			"upload_interval: 上传周期毫秒数",
		},
	}
}

// ValidateConfig 验证配置是否有效
func ValidateConfig(northboundType string, config map[string]interface{}) error {
	switch northboundType {
	case "http":
		if url, ok := config["url"].(string); !ok || url == "" {
			return fmt.Errorf("url is required for HTTP adapter")
		}
	case "mqtt":
		if broker, ok := config["broker"].(string); !ok || broker == "" {
			return fmt.Errorf("broker is required for MQTT adapter")
		}
		if topic, ok := config["topic"].(string); !ok || topic == "" {
			return fmt.Errorf("topic is required for MQTT adapter")
		}
	case "xunji":
		if serverUrl, ok := config["serverUrl"].(string); !ok || serverUrl == "" {
			return fmt.Errorf("serverUrl is required for XunJi adapter")
		}
		if productKey, ok := config["productKey"].(string); !ok || productKey == "" {
			return fmt.Errorf("productKey is required for XunJi adapter")
		}
		if deviceKey, ok := config["deviceKey"].(string); !ok || deviceKey == "" {
			return fmt.Errorf("deviceKey is required for XunJi adapter")
		}
	default:
		return fmt.Errorf("unknown northbound type: %s", northboundType)
	}
	return nil
}
