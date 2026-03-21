package app

import "net/http"

func registerDriverRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /drivers", apiDeps.driver.GetDrivers)
	api.HandleFunc("GET /drivers/runtime", apiDeps.driver.GetDriverRuntimeList)
	api.HandleFunc("GET /drivers/files", apiDeps.driver.ListDriverFiles)
	api.HandleFunc("POST /drivers", apiDeps.driver.CreateDriver)
	api.HandleFunc("PUT /drivers/{id}", apiDeps.driver.UpdateDriver)
	api.HandleFunc("DELETE /drivers/{id}", apiDeps.driver.DeleteDriver)
	api.HandleFunc("GET /drivers/{id}/runtime", apiDeps.driver.GetDriverRuntime)
	api.HandleFunc("POST /drivers/{id}/reload", apiDeps.driver.ReloadDriver)
	api.HandleFunc("GET /drivers/{id}/download", apiDeps.driver.DownloadDriver)
	api.HandleFunc("POST /drivers/upload", apiDeps.driver.UploadDriverFile)
}
