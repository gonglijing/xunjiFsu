package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// 阈值管理
func (h *Handler) GetThresholds(w http.ResponseWriter, r *http.Request) {
	thresholds, err := database.GetAllThresholds()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, thresholds)
}

func (h *Handler) CreateThreshold(w http.ResponseWriter, r *http.Request) {
	var threshold models.Threshold
	if err := ParseRequest(r, &threshold); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	id, err := database.CreateThreshold(&threshold)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	threshold.ID = id
	WriteCreated(w, threshold)
}

func (h *Handler) UpdateThreshold(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	var threshold models.Threshold
	if err := ParseRequest(r, &threshold); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	threshold.ID = id
	if err := database.UpdateThreshold(&threshold); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, threshold)
}

func (h *Handler) DeleteThreshold(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	if err := database.DeleteThreshold(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, nil)
}

// 报警日志
func (h *Handler) GetAlarmLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := database.GetRecentAlarmLogs(100)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, logs)
}

func (h *Handler) AcknowledgeAlarm(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	if err := database.AcknowledgeAlarmLog(id, "admin"); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, map[string]string{"status": "acknowledged"})
}

// 数据缓存
func (h *Handler) GetDataCache(w http.ResponseWriter, r *http.Request) {
	cache, err := database.GetAllDataCache()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, cache)
}

func (h *Handler) GetDataCacheByDeviceID(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	cache, err := database.GetDataCacheByDeviceID(id)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, cache)
}

// 历史数据
func (h *Handler) GetDataPoints(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	points, err := database.GetDataPointsByDevice(id, 1000)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, points)
}

func (h *Handler) GetHistoryData(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	fieldName := r.URL.Query().Get("field_name")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	var (
		points []*database.DataPoint
		err    error
	)
	var startTime, endTime time.Time
	if startStr != "" {
		parsed, parseErr := parseTimeParam(startStr)
		if parseErr != nil {
			WriteBadRequest(w, "Invalid start time")
			return
		}
		startTime = parsed
	}
	if endStr != "" {
		parsed, parseErr := parseTimeParam(endStr)
		if parseErr != nil {
			WriteBadRequest(w, "Invalid end time")
			return
		}
		endTime = parsed
	}

	if deviceID != "" {
		id, parseErr := strconv.ParseInt(deviceID, 10, 64)
		if parseErr != nil {
			WriteBadRequest(w, "Invalid device_id")
			return
		}
		if fieldName != "" {
			points, err = database.GetDataPointsByDeviceFieldAndTime(id, fieldName, startTime, endTime, 2000)
		} else if !startTime.IsZero() || !endTime.IsZero() {
			points, err = database.GetDataPointsByDeviceAndTimeLimit(id, startTime, endTime, 2000)
		} else {
			points, err = database.GetDataPointsByDevice(id, 1000)
		}
	} else {
		points, err = database.GetLatestDataPoints(1000)
	}
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, points)
}

func parseTimeParam(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02T15:04", value); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return ts, nil
	}
	return time.Time{}, fmt.Errorf("invalid time format")
}

func (h *Handler) GetLatestDataPoints(w http.ResponseWriter, r *http.Request) {
	points, err := database.GetLatestDataPoints(100)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, points)
}

// 存储配置
func (h *Handler) GetStorageConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllStorageConfigs()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, configs)
}

func (h *Handler) CreateStorageConfig(w http.ResponseWriter, r *http.Request) {
	var config database.StorageConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	id, err := database.CreateStorageConfig(&config)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	config.ID = id
	WriteCreated(w, config)
}

func (h *Handler) UpdateStorageConfig(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	var config database.StorageConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	config.ID = id
	if err := database.UpdateStorageConfig(&config); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, config)
}

func (h *Handler) DeleteStorageConfig(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}
	if err := database.DeleteStorageConfig(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, nil)
}

// CleanupData 清理过期数据
func (h *Handler) CleanupData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Before string `json:"before"`
	}
	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}
	count, err := database.CleanupData(req.Before)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, map[string]interface{}{
		"deleted_count": count,
		"message":       "Data cleaned up successfully",
	})
}

// CleanupDataByPolicy 按存储策略立即执行一次清理
func (h *Handler) CleanupDataByPolicy(w http.ResponseWriter, r *http.Request) {
	count, err := database.CleanupOldDataByConfig()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, map[string]interface{}{
		"deleted_count": count,
		"message":       "Cleanup by policy finished",
	})
}
