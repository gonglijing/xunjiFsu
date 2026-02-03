package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/graceful"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"golang.org/x/crypto/acme/autocert"
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
	database.StartRetentionCleanup(6 * time.Hour)

	if cfg.ThresholdCacheEnabled {
		logger.Info("Starting threshold cache...")
		collector.StartThresholdCache()
	}

	driverManager := driver.NewDriverManager()
	driverExecutor := driver.NewDriverExecutor(driverManager)

	// 加载所有启用的驱动
	if err := loadEnabledDrivers(driverManager); err != nil {
		logger.Warn("Failed to load drivers", "error", err)
	}

	northboundMgr := northbound.NewNorthboundManager()

	loadEnabledNorthboundConfigs(northboundMgr)

	// 开启北向上传调度（按配置）
	startNorthboundSchedulers(northboundMgr)
	northboundMgr.Start()

	collect := collector.NewCollector(driverExecutor, northboundMgr)
	authManager := auth.NewJWTManager(secretKey)
	h := handlers.NewHandler(authManager, collect, driverManager, northboundMgr)

	router := buildRouter(h, authManager)
	finalHandler := buildHandlerChain(cfg, router)

	if err := collect.Start(); err != nil {
		logger.Warn("Failed to start collector", "error", err)
	}

	gracefulMgr := graceful.NewGracefulShutdown(30 * time.Second)
	registerShutdown(gracefulMgr, collect, northboundMgr, cfg)
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
		m := &autocert.Manager{
			Cache:      autocert.DirCache(cfg.TLSCacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(cfg.TLSDomain),
		}
		server.TLSConfig = &tls.Config{
			GetCertificate: m.GetCertificate,
			MinVersion:     tls.VersionTLS12,
		}
		go func() {
			_ = http.ListenAndServe(":80", m.HTTPHandler(nil))
		}()
		logger.Info("Starting HTTPS (auto-cert)", "addr", cfg.ListenAddr, "domain", cfg.TLSDomain)
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
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
func loadEnabledDrivers(manager *driver.DriverManager) error {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		return fmt.Errorf("failed to get drivers: %w", err)
	}

	loaded := 0
	for _, d := range drivers {
		if d.Enabled != 1 {
			continue
		}
		// 跳过 file_path 为空的驱动
		if d.FilePath == "" {
			logger.Debug("Skipping driver with empty file_path", "id", d.ID, "name", d.Name)
			continue
		}
		// 读取 WASM 文件
		wasmData, err := os.ReadFile(d.FilePath)
		if err != nil {
			logger.Warn("Failed to read driver file", "id", d.ID, "name", d.Name, "file", d.FilePath, "error", err)
			continue
		}
		if err := manager.LoadDriver(d, wasmData, 0); err != nil {
			logger.Warn("Failed to load driver", "id", d.ID, "name", d.Name, "error", err)
			continue
		}
		loaded++
		logger.Info("Loaded driver", "id", d.ID, "name", d.Name)
	}
	logger.Info("Drivers loaded", "count", loaded)
	return nil
}

func registerShutdown(gracefulMgr *graceful.GracefulShutdown, collect *collector.Collector, northMgr *northbound.NorthboundManager, cfg *config.Config) {
	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping collector...")
		return collect.Stop()
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
