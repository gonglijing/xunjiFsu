package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type DriverAPI struct {
	service *service.DriverService
}

func NewDriverAPI(driverService *service.DriverService) *DriverAPI {
	return &DriverAPI{service: driverService}
}

var (
	errListDriversFailed      = APIErrorDef{Code: "E_LIST_DRIVERS_FAILED", Message: "获取驱动列表失败"}
	errCreateDriverFailed     = APIErrorDef{Code: "E_CREATE_DRIVER_FAILED", Message: "创建驱动失败"}
	errUpdateDriverFailed     = APIErrorDef{Code: "E_UPDATE_DRIVER_FAILED", Message: "更新驱动失败"}
	errDeleteDriverFailed     = APIErrorDef{Code: "E_DELETE_DRIVER_FAILED", Message: "删除驱动失败"}
	errReloadDriverFailed     = APIErrorDef{Code: "E_RELOAD_DRIVER_FAILED", Message: "重载驱动失败"}
	errGetDriverRuntimeFailed = APIErrorDef{Code: "E_GET_DRIVER_RUNTIME_FAILED", Message: "获取驱动运行态失败"}
	errListDriverFilesFailed  = APIErrorDef{Code: "E_LIST_DRIVER_FILES_FAILED", Message: "读取驱动目录失败"}
	errSaveDriverFileFailed   = APIErrorDef{Code: "E_SAVE_DRIVER_FILE_FAILED", Message: "保存驱动文件失败"}
	errCreateDriversDirFailed = APIErrorDef{Code: "E_CREATE_DRIVERS_DIR_FAILED", Message: "创建驱动目录失败"}
	errWriteDriverFileFailed  = APIErrorDef{Code: "E_WRITE_DRIVER_FILE_FAILED", Message: "写入驱动文件失败"}
	errDriverNotFound         = APIErrorDef{Code: "E_DRIVER_NOT_FOUND", Message: "Driver not found"}
	errDriverNameRequired     = APIErrorDef{Code: "E_DRIVER_NAME_REQUIRED", Message: "driver name is required"}
	errDriverWasmNotFound     = APIErrorDef{Code: "E_DRIVER_WASM_NOT_FOUND", Message: "driver wasm file not found"}
)
