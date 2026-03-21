package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type batchDeleteAlarmsRequest struct {
	IDs []int64 `json:"ids"`
}

func (api *AlarmAPI) loadAlarmByRequest(w http.ResponseWriter, r *http.Request) (*models.AlarmLog, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}

	alarmLog, err := api.service.LoadAlarm(id)
	if err != nil {
		WriteNotFoundDef(w, errAlarmNotFound)
		return nil, false
	}
	return alarmLog, true
}

func parseBatchDeleteAlarmsRequest(w http.ResponseWriter, r *http.Request) (*batchDeleteAlarmsRequest, bool) {
	var req batchDeleteAlarmsRequest
	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	if len(req.IDs) == 0 {
		WriteBadRequestDef(w, errAlarmIDsRequired)
		return nil, false
	}
	return &req, true
}
