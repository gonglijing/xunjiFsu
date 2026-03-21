package app

import "net/http"

func registerThresholdRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /thresholds", apiDeps.threshold.GetThresholds)
	api.HandleFunc("POST /thresholds", apiDeps.threshold.CreateThreshold)
	api.HandleFunc("GET /thresholds/repeat-interval", apiDeps.threshold.GetAlarmRepeatInterval)
	api.HandleFunc("POST /thresholds/repeat-interval", apiDeps.threshold.UpdateAlarmRepeatInterval)
	api.HandleFunc("PUT /thresholds/{id}", apiDeps.threshold.UpdateThreshold)
	api.HandleFunc("DELETE /thresholds/{id}", apiDeps.threshold.DeleteThreshold)
}
