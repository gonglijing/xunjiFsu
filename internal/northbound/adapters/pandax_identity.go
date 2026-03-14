package adapters

import (
	"strconv"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *PandaXAdapter) defaultIdentity() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config == nil {
		return "", ""
	}
	return strings.TrimSpace(a.config.ProductKey), strings.TrimSpace(a.config.DeviceKey)
}

func (a *PandaXAdapter) resolveSubDeviceToken(data *models.CollectData) string {
	a.mu.RLock()
	cfg := a.config
	a.mu.RUnlock()

	if cfg == nil {
		if strings.TrimSpace(data.DeviceName) != "" {
			return strings.TrimSpace(data.DeviceName)
		}
		if strings.TrimSpace(data.DeviceKey) != "" {
			return strings.TrimSpace(data.DeviceKey)
		}
		return defaultDeviceToken(data.DeviceID)
	}

	pk := pickFirstNonEmpty2(data.ProductKey, cfg.ProductKey)
	name := pickFirstNonEmpty2(data.DeviceName, data.DeviceKey)
	dk := pickFirstNonEmpty2(data.DeviceKey, data.DeviceName)

	switch strings.ToLower(strings.TrimSpace(cfg.SubDeviceTokenMode)) {
	case "devicekey", "device_key":
		if dk != "" {
			return dk
		}
	case "product_devicekey", "product_device_key":
		if pk != "" && dk != "" {
			return pk + "_" + dk
		}
	case "product_devicename", "product_device_name":
		if pk != "" && name != "" {
			return pk + "_" + name
		}
	default:
		if name != "" {
			return name
		}
	}

	if dk != "" {
		return dk
	}
	if name != "" {
		return name
	}
	return defaultDeviceToken(data.DeviceID)
}

func (a *PandaXAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func (a *PandaXAdapter) nextID(prefix string) string {
	return nextPrefixedID(prefix, &a.seq)
}

func formatMetricFloat2(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}
