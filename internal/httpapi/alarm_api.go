package httpapi

import "github.com/gonglijing/xunjiFsu/internal/service"

type AlarmAPI struct {
	service *service.AlarmService
}

func NewAlarmAPI(alarmService *service.AlarmService) *AlarmAPI {
	return &AlarmAPI{service: alarmService}
}

var (
	errListAlarmLogsFailed    = APIErrorDef{Code: "E_LIST_ALARM_LOGS_FAILED", Message: "获取报警日志失败"}
	errAcknowledgeAlarmFailed = APIErrorDef{Code: "E_ACKNOWLEDGE_ALARM_FAILED", Message: "确认报警失败"}
	errDeleteAlarmFailed      = APIErrorDef{Code: "E_DELETE_ALARM_FAILED", Message: "删除报警失败"}
	errBatchDeleteAlarmFailed = APIErrorDef{Code: "E_BATCH_DELETE_ALARM_FAILED", Message: "批量删除报警失败"}
	errClearAlarmLogsFailed   = APIErrorDef{Code: "E_CLEAR_ALARM_LOGS_FAILED", Message: "清空报警失败"}
	errAlarmNotFound          = APIErrorDef{Code: "E_ALARM_NOT_FOUND", Message: "alarm not found"}
	errAlarmIDsRequired       = APIErrorDef{Code: "E_ALARM_IDS_REQUIRED", Message: "ids is required"}
)
