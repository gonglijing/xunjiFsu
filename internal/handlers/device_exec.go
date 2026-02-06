package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
)

func (h *Handler) ExecuteDriverFunction(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	var req struct {
		Function string                 `json:"function"`
		Params   map[string]interface{} `json:"params"`
	}
	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	requestFunc, pluginFunc, configFunc := normalizeExecuteFunction(req.Function)

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFound(w, "Device not found")
		return
	}
	if device.DriverID == nil {
		WriteBadRequest(w, "Device has no driver")
		return
	}

	driverModel, err := database.GetDriverByID(*device.DriverID)
	if err != nil {
		WriteServerError(w, "driver not found")
		return
	}
	if !h.driverManager.IsLoaded(*device.DriverID) {
		if err := h.driverManager.LoadDriverFromModel(driverModel, 0); err != nil {
			WriteServerError(w, "driver load failed: "+err.Error())
			return
		}
	}

	config := make(map[string]string, len(req.Params)+2)
	for key, value := range req.Params {
		config[key] = stringifyParamValue(value)
	}
	config["device_address"] = device.DeviceAddress
	config["func_name"] = configFunc

	resourceID := int64(0)
	if device.ResourceID != nil {
		resourceID = *device.ResourceID
	}
	ctx := &driver.DriverContext{
		DeviceID:     device.ID,
		DeviceName:   device.Name,
		ResourceID:   resourceID,
		ResourceType: inferDeviceResourceType(device),
		Config:       config,
		DeviceConfig: "",
	}

	result, err := h.driverManager.ExecuteDriver(*device.DriverID, pluginFunc, ctx)
	if err != nil {
		if errors.Is(err, driver.ErrDriverNotFound) {
			WriteBadRequest(w, "driver is not loaded")
			return
		}
		WriteServerError(w, fmt.Sprintf("Failed to execute %s: %v", requestFunc, err))
		return
	}

	WriteSuccess(w, result)
}

// GetDeviceWritables 返回设备驱动声明的可写寄存器元数据（来自 Driver.ConfigSchema.writable）
func (h *Handler) GetDeviceWritables(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFound(w, "Device not found")
		return
	}
	if device.DriverID == nil {
		WriteSuccess(w, []interface{}{})
		return
	}

	driverModel, err := database.GetDriverByID(*device.DriverID)
	if err != nil {
		WriteServerError(w, "driver not found")
		return
	}

	var cfg struct {
		Writable []interface{} `json:"writable"`
	}
	if driverModel.ConfigSchema != "" {
		if err := json.Unmarshal([]byte(driverModel.ConfigSchema), &cfg); err != nil {
			WriteBadRequest(w, "driver config_schema is invalid JSON")
			return
		}
	}

	WriteSuccess(w, cfg.Writable)
}
