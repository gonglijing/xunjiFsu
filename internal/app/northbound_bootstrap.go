package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

func loadEnabledNorthboundConfigs(northboundMgr *northbound.NorthboundManager) {
	logger.Info("Loading enabled northbound configs...")
	configs, err := database.ListNorthboundConfigs()
	if err != nil {
		logger.Warn("Failed to load northbound configs", "error", err)
		return
	}

	sysStatsCollector := collector.GetSystemStatsCollector()

	enabledCount := 0
	for _, config := range configs {
		if !shouldLoadNorthboundConfig(config) {
			continue
		}
		if err := registerNorthboundAdapter(northboundMgr, config, sysStatsCollector); err != nil {
			logger.Warn("Failed to register northbound config", "name", config.Name, "error", err)
			continue
		}
		logger.Info("Northbound config registered",
			"name", config.Name,
			"type", config.Type,
			"upload_interval", config.UploadInterval)
		enabledCount++
	}

	logger.Info("Loaded enabled northbound configs", "count", enabledCount)
}

func registerNorthboundAdapter(northboundMgr *northbound.NorthboundManager, config *models.NorthboundConfig, sysStatsCollector *collector.SystemStatsCollector) error {
	adapter := adapters.NewAdapter(config.Type, config.Name)
	if adapter == nil {
		return fmt.Errorf("unsupported northbound type: %s", config.Type)
	}

	if err := adapter.Initialize(buildNorthboundConfigPayload(config)); err != nil {
		return fmt.Errorf("initialize northbound adapter %s: %w", config.Name, err)
	}

	interval := time.Duration(config.UploadInterval) * time.Millisecond
	adapter.SetInterval(interval)

	registerPandaxSystemStatsProvider(adapter, config, sysStatsCollector)
	northboundMgr.RegisterAdapter(config.Name, adapter)
	return nil
}

func shouldLoadNorthboundConfig(config *models.NorthboundConfig) bool {
	return config != nil && config.Enabled == 1
}

func buildNorthboundConfigPayload(config *models.NorthboundConfig) string {
	if config == nil {
		return ""
	}
	trimmed := strings.TrimSpace(config.Config)
	if trimmed != "" && trimmed != "{}" {
		return config.Config
	}
	return adapters.BuildConfigFromModel(config)
}

func registerPandaxSystemStatsProvider(adapter adapters.NorthboundAdapter, config *models.NorthboundConfig, sysStatsCollector *collector.SystemStatsCollector) {
	if config == nil || sysStatsCollector == nil || config.Type != "pandax" {
		return
	}
	pandaxAdapter, ok := adapter.(*adapters.PandaXAdapter)
	if !ok {
		return
	}
	pandaxAdapter.SetSystemStatsProvider(sysStatsCollector)
	logger.Info("Set system stats provider for pandax adapter")
}
