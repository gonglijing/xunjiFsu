package service

import (
	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type DeviceService struct {
	collector *collectorpkg.Collector
}

func NewDeviceService(collector *collectorpkg.Collector) *DeviceService {
	return &DeviceService{collector: collector}
}

func (s *DeviceService) LoadDevice(id int64) (*models.Device, error) {
	return database.GetDeviceByID(id)
}

func (s *DeviceService) CreateDevice(device *models.Device) (*models.Device, error) {
	if device == nil {
		return nil, nil
	}

	id, err := database.CreateDevice(device)
	if err != nil {
		return nil, err
	}
	device.ID = id
	return device, nil
}

func (s *DeviceService) UpdateDevice(device *models.Device) (*models.Device, error) {
	if device == nil {
		return nil, nil
	}
	if err := database.UpdateDevice(device); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *DeviceService) DeleteDevice(id int64) error {
	return database.DeleteDevice(id)
}

func (s *DeviceService) ToggleDeviceEnabled(id int64) (int, error) {
	device, err := database.GetDeviceByID(id)
	if err != nil {
		return 0, err
	}

	nextState := nextDeviceEnabledState(device.Enabled)
	if err := database.UpdateDeviceEnabled(device.ID, nextState); err != nil {
		return 0, err
	}
	return nextState, nil
}

func nextDeviceEnabledState(enabled int) int {
	if enabled == 0 {
		return 1
	}
	return 0
}
