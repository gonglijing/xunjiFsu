package httpapi

import "net/http"

func (api *DeviceRuntimeAPI) GetDeviceRuntimeStatuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := api.service.ListDeviceRuntimeStatuses()
	if err != nil {
		writeServerErrorWithLog(w, errListDevicesFailed, err)
		return
	}
	WriteSuccess(w, statuses)
}

func (api *DeviceRuntimeAPI) GetDeviceRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	status, ok := api.loadDeviceRuntimeStatusByRequest(w, r)
	if !ok {
		return
	}

	WriteSuccess(w, status)
}
