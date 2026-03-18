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

	applyCollectorRuntimeTuning(cfg, collect)
	applyDriverRuntimeTuning(cfg, driverExecutor)
	applyNorthboundRuntimeTuning(cfg, northboundMgr)
}

func applyCollectorRuntimeTuning(cfg *config.Config, collect *collector.Collector) {
	if cfg == nil || collect == nil {
		return
	}

	collect.SetRuntimeIntervals(cfg.CollectorDeviceSyncInterval, cfg.CollectorCommandPollInterval)
	collect.SetMaxConcurrentCollects(cfg.CollectorWorkers)
}

func applyDriverRuntimeTuning(cfg *config.Config, driverExecutor *driver.DriverExecutor) {
	if cfg == nil || driverExecutor == nil {
		return
	}

	driverExecutor.SetTimeouts(cfg.DriverSerialReadTimeout, cfg.DriverTCPDialTimeout, cfg.DriverTCPReadTimeout)
	driverExecutor.SetRetries(cfg.DriverSerialOpenRetries, cfg.DriverTCPDialRetries, cfg.DriverSerialOpenBackoff, cfg.DriverTCPDialBackoff)
}

func applyNorthboundRuntimeTuning(cfg *config.Config, northboundMgr *northbound.NorthboundManager) {
	if cfg == nil || northboundMgr == nil {
		return
	}

	applyNorthboundRuntimeConfig(cfg, northboundMgr)
}
