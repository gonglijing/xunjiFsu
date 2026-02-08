package app

import (
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

func applyRuntimeTuning(cfg *config.Config, collect *collector.Collector, driverExecutor *driver.DriverExecutor, northboundMgr *northbound.NorthboundManager) {
	if cfg == nil {
		return
	}

	if collect != nil {
		collect.SetRuntimeIntervals(cfg.CollectorDeviceSyncInterval, cfg.CollectorCommandPollInterval)
	}

	if driverExecutor != nil {
		driverExecutor.SetTimeouts(cfg.DriverSerialReadTimeout, cfg.DriverTCPDialTimeout, cfg.DriverTCPReadTimeout)
		driverExecutor.SetRetries(cfg.DriverSerialOpenRetries, cfg.DriverTCPDialRetries, cfg.DriverSerialOpenBackoff, cfg.DriverTCPDialBackoff)
	}

	if northboundMgr != nil {
		applyNorthboundRuntimeConfig(cfg, northboundMgr)
	}
}
