package handlers

import (
	"net/http"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type deviceListItem struct {
	*models.Device
	CollectRuntime collectorpkg.DeviceRuntimeStatus `json:"collect_runtime"`
}

// GetDevices 获取所有设备
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListDevicesFailed, err)
		return
	}

	resources, _ := database.ListResources()
	drivers, _ := database.GetAllDrivers()
	resourceMap := buildResourceMap(resources)
	driverNameMap := buildDriverNameMap(drivers)
	runtimeStatusMap := make(map[int64]collectorpkg.DeviceRuntimeStatus)
	if h.collector != nil {
		runtimeStatusMap = h.collector.ListDeviceRuntimeStatus()
	}
	deviceList := make([]*deviceListItem, 0, len(devices))

	for _, device := range devices {
		if device == nil {
			continue
		}
		enrichDeviceDisplay(device, driverNameMap, resourceMap)
		deviceList = append(deviceList, buildDeviceListItem(device, runtimeStatusMap))
	}

	WriteSuccess(w, deviceList)
}

// CreateDevice 创建设备
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	device, ok := parseAndNormalizeDevice(w, r)
	if !ok {
		return
	}

	id, err := database.CreateDevice(device)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateDeviceFailed, err)
		return
	}

	device.ID = id
	WriteCreated(w, device)
}

// UpdateDevice 更新设备
func (h *Handler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	device, ok := loadDeviceOrWriteNotFound(w, r)
	if !ok {
		return
	}

	payload, ok := parseAndNormalizeDevice(w, r)
	if !ok {
		return
	}

	payload.ID = device.ID
	if err := database.UpdateDevice(payload); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateDeviceFailed, err)
		return
	}

	WriteSuccess(w, payload)
}

// DeleteDevice 删除设备
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	device, ok := loadDeviceOrWriteNotFound(w, r)
	if !ok {
		return
	}

	if err := database.DeleteDevice(device.ID); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteDeviceFailed, err)
		return
	}

	WriteDeleted(w)
}

// ToggleDeviceEnable 切换设备使能状态
func (h *Handler) ToggleDeviceEnable(w http.ResponseWriter, r *http.Request) {
	device, ok := loadDeviceOrWriteNotFound(w, r)
	if !ok {
		return
	}

	nextState := nextDeviceEnabledState(device.Enabled)

	if err := database.UpdateDeviceEnabled(device.ID, nextState); err != nil {
		writeServerErrorWithLog(w, apiErrToggleDeviceFailed, err)
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": nextState,
	})
}

func parseAndNormalizeDevice(w http.ResponseWriter, r *http.Request) (*models.Device, bool) {
	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, errInvalidRequestBodyWithDetailPrefix+err.Error())
		return nil, false
	}
	if err := normalizeDeviceInput(&device); err != nil {
		WriteBadRequestDef(w, apiErrDeviceNameRequired)
		return nil, false
	}
	return &device, true
}

func loadDeviceOrWriteNotFound(w http.ResponseWriter, r *http.Request) (*models.Device, bool) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return nil, false
	}

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return nil, false
	}

	return device, true
}

func nextDeviceEnabledState(enabled int) int {
	if enabled == 0 {
		return 1
	}
	return 0
}

func buildDeviceListItem(device *models.Device, runtimeStatusMap map[int64]collectorpkg.DeviceRuntimeStatus) *deviceListItem {
	if device == nil {
		return nil
	}

	item := &deviceListItem{Device: device}
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
