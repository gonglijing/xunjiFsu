package app

import (
	"fmt"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

func loadEnabledNorthboundConfigs(northboundMgr *northbound.NorthboundManager) {
	logger.Info("Loading enabled northbound configs...")
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		logger.Warn("Failed to load northbound configs", "error", err)
		return
	}

	enabledCount := 0
	for _, config := range configs {
		if config.Enabled != 1 {
			continue
		}
		if err := registerNorthboundAdapter(northboundMgr, config); err != nil {
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

func registerNorthboundAdapter(northboundMgr *northbound.NorthboundManager, config *models.NorthboundConfig) error {
	var adapter northbound.Northbound

	switch config.Type {
	case "xunji":
		adapter = northbound.NewXunJiAdapter()
	case "http":
		adapter = northbound.NewHTTPAdapter()
	case "mqtt":
		adapter = northbound.NewMQTTAdapter()
	default:
		return fmt.Errorf("unknown northbound type: %s", config.Type)
	}

	if err := adapter.Initialize(config.Config); err != nil {
		return fmt.Errorf("initialize northbound adapter %s: %w", config.Name, err)
	}

	northboundMgr.RegisterAdapter(config.Name, adapter)
	return nil
}
