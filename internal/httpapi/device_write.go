package httpapi

import (
	"net/http"
)

const errInvalidRequestBodyWithDetailPrefix = "Invalid request body: "

var (
	errInvalidID          = APIErrorDef{Code: "E_INVALID_ID", Message: "Invalid ID"}
	errDeviceNotFound     = APIErrorDef{Code: "E_DEVICE_NOT_FOUND", Message: "Device not found"}
	errDeviceNameRequired = APIErrorDef{Code: "E_DEVICE_NAME_REQUIRED", Message: "device name is required"}
	errCreateDeviceFailed = APIErrorDef{Code: "E_CREATE_DEVICE_FAILED", Message: "创建设备失败"}
	errUpdateDeviceFailed = APIErrorDef{Code: "E_UPDATE_DEVICE_FAILED", Message: "更新设备失败"}
	errDeleteDeviceFailed = APIErrorDef{Code: "E_DELETE_DEVICE_FAILED", Message: "删除设备失败"}
)

func (api *DeviceAPI) CreateDevice(w http.ResponseWriter, r *http.Request) {
	device, ok := parseAndNormalizeDevice(w, r)
	if !ok {
		return
	}

	device, err := api.service.CreateDevice(device)
	if err != nil {
		writeServerErrorWithLog(w, errCreateDeviceFailed, err)
		return
	}
	WriteCreated(w, device)
}

func (api *DeviceAPI) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	deviceModel, ok := api.loadDeviceByRequest(w, r)
	if !ok {
		return
	}

	device, ok := parseAndNormalizeDevice(w, r)
	if !ok {
		return
	}
	device.ID = deviceModel.ID

	device, err := api.service.UpdateDevice(device)
	if err != nil {
		writeServerErrorWithLog(w, errUpdateDeviceFailed, err)
		return
	}
	WriteSuccess(w, device)
}

func (api *DeviceAPI) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	device, ok := api.loadDeviceByRequest(w, r)
	if !ok {
		return
	}
	if err := api.service.DeleteDevice(device.ID); err != nil {
		writeServerErrorWithLog(w, errDeleteDeviceFailed, err)
		return
	}
	WriteDeleted(w)
}

func (api *DeviceAPI) ToggleDeviceEnabled(w http.ResponseWriter, r *http.Request) {
	device, ok := api.loadDeviceByRequest(w, r)
	if !ok {
		return
	}

	nextState, err := api.service.ToggleDeviceEnabled(device.ID)
	if err != nil {
		WriteNotFoundDef(w, errDeviceNotFound)
		return
	}
	WriteSuccess(w, enabledStateView{Enabled: nextState})
}
