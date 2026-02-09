package app

import (
	"fmt"
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
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		logger.Warn("Failed to load northbound configs", "error", err)
		return
	}

	// 获取系统属性采集器
	sysStatsCollector := collector.GetSystemStatsCollector()

	enabledCount := 0
	for _, config := range configs {
		if config.Enabled != 1 {
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
	// 从模型字段生成配置JSON
	configJSON := adapters.BuildConfigFromModel(config)

	// 使用内置适配器
	adapter := adapters.NewAdapter(config.Type, config.Name)
	if adapter == nil {
		return fmt.Errorf("unsupported northbound type: %s", config.Type)
	}

	if err := adapter.Initialize(configJSON); err != nil {
		return fmt.Errorf("initialize northbound adapter %s: %w", config.Name, err)
	}

	// 设置上传周期
	interval := time.Duration(config.UploadInterval) * time.Millisecond
	adapter.SetInterval(interval)

	// 对于 pandax 适配器，设置系统属性提供者
	if config.Type == "pandax" && sysStatsCollector != nil {
		if pandaxAdapter, ok := adapter.(*adapters.PandaXAdapter); ok {
			pandaxAdapter.SetSystemStatsProvider(sysStatsCollector)
			logger.Info("Set system stats provider for pandax adapter")
		}
	}

	// 注册到管理器
	northboundMgr.RegisterAdapter(config.Name, adapter)

	return nil
}
