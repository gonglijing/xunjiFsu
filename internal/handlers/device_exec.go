package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
)

func (h *Handler) ExecuteDriverFunction(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	var req struct {
		Function string                 `json:"function"`
		Params   map[string]interface{} `json:"params"`
	}
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return
	}

	_, pluginFunc, configFunc := normalizeExecuteFunction(req.Function)

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return
	}
	if device.DriverID == nil {
		WriteBadRequestDef(w, apiErrDeviceHasNoDriver)
		return
	}

	driverModel, err := database.GetDriverByID(*device.DriverID)
	if err != nil {
		WriteServerErrorDef(w, apiErrDriverLookupFailed)
		return
	}
	if !h.driverManager.IsLoaded(*device.DriverID) {
		if err := h.driverManager.LoadDriverFromModel(driverModel, 0); err != nil {
			writeServerErrorWithLog(w, apiErrLoadDriverFailed, err)
			return
		}
	}

	config := make(map[string]string, len(req.Params)+2)
	for key, value := range req.Params {
		config[key] = stringifyParamValue(value)
	}
	config["device_address"] = device.DeviceAddress
	config["func_name"] = configFunc
	enrichExecuteIdentity(config, device)
	if configFunc == "write" {
		if err := normalizeWriteParams(config, req.Params); err != nil {
			WriteBadRequestCode(w, apiErrExecuteDriverParamInvalid.Code, apiErrExecuteDriverParamInvalid.Message+": "+err.Error())
			return
		}
	}

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
			WriteBadRequestDef(w, apiErrDriverNotLoaded)
			return
		}
		writeServerErrorWithLog(w, apiErrExecuteDriverFailed, err)
		return
	}

	WriteSuccess(w, result)
}

// GetDeviceWritables 返回设备驱动声明的可写寄存器元数据（来自 Driver.ConfigSchema.writable）
func (h *Handler) GetDeviceWritables(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return
	}
	if device.DriverID == nil {
		WriteSuccess(w, []interface{}{})
		return
	}

	driverModel, err := database.GetDriverByID(*device.DriverID)
	if err != nil {
		WriteServerErrorDef(w, apiErrDriverLookupFailed)
		return
	}

	var cfg struct {
		Writable []interface{} `json:"writable"`
	}
	if driverModel.ConfigSchema != "" {
		if err := json.Unmarshal([]byte(driverModel.ConfigSchema), &cfg); err != nil {
			WriteBadRequestDef(w, apiErrDriverConfigSchemaInvalid)
			return
		}
	}

	WriteSuccess(w, cfg.Writable)
}
