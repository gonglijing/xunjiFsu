package handlers

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ToggleNorthboundEnable 切换北向使能状态
func (h *Handler) ToggleNorthboundEnable(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrNorthboundConfigNotFound)
		return
	}

	prevState := config.Enabled
	nextState := 0
	if prevState == 0 {
		nextState = 1
	}

	if err := database.UpdateNorthboundEnabled(id, nextState); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	config.Enabled = nextState
	if err := h.rebuildNorthboundRuntime(config); err != nil {
		_ = database.UpdateNorthboundEnabled(id, prevState)
		config.Enabled = prevState
		_ = h.rebuildNorthboundRuntime(config)
		WriteBadRequest(w, "北向初始化失败: "+err.Error())
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": nextState,
	})
}

// ReloadNorthboundConfig 重载单个北向运行时
func (h *Handler) ReloadNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrNorthboundConfigNotFound)
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
		if cfg == nil {
			continue
		}
		name := normalizeNorthboundName(cfg.Name)
		if name == "" {
			continue
		}
		configByName[name] = cfg
	}

	nameSet := make(map[string]struct{}, len(configByName)+8)
	for _, name := range h.northboundMgr.ListRuntimeNames() {
		name = normalizeNorthboundName(name)
		if name != "" {
			nameSet[name] = struct{}{}
		}
	}
	for name := range configByName {
		nameSet[name] = struct{}{}
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]map[string]interface{}, 0, len(names))
	for _, name := range names {
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

func normalizeNorthboundName(name string) string {
	return strings.TrimSpace(name)
}
