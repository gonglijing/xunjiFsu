package handlers

import (
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const defaultGatewayName = "HuShu智能网关"

func normalizeGatewayConfigInput(cfg *models.GatewayConfig) {
	if cfg == nil {
		return
	}
	cfg.GatewayName = strings.TrimSpace(cfg.GatewayName)
	if cfg.GatewayName == "" {
		cfg.GatewayName = defaultGatewayName
	}
	if cfg.DataRetentionDays <= 0 {
		cfg.DataRetentionDays = database.DefaultRetentionDays
	}
}

func toDatabaseGatewayConfig(cfg *models.GatewayConfig) *database.GatewayConfig {
	if cfg == nil {
		return nil
	}
	return &database.GatewayConfig{
		ID:                cfg.ID,
		GatewayName:       cfg.GatewayName,
		DataRetentionDays: cfg.DataRetentionDays,
	}
}

func loadGatewayConfigOrWriteServerError(w http.ResponseWriter) (*database.GatewayConfig, bool) {
	cfg, err := database.GetGatewayConfig()
	if err != nil {
		writeServerErrorWithLog(w, apiErrGetGatewayConfigFailed, err)
		return nil, false
	}
	return cfg, true
}
