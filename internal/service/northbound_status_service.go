package service

import (
	"slices"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

type NorthboundStatusItem struct {
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

func (s *NorthboundService) ListStatusItems() ([]NorthboundStatusItem, error) {
	configs, err := database.ListNorthboundConfigs()
	if err != nil {
		return nil, err
	}

	runtimeNames := []string(nil)
	if s.manager != nil {
		runtimeNames = s.manager.ListRuntimeNames()
	}

	configByName := BuildNorthboundConfigIndex(configs)
	names := ListNorthboundStatusNames(configByName, runtimeNames)
	return BuildNorthboundStatusItems(configByName, names, s.manager), nil
}

func BuildNorthboundConfigIndex(configs []*models.NorthboundConfig) map[string]*models.NorthboundConfig {
	configByName := make(map[string]*models.NorthboundConfig, len(configs))
	for _, cfg := range configs {
		if cfg == nil {
			continue
		}
		name := strings.TrimSpace(cfg.Name)
		if name == "" {
			continue
		}
		configByName[name] = cfg
	}
	return configByName
}

func ListNorthboundStatusNames(configByName map[string]*models.NorthboundConfig, runtimeNames []string) []string {
	names := make([]string, 0, len(configByName)+len(runtimeNames))
	for name := range configByName {
		names = append(names, name)
	}
	for _, runtimeName := range runtimeNames {
		name := strings.TrimSpace(runtimeName)
		if name == "" {
			continue
		}
		if _, exists := configByName[name]; exists {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func BuildNorthboundStatusItems(configByName map[string]*models.NorthboundConfig, names []string, mgr *northbound.NorthboundManager) []NorthboundStatusItem {
	items := make([]NorthboundStatusItem, 0, len(names))
	for _, name := range names {
		runtimeStatus := northbound.RuntimeStatus{Name: name}
		if mgr != nil {
			runtimeStatus = mgr.RuntimeStatus(name)
		}
		items = append(items, buildNorthboundStatusItem(name, configByName[name], runtimeStatus))
	}
	return items
}

func buildNorthboundStatusItem(name string, cfg *models.NorthboundConfig, runtimeStatus northbound.RuntimeStatus) NorthboundStatusItem {
	item := NorthboundStatusItem{
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
		item.Type = strings.ToLower(strings.TrimSpace(cfg.Type))
		item.DBEnabled = cfg.Enabled == 1
		item.DBUploadInterval = cfg.UploadInterval
	}
	if !runtimeStatus.LastSentAt.IsZero() {
		item.LastSentAt = runtimeStatus.LastSentAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return item
}
