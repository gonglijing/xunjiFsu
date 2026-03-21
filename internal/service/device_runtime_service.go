package service

import (
	"cmp"
	"slices"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type DeviceRuntimeService struct {
	collector *collectorpkg.Collector
}

func NewDeviceRuntimeService(collector *collectorpkg.Collector) *DeviceRuntimeService {
	return &DeviceRuntimeService{collector: collector}
}

func (s *DeviceRuntimeService) ListDeviceRuntimeStatuses() ([]collectorpkg.DeviceRuntimeStatus, error) {
	devices, err := database.ListDevices()
	if err != nil {
		return nil, err
	}

	runtimeStatusMap := make(map[int64]collectorpkg.DeviceRuntimeStatus)
	if s.collector != nil {
		runtimeStatusMap = s.collector.ListDeviceRuntimeStatus()
	}

	return buildDeviceRuntimeStatusList(devices, runtimeStatusMap), nil
}

func (s *DeviceRuntimeService) LoadDeviceRuntimeStatus(id int64) (collectorpkg.DeviceRuntimeStatus, error) {
	if _, err := database.LoadDevice(id); err != nil {
		return collectorpkg.DeviceRuntimeStatus{}, err
	}

	status := collectorpkg.DeviceRuntimeStatus{
		DeviceID:   id,
		Registered: false,
	}
	if s.collector != nil {
		if runtimeStatus, exists := s.collector.GetDeviceRuntimeStatus(id); exists {
			status = runtimeStatus
		}
	}
	if status.DeviceID == 0 {
		status.DeviceID = id
	}
	return status, nil
}

func buildDeviceRuntimeStatusList(devices []*models.Device, runtimeStatusMap map[int64]collectorpkg.DeviceRuntimeStatus) []collectorpkg.DeviceRuntimeStatus {
	statuses := make([]collectorpkg.DeviceRuntimeStatus, 0, len(devices))
	for _, device := range devices {
		if device == nil {
			continue
		}

		status, ok := runtimeStatusMap[device.ID]
		if !ok {
			status = collectorpkg.DeviceRuntimeStatus{
				DeviceID:   device.ID,
				Registered: false,
			}
		}
		if status.DeviceID == 0 {
			status.DeviceID = device.ID
		}
		statuses = append(statuses, status)
	}

	slices.SortFunc(statuses, func(a, b collectorpkg.DeviceRuntimeStatus) int {
		return cmp.Compare(a.DeviceID, b.DeviceID)
	})
	return statuses
}
