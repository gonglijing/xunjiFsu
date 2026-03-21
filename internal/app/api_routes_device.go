package app

import "net/http"

func registerDeviceRoutes(api *http.ServeMux, apiDeps *apiRouteDeps) {
	api.HandleFunc("GET /devices", apiDeps.device.ListDevices)
	api.HandleFunc("GET /devices/runtime", apiDeps.deviceRuntime.GetDeviceRuntimeStatuses)
	api.HandleFunc("POST /devices", apiDeps.device.CreateDevice)
	api.HandleFunc("PUT /devices/{id}", apiDeps.device.UpdateDevice)
	api.HandleFunc("DELETE /devices/{id}", apiDeps.device.DeleteDevice)
	api.HandleFunc("POST /devices/{id}/toggle", apiDeps.device.ToggleDeviceEnabled)
	api.HandleFunc("POST /devices/{id}/execute", apiDeps.deviceExec.ExecuteDriverFunction)
	api.HandleFunc("GET /devices/{id}/runtime", apiDeps.deviceRuntime.GetDeviceRuntimeStatus)
	api.HandleFunc("GET /devices/{id}/writables", apiDeps.deviceExec.GetDeviceWritables)
}
