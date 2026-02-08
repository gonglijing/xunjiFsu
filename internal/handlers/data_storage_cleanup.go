package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

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
	if !parseRequestOrWriteBadRequestDefault(w, r, &config) {
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
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	var config database.StorageConfig
	if !parseRequestOrWriteBadRequestDefault(w, r, &config) {
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
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if err := database.DeleteStorageConfig(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteDeleted(w)
}

// CleanupData 清理过期数据
func (h *Handler) CleanupData(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Before string `json:"before"`
	}
	if !parseRequestOrWriteBadRequestDefault(w, r, &req) {
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
