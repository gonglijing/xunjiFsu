package app

import (
	"fmt"
	"log/slog"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/platform/config"
)

func initDatabases(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	applyDatabaseRuntimeConfig(cfg)
	if err := initParamDatabase(cfg); err != nil {
		return err
	}
	if err := initDataDatabase(cfg); err != nil {
		return err
	}

	return nil
}

func applyDatabaseRuntimeConfig(cfg *config.Config) {
	if cfg == nil {
		return
	}

	database.ApplyRuntimeLimits(cfg.MaxDataPoints, cfg.MaxDataCache)
	database.ApplySyncInterval(cfg.SyncInterval)
}

func initParamDatabase(cfg *config.Config) error {
	slog.Info("Initializing param database (persistent mode)...")
	if err := database.InitParamDBWithPath(cfg.ParamDBPath); err != nil {
		return fmt.Errorf("failed to initialize param database: %w", err)
	}
	return nil
}

func initDataDatabase(cfg *config.Config) error {
	slog.Info("Initializing data database (memory mode + batch sync)...")
	if err := database.InitDataDBWithPath(cfg.DataDBPath); err != nil {
		return fmt.Errorf("failed to initialize data database: %w", err)
	}
	return nil
}

func initSchemasAndDefaultData() error {
	if err := initParamDatabaseSchema(); err != nil {
		return err
	}
	if err := initGatewayDatabaseTables(); err != nil {
		return err
	}
	if err := initDeviceDatabaseTables(); err != nil {
		return err
	}
	if err := initDataDatabaseSchema(); err != nil {
		return err
	}
	if err := initDefaultGatewayData(); err != nil {
		return err
	}

	return nil
}

func initParamDatabaseSchema() error {
	slog.Info("Initializing param database schema...")
	if err := database.InitParamSchema(); err != nil {
		return fmt.Errorf("failed to initialize param schema: %w", err)
	}
	return nil
}

func initGatewayDatabaseTables() error {
	slog.Info("Initializing resource table...")
	if err := database.InitResourceTable(); err != nil {
		return fmt.Errorf("failed to initialize resource table: %w", err)
	}
	database.EnsureDeviceResourceColumn()

	slog.Info("Initializing gateway config table...")
	if err := database.InitGatewayConfigTable(); err != nil {
		return fmt.Errorf("failed to initialize gateway config table: %w", err)
	}

	slog.Info("Initializing runtime config audit table...")
	if err := database.InitRuntimeConfigAuditTable(); err != nil {
		return fmt.Errorf("failed to initialize runtime config audit table: %w", err)
	}
	return nil
}

func initDeviceDatabaseTables() error {
	slog.Info("Initializing device table...")
	if err := database.InitDeviceTable(); err != nil {
		return fmt.Errorf("failed to initialize device table: %w", err)
	}
	return nil
}

func initDataDatabaseSchema() error {
	slog.Info("Initializing data database schema...")
	if err := database.InitDataSchema(); err != nil {
		return fmt.Errorf("failed to initialize data schema: %w", err)
	}
	return nil
}

func initDefaultGatewayData() error {
	slog.Info("Initializing default data...")
	if err := database.InitDefaultData(); err != nil {
		return fmt.Errorf("failed to initialize default data: %w", err)
	}
	return nil
}
