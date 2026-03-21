package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func parseGatewayConfigRequest(w http.ResponseWriter, r *http.Request) (*models.GatewayConfig, bool) {
	var cfg models.GatewayConfig
	if err := ParseRequest(r, &cfg); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}

	service.NormalizeGatewayConfigInput(&cfg)
	return &cfg, true
}
