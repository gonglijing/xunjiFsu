package httpapi

import (
	"time"

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
	Runtime        northboundRuntimeView `json:"runtime"`
	Connection     *ConnectionView       `json:"connection,omitempty"`
	SupportedTypes []string              `json:"supported_types,omitempty"`
	SchemaFields   []SchemaFieldView     `json:"schema_fields,omitempty"`
}

type SchemaFieldView struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Optional    bool   `json:"optional"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

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

func (api *NorthboundAPI) buildNorthboundConfigView(config *models.NorthboundConfig) *northboundConfigView {
	if config == nil {
		return nil
	}
	config.Type = normalizeNorthboundType(config.Type)
	return &northboundConfigView{
		NorthboundConfig: config,
		Runtime:          api.buildNorthboundRuntimeView(config),
		Connection:       buildNorthboundConnectionView(config),
		SupportedTypes:   adapters.SupportedTypes(),
		SchemaFields:     buildSchemaFieldViews(config.Type),
	}
}

func (api *NorthboundAPI) buildNorthboundRuntimeView(config *models.NorthboundConfig) northboundRuntimeView {
	runtime := northboundRuntimeView{Connected: config.Connected}
	if api.manager != nil {
		runtime.Registered = api.manager.HasAdapter(config.Name)
		runtime.Enabled = api.manager.IsEnabled(config.Name)
		runtime.UploadInterval = api.manager.GetInterval(config.Name).Milliseconds()
		runtime.Pending = api.manager.HasPending(config.Name)
		runtime.BreakerState = api.manager.GetBreakerState(config.Name).String()
		if ts := api.manager.GetLastUploadTime(config.Name); !ts.IsZero() {
			runtime.LastSentAt = ts.Format(time.RFC3339)
		}
	}
	if runtime.UploadInterval <= 0 {
		runtime.UploadInterval = int64(config.UploadInterval)
	}
	return runtime
}

func buildNorthboundConnectionView(config *models.NorthboundConfig) *ConnectionView {
	return &ConnectionView{
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
}
