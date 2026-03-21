package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type StatusAPI struct {
	service *service.StatusService
}

func NewStatusAPI(statusService *service.StatusService) *StatusAPI {
	return &StatusAPI{service: statusService}
}

var (
	errGetStatusFailed       = APIErrorDef{Code: "E_GET_STATUS_FAILED", Message: "获取系统状态失败"}
	errStartCollectorFailed  = APIErrorDef{Code: "E_START_COLLECTOR_FAILED", Message: "启动采集器失败"}
	errListDataCacheFailed   = APIErrorDef{Code: "E_LIST_DATA_CACHE_FAILED", Message: "获取数据缓存失败"}
	errGetDeviceCacheFailed  = APIErrorDef{Code: "E_GET_DEVICE_DATA_CACHE_FAILED", Message: "获取设备缓存失败"}
	errQueryHistoryData      = APIErrorDef{Code: "E_QUERY_HISTORY_DATA_FAILED", Message: "查询历史数据失败"}
	errClearHistoryData      = APIErrorDef{Code: "E_CLEAR_HISTORY_DATA_FAILED", Message: "清除历史数据失败"}
	errHistoryDataQueryDef   = APIErrorDef{Code: "E_HISTORY_DATA_QUERY_INVALID", Message: "历史数据查询参数无效"}
	errHistoryPointQueryDef  = APIErrorDef{Code: "E_HISTORY_POINT_QUERY_INVALID", Message: "历史测点参数无效"}
	errInvalidDeviceID       = "Invalid device_id"
	errHistoryStartAfterEnd  = "start time must be before end time"
	errHistoryFilterRequires = "device_id is required when using field_name/start/end filters"
	errHistoryFieldRequired  = "field_name is required"
)
