package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
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
	config, ok := loadNorthboundConfigOrWriteNotFound(w, r)
	if !ok {
		return
	}

	prevState := config.Enabled
	nextState := 0
	if prevState == 0 {
		nextState = 1
	}

	if err := database.UpdateNorthboundEnabled(config.ID, nextState); err != nil {
		writeServerErrorWithLog(w, apiErrToggleNorthboundFailed, err)
		return
	}

	config.Enabled = nextState
	if err := h.rebuildNorthboundRuntime(config); err != nil {
		_ = database.UpdateNorthboundEnabled(config.ID, prevState)
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
	config, ok := loadNorthboundConfigOrWriteNotFound(w, r)
	if !ok {
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
	config, ok := loadNorthboundConfigOrWriteNotFound(w, r)
	if !ok {
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

	adapter, err := h.runtimeAdapterForConfig(config)
	if err != nil {
		return err
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

	configByName := northboundConfigsByName(configs)
	names := listNorthboundStatusNames(configByName, h.northboundMgr.ListRuntimeNames())
	items := buildNorthboundStatusItems(configByName, names, h.northboundMgr)

	WriteSuccess(w, items)
}

func loadNorthboundConfigOrWriteNotFound(w http.ResponseWriter, r *http.Request) (*models.NorthboundConfig, bool) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return nil, false
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrNorthboundConfigNotFound)
		return nil, false
	}
	return config, true
}

func (h *Handler) runtimeAdapterForConfig(config *models.NorthboundConfig) (adapters.NorthboundAdapter, error) {
	adapter, err := h.northboundMgr.GetAdapter(config.Name)
	if err == nil {
		return adapter, nil
	}

	if rebuildErr := h.rebuildNorthboundRuntime(config); rebuildErr != nil {
		return nil, fmt.Errorf("rebuild runtime failed: %w", rebuildErr)
	}

	adapter, err = h.northboundMgr.GetAdapter(config.Name)
	if err != nil {
		return nil, fmt.Errorf("get adapter failed: %w", err)
	}
	return adapter, nil
}

func northboundConfigsByName(configs []*models.NorthboundConfig) map[string]*models.NorthboundConfig {
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
	return configByName
}

func listNorthboundStatusNames(configByName map[string]*models.NorthboundConfig, runtimeNames []string) []string {
	names := make([]string, 0, len(configByName)+len(runtimeNames))
	for name := range configByName {
		names = append(names, name)
	}
	for _, runtimeName := range runtimeNames {
		name := normalizeNorthboundName(runtimeName)
		if name == "" {
			continue
		}
		if _, exists := configByName[name]; exists {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func buildNorthboundStatusItems(configByName map[string]*models.NorthboundConfig, names []string, mgr *northbound.NorthboundManager) []northboundStatusItem {
	items := make([]northboundStatusItem, 0, len(names))
	for _, name := range names {
		items = append(items, buildNorthboundStatusItem(name, configByName[name], mgr.RuntimeStatus(name)))
	}
	return items
}

func buildNorthboundStatusItem(name string, cfg *models.NorthboundConfig, runtimeStatus northbound.RuntimeStatus) northboundStatusItem {
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
	return item
}

func normalizeNorthboundName(name string) string {
	return strings.TrimSpace(name)
}
