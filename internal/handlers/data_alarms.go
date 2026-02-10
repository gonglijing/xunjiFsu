package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

type batchDeleteAlarmsRequest struct {
	IDs []int64 `json:"ids"`
}

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

func (h *Handler) DeleteAlarm(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if err := database.DeleteAlarmLog(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteAlarmFailed, err)
		return
	}
	WriteDeleted(w)
}

func (h *Handler) BatchDeleteAlarms(w http.ResponseWriter, r *http.Request) {
	var req batchDeleteAlarmsRequest
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
		return
	}
	if len(req.IDs) == 0 {
		WriteBadRequestDef(w, apiErrAlarmIDsRequired)
		return
	}

	deleted, err := database.DeleteAlarmLogsByIDs(req.IDs)
	if err != nil {
		writeServerErrorWithLog(w, apiErrBatchDeleteAlarmFailed, err)
		return
	}
	WriteSuccess(w, map[string]int64{"deleted": deleted})
}

func (h *Handler) ClearAlarms(w http.ResponseWriter, r *http.Request) {
	deleted, err := database.ClearAlarmLogs()
	if err != nil {
		writeServerErrorWithLog(w, apiErrClearAlarmLogsFailed, err)
		return
	}
	WriteSuccess(w, map[string]int64{"deleted": deleted})
}
