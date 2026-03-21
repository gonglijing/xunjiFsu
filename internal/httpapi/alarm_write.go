package httpapi

import "net/http"

func (api *AlarmAPI) AcknowledgeAlarm(w http.ResponseWriter, r *http.Request) {
	alarmLog, ok := api.loadAlarmByRequest(w, r)
	if !ok {
		return
	}
	if err := api.service.AcknowledgeAlarm(alarmLog.ID, "admin"); err != nil {
		writeServerErrorWithLog(w, errAcknowledgeAlarmFailed, err)
		return
	}
	WriteSuccess(w, operationStatusView{Status: "acknowledged"})
}

func (api *AlarmAPI) DeleteAlarm(w http.ResponseWriter, r *http.Request) {
	alarmLog, ok := api.loadAlarmByRequest(w, r)
	if !ok {
		return
	}
	if err := api.service.DeleteAlarm(alarmLog.ID); err != nil {
		writeServerErrorWithLog(w, errDeleteAlarmFailed, err)
		return
	}
	WriteDeleted(w)
}

func (api *AlarmAPI) BatchDeleteAlarms(w http.ResponseWriter, r *http.Request) {
	req, ok := parseBatchDeleteAlarmsRequest(w, r)
	if !ok {
		return
	}

	deleted, err := api.service.BatchDeleteAlarms(req.IDs)
	if err != nil {
		writeServerErrorWithLog(w, errBatchDeleteAlarmFailed, err)
		return
	}
	WriteSuccess(w, deletedCountView{Deleted: deleted})
}

func (api *AlarmAPI) ClearAlarms(w http.ResponseWriter, r *http.Request) {
	deleted, err := api.service.ClearAlarms()
	if err != nil {
		writeServerErrorWithLog(w, errClearAlarmLogsFailed, err)
		return
	}
	WriteSuccess(w, deletedCountView{Deleted: deleted})
}
