package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *DeviceExecAPI) GetDeviceWritables(w http.ResponseWriter, r *http.Request) {
	device, driverModel, ok := api.loadDeviceDriverForLookupRequest(w, r)
	if !ok {
		return
	}
	if device.DriverID == nil {
		WriteSuccess(w, []any{})
		return
	}

	writables, err := service.ParseDriverWritables(driverModel.ConfigSchema)
	if err != nil {
		WriteBadRequestDef(w, errDriverSchemaInvalid)
		return
	}
	WriteSuccess(w, writables)
}
