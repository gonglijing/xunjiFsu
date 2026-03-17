package app

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/graceful"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

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

	logger.Info("Starting data sync task...")
	database.StartDataSync()
	logger.Info("Starting retention cleanup task...")
	database.StartRetentionCleanup(24 * time.Hour)

	if cfg.ThresholdCacheEnabled {
		logger.Info("Starting threshold cache...")
		collector.StartThresholdCache()
	}

	driverManager := driver.NewDriverManager()
	driverExecutor := driver.NewDriverExecutor(driverManager)
	driverManager.SetCallTimeout(cfg.DriverCallTimeout)

	// 加载所有启用的驱动
	if err := loadEnabledDrivers(cfg, driverManager); err != nil {
		logger.Warn("Failed to load drivers", "error", err)
	}

	// 创建北向管理器（使用内置适配器，不再需要插件目录）
	northboundMgr := northbound.NewNorthboundManager()

	loadEnabledNorthboundConfigs(northboundMgr)

	// 开启北向上传调度（按配置）
	startNorthboundSchedulers(northboundMgr)
	northboundMgr.Start()

	collect := collector.NewCollectorWithIntervals(driverExecutor, northboundMgr, cfg.CollectorDeviceSyncInterval, cfg.CollectorCommandPollInterval)
	applyRuntimeTuning(cfg, collect, driverExecutor, northboundMgr)
	authManager := auth.NewJWTManager(secretKey)
	h := handlers.NewHandler(authManager, collect, cfg, driverExecutor, driverManager, northboundMgr, cfg.DriversDir)

	router := buildRouter(h, authManager)
	finalHandler := buildHandlerChain(cfg, router)

	if err := collect.Start(); err != nil {
		logger.Warn("Failed to start collector", "error", err)
	}

	// 启动系统属性采集器
	sysCollector := collector.GetSystemStatsCollector()
	sysCollector.SetNorthboundManager(northboundMgr)
	if err := sysCollector.Start(); err != nil {
		logger.Warn("Failed to start system stats collector", "error", err)
	}

	gracefulMgr := graceful.NewGracefulShutdown(30 * time.Second)
	registerShutdown(gracefulMgr, collect, northboundMgr, sysCollector, cfg)
	gracefulMgr.Start()

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      finalHandler,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	gracefulMgr.SetHTTPServer(server)

	// TLS 优先级：1) 自动证书 2) 指定证书 3) HTTP
	switch {
	case cfg.TLSAuto && cfg.TLSDomain != "":
		if err := listenAndServeWithAutoCert(server, cfg); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case cfg.TLSCertFile != "" && cfg.TLSKeyFile != "":
		logger.Info("Starting HTTPS", "addr", cfg.ListenAddr, "cert", cfg.TLSCertFile)
		if err := server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	default:
		logger.Info("Starting HTTP", "addr", cfg.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	}

	gracefulMgr.Wait()
	return nil
}

// loadEnabledDrivers 从数据库加载所有启用的驱动
func loadEnabledDrivers(cfg *config.Config, manager *driver.DriverManager) error {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		return fmt.Errorf("failed to get drivers: %w", err)
	}

	loaded := 0
	for _, driverModel := range drivers {
		if !shouldLoadDriver(driverModel) {
			continue
		}
		driverModel.FilePath = resolveDriverFilePath(cfg, driverModel)
		if driverModel.FilePath == "" {
			logger.Debug("Skipping driver with empty file_path", "id", driverModel.ID, "name", driverModel.Name)
			continue
		}
		if err := manager.LoadDriverFromModel(driverModel, 0); err != nil {
			logger.Warn("Failed to load driver", "id", driverModel.ID, "name", driverModel.Name, "error", err)
			continue
		}
		loaded++
		logger.Info("Loaded driver", "id", driverModel.ID, "name", driverModel.Name)
	}
	logger.Info("Drivers loaded", "count", loaded)
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
		logger.Info("Stopping collector...")
		return collect.Stop()
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping system stats collector...")
		if sysCollector != nil {
			return sysCollector.Stop()
		}
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping data sync...")
		database.StopDataSync()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping retention cleanup...")
		database.StopRetentionCleanup()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Final sync to disk...")
		return database.SyncDataToDisk()
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping northbound manager...")
		northMgr.Stop()
		return nil
	})

	if cfg.ThresholdCacheEnabled {
		gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
			logger.Info("Stopping threshold cache...")
			collector.StopThresholdCache()
			return nil
		})
	}
}
