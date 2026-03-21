package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type DeviceExecAPI struct {
	service *service.DeviceExecService
}

func NewDeviceExecAPI(deviceExecService *service.DeviceExecService) *DeviceExecAPI {
	return &DeviceExecAPI{service: deviceExecService}
}

type executeDriverPayload struct {
	Function string                 `json:"function"`
	Params   map[string]interface{} `json:"params"`
}

var (
	errLoadDriverFailed       = APIErrorDef{Code: "E_LOAD_DRIVER_FAILED", Message: "加载驱动失败"}
	errDriverNotLoaded        = APIErrorDef{Code: "E_DRIVER_NOT_LOADED", Message: "driver is not loaded"}
	errDriverSchemaInvalid    = APIErrorDef{Code: "E_DRIVER_CONFIG_SCHEMA_INVALID", Message: "driver config_schema is invalid JSON"}
	errDeviceHasNoDriver      = APIErrorDef{Code: "E_DEVICE_HAS_NO_DRIVER", Message: "Device has no driver"}
	errExecuteDriverFailed    = APIErrorDef{Code: "E_EXECUTE_DRIVER_FAILED", Message: "执行驱动函数失败"}
	errExecuteDriverParamFail = APIErrorDef{Code: "E_EXECUTE_DRIVER_PARAM_INVALID", Message: "驱动执行参数无效"}
)
