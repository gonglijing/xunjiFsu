package handlers

import (
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func summarizeDeviceStats(devices []*models.Device) DeviceStats {
	stats := DeviceStats{Total: len(devices)}
	for _, device := range devices {
		if device != nil && device.Enabled == 1 {
			stats.Enabled++
		}
	}
	return stats
}

func summarizeNorthboundStats(configs []*models.NorthboundConfig) NorthboundStats {
	stats := NorthboundStats{Total: len(configs)}
	for _, config := range configs {
		if config != nil && config.Enabled == 1 {
			stats.Enabled++
		}
	}
	return stats
}

func summarizeAlarmStats(alarms []*models.AlarmLog, now time.Time) AlarmStats {
	stats := AlarmStats{Total: len(alarms)}
	today := now.Truncate(24 * time.Hour)
	for _, alarm := range alarms {
		if alarm == nil {
			continue
		}
		if alarm.Acknowledged == 0 {
			stats.Unacked++
		}
		if alarm.TriggeredAt.After(today) {
			stats.Today++
		}
	}
	return stats
}
