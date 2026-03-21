package service

import (
	"fmt"
	"path/filepath"
	"strings"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type DeviceListItem struct {
	*models.Device
	CollectRuntime collectorpkg.DeviceRuntimeStatus `json:"collect_runtime"`
}

func (s *DeviceService) ListDevices() ([]*DeviceListItem, error) {
	devices, err := database.GetAllDevices()
	if err != nil {
		return nil, err
	}

	resources, _ := database.ListResources()
	drivers, _ := database.GetAllDrivers()
	resourceMap := buildResourceMap(resources)
	driverNameMap := buildDriverNameMap(drivers)
	runtimeStatusMap := s.deviceRuntimeStatusMap()
	deviceList := make([]*DeviceListItem, 0, len(devices))

	for _, device := range devices {
		if device == nil {
			continue
		}
		enrichDeviceDisplay(device, driverNameMap, resourceMap)
		deviceList = append(deviceList, buildDeviceListItem(device, runtimeStatusMap))
	}

	return deviceList, nil
}

func (s *DeviceService) deviceRuntimeStatusMap() map[int64]collectorpkg.DeviceRuntimeStatus {
	if s == nil || s.collector == nil {
		return make(map[int64]collectorpkg.DeviceRuntimeStatus)
	}
	return s.collector.ListDeviceRuntimeStatus()
}

func resolveDriverDisplayName(driverModel *models.Driver) string {
	if driverModel == nil {
		return ""
	}
	if driverModel.FilePath != "" {
		name := filepath.Base(driverModel.FilePath)
		return strings.TrimSuffix(name, ".wasm")
	}
	return strings.TrimSpace(driverModel.Name)
}

func buildDriverNameMap(drivers []*models.Driver) map[int64]string {
	nameMap := make(map[int64]string, len(drivers))
	for _, driverModel := range drivers {
		if driverModel == nil {
			continue
		}
		if name := resolveDriverDisplayName(driverModel); name != "" {
			nameMap[driverModel.ID] = name
		}
	}
	return nameMap
}

func buildResourceMap(resources []*models.Resource) map[int64]*models.Resource {
	resourceMap := make(map[int64]*models.Resource, len(resources))
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		resourceMap[resource.ID] = resource
	}
	return resourceMap
}

func enrichDeviceDisplay(device *models.Device, driverNameMap map[int64]string, resourceMap map[int64]*models.Resource) {
	if device == nil {
		return
	}

	if device.DriverID != nil {
		if name, ok := driverNameMap[*device.DriverID]; ok {
			device.DriverName = name
		} else {
			device.DriverName = fmt.Sprintf("驱动 #%d", *device.DriverID)
		}
	}

	if device.ResourceID == nil {
		return
	}

	resource, ok := resourceMap[*device.ResourceID]
	if !ok || resource == nil {
		return
	}

	device.ResourceName = resource.Name
	device.ResourceType = resource.Type
	device.ResourcePath = resource.Path
}

func buildDeviceListItem(device *models.Device, runtimeStatusMap map[int64]collectorpkg.DeviceRuntimeStatus) *DeviceListItem {
	if device == nil {
		return nil
	}

	item := &DeviceListItem{Device: device}
	if status, ok := runtimeStatusMap[device.ID]; ok {
		item.CollectRuntime = status
	} else {
		item.CollectRuntime = collectorpkg.DeviceRuntimeStatus{
			DeviceID:   device.ID,
			Registered: false,
		}
	}
	if item.CollectRuntime.DeviceID == 0 {
		item.CollectRuntime.DeviceID = device.ID
	}
	return item
}
