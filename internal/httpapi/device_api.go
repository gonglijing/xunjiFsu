package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type DeviceAPI struct {
	service *service.DeviceService
}

func NewDeviceAPI(deviceService *service.DeviceService) *DeviceAPI {
	return &DeviceAPI{service: deviceService}
}
