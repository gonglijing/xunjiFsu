package handlers

import (
	"fmt"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetDevices 获取所有设备
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListDevicesFailed, err)
		return
	}

	resources, _ := database.ListResources()
	resourceMap := make(map[int64]*models.Resource, len(resources))
	for _, res := range resources {
		if res == nil {
			continue
		}
		resourceMap[res.ID] = res
	}

	drivers, _ := database.GetAllDrivers()
	driverNameMap := buildDriverNameMap(drivers)

	for _, device := range devices {
		if device == nil {
			continue
		}
		if device.DriverID != nil {
			if name, ok := driverNameMap[*device.DriverID]; ok {
				device.DriverName = name
			} else {
				device.DriverName = fmt.Sprintf("驱动 #%d", *device.DriverID)
			}
		}
		if device.ResourceID != nil {
			if res, ok := resourceMap[*device.ResourceID]; ok {
				device.ResourceName = res.Name
				device.ResourceType = res.Type
				device.ResourcePath = res.Path
			}
		}
	}

	WriteSuccess(w, devices)
}

// CreateDevice 创建设备
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, errInvalidRequestBodyWithDetailPrefix+err.Error())
		return
	}
	if err := normalizeDeviceInput(&device); err != nil {
		WriteBadRequestDef(w, apiErrDeviceNameRequired)
		return
	}

	id, err := database.CreateDevice(&device)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateDeviceFailed, err)
		return
	}

	device.ID = id
	WriteCreated(w, device)
}

// UpdateDevice 更新设备
func (h *Handler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if _, err := database.GetDeviceByID(id); err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return
	}

	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, errInvalidRequestBodyWithDetailPrefix+err.Error())
		return
	}
	if err := normalizeDeviceInput(&device); err != nil {
		WriteBadRequestDef(w, apiErrDeviceNameRequired)
		return
	}

	device.ID = id
	if err := database.UpdateDevice(&device); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateDeviceFailed, err)
		return
	}

	WriteSuccess(w, device)
}

// DeleteDevice 删除设备
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if _, err := database.GetDeviceByID(id); err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return
	}

	if err := database.DeleteDevice(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteDeviceFailed, err)
		return
	}

	WriteDeleted(w)
}

// ToggleDeviceEnable 切换设备使能状态
func (h *Handler) ToggleDeviceEnable(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return
	}

	nextState := 0
	if device.Enabled == 0 {
		nextState = 1
	}

	if err := database.UpdateDeviceEnabled(id, nextState); err != nil {
		writeServerErrorWithLog(w, apiErrToggleDeviceFailed, err)
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": nextState,
	})
}
