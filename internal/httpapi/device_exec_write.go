package httpapi

import (
	"errors"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/driver"
)

func (api *DeviceExecAPI) ExecuteDriverFunction(w http.ResponseWriter, r *http.Request) {
	payload, ok := parseExecuteDriverPayload(w, r)
	if !ok {
		return
	}

	device, _, ok := api.loadDeviceDriverForExecutionRequest(w, r)
	if !ok {
		return
	}

	driverID := *device.DriverID
	_, pluginFunc, configFunc := normalizeExecuteFunction(payload.Function)

	config, err := buildExecuteDriverConfig(payload.Params, device, configFunc)
	if err != nil {
		WriteBadRequestCode(w, errExecuteDriverParamFail.Code, errExecuteDriverParamFail.Message+": "+err.Error())
		return
	}

	ctx := buildExecuteDriverContext(device, config)
	result, err := api.service.ExecuteDriverFunction(driverID, pluginFunc, ctx)
	if err != nil {
		if errors.Is(err, driver.ErrDriverNotFound) {
			WriteBadRequestDef(w, errDriverNotLoaded)
			return
		}
		writeServerErrorWithLog(w, errExecuteDriverFailed, err)
		return
	}

	WriteSuccess(w, result)
}
