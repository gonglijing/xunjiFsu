package app

import "net/http"

func registerDataRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /data", apiDeps.data.GetDataCache)
	api.HandleFunc("GET /data/cache/{id}", apiDeps.data.GetDataCacheByDeviceID)
	api.HandleFunc("GET /data/history", apiDeps.data.GetHistoryData)
	api.HandleFunc("DELETE /data/history", apiDeps.data.ClearHistoryData)
}
