package httpapi

import "net/http"

func (api *AlarmAPI) GetAlarmLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := api.service.ListRecentAlarmLogs(100)
	if err != nil {
		writeServerErrorWithLog(w, errListAlarmLogsFailed, err)
		return
	}
	WriteSuccess(w, logs)
}
