package app

import "net/http"

func registerCollectorRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("POST /collector/start", apiDeps.status.StartCollector)
	api.HandleFunc("POST /collector/stop", apiDeps.status.StopCollector)
}
