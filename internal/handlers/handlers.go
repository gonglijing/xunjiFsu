package handlers

import (
	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

// Handler Web处理器
type Handler struct {
	authManager    *auth.JWTManager
	collector      *collector.Collector
	appConfig      *config.Config
	driverExecutor *driver.DriverExecutor
	driverManager  *driver.DriverManager
	northboundMgr  *northbound.NorthboundManager
	driversDir     string
}

// NewHandler 创建处理器
func NewHandler(
	authManager *auth.JWTManager,
	collector *collector.Collector,
	appConfig *config.Config,
	driverExecutor *driver.DriverExecutor,
	driverManager *driver.DriverManager,
	northboundMgr *northbound.NorthboundManager,
	driversDir string,
) *Handler {
	return &Handler{
		authManager:    authManager,
		collector:      collector,
		appConfig:      appConfig,
		driverExecutor: driverExecutor,
		driverManager:  driverManager,
		northboundMgr:  northboundMgr,
		driversDir:     driversDir,
	}
}
