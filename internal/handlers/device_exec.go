package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
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
	driverID := *device.DriverID

	driverModel, err := database.GetDriverByID(driverID)
	if err != nil {
		WriteServerErrorDef(w, apiErrDriverLookupFailed)
		return
	}
	if err := h.ensureDriverLoaded(driverID, driverModel); err != nil {
		writeServerErrorWithLog(w, apiErrLoadDriverFailed, err)
		return
	}

	config, err := buildExecuteDriverConfig(req.Params, device, configFunc)
	if err != nil {
		WriteBadRequestCode(w, apiErrExecuteDriverParamInvalid.Code, apiErrExecuteDriverParamInvalid.Message+": "+err.Error())
		return
	}

	ctx := buildExecuteDriverContext(device, config)

	result, err := h.driverManager.ExecuteDriver(driverID, pluginFunc, ctx)
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

func (h *Handler) ensureDriverLoaded(driverID int64, driverModel *models.Driver) error {
	if h.driverManager.IsLoaded(driverID) {
		return nil
	}
	return h.driverManager.LoadDriverFromModel(driverModel, 0)
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
