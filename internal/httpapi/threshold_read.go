package httpapi

import "net/http"

func (api *ThresholdAPI) GetThresholds(w http.ResponseWriter, r *http.Request) {
	thresholds, err := api.service.ListThresholds()
	if err != nil {
		writeServerErrorWithLog(w, errListThresholdsFailed, err)
		return
	}
	WriteSuccess(w, thresholds)
}

func (api *ThresholdAPI) GetAlarmRepeatInterval(w http.ResponseWriter, r *http.Request) {
	seconds, err := api.service.LoadAlarmRepeatInterval()
	if err != nil {
		writeServerErrorWithLog(w, errGetAlarmRepeatIntervalFailed, err)
		return
	}

	WriteSuccess(w, alarmRepeatIntervalView{Seconds: seconds})
}
