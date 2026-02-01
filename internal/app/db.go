package app

import (
	"fmt"

	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

func initDatabases(cfg *config.Config) error {
	logger.Info("Initializing param database (persistent mode)...")
	if err := database.InitParamDBWithPath(cfg.ParamDBPath); err != nil {
		return fmt.Errorf("failed to initialize param database: %w", err)
	}

	logger.Info("Initializing data database (memory mode + batch sync)...")
	if err := database.InitDataDBWithPath(cfg.DataDBPath); err != nil {
		return fmt.Errorf("failed to initialize data database: %w", err)
	}

	return nil
}

func initSchemasAndDefaultData() error {
	logger.Info("Initializing param database schema...")
	if err := database.InitParamSchema(); err != nil {
		return fmt.Errorf("failed to initialize param schema: %w", err)
	}

	logger.Info("Initializing resource table...")
	if err := database.InitResourceTable(); err != nil {
		return fmt.Errorf("failed to initialize resource table: %w", err)
	}
	database.EnsureDeviceResourceColumn()

	logger.Info("Initializing storage policy table...")
	if err := database.InitStoragePolicyTable(); err != nil {
		return fmt.Errorf("failed to initialize storage policy table: %w", err)
	}

	logger.Info("Initializing device table...")
	if err := database.InitDeviceTable(); err != nil {
		return fmt.Errorf("failed to initialize device table: %w", err)
	}

	logger.Info("Initializing data database schema...")
	if err := database.InitDataSchema(); err != nil {
		return fmt.Errorf("failed to initialize data schema: %w", err)
	}

	logger.Info("Initializing default data...")
	if err := database.InitDefaultData(); err != nil {
		return fmt.Errorf("failed to initialize default data: %w", err)
	}

	return nil
}
