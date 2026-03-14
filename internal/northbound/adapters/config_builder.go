package adapters

import (
	"encoding/json"

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
