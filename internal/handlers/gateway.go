package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetGatewayConfig 获取网关配置
func (h *Handler) GetGatewayConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := database.GetGatewayConfig()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, cfg)
}

// UpdateGatewayConfig 更新网关配置
func (h *Handler) UpdateGatewayConfig(w http.ResponseWriter, r *http.Request) {
	var cfg models.GatewayConfig
	if err := ParseRequest(r, &cfg); err != nil {
		WriteBadRequest(w, "invalid body")
		return
	}

	// 转换为数据库模型
	dbCfg := &database.GatewayConfig{
		ID:          cfg.ID,
		ProductKey:  cfg.ProductKey,
		DeviceKey:   cfg.DeviceKey,
		GatewayName: cfg.GatewayName,
	}

	if err := database.UpdateGatewayConfig(dbCfg); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 返回更新后的配置
	updatedCfg, err := database.GetGatewayConfig()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, updatedCfg)
}
