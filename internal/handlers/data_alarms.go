package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// 报警日志
func (h *Handler) GetAlarmLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := database.GetRecentAlarmLogs(100)
	if err != nil {
		writeServerErrorWithLog(w, apiErrListAlarmLogsFailed, err)
		return
	}
	WriteSuccess(w, logs)
}

func (h *Handler) AcknowledgeAlarm(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if err := database.AcknowledgeAlarmLog(id, "admin"); err != nil {
		writeServerErrorWithLog(w, apiErrAcknowledgeAlarmFailed, err)
		return
	}
	WriteSuccess(w, map[string]string{"status": "acknowledged"})
}
