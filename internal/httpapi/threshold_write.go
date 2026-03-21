package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *ThresholdAPI) CreateThreshold(w http.ResponseWriter, r *http.Request) {
	threshold, ok := parseThresholdRequest(w, r)
	if !ok {
		return
	}

	threshold, err := api.service.CreateThreshold(threshold)
	if err != nil {
		writeServerErrorWithLog(w, errCreateThresholdFailed, err)
		return
	}
	WriteCreated(w, threshold)
}

func (api *ThresholdAPI) UpdateThreshold(w http.ResponseWriter, r *http.Request) {
	thresholdModel, ok := api.loadThresholdByRequest(w, r)
	if !ok {
		return
	}

	threshold, ok := parseThresholdRequest(w, r)
	if !ok {
		return
	}

	threshold.ID = thresholdModel.ID
	threshold, err := api.service.UpdateThreshold(threshold)
	if err != nil {
		writeServerErrorWithLog(w, errUpdateThresholdFailed, err)
		return
	}

	WriteSuccess(w, threshold)
}

func (api *ThresholdAPI) DeleteThreshold(w http.ResponseWriter, r *http.Request) {
	threshold, ok := api.loadThresholdByRequest(w, r)
	if !ok {
		return
	}

	if err := api.service.DeleteThreshold(threshold.ID); err != nil {
		writeServerErrorWithLog(w, errDeleteThresholdFailed, err)
		return
	}

	WriteDeleted(w)
}

func (api *ThresholdAPI) UpdateAlarmRepeatInterval(w http.ResponseWriter, r *http.Request) {
	var payload alarmRepeatIntervalPayload
	if err := ParseRequest(r, &payload); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return
	}

	if err := service.ValidateAlarmRepeatIntervalSeconds(payload.Seconds); err != nil {
		WriteBadRequestDef(w, errInvalidAlarmRepeatInterval)
		return
	}

	if err := api.service.UpdateAlarmRepeatInterval(payload.Seconds); err != nil {
		writeServerErrorWithLog(w, errUpdateAlarmRepeatIntervalFailed, err)
		return
	}
	WriteSuccess(w, alarmRepeatIntervalView{Seconds: payload.Seconds})
}
