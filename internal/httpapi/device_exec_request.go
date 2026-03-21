package httpapi

import (
	"errors"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func parseExecuteDriverPayload(w http.ResponseWriter, r *http.Request) (*executeDriverPayload, bool) {
	var payload executeDriverPayload
	if err := ParseRequest(r, &payload); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	return &payload, true
}

func (api *DeviceExecAPI) loadDeviceDriverForExecutionRequest(w http.ResponseWriter, r *http.Request) (*models.Device, *models.Driver, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, nil, false
	}
	return api.loadDeviceDriverForExecution(w, id)
}

func (api *DeviceExecAPI) loadDeviceDriverForLookupRequest(w http.ResponseWriter, r *http.Request) (*models.Device, *models.Driver, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, nil, false
	}
	return api.loadDeviceDriverForLookup(w, id)
}

func (api *DeviceExecAPI) loadDeviceDriverForExecution(w http.ResponseWriter, id int64) (*models.Device, *models.Driver, bool) {
	device, driverModel, err := api.service.LoadDeviceDriverForExecution(id)
	if err != nil {
		if errors.Is(err, driver.ErrDriverNotFound) {
			writeServerErrorWithLog(w, errLoadDriverFailed, err)
			return nil, nil, false
		}
		if err.Error() == "device has no driver" {
			WriteBadRequestDef(w, errDeviceHasNoDriver)
			return nil, nil, false
		}
		WriteNotFoundDef(w, errDeviceNotFound)
		return nil, nil, false
	}
	if device.DriverID == nil {
		WriteBadRequestDef(w, errDeviceHasNoDriver)
		return nil, nil, false
	}
	return device, driverModel, true
}

func (api *DeviceExecAPI) loadDeviceDriverForLookup(w http.ResponseWriter, id int64) (*models.Device, *models.Driver, bool) {
	device, driverModel, err := api.service.LoadDeviceDriverForLookup(id)
	if err != nil {
		WriteNotFoundDef(w, errDeviceNotFound)
		return nil, nil, false
	}
	return device, driverModel, true
}
