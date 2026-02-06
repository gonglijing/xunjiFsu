package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// ==================== 北向配置管理 ====================

// GetNorthboundConfigs 获取所有北向配置
func (h *Handler) GetNorthboundConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	views := make([]*northboundConfigView, 0, len(configs))
	for _, cfg := range configs {
		views = append(views, h.buildNorthboundConfigView(cfg))
	}

	WriteSuccess(w, views)
}

// CreateNorthboundConfig 创建北向配置
func (h *Handler) CreateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	var config models.NorthboundConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	normalizeNorthboundConfig(&config)
	if err := validateNorthboundConfig(&config); err != nil {
		WriteBadRequest(w, err.Error())
		return
	}
	if err := enrichNorthboundConfigWithGatewayIdentity(&config); err != nil {
		WriteBadRequest(w, "config 参数无效: "+err.Error())
		return
	}

	if config.Enabled == 1 {
		if err := h.registerNorthboundAdapter(&config); err != nil {
			WriteBadRequest(w, "北向初始化失败: "+err.Error())
			return
		}
		h.northboundMgr.SetEnabled(config.Name, true)
	}
	h.northboundMgr.SetInterval(config.Name, time.Duration(config.UploadInterval)*time.Millisecond)
	h.northboundMgr.SetEnabled(config.Name, config.Enabled == 1)

	id, err := database.CreateNorthboundConfig(&config)
	if err != nil {
		h.northboundMgr.RemoveAdapter(config.Name)
		WriteServerError(w, err.Error())
		return
	}

	config.ID = id

	WriteCreated(w, h.buildNorthboundConfigView(&config))
}

// UpdateNorthboundConfig 更新北向配置
func (h *Handler) UpdateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	oldConfig, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFound(w, "Northbound config not found")
		return
	}

	var config models.NorthboundConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	normalizeNorthboundConfig(&config)
	if err := validateNorthboundConfig(&config); err != nil {
		WriteBadRequest(w, err.Error())
		return
	}
	if err := enrichNorthboundConfigWithGatewayIdentity(&config); err != nil {
		WriteBadRequest(w, "config 参数无效: "+err.Error())
		return
	}

	if oldConfig.Name != config.Name {
		h.northboundMgr.RemoveAdapter(oldConfig.Name)
	}

	if err := h.rebuildNorthboundRuntime(&config); err != nil {
		if oldConfig != nil {
			_ = h.rebuildNorthboundRuntime(oldConfig)
		}
		WriteBadRequest(w, "北向初始化失败: "+err.Error())
		return
	}

	config.ID = id
	if err := database.UpdateNorthboundConfig(&config); err != nil {
		h.northboundMgr.RemoveAdapter(config.Name)
		if oldConfig != nil {
			_ = h.rebuildNorthboundRuntime(oldConfig)
		}
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, h.buildNorthboundConfigView(&config))
}

// DeleteNorthboundConfig 删除北向配置
func (h *Handler) DeleteNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	// 获取配置信息，删除时移除适配器
	config, err := database.GetNorthboundConfigByID(id)
	if err == nil {
		h.northboundMgr.RemoveAdapter(config.Name)
	}

	if err := database.DeleteNorthboundConfig(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
}

// registerNorthboundAdapter 内部辅助函数：注册北向适配器
func (h *Handler) registerNorthboundAdapter(config *models.NorthboundConfig) error {
	h.northboundMgr.RemoveAdapter(config.Name)
	adapter, err := northbound.NewAdapterFromConfig(h.northboundMgr.PluginDir(), config)
	if err != nil {
		return err
	}
	h.northboundMgr.RegisterAdapter(config.Name, adapter)
	return nil
}

// ToggleNorthboundEnable 切换北向使能状态
func (h *Handler) ToggleNorthboundEnable(w http.ResponseWriter, r *http.Request) {
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

	// 切换状态
	newState := 0
	if config.Enabled == 0 {
		newState = 1
	}

	if err := database.UpdateNorthboundEnabled(id, newState); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 更新适配器
	if newState == 1 {
		if err := h.registerNorthboundAdapter(config); err != nil {
			_ = database.UpdateNorthboundEnabled(id, 0)
			WriteBadRequest(w, "北向初始化失败: "+err.Error())
			return
		}
		h.northboundMgr.SetEnabled(config.Name, true)
	} else {
		h.northboundMgr.RemoveAdapter(config.Name)
		h.northboundMgr.SetEnabled(config.Name, false)
	}
	h.northboundMgr.SetInterval(config.Name, time.Duration(config.UploadInterval)*time.Millisecond)

	WriteSuccess(w, map[string]interface{}{
		"enabled": newState,
	})
}

// ReloadNorthboundConfig 重载单个北向运行时
func (h *Handler) ReloadNorthboundConfig(w http.ResponseWriter, r *http.Request) {
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

	if err := h.rebuildNorthboundRuntime(config); err != nil {
		WriteBadRequest(w, "北向重载失败: "+err.Error())
		return
	}

	WriteSuccess(w, h.buildNorthboundConfigView(config))
}

// GetNorthboundStatus 获取北向运行态
func (h *Handler) GetNorthboundStatus(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	configByName := make(map[string]*models.NorthboundConfig, len(configs))
	for _, cfg := range configs {
		configByName[cfg.Name] = cfg
	}

	names := h.northboundMgr.ListRuntimeNames()
	for name := range configByName {
		names = append(names, name)
	}

	seen := make(map[string]struct{}, len(names))
	items := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		cfg := configByName[name]
		item := map[string]interface{}{
			"name":            name,
			"configured":      cfg != nil,
			"registered":      h.northboundMgr.HasAdapter(name),
			"enabled":         h.northboundMgr.IsEnabled(name),
			"upload_interval": h.northboundMgr.GetInterval(name).Milliseconds(),
			"pending":         h.northboundMgr.HasPending(name),
			"breaker_state":   h.northboundMgr.GetBreakerState(name).String(),
		}
		if cfg != nil {
			item["id"] = cfg.ID
			item["type"] = cfg.Type
			item["db_enabled"] = cfg.Enabled == 1
			item["db_upload_interval"] = cfg.UploadInterval
		}
		if ts := h.northboundMgr.GetLastUploadTime(name); !ts.IsZero() {
			item["last_sent_at"] = ts.Format(time.RFC3339)
		}
		items = append(items, item)
	}

	WriteSuccess(w, items)
}
