package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type executeDriverPayload struct {
	Function string                 `json:"function"`
	Params   map[string]interface{} `json:"params"`
}

func (h *Handler) ExecuteDriverFunction(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	payload, ok := parseExecuteDriverPayload(w, r)
	if !ok {
		return
	}
	device, _, ok := h.loadDeviceDriverForExecution(w, id)
	if !ok {
		return
	}
	driverID := *device.DriverID
	_, pluginFunc, configFunc := normalizeExecuteFunction(payload.Function)

	config, err := buildExecuteDriverConfig(payload.Params, device, configFunc)
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

	device, driverModel, ok := h.loadDeviceDriverForLookup(w, id)
	if !ok {
		return
	}
	if device.DriverID == nil {
		WriteSuccess(w, []interface{}{})
		return
	}

	writables, err := parseDriverWritables(driverModel.ConfigSchema)
	if err != nil {
		WriteBadRequestDef(w, apiErrDriverConfigSchemaInvalid)
		return
	}

	WriteSuccess(w, writables)
}

func parseExecuteDriverPayload(w http.ResponseWriter, r *http.Request) (*executeDriverPayload, bool) {
	var payload executeDriverPayload
	if !parseRequestOrWriteBadRequestDefault(w, r, &payload) {
		return nil, false
	}
	return &payload, true
}

func (h *Handler) loadDeviceDriverForExecution(w http.ResponseWriter, id int64) (*models.Device, *models.Driver, bool) {
	device, ok := loadDeviceByIDOrWriteNotFound(w, id)
	if !ok {
		return nil, nil, false
	}
	if device.DriverID == nil {
		WriteBadRequestDef(w, apiErrDeviceHasNoDriver)
		return nil, nil, false
	}

	driverModel, ok := loadDriverByIDOrWriteLookupError(w, *device.DriverID)
	if !ok {
		return nil, nil, false
	}
	if err := h.ensureDriverLoaded(*device.DriverID, driverModel); err != nil {
		writeServerErrorWithLog(w, apiErrLoadDriverFailed, err)
		return nil, nil, false
	}

	return device, driverModel, true
}

func (h *Handler) loadDeviceDriverForLookup(w http.ResponseWriter, id int64) (*models.Device, *models.Driver, bool) {
	device, ok := loadDeviceByIDOrWriteNotFound(w, id)
	if !ok {
		return nil, nil, false
	}
	if device.DriverID == nil {
		return device, nil, true
	}

	driverModel, ok := loadDriverByIDOrWriteLookupError(w, *device.DriverID)
	if !ok {
		return nil, nil, false
	}

	return device, driverModel, true
}

func loadDeviceByIDOrWriteNotFound(w http.ResponseWriter, id int64) (*models.Device, bool) {
	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrDeviceNotFound)
		return nil, false
	}
	return device, true
}

func loadDriverByIDOrWriteLookupError(w http.ResponseWriter, driverID int64) (*models.Driver, bool) {
	driverModel, err := database.GetDriverByID(driverID)
	if err != nil {
		WriteServerErrorDef(w, apiErrDriverLookupFailed)
		return nil, false
	}
	return driverModel, true
}

func parseDriverWritables(configSchema string) ([]interface{}, error) {
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
