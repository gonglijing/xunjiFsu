package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

type northboundRuntimeView struct {
	Registered     bool   `json:"registered"`
	Enabled        bool   `json:"enabled"`
	UploadInterval int64  `json:"upload_interval"`
	Pending        bool   `json:"pending"`
	LastSentAt     string `json:"last_sent_at,omitempty"`
	BreakerState   string `json:"breaker_state"`
	Connected      bool   `json:"connected"`
}

type northboundConfigView struct {
	*models.NorthboundConfig
	Runtime      northboundRuntimeView `json:"runtime"`
	Connection   *ConnectionView       `json:"connection,omitempty"`
	SupportedTypes []string           `json:"supported_types,omitempty"`
}

// ConnectionView 连接信息视图（不返回密码）
type ConnectionView struct {
	Type       string `json:"type"`
	ServerURL  string `json:"server_url"`
	Port       int    `json:"port"`
	Path       string `json:"path,omitempty"`
	Username   string `json:"username,omitempty"`
	ClientID   string `json:"client_id,omitempty"`
	Topic      string `json:"topic,omitempty"`
	AlarmTopic string `json:"alarm_topic,omitempty"`
	ProductKey string `json:"product_key,omitempty"`
	DeviceKey  string `json:"device_key,omitempty"`
	QOS        int    `json:"qos"`
	Retain     bool   `json:"retain"`
	KeepAlive  int    `json:"keep_alive"`
	Timeout    int    `json:"timeout"`
	Connected  bool   `json:"connected"`
}

func normalizeNorthboundConfig(config *models.NorthboundConfig) {
	if config == nil {
		return
	}
	config.Name = strings.TrimSpace(config.Name)
	config.Type = strings.TrimSpace(config.Type)
	config.ServerURL = strings.TrimSpace(config.ServerURL)
	config.Path = strings.TrimSpace(config.Path)
	config.Username = strings.TrimSpace(config.Username)
	config.Password = strings.TrimSpace(config.Password)
	config.ClientID = strings.TrimSpace(config.ClientID)
	config.Topic = strings.TrimSpace(config.Topic)
	config.AlarmTopic = strings.TrimSpace(config.AlarmTopic)
	config.ProductKey = strings.TrimSpace(config.ProductKey)
	config.DeviceKey = strings.TrimSpace(config.DeviceKey)

	if config.Enabled != 1 {
		config.Enabled = 0
	}
	if config.UploadInterval <= 0 {
		config.UploadInterval = 5000
	}
	if config.Port <= 0 {
		switch config.Type {
		case "http":
			config.Port = 80
		case "mqtt", "xunji":
			config.Port = 1883
		}
	}
	if config.QOS < 0 || config.QOS > 2 {
		config.QOS = 0
	}
	if config.KeepAlive <= 0 {
		config.KeepAlive = 60
	}
	if config.Timeout <= 0 {
		config.Timeout = 30
	}
}

func validateNorthboundConfig(config *models.NorthboundConfig) error {
	if config == nil {
		return fmt.Errorf("northbound config is nil")
	}
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}
	if config.Type == "" {
		return fmt.Errorf("type is required")
	}

	// 验证类型
	validTypes := []string{"http", "mqtt", "xunji"}
	isValid := false
	for _, t := range validTypes {
		if strings.ToLower(config.Type) == t {
			isValid = true
			config.Type = t
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid type: %s, must be one of: http, mqtt, xunji", config.Type)
	}

	// 根据类型验证必填字段
	switch config.Type {
	case "http":
		if config.ServerURL == "" {
			return fmt.Errorf("server_url is required for HTTP type")
		}
	case "mqtt":
		if config.ServerURL == "" {
			return fmt.Errorf("server_url is required for MQTT type")
		}
		if config.Topic == "" {
			return fmt.Errorf("topic is required for MQTT type")
		}
	case "xunji":
		if config.ServerURL == "" {
			return fmt.Errorf("server_url is required for XunJi type")
		}
		if config.ProductKey == "" {
			return fmt.Errorf("product_key is required for XunJi type")
		}
		if config.DeviceKey == "" {
			return fmt.Errorf("device_key is required for XunJi type")
		}
	}

	return nil
}

func enrichNorthboundConfigWithGatewayIdentity(config *models.NorthboundConfig) error {
	if config == nil {
		return nil
	}
	if strings.ToLower(config.Type) != "xunji" {
		return nil
	}

	// 如果 product_key 为空，从网关配置获取
	if config.ProductKey == "" {
		gwPK, _ := database.GetGatewayIdentity()
		if gwPK != "" {
			config.ProductKey = gwPK
		}
	}

	// 如果 device_key 为空，从网关配置获取
	if config.DeviceKey == "" {
		_, gwDK := database.GetGatewayIdentity()
		if gwDK != "" {
			config.DeviceKey = gwDK
		}
	}

	return nil
}

func (h *Handler) buildNorthboundConfigView(config *models.NorthboundConfig) *northboundConfigView {
	if config == nil {
		return nil
	}

	runtime := northboundRuntimeView{
		Registered:     h.northboundMgr.HasAdapter(config.Name),
		Enabled:        h.northboundMgr.IsEnabled(config.Name),
		UploadInterval: h.northboundMgr.GetInterval(config.Name).Milliseconds(),
		Pending:        h.northboundMgr.HasPending(config.Name),
		BreakerState:   h.northboundMgr.GetBreakerState(config.Name).String(),
		Connected:      config.Connected,
	}

	if runtime.UploadInterval <= 0 {
		runtime.UploadInterval = int64(config.UploadInterval)
	}

	if ts := h.northboundMgr.GetLastUploadTime(config.Name); !ts.IsZero() {
		runtime.LastSentAt = ts.Format(time.RFC3339)
	}

	// 构建连接信息视图（不返回密码）
	connView := &ConnectionView{
		Type:       config.Type,
		ServerURL:  config.ServerURL,
		Port:       config.Port,
		Path:       config.Path,
		Username:   config.Username,
		ClientID:   config.ClientID,
		Topic:      config.Topic,
		AlarmTopic: config.AlarmTopic,
		ProductKey: config.ProductKey,
		DeviceKey:  config.DeviceKey,
		QOS:        config.QOS,
		Retain:     config.Retain,
		KeepAlive:  config.KeepAlive,
		Timeout:    config.Timeout,
		Connected:  config.Connected,
	}

	// 获取支持的类型列表
	supportedTypes := adapters.SupportedTypes()

	return &northboundConfigView{
		NorthboundConfig: config,
		Runtime:         runtime,
		Connection:      connView,
		SupportedTypes:  supportedTypes,
	}
}

func (h *Handler) rebuildNorthboundRuntime(cfg *models.NorthboundConfig) error {
	if cfg == nil {
		return fmt.Errorf("northbound config is nil")
	}
	normalizeNorthboundConfig(cfg)
	if err := enrichNorthboundConfigWithGatewayIdentity(cfg); err != nil {
		return err
	}

	h.northboundMgr.RemoveAdapter(cfg.Name)
	h.northboundMgr.SetInterval(cfg.Name, time.Duration(cfg.UploadInterval)*time.Millisecond)

	if cfg.Enabled == 0 {
		h.northboundMgr.SetEnabled(cfg.Name, false)
		return nil
	}

	if err := h.registerNorthboundAdapter(cfg); err != nil {
		h.northboundMgr.SetEnabled(cfg.Name, false)
		return err
	}
	h.northboundMgr.SetEnabled(cfg.Name, true)
	return nil
}

func (h *Handler) registerNorthboundAdapter(config *models.NorthboundConfig) error {
	// 从模型字段生成配置JSON
	configJSON := adapters.BuildConfigFromModel(config)

	// 使用内置适配器
	adapter := adapters.NewAdapter(config.Type, config.Name)
	if adapter == nil {
		return fmt.Errorf("unsupported northbound type: %s", config.Type)
	}

	if err := adapter.Initialize(configJSON); err != nil {
		return fmt.Errorf("initialize northbound adapter %s: %w", config.Name, err)
	}

	// 设置上传周期
	adapter.SetInterval(time.Duration(config.UploadInterval) * time.Millisecond)

	// 注册到管理器
	h.northboundMgr.RegisterAdapter(config.Name, adapter)
	return nil
}

// GetNorthboundSupportedTypes 获取支持的北向类型
func (h *Handler) GetNorthboundSupportedTypes(w http.ResponseWriter, r *http.Request) {
	types := adapters.GetSupportedTypes()
	WriteSuccess(w, types)
}

// GetNorthboundConnectionInfo 获取单个北向的连接信息
func (h *Handler) GetNorthboundConnectionInfo(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFound(w, "Northbound config not found")
		return
	}

	connInfo := adapters.ParseConnectionInfoFromModel(config)
	WriteSuccess(w, connInfo)
}
