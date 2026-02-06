package handlers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

type northboundRuntimeView struct {
	Registered     bool   `json:"registered"`
	Enabled        bool   `json:"enabled"`
	UploadInterval int64  `json:"upload_interval"`
	Pending        bool   `json:"pending"`
	LastSentAt     string `json:"last_sent_at,omitempty"`
	BreakerState   string `json:"breaker_state"`
}

type northboundConfigView struct {
	*models.NorthboundConfig
	Runtime northboundRuntimeView `json:"runtime"`
}

func normalizeNorthboundConfig(config *models.NorthboundConfig) {
	if config == nil {
		return
	}
	config.Name = strings.TrimSpace(config.Name)
	config.Type = strings.TrimSpace(config.Type)
	if strings.TrimSpace(config.Config) == "" {
		config.Config = "{}"
	}
	if config.Enabled != 1 {
		config.Enabled = 0
	}
	if config.UploadInterval <= 0 {
		config.UploadInterval = 5000
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
	if !json.Valid([]byte(config.Config)) {
		return fmt.Errorf("config must be valid JSON")
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

	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config.Config), &raw); err != nil {
		return err
	}

	pk, _ := raw["productKey"].(string)
	dk, _ := raw["deviceKey"].(string)
	gwPK, gwDK := database.GetGatewayIdentity()

	if strings.TrimSpace(pk) == "" && gwPK != "" {
		raw["productKey"] = gwPK
	}
	if strings.TrimSpace(dk) == "" && gwDK != "" {
		raw["deviceKey"] = gwDK
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	config.Config = string(b)
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
	}
	if runtime.UploadInterval <= 0 {
		runtime.UploadInterval = int64(config.UploadInterval)
	}
	if ts := h.northboundMgr.GetLastUploadTime(config.Name); !ts.IsZero() {
		runtime.LastSentAt = ts.Format(time.RFC3339)
	}
	return &northboundConfigView{NorthboundConfig: config, Runtime: runtime}
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
	h.northboundMgr.RemoveAdapter(config.Name)
	adapter, err := northbound.NewAdapterFromConfig(h.northboundMgr.PluginDir(), config)
	if err != nil {
		return err
	}
	h.northboundMgr.RegisterAdapter(config.Name, adapter)
	return nil
}
