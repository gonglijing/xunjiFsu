package handlers

import (
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type alarmRepeatIntervalPayload struct {
	Seconds int `json:"seconds"`
}

// 阈值管理
func (h *Handler) GetThresholds(w http.ResponseWriter, r *http.Request) {
	thresholds, err := database.GetAllThresholds()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListThresholdsFailed, err)
		return
	}
	WriteSuccess(w, thresholds)
}

func (h *Handler) CreateThreshold(w http.ResponseWriter, r *http.Request) {
	var threshold models.Threshold
	if !parseRequestOrWriteBadRequestDefault(w, r, &threshold) {
		return
	}
	normalizeThresholdInput(&threshold)

	id, err := database.CreateThreshold(&threshold)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateThresholdFailed, err)
		return
	}
	collector.InvalidateDeviceCache(threshold.DeviceID)

	threshold.ID = id
	WriteCreated(w, threshold)
}

func (h *Handler) UpdateThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	oldThreshold, _ := database.GetThresholdByID(id)

	var threshold models.Threshold
	if !parseRequestOrWriteBadRequestDefault(w, r, &threshold) {
		return
	}
	normalizeThresholdInput(&threshold)

	threshold.ID = id
	if err := database.UpdateThreshold(&threshold); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateThresholdFailed, err)
		return
	}

	collector.InvalidateDeviceCache(threshold.DeviceID)
	if oldThreshold != nil && oldThreshold.DeviceID != threshold.DeviceID {
		collector.InvalidateDeviceCache(oldThreshold.DeviceID)
	}

	WriteSuccess(w, threshold)
}

func (h *Handler) DeleteThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	threshold, _ := database.GetThresholdByID(id)
	if err := database.DeleteThreshold(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteThresholdFailed, err)
		return
	}

	if threshold != nil {
		collector.InvalidateDeviceCache(threshold.DeviceID)
	}

	WriteDeleted(w)
}

func (h *Handler) GetAlarmRepeatInterval(w http.ResponseWriter, r *http.Request) {
	seconds, err := database.GetAlarmRepeatIntervalSeconds()
	if err != nil {
		writeServerErrorWithLog(w, apiErrGetAlarmRepeatIntervalFailed, err)
		return
	}

	WriteSuccess(w, map[string]int{"seconds": seconds})
}

func (h *Handler) UpdateAlarmRepeatInterval(w http.ResponseWriter, r *http.Request) {
	var payload alarmRepeatIntervalPayload
	if !parseRequestOrWriteBadRequestDefault(w, r, &payload) {
		return
	}

	if payload.Seconds <= 0 {
		WriteBadRequestDef(w, apiErrInvalidAlarmRepeatInterval)
		return
	}

	if err := database.UpdateAlarmRepeatIntervalSeconds(payload.Seconds); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateAlarmRepeatIntervalFailed, err)
		return
	}

	collector.InvalidateAlarmRepeatIntervalCache()
	WriteSuccess(w, map[string]int{"seconds": payload.Seconds})
}

func normalizeThresholdInput(threshold *models.Threshold) {
	if threshold == nil {
		return
	}

	threshold.FieldName = strings.TrimSpace(threshold.FieldName)
	threshold.Operator = strings.TrimSpace(threshold.Operator)
	threshold.Severity = strings.TrimSpace(threshold.Severity)
	threshold.Message = strings.TrimSpace(threshold.Message)
	if threshold.Shielded != 1 {
		threshold.Shielded = 0
	}
}
