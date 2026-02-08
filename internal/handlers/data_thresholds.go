package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

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
	id, err := database.CreateThreshold(&threshold)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateThresholdFailed, err)
		return
	}
	threshold.ID = id
	WriteCreated(w, threshold)
}

func (h *Handler) UpdateThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	var threshold models.Threshold
	if !parseRequestOrWriteBadRequestDefault(w, r, &threshold) {
		return
	}
	threshold.ID = id
	if err := database.UpdateThreshold(&threshold); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateThresholdFailed, err)
		return
	}
	WriteSuccess(w, threshold)
}

func (h *Handler) DeleteThreshold(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if err := database.DeleteThreshold(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteThresholdFailed, err)
		return
	}
	WriteDeleted(w)
}
