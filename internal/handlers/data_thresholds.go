package handlers

import (
	"fmt"
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
	threshold, ok := parseThresholdPayload(w, r)
	if !ok {
		return
	}

	id, err := database.CreateThreshold(threshold)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateThresholdFailed, err)
		return
	}
	invalidateThresholdDeviceCaches(threshold, nil)

	threshold.ID = id
	WriteCreated(w, threshold)
}

func (h *Handler) UpdateThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	oldThreshold, _ := database.GetThresholdByID(id)
	threshold, ok := parseThresholdPayload(w, r)
	if !ok {
		return
	}

	threshold.ID = id
	if err := database.UpdateThreshold(threshold); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateThresholdFailed, err)
		return
	}

	invalidateThresholdDeviceCaches(threshold, oldThreshold)

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

	invalidateThresholdDeviceCaches(nil, threshold)

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

	if err := validateAlarmRepeatIntervalSeconds(payload.Seconds); err != nil {
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

func parseThresholdPayload(w http.ResponseWriter, r *http.Request) (*models.Threshold, bool) {
	var threshold models.Threshold
	if !parseRequestOrWriteBadRequestDefault(w, r, &threshold) {
		return nil, false
	}
	normalizeThresholdInput(&threshold)
	return &threshold, true
}

func invalidateThresholdDeviceCaches(current *models.Threshold, previous *models.Threshold) {
	for _, deviceID := range buildThresholdCacheDeviceIDs(current, previous) {
		collector.InvalidateDeviceCache(deviceID)
	}
}

func buildThresholdCacheDeviceIDs(current *models.Threshold, previous *models.Threshold) []int64 {
	if current == nil && previous == nil {
		return nil
	}

	deviceIDs := make([]int64, 0, 2)
	appendDeviceID := func(deviceID int64) {
		if deviceID <= 0 {
			return
		}
		for _, existing := range deviceIDs {
			if existing == deviceID {
				return
			}
		}
		deviceIDs = append(deviceIDs, deviceID)
	}

	if current != nil {
		appendDeviceID(current.DeviceID)
	}
	if previous != nil {
		appendDeviceID(previous.DeviceID)
	}

	return deviceIDs
}

func validateAlarmRepeatIntervalSeconds(seconds int) error {
	if seconds <= 0 {
		return fmt.Errorf("alarm repeat interval must be > 0")
	}
	return nil
}
