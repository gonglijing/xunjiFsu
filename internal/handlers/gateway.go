package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetGatewayConfig 获取网关配置
func (h *Handler) GetGatewayConfig(w http.ResponseWriter, r *http.Request) {
	cfg, ok := loadGatewayConfigOrWriteServerError(w)
	if !ok {
		return
	}
	WriteSuccess(w, cfg)
}

// UpdateGatewayConfig 更新网关配置
func (h *Handler) UpdateGatewayConfig(w http.ResponseWriter, r *http.Request) {
	cfg, ok := parseGatewayConfigPayload(w, r)
	if !ok {
		return
	}

	if err := database.UpdateGatewayConfig(toDatabaseGatewayConfig(cfg)); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateGatewayConfigFailed, err)
		return
	}

	updatedCfg, ok := loadGatewayConfigOrWriteServerError(w)
	if !ok {
		return
	}

	WriteSuccess(w, updatedCfg)
}

func parseGatewayConfigPayload(w http.ResponseWriter, r *http.Request) (*models.GatewayConfig, bool) {
	var cfg models.GatewayConfig
	if !parseRequestOrWriteBadRequestDefault(w, r, &cfg) {
		return nil, false
	}

	normalizeGatewayConfigInput(&cfg)
	return &cfg, true
}
