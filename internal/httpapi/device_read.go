package httpapi

import "net/http"

var (
	errListDevicesFailed = APIErrorDef{Code: "E_LIST_DEVICES_FAILED", Message: "获取设备列表失败"}
)

func (api *DeviceAPI) ListDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := api.service.ListDevices()
	if err != nil {
		writeServerErrorWithLog(w, errListDevicesFailed, err)
		return
	}
	WriteSuccess(w, devices)
}
