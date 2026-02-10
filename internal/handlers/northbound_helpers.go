package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
	"github.com/gonglijing/xunjiFsu/internal/northbound/nbtype"
	northboundschema "github.com/gonglijing/xunjiFsu/internal/northbound/schema"
)

type requiredFieldRule struct {
	fieldName string
	present   func(*models.NorthboundConfig) bool
}

var northboundRequiredFieldRules = map[string][]requiredFieldRule{
	nbtype.TypeMQTT: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "topic", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.Topic) != "" }},
	},
	nbtype.TypeSagoo: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "product_key", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ProductKey) != "" }},
		{fieldName: "device_key", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.DeviceKey) != "" }},
	},
	nbtype.TypePandaX: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "username", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.Username) != "" }},
	},
	nbtype.TypeIThings: {
		{fieldName: "server_url", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.ServerURL) != "" }},
		{fieldName: "username", present: func(cfg *models.NorthboundConfig) bool { return strings.TrimSpace(cfg.Username) != "" }},
	},
}

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
	Runtime        northboundRuntimeView `json:"runtime"`
	Connection     *ConnectionView       `json:"connection,omitempty"`
	SupportedTypes []string              `json:"supported_types,omitempty"`
	SchemaFields   []SchemaFieldView     `json:"schema_fields,omitempty"`
}

// SchemaFieldView 用于前端展示 schema 字段信息
type SchemaFieldView struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Optional    bool   `json:"optional"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
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
	config.Type = normalizeNorthboundType(config.Type)
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
		case nbtype.TypeMQTT, nbtype.TypeSagoo, nbtype.TypePandaX, nbtype.TypeIThings:
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

	config.Type = normalizeNorthboundType(config.Type)
	if !nbtype.IsSupported(config.Type) {
		return fmt.Errorf("invalid type: %s, must be one of: %s", config.Type, strings.Join(nbtype.SupportedTypes(), ", "))
	}

	// 如果有 config JSON 字段，验证 schema
	if hasSchemaConfig(config) {
		if err := validateConfigBySchema(config.Type, config.Config); err != nil {
			return err
		}
	}

	if err := validateRequiredFields(config); err != nil {
		return err
	}

	return nil
}

func normalizeNorthboundType(raw string) string {
	return nbtype.Normalize(raw)
}

func hasSchemaConfig(config *models.NorthboundConfig) bool {
	if config == nil {
		return false
	}
	trimmed := strings.TrimSpace(config.Config)
	return trimmed != "" && trimmed != "{}"
}

func validateRequiredFields(config *models.NorthboundConfig) error {
	if config == nil || hasSchemaConfig(config) {
		return nil
	}

	rules, ok := northboundRequiredFieldRules[config.Type]
	if !ok {
		return nil
	}

	typeName := nbtype.DisplayName(config.Type)
	for _, rule := range rules {
		if !rule.present(config) {
			return fmt.Errorf("%s or config is required for %s type", rule.fieldName, typeName)
		}
	}

	return nil
}

// validateConfigBySchema 使用 schema 验证配置 JSON
func validateConfigBySchema(nbType string, configJSON string) error {
	fields, ok := northboundschema.FieldsByType(nbType)
	if !ok {
		// 不支持的类型，跳过 schema 验证
		return nil
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("config is invalid JSON: %w", err)
	}

	// 验证必填字段
	for _, field := range fields {
		if !field.Required {
			continue
		}
		value, exists := config[field.Key]
		if !exists || value == nil {
			return fmt.Errorf("%s is required", field.Label)
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			return fmt.Errorf("%s is required", field.Label)
		}
	}

	return nil
}

func (h *Handler) buildNorthboundConfigView(config *models.NorthboundConfig) *northboundConfigView {
	if config == nil {
		return nil
	}

	config.Type = normalizeNorthboundType(config.Type)

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

	// 获取 schema 字段
	var schemaFields []SchemaFieldView
	if fields, ok := northboundschema.FieldsByType(config.Type); ok {
		for _, f := range fields {
			schemaFields = append(schemaFields, SchemaFieldView{
				Key:         f.Key,
				Label:       f.Label,
				Type:        string(f.Type),
				Required:    f.Required,
				Optional:    f.Optional,
				Default:     f.Default,
				Description: f.Description,
			})
		}
	}

	// 获取支持的类型列表
	supportedTypes := adapters.SupportedTypes()

	return &northboundConfigView{
		NorthboundConfig: config,
		Runtime:          runtime,
		Connection:       connView,
		SupportedTypes:   supportedTypes,
		SchemaFields:     schemaFields,
	}
}

func (h *Handler) rebuildNorthboundRuntime(cfg *models.NorthboundConfig) error {
	if cfg == nil {
		return fmt.Errorf("northbound config is nil")
	}
	normalizeNorthboundConfig(cfg)

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

	// 如果已经有 config 字段，优先使用它（支持前端 schema 方式）
	if config.Config != "" && config.Config != "{}" {
		configJSON = config.Config
	}

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
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrNorthboundConfigNotFound)
		return
	}

	connInfo := adapters.ParseConnectionInfoFromModel(config)
	WriteSuccess(w, connInfo)
}
