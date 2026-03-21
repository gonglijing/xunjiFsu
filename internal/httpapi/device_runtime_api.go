package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type DeviceRuntimeAPI struct {
	service *service.DeviceRuntimeService
}

func NewDeviceRuntimeAPI(runtimeService *service.DeviceRuntimeService) *DeviceRuntimeAPI {
	return &DeviceRuntimeAPI{service: runtimeService}
}
