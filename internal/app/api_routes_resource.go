package app

import "net/http"

func registerResourceRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /resources", apiDeps.resource.GetResources)
	api.HandleFunc("POST /resources", apiDeps.resource.CreateResource)
	api.HandleFunc("PUT /resources/{id}", apiDeps.resource.UpdateResource)
	api.HandleFunc("DELETE /resources/{id}", apiDeps.resource.DeleteResource)
	api.HandleFunc("POST /resources/{id}/toggle", apiDeps.resource.ToggleResource)
}
