package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
)

// NorthboundConfigBuilder 用于从数据库字段生成适配器配置JSON
type NorthboundConfigBuilder struct {
	northboundType string
	config         map[string]interface{}
}

// NewConfigBuilder 创建配置构建器
func NewConfigBuilder(northboundType string) *NorthboundConfigBuilder {
	return &NorthboundConfigBuilder{
		northboundType: nbtype.Normalize(northboundType),
		config:         make(map[string]interface{}),
	}
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
	if defaults, ok := northboundConfigDefaults[nbtype.Normalize(b.northboundType)]; ok {
		b.applyDefaults(defaults)
	}

	data, _ := json.Marshal(b.config)
	return string(data)
}

// BuildFromModel 从 NorthboundConfig 模型生成配置JSON
func BuildConfigFromModel(cfg *models.NorthboundConfig) string {
	builder := NewConfigBuilder(cfg.Type)

	switch nbtype.Normalize(cfg.Type) {
	case nbtype.TypeMQTT:
		applySharedModelFields(builder, cfg, modelBuildOptions{
			includeTopic:      true,
			includeAlarmTopic: true,
		})

	case nbtype.TypeXunji:
		applySharedModelFields(builder, cfg, modelBuildOptions{
			includeTopic:          true,
			includeAlarmTopic:     true,
			includeUploadInterval: true,
		})

	case nbtype.TypeSagoo:
		applySharedModelFields(builder, cfg, modelBuildOptions{
			includeTopic:           true,
			includeAlarmTopic:      true,
			includeProductIdentity: true,
			includeUploadInterval:  true,
		})

	case nbtype.TypePandaX:
		applySharedModelFields(builder, cfg, modelBuildOptions{
			includeUploadInterval: true,
		})

	case nbtype.TypeIThings:
		applySharedModelFields(builder, cfg, modelBuildOptions{
			includeProductIdentity: true,
			includeUploadInterval:  true,
		})
	}

	return builder.Build()
}

// buildBrokerURL 构建完整的Broker URL
func buildBrokerURL(serverURL string, port int) string {
	return normalizeServerURLWithPort(serverURL, "tcp", port)
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
		Type:       cfg.Type,
		Server:     cfg.ServerURL,
		Port:       cfg.Port,
		Path:       cfg.Path,
		Username:   cfg.Username,
		ClientID:   cfg.ClientID,
		Topic:      cfg.Topic,
		AlarmTopic: cfg.AlarmTopic,
		Connected:  cfg.Connected,
	}
	return info
}

// GetSupportedTypes 返回支持的北向类型及其字段描述
func GetSupportedTypes() map[string][]string {
	return map[string][]string{
		nbtype.TypeMQTT: {
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
		nbtype.TypeXunji: {
			"server_url: MQTT Broker 地址",
			"port: 端口 (默认1883)",
			"topic: 实时上报 Topic（默认 v1/gateway/{gatewayname}）",
			"username: 用户名 (可选)",
			"password: 密码 (可选)",
			"client_id: 客户端ID (可选)",
			"qos: QoS等级 (0-2)",
			"retain: 是否保留消息",
			"keep_alive: 心跳周期秒数",
			"timeout: 连接超时秒数",
			"upload_interval: 上传周期毫秒数",
		},
		nbtype.TypeSagoo: {
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
		nbtype.TypePandaX: {
			"server_url: PandaX Broker 地址",
			"port: 端口 (默认1883)",
			"username: 设备 Token（MQTT Username）",
			"password: 密码 (可选)",
			"client_id: 客户端ID (可选)",
			"qos: QoS等级 (0-2)",
			"retain: 是否保留消息",
			"keep_alive: 心跳周期秒数",
			"timeout: 连接超时秒数",
			"upload_interval: 上传周期毫秒数",
		},
		nbtype.TypeIThings: {
			"server_url: iThings Broker 地址",
			"port: 端口 (默认1883)",
			"username: MQTT 用户名",
			"password: MQTT 密码",
			"product_key: 网关产品ID",
			"device_key: 网关设备名",
			"gateway_mode: 仅支持 true（网关+子设备）",
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
	switch nbtype.Normalize(northboundType) {
	case nbtype.TypeMQTT:
		if broker, ok := config["broker"].(string); !ok || strings.TrimSpace(broker) == "" {
			return fmt.Errorf("broker is required for MQTT adapter")
		}
		if topic, ok := config["topic"].(string); !ok || strings.TrimSpace(topic) == "" {
			return fmt.Errorf("topic is required for MQTT adapter")
		}
	case nbtype.TypeXunji:
		if strings.TrimSpace(configServerURL(config)) == "" {
			return fmt.Errorf("serverUrl is required for Xunji adapter")
		}
	case nbtype.TypeSagoo:
		if serverURL, ok := config["serverUrl"].(string); !ok || strings.TrimSpace(serverURL) == "" {
			return fmt.Errorf("serverUrl is required for Sagoo adapter")
		}
		if productKey, ok := config["productKey"].(string); !ok || strings.TrimSpace(productKey) == "" {
			return fmt.Errorf("productKey is required for Sagoo adapter")
		}
		if deviceKey, ok := config["deviceKey"].(string); !ok || strings.TrimSpace(deviceKey) == "" {
			return fmt.Errorf("deviceKey is required for Sagoo adapter")
		}
	case nbtype.TypePandaX:
		if strings.TrimSpace(configServerURL(config)) == "" {
			return fmt.Errorf("serverUrl is required for PandaX adapter")
		}
		if username, ok := config["username"].(string); !ok || strings.TrimSpace(username) == "" {
			return fmt.Errorf("username is required for PandaX adapter")
		}
		if _, ok := config["gatewayMode"]; ok && !pickConfigBool(config, true, "gatewayMode") {
			return fmt.Errorf("gatewayMode must be true for PandaX adapter")
		}
	case nbtype.TypeIThings:
		if strings.TrimSpace(configServerURL(config)) == "" {
			return fmt.Errorf("serverUrl is required for iThings adapter")
		}
		if username, ok := config["username"].(string); !ok || strings.TrimSpace(username) == "" {
			return fmt.Errorf("username is required for iThings adapter")
		}
		if productKey, ok := config["productKey"].(string); !ok || strings.TrimSpace(productKey) == "" {
			return fmt.Errorf("productKey is required for iThings adapter")
		}
		if deviceKey, ok := config["deviceKey"].(string); !ok || strings.TrimSpace(deviceKey) == "" {
			return fmt.Errorf("deviceKey is required for iThings adapter")
		}
	default:
		return fmt.Errorf("unknown northbound type: %s", northboundType)
	}

	return nil
}
