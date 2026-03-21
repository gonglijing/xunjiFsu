package service

import (
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type StatusData struct {
	CollectorRunning bool            `json:"collector_running"`
	Devices          DeviceStats     `json:"devices"`
	Northbound       NorthboundStats `json:"northbound"`
	Alarms           AlarmStats      `json:"alarms"`
	Drivers          DriverStats     `json:"drivers"`
	Timestamp        time.Time       `json:"timestamp"`
}

type DeviceStats struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
}

type NorthboundStats struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
}

type AlarmStats struct {
	Total   int `json:"total"`
	Unacked int `json:"unacked"`
	Today   int `json:"today"`
}

type DriverStats struct {
	Total int `json:"total"`
}

type CollectorController interface {
	Start() error
	Stop() error
	IsRunning() bool
}

type StatusService struct {
	collector   CollectorController
	driverCount func() int
}

func NewStatusService(collector CollectorController, driverCount func() int) *StatusService {
	return &StatusService{collector: collector, driverCount: driverCount}
}

func (s *StatusService) LoadStatus(now time.Time) (StatusData, error) {
	devices, err := database.GetAllDevices()
	if err != nil {
		return StatusData{}, err
	}
	configs, _ := database.GetAllNorthboundConfigs()
	alarms, _ := database.GetRecentAlarmLogs(1000)

	driverCount := 0
	if s.driverCount != nil {
		driverCount = s.driverCount()
	}
	collectorRunning := false
	if s.collector != nil {
		collectorRunning = s.collector.IsRunning()
	}

	return BuildStatusData(devices, configs, alarms, driverCount, collectorRunning, now), nil
}

func (s *StatusService) StartCollector() error {
	if s.collector == nil {
		return nil
	}
	return s.collector.Start()
}

func (s *StatusService) StopCollector() {
	if s.collector != nil {
		_ = s.collector.Stop()
	}
}

func BuildDeviceStats(devices []*models.Device) DeviceStats {
	stats := DeviceStats{Total: len(devices)}
	for _, device := range devices {
		if device != nil && device.Enabled == 1 {
			stats.Enabled++
		}
	}
	return stats
}

func BuildNorthboundStats(configs []*models.NorthboundConfig) NorthboundStats {
	stats := NorthboundStats{Total: len(configs)}
	for _, config := range configs {
		if config != nil && config.Enabled == 1 {
			stats.Enabled++
		}
	}
	return stats
}

func BuildAlarmStats(alarms []*models.AlarmLog, now time.Time) AlarmStats {
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

func BuildStatusData(
	devices []*models.Device,
	configs []*models.NorthboundConfig,
	alarms []*models.AlarmLog,
	driverCount int,
	collectorRunning bool,
	now time.Time,
) StatusData {
	return StatusData{
		CollectorRunning: collectorRunning,
		Devices:          BuildDeviceStats(devices),
		Northbound:       BuildNorthboundStats(configs),
		Alarms:           BuildAlarmStats(alarms, now),
		Drivers: DriverStats{
			Total: driverCount,
		},
		Timestamp: now,
	}
}

func BuildCollectorStatusResponse(status string) map[string]string {
	return map[string]string{"status": status}
}
