package app

import (
	"context"
	"path/filepath"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/graceful"
	"github.com/gonglijing/xunjiFsu/internal/httpapi"
	"log/slog"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

const retentionCleanupInterval = 24 * time.Hour

// Run boots the application and blocks until shutdown completes.
func Run(cfg *config.Config) error {
	secretKey := loadOrGenerateSecretKey()

	if err := initDatabases(cfg); err != nil {
		return err
	}
	defer database.ParamDB.Close()
	defer database.DataDB.Close()

	if err := initSchemasAndDefaultData(); err != nil {
		return err
	}

	startBackgroundTasks(cfg)

	driverManager := driver.NewDriverManager()
	driverExecutor := driver.NewDriverExecutor(driverManager)
	driverManager.SetCallTimeout(cfg.DriverCallTimeout)

	if err := loadEnabledDrivers(cfg, driverManager); err != nil {
		slog.Warn("Failed to load drivers", "error", err)
	}

	northboundMgr := northbound.NewNorthboundManager()
	loadEnabledNorthboundConfigs(northboundMgr)
	startNorthboundSchedulers(northboundMgr)
	northboundMgr.Start()

	collect := collector.NewCollectorWithIntervals(driverExecutor, northboundMgr, cfg.CollectorDeviceSyncInterval, cfg.CollectorCommandPollInterval)
	applyRuntimeTuning(cfg, collect, driverExecutor, northboundMgr)
	authManager := auth.NewJWTManager(secretKey)
	pageHandler := httpapi.NewAuthHandler(authManager)
	apiDeps := newAPIRouteDeps(cfg, collect, driverExecutor, driverManager, northboundMgr, authManager)

	router := buildRouter(pageHandler, apiDeps, authManager)
	finalHandler := buildHandlerChain(cfg, router)

	if err := collect.Start(); err != nil {
		slog.Warn("Failed to start collector", "error", err)
	}

	sysCollector := startSystemStatsCollector(northboundMgr)

	gracefulMgr := graceful.NewGracefulShutdown(30 * time.Second)
	registerShutdown(gracefulMgr, collect, northboundMgr, sysCollector, cfg)
	gracefulMgr.Start()

	server := buildHTTPServer(cfg, finalHandler)
	gracefulMgr.SetHTTPServer(server)

	if err := serveHTTPServer(server, cfg); err != nil {
		return err
	}

	gracefulMgr.Wait()
	return nil
}

func startBackgroundTasks(cfg *config.Config) {
	slog.Info("Starting collect data writer...")
	database.StartCollectDataWriter()

	slog.Info("Starting data sync task...")
	database.StartDataSync()

	slog.Info("Starting retention cleanup task...")
	database.StartRetentionCleanup(retentionCleanupInterval)

	if cfg != nil && cfg.ThresholdCacheEnabled {
		slog.Info("Starting threshold cache...")
		collector.StartThresholdCache()
	}
}

func startSystemStatsCollector(northboundMgr *northbound.NorthboundManager) *collector.SystemStatsCollector {
	sysCollector := collector.GetSystemStatsCollector()
	sysCollector.SetNorthboundManager(northboundMgr)
	if err := sysCollector.Start(); err != nil {
		slog.Warn("Failed to start system stats collector", "error", err)
	}
	return sysCollector
}

func loadEnabledDrivers(cfg *config.Config, manager *driver.DriverManager) error {
	drivers, err := database.ListDrivers()
	if err != nil {
		return errDriverQuery(err)
	}

	loaded := 0
	for _, driverModel := range drivers {
		if !shouldLoadDriver(driverModel) {
			continue
		}
		driverModel.FilePath = resolveDriverFilePath(cfg, driverModel)
		if driverModel.FilePath == "" {
			slog.Debug("Skipping driver with empty file_path", "id", driverModel.ID, "name", driverModel.Name)
			continue
		}
		if err := manager.LoadDriverFromModel(driverModel, 0); err != nil {
			slog.Warn("Failed to load driver", "id", driverModel.ID, "name", driverModel.Name, "error", err)
			continue
		}
		loaded++
		slog.Info("Loaded driver", "id", driverModel.ID, "name", driverModel.Name)
	}
	slog.Info("Drivers loaded", "count", loaded)
	return nil
}

func shouldLoadDriver(driverModel *models.Driver) bool {
	return driverModel != nil && driverModel.Enabled == 1
}

func resolveDriverFilePath(cfg *config.Config, driverModel *models.Driver) string {
	if driverModel == nil {
		return ""
	}
	if driverModel.FilePath != "" || cfg == nil {
		return driverModel.FilePath
	}
	return filepath.Join(cfg.DriversDir, driverModel.Name+".wasm")
}

func registerShutdown(gracefulMgr *graceful.GracefulShutdown, collect *collector.Collector, northMgr *northbound.NorthboundManager, sysCollector *collector.SystemStatsCollector, cfg *config.Config) {
	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Stopping collector...")
		return collect.Stop()
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Stopping system stats collector...")
		if sysCollector != nil {
			return sysCollector.Stop()
		}
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Stopping collect data writer...")
		database.StopCollectDataWriter()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Stopping data sync...")
		database.StopDataSync()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Stopping retention cleanup...")
		database.StopRetentionCleanup()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Final sync to disk...")
		return database.SyncDataToDisk()
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		slog.Info("Stopping northbound manager...")
		northMgr.Stop()
		return nil
	})

	if cfg.ThresholdCacheEnabled {
		gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
			slog.Info("Stopping threshold cache...")
			collector.StopThresholdCache()
			return nil
		})
	}
}
