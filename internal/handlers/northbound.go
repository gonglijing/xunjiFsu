package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

// ==================== 北向配置管理 ====================

// GetNorthboundConfigs 获取所有北向配置
func (h *Handler) GetNorthboundConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "northbound.html", map[string]interface{}{"Configs": configs}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	WriteSuccess(w, configs)
}

// CreateNorthboundConfig 创建北向配置
func (h *Handler) CreateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	var config models.NorthboundConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	id, err := database.CreateNorthboundConfig(&config)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	config.ID = id

	// 如果启用了，自动注册适配器
	if config.Enabled == 1 {
		h.registerNorthboundAdapter(&config)
	}

	WriteCreated(w, config)
}

// UpdateNorthboundConfig 更新北向配置
func (h *Handler) UpdateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	var config models.NorthboundConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	config.ID = id
	if err := database.UpdateNorthboundConfig(&config); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 处理使能状态变化
	if config.Enabled == 1 {
		h.northboundMgr.UnregisterAdapter(config.Name)
		h.registerNorthboundAdapter(&config)
	} else {
		h.northboundMgr.UnregisterAdapter(config.Name)
	}

	WriteSuccess(w, config)
}

// DeleteNorthboundConfig 删除北向配置
func (h *Handler) DeleteNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	// 获取配置信息，删除时移除适配器
	config, err := database.GetNorthboundConfigByID(id)
	if err == nil {
		h.northboundMgr.RemoveAdapter(config.Name)
	}

	if err := database.DeleteNorthboundConfig(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
}

// registerNorthboundAdapter 内部辅助函数：注册北向适配器
func (h *Handler) registerNorthboundAdapter(config *models.NorthboundConfig) {
	var adapter northbound.Northbound

	switch config.Type {
	case "xunji":
		adapter = northbound.NewXunJiAdapter()
	case "http":
		adapter = northbound.NewHTTPAdapter()
	case "mqtt":
		adapter = northbound.NewMQTTAdapter()
	default:
		return
	}

	if err := adapter.Initialize(config.Config); err != nil {
		return
	}

	h.northboundMgr.RegisterAdapter(config.Name, adapter)
}

// ToggleNorthboundEnable 切换北向使能状态
func (h *Handler) ToggleNorthboundEnable(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFound(w, "Northbound config not found")
		return
	}

	// 切换状态
	newState := 0
	if config.Enabled == 0 {
		newState = 1
	}

	if err := database.UpdateNorthboundEnabled(id, newState); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 更新适配器
	if newState == 1 {
		h.registerNorthboundAdapter(config)
	} else {
		h.northboundMgr.RemoveAdapter(config.Name)
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": newState,
		"message": "Northbound enabled"[:9] + "disabled"[(newState*9):],
	})
}
