package handlers

import (
	"net/http"
	"strconv"

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

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "thresholds.html", map[string]interface{}{"Thresholds": thresholds}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "alarms.html", map[string]interface{}{"Alarms": logs}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		// 转换为模板需要的格式
		pointsMap := make([]map[string]interface{}, len(points))
		for i, p := range points {
			pointsMap[i] = map[string]interface{}{
				"Timestamp":  p.CollectedAt.Format("2006-01-02 15:04:05"),
				"DeviceName": p.DeviceName,
				"Variable":   p.FieldName,
				"Value":      p.Value,
				"Unit":       "",
			}
		}
		if err := tmpl.ExecuteTemplate(w, "history.html", map[string]interface{}{"Points": pointsMap}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	WriteSuccess(w, points)
}

// GetHistoryData 获取历史数据（带时间范围过滤）
func (h *Handler) GetHistoryData(w http.ResponseWriter, r *http.Request) {
	// 解析查询参数
	deviceID := r.URL.Query().Get("device_id")

	var points []*database.DataPoint
	var err error

	if deviceID != "" {
		id, parseErr := strconv.ParseInt(deviceID, 10, 64)
		if parseErr != nil {
			WriteBadRequest(w, "Invalid device_id")
			return
		}
		points, err = database.GetDataPointsByDevice(id, 1000)
	} else {
		points, err = database.GetLatestDataPoints(1000)
	}

	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		// 转换为模板需要的格式
		pointsMap := make([]map[string]interface{}, len(points))
		for i, p := range points {
			pointsMap[i] = map[string]interface{}{
				"Timestamp":  p.CollectedAt.Format("2006-01-02 15:04:05"),
				"DeviceName": p.DeviceName,
				"Variable":   p.FieldName,
				"Value":      p.Value,
				"Unit":       "",
			}
		}
		if err := tmpl.ExecuteTemplate(w, "history.html", map[string]interface{}{"Points": pointsMap}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		// 转换为模板需要的格式
		pointsMap := make([]map[string]interface{}, len(points))
		for i, p := range points {
			pointsMap[i] = map[string]interface{}{
				"Timestamp":  p.CollectedAt.Format("2006-01-02 15:04:05"),
				"DeviceName": p.DeviceName,
				"Variable":   p.FieldName,
				"Value":      p.Value,
				"Unit":       "",
			}
		}
		if err := tmpl.ExecuteTemplate(w, "realtime.html", map[string]interface{}{"DataPoints": pointsMap}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
