package app

import "net/http"

func registerNorthboundRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /northbound", apiDeps.northbound.GetNorthboundConfigs)
	api.HandleFunc("GET /northbound/status", apiDeps.northbound.GetNorthboundStatus)
	api.HandleFunc("GET /northbound/schema", apiDeps.northbound.GetNorthboundSchema)
	api.HandleFunc("POST /northbound", apiDeps.northbound.CreateNorthboundConfig)
	api.HandleFunc("PUT /northbound/{id}", apiDeps.northbound.UpdateNorthboundConfig)
	api.HandleFunc("DELETE /northbound/{id}", apiDeps.northbound.DeleteNorthboundConfig)
	api.HandleFunc("POST /northbound/{id}/toggle", apiDeps.northbound.ToggleNorthboundEnable)
	api.HandleFunc("POST /northbound/{id}/reload", apiDeps.northbound.ReloadNorthboundConfig)
	api.HandleFunc("POST /northbound/{id}/sync-devices", apiDeps.northbound.SyncNorthboundDevices)
}
