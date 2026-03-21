package httpapi

import (
	"net/http"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
)

func (api *DeviceRuntimeAPI) loadDeviceRuntimeStatusByRequest(w http.ResponseWriter, r *http.Request) (collectorpkg.DeviceRuntimeStatus, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, errInvalidID)
	if !ok {
		return collectorpkg.DeviceRuntimeStatus{}, false
	}

	status, err := api.service.LoadDeviceRuntimeStatus(id)
	if err != nil {
		WriteNotFoundDef(w, errDeviceNotFound)
		return collectorpkg.DeviceRuntimeStatus{}, false
	}
	return status, true
}
