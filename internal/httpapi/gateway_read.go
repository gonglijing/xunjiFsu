package httpapi

import "net/http"

func (api *GatewayAPI) GetGatewayConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := api.configService.LoadGatewayConfig()
	if err != nil {
		writeServerErrorWithLog(w, errGetGatewayConfigFailed, err)
		return
	}
	WriteSuccess(w, cfg)
}
