package app

import (
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

func applyNorthboundRuntimeConfig(cfg *config.Config, nm *northbound.NorthboundManager) {
	if cfg == nil || nm == nil {
		return
		return
	}

	for _, name := range nm.ListRuntimeNames() {
		adapter, err := nm.GetAdapter(name)
		if err != nil || adapter == nil {
			continue
		}
		mqttAdapter, ok := adapter.(*adapters.MQTTAdapter)
		if !ok {
			continue
		}
		mqttAdapter.SetReconnectInterval(cfg.NorthboundMQTTReconnectInterval)
		logger.Info("Applied MQTT reconnect interval", "name", name, "interval", cfg.NorthboundMQTTReconnectInterval)
	}
}
