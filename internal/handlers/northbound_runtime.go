package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

type northboundStatusItem struct {
	ID               int64  `json:"id,omitempty"`
	Name             string `json:"name"`
	Type             string `json:"type,omitempty"`
	Configured       bool   `json:"configured"`
	Registered       bool   `json:"registered"`
	Enabled          bool   `json:"enabled"`
	Connected        bool   `json:"connected"`
	UploadInterval   int64  `json:"upload_interval"`
	Pending          bool   `json:"pending"`
	BreakerState     string `json:"breaker_state"`
	DBEnabled        bool   `json:"db_enabled,omitempty"`
	DBUploadInterval int    `json:"db_upload_interval,omitempty"`
	LastSentAt       string `json:"last_sent_at,omitempty"`
}

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
		writeServerErrorWithLog(w, apiErrToggleNorthboundFailed, err)
		return
	}

	config.Enabled = nextState
	if err := h.rebuildNorthboundRuntime(config); err != nil {
		_ = database.UpdateNorthboundEnabled(id, prevState)
		config.Enabled = prevState
		_ = h.rebuildNorthboundRuntime(config)
		WriteBadRequestCode(w, apiErrNorthboundInitializeFailed.Code, apiErrNorthboundInitializeFailed.Message+": "+err.Error())
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
		WriteBadRequestCode(w, apiErrNorthboundReloadFailed.Code, apiErrNorthboundReloadFailed.Message+": "+err.Error())
		return
	}

	WriteSuccess(w, h.buildNorthboundConfigView(config))
}

// SyncNorthboundDevices 触发北向设备同步（PandaX）
func (h *Handler) SyncNorthboundDevices(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrNorthboundConfigNotFound)
		return
	}

	if err := h.syncNorthboundDevices(config); err != nil {
		WriteBadRequestCode(w, apiErrNorthboundSyncDevicesFailed.Code, apiErrNorthboundSyncDevicesFailed.Message+": "+err.Error())
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"id":      config.ID,
		"name":    config.Name,
		"type":    config.Type,
		"message": "同步设备已触发",
	})
}

func (h *Handler) syncNorthboundDevices(config *models.NorthboundConfig) error {
	if config == nil {
		return fmt.Errorf("northbound config is nil")
	}
	if config.Enabled == 0 {
		return fmt.Errorf("northbound is disabled")
	}

	adapter, err := h.northboundMgr.GetAdapter(config.Name)
	if err != nil {
		if err := h.rebuildNorthboundRuntime(config); err != nil {
			return fmt.Errorf("rebuild runtime failed: %w", err)
		}
		adapter, err = h.northboundMgr.GetAdapter(config.Name)
		if err != nil {
			return fmt.Errorf("get adapter failed: %w", err)
		}
	}

	deviceSyncAdapter, ok := adapter.(adapters.NorthboundAdapterWithDeviceSync)
	if !ok {
		return fmt.Errorf("adapter type %s does not support device sync", normalizeNorthboundType(config.Type))
	}

	if err := deviceSyncAdapter.SyncDevices(); err != nil {
		return err
	}
	return nil
}

// GetNorthboundStatus 获取北向运行态
func (h *Handler) GetNorthboundStatus(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListNorthboundStatusFailed, err)
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

	names := make([]string, 0, len(configByName)+8)
	for name := range configByName {
		names = append(names, name)
	}
	for _, runtimeName := range h.northboundMgr.ListRuntimeNames() {
		name := normalizeNorthboundName(runtimeName)
		if name == "" {
			continue
		}
		if _, exists := configByName[name]; exists {
			continue
		}
		configByName[name] = nil
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]northboundStatusItem, 0, len(names))
	for _, name := range names {
		cfg := configByName[name]
		runtimeStatus := h.northboundMgr.RuntimeStatus(name)
		item := northboundStatusItem{
			Name:           name,
			Configured:     cfg != nil,
			Registered:     runtimeStatus.Registered,
			Enabled:        runtimeStatus.Enabled,
			Connected:      runtimeStatus.Connected,
			UploadInterval: runtimeStatus.UploadIntervalMS,
			Pending:        runtimeStatus.Pending,
			BreakerState:   runtimeStatus.BreakerState,
		}
		if cfg != nil {
			item.ID = cfg.ID
			item.Type = normalizeNorthboundType(cfg.Type)
			item.DBEnabled = cfg.Enabled == 1
			item.DBUploadInterval = cfg.UploadInterval
		}
		if !runtimeStatus.LastSentAt.IsZero() {
			item.LastSentAt = runtimeStatus.LastSentAt.Format("2006-01-02T15:04:05Z07:00")
		}
		items = append(items, item)
	}

	WriteSuccess(w, items)
}

func normalizeNorthboundName(name string) string {
	return strings.TrimSpace(name)
}
