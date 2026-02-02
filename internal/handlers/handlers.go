package handlers

import (
	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

// Handler Web处理器
type Handler struct {
	authManager   *auth.JWTManager
	collector     *collector.Collector
	driverManager *driver.DriverManager
	northboundMgr *northbound.NorthboundManager
}

// NewHandler 创建处理器
func NewHandler(
	authManager *auth.JWTManager,
	collector *collector.Collector,
	driverManager *driver.DriverManager,
	northboundMgr *northbound.NorthboundManager,
) *Handler {
	return &Handler{
		authManager:   authManager,
		collector:     collector,
		driverManager: driverManager,
		northboundMgr: northboundMgr,
	}
}
