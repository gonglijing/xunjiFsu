package httpapi

import (
	"net/http"
)

func (api *GatewayAPI) UpdateGatewayConfig(w http.ResponseWriter, r *http.Request) {
	cfg, ok := parseGatewayConfigRequest(w, r)
	if !ok {
		return
	}

	updatedCfg, err := api.configService.UpdateGatewayConfig(cfg)
	if err != nil {
		writeServerErrorWithLog(w, errUpdateGatewayConfigFailed, err)
		return
	}

	WriteSuccess(w, updatedCfg)
}
