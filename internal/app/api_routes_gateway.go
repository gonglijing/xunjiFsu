package app

import "net/http"

func registerGatewayRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /gateway/config", apiDeps.gateway.GetGatewayConfig)
	api.HandleFunc("PUT /gateway/config", apiDeps.gateway.UpdateGatewayConfig)
	api.HandleFunc("GET /gateway/runtime", apiDeps.gateway.GetGatewayRuntimeConfig)
	api.HandleFunc("PUT /gateway/runtime", apiDeps.gateway.UpdateGatewayRuntimeConfig)
	api.HandleFunc("GET /gateway/runtime/audits", apiDeps.gateway.GetGatewayRuntimeAudits)
}
