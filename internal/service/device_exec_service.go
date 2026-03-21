package service

import (
	"encoding/json"
	"errors"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type DeviceDriverExecutor interface {
	ExecuteDriver(id int64, function string, driverCtx *driver.DriverContext) (*driver.DriverResult, error)
	IsLoaded(id int64) bool
	LoadDriverFromModel(driver *models.Driver, resourceID int64) error
}

type DeviceExecService struct {
	driverManager DeviceDriverExecutor
}

func NewDeviceExecService(driverManager DeviceDriverExecutor) *DeviceExecService {
	return &DeviceExecService{driverManager: driverManager}
}

func (s *DeviceExecService) EnsureDriverLoaded(driverID int64, driverModel *models.Driver) error {
	if s.driverManager == nil {
		return driver.ErrDriverNotFound
	}
	if s.driverManager.IsLoaded(driverID) {
		return nil
	}
	return s.driverManager.LoadDriverFromModel(driverModel, 0)
}

func (s *DeviceExecService) LoadDeviceDriverForExecution(id int64) (*models.Device, *models.Driver, error) {
	device, err := database.LoadDevice(id)
	if err != nil {
		return nil, nil, err
	}
	if device.DriverID == nil {
		return nil, nil, errors.New("device has no driver")
	}

	driverModel, err := database.LoadDriver(*device.DriverID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.EnsureDriverLoaded(*device.DriverID, driverModel); err != nil {
		return nil, nil, err
	}

	return device, driverModel, nil
}

func (s *DeviceExecService) LoadDeviceDriverForLookup(id int64) (*models.Device, *models.Driver, error) {
	device, err := database.LoadDevice(id)
	if err != nil {
		return nil, nil, err
	}
	if device.DriverID == nil {
		return device, nil, nil
	}

	driverModel, err := database.LoadDriver(*device.DriverID)
	if err != nil {
		return nil, nil, err
	}
	return device, driverModel, nil
}

func (s *DeviceExecService) ExecuteDriverFunction(driverID int64, pluginFunc string, ctx *driver.DriverContext) (*driver.DriverResult, error) {
	if s.driverManager == nil {
		return nil, driver.ErrDriverNotFound
	}
	return s.driverManager.ExecuteDriver(driverID, pluginFunc, ctx)
}

func ParseDriverWritables(configSchema string) ([]interface{}, error) {
	if configSchema == "" {
		return nil, nil
	}

	var cfg struct {
		Writable []interface{} `json:"writable"`
	}
	if err := json.Unmarshal([]byte(configSchema), &cfg); err != nil {
		return nil, err
	}
	return cfg.Writable, nil
}
