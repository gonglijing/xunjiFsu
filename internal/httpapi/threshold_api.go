package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type ThresholdAPI struct {
	service *service.ThresholdService
}

func NewThresholdAPI(thresholdService *service.ThresholdService) *ThresholdAPI {
	return &ThresholdAPI{service: thresholdService}
}

type alarmRepeatIntervalPayload struct {
	Seconds int `json:"seconds"`
}

type alarmRepeatIntervalView struct {
	Seconds int `json:"seconds"`
}

var (
	errListThresholdsFailed            = APIErrorDef{Code: "E_LIST_THRESHOLDS_FAILED", Message: "获取阈值列表失败"}
	errCreateThresholdFailed           = APIErrorDef{Code: "E_CREATE_THRESHOLD_FAILED", Message: "创建阈值失败"}
	errUpdateThresholdFailed           = APIErrorDef{Code: "E_UPDATE_THRESHOLD_FAILED", Message: "更新阈值失败"}
	errDeleteThresholdFailed           = APIErrorDef{Code: "E_DELETE_THRESHOLD_FAILED", Message: "删除阈值失败"}
	errThresholdNotFound               = APIErrorDef{Code: "E_THRESHOLD_NOT_FOUND", Message: "threshold not found"}
	errGetAlarmRepeatIntervalFailed    = APIErrorDef{Code: "E_GET_ALARM_REPEAT_INTERVAL_FAILED", Message: "获取报警重复触发间隔失败"}
	errUpdateAlarmRepeatIntervalFailed = APIErrorDef{Code: "E_UPDATE_ALARM_REPEAT_INTERVAL_FAILED", Message: "更新报警重复触发间隔失败"}
	errInvalidAlarmRepeatInterval      = APIErrorDef{Code: "E_INVALID_ALARM_REPEAT_INTERVAL", Message: "alarm repeat interval must be > 0"}
)
