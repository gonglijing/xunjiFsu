package app

import (
	"log/slog"

	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
	"github.com/gonglijing/xunjiFsu/internal/platform/config"
)

func applyNorthboundRuntimeConfig(cfg *config.Config, nm *northbound.NorthboundManager) {
	if cfg == nil || nm == nil {
		return
	}

	for _, name := range nm.ListRuntimeNames() {
		applyMQTTReconnectInterval(name, cfg, nm)
	}
}

func applyMQTTReconnectInterval(name string, cfg *config.Config, nm *northbound.NorthboundManager) {
	adapter, err := nm.GetAdapter(name)
	if err != nil || adapter == nil {
		return
	}

	mqttAdapter, ok := adapter.(*adapters.MQTTAdapter)
	if !ok {
		return
	}

	mqttAdapter.SetReconnectInterval(cfg.NorthboundMQTTReconnectInterval)
	slog.Info("Applied MQTT reconnect interval", "name", name, "interval", cfg.NorthboundMQTTReconnectInterval)
}
