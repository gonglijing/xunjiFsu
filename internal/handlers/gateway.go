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
	normalizeGatewayConfigInput(&cfg)

	if err := database.UpdateGatewayConfig(toDatabaseGatewayConfig(&cfg)); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	updatedCfg, err := database.GetGatewayConfig()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, updatedCfg)
}
