package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 阈值管理 ====================

// GetThresholds 获取所有阈值配置
func (h *Handler) GetThresholds(w http.ResponseWriter, r *http.Request) {
	thresholds, err := database.GetAllThresholds()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, thresholds)
}

// CreateThreshold 创建阈值配置
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

// UpdateThreshold 更新阈值配置
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

// DeleteThreshold 删除阈值配置
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

// ==================== 报警日志 ====================

// GetAlarmLogs 获取报警日志
func (h *Handler) GetAlarmLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := database.GetRecentAlarmLogs(100)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, logs)
}

// AcknowledgeAlarm 确认报警
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

// ==================== 数据缓存 ====================

// GetDataCache 获取数据缓存
func (h *Handler) GetDataCache(w http.ResponseWriter, r *http.Request) {
	cache, err := database.GetAllDataCache()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, cache)
}

// GetDataCacheByDeviceID 获取指定设备的数据缓存
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

// ==================== 历史数据 ====================

// GetDataPoints 获取历史数据点
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

// GetLatestDataPoints 获取最新的数据点
func (h *Handler) GetLatestDataPoints(w http.ResponseWriter, r *http.Request) {
	points, err := database.GetLatestDataPoints(100)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, points)
}

// ==================== 存储配置 ====================

// GetStorageConfigs 获取存储配置
func (h *Handler) GetStorageConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllStorageConfigs()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, configs)
}

// CreateStorageConfig 创建存储配置
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

// UpdateStorageConfig 更新存储配置
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

// DeleteStorageConfig 删除存储配置
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
