package app

import (
	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gorilla/mux"
)

func registerAPIRoutes(r *mux.Router, h *handlers.Handler, authManager *auth.JWTManager) {
	api := r.PathPrefix("/api").Subrouter()
	api.Use(authManager.RequireAuth)

	api.HandleFunc("/status", h.GetStatus).Methods("GET")

	registerCollectorRoutes(api, h)
	registerDriverRoutes(api, h)
	registerDeviceRoutes(api, h)
	registerNorthboundRoutes(api, h)
	registerThresholdRoutes(api, h)
	registerAlarmRoutes(api, h)
	registerDataRoutes(api, h)
	registerStorageRoutes(api, h)
	registerUserRoutes(api, h)
	registerResourceRoutes(api, h)
	registerGatewayRoutes(api, h)
}

func registerCollectorRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/collector/start", h.StartCollector).Methods("POST")
	api.HandleFunc("/collector/stop", h.StopCollector).Methods("POST")
}

func registerDriverRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/drivers", h.GetDrivers).Methods("GET")
	api.HandleFunc("/drivers/runtime", h.GetDriverRuntimeList).Methods("GET")
	api.HandleFunc("/drivers/files", h.ListDriverFiles).Methods("GET")
	api.HandleFunc("/drivers", h.CreateDriver).Methods("POST")
	api.HandleFunc("/drivers/{id}", h.UpdateDriver).Methods("PUT")
	api.HandleFunc("/drivers/{id}", h.DeleteDriver).Methods("DELETE")
	api.HandleFunc("/drivers/{id}/runtime", h.GetDriverRuntime).Methods("GET")
	api.HandleFunc("/drivers/{id}/reload", h.ReloadDriver).Methods("POST")
	api.HandleFunc("/drivers/{id}/download", h.DownloadDriver).Methods("GET")
	api.HandleFunc("/drivers/upload", h.UploadDriverFile).Methods("POST")
}

func registerDeviceRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/devices", h.GetDevices).Methods("GET")
	api.HandleFunc("/devices", h.CreateDevice).Methods("POST")
	api.HandleFunc("/devices/{id}", h.UpdateDevice).Methods("PUT")
	api.HandleFunc("/devices/{id}", h.DeleteDevice).Methods("DELETE")
	api.HandleFunc("/devices/{id}/toggle", h.ToggleDeviceEnable).Methods("POST")
	api.HandleFunc("/devices/{id}/execute", h.ExecuteDriverFunction).Methods("POST")
	api.HandleFunc("/devices/{id}/writables", h.GetDeviceWritables).Methods("GET")
}

func registerNorthboundRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/northbound", h.GetNorthboundConfigs).Methods("GET")
	api.HandleFunc("/northbound/status", h.GetNorthboundStatus).Methods("GET")
	api.HandleFunc("/northbound/schema", h.GetNorthboundSchema).Methods("GET")
	api.HandleFunc("/northbound", h.CreateNorthboundConfig).Methods("POST")
	api.HandleFunc("/northbound/{id}", h.UpdateNorthboundConfig).Methods("PUT")
	api.HandleFunc("/northbound/{id}", h.DeleteNorthboundConfig).Methods("DELETE")
	api.HandleFunc("/northbound/{id}/toggle", h.ToggleNorthboundEnable).Methods("POST")
	api.HandleFunc("/northbound/{id}/reload", h.ReloadNorthboundConfig).Methods("POST")
}

func registerThresholdRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/thresholds", h.GetThresholds).Methods("GET")
	api.HandleFunc("/thresholds", h.CreateThreshold).Methods("POST")
	api.HandleFunc("/thresholds/{id}", h.UpdateThreshold).Methods("PUT")
	api.HandleFunc("/thresholds/{id}", h.DeleteThreshold).Methods("DELETE")
}

func registerAlarmRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/alarms", h.GetAlarmLogs).Methods("GET")
	api.HandleFunc("/alarms/{id}/acknowledge", h.AcknowledgeAlarm).Methods("POST")
}

func registerDataRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/data", h.GetDataCache).Methods("GET")
	api.HandleFunc("/data/cache/{id}", h.GetDataCacheByDeviceID).Methods("GET")
	api.HandleFunc("/data/history", h.GetHistoryData).Methods("GET")
}

func registerStorageRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/storage", h.GetStorageConfigs).Methods("GET")
	api.HandleFunc("/storage", h.CreateStorageConfig).Methods("POST")
	api.HandleFunc("/storage/{id}", h.UpdateStorageConfig).Methods("PUT")
	api.HandleFunc("/storage/{id}", h.DeleteStorageConfig).Methods("DELETE")
	api.HandleFunc("/storage/cleanup", h.CleanupData).Methods("POST")
	api.HandleFunc("/storage/run", h.CleanupDataByPolicy).Methods("POST")
}

func registerUserRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/users", h.GetUsers).Methods("GET")
	api.HandleFunc("/users", h.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", h.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE")
	api.HandleFunc("/users/password", h.ChangePassword).Methods("PUT")
}

func registerResourceRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/resources", h.GetResources).Methods("GET")
	api.HandleFunc("/resources", h.CreateResource).Methods("POST")
	api.HandleFunc("/resources/{id}", h.UpdateResource).Methods("PUT")
	api.HandleFunc("/resources/{id}", h.DeleteResource).Methods("DELETE")
	api.HandleFunc("/resources/{id}/toggle", h.ToggleResource).Methods("POST")
}

func registerGatewayRoutes(api *mux.Router, h *handlers.Handler) {
	api.HandleFunc("/gateway/config", h.GetGatewayConfig).Methods("GET")
	api.HandleFunc("/gateway/config", h.UpdateGatewayConfig).Methods("PUT")
	api.HandleFunc("/gateway/runtime", h.GetGatewayRuntimeConfig).Methods("GET")
	api.HandleFunc("/gateway/runtime", h.UpdateGatewayRuntimeConfig).Methods("PUT")
	api.HandleFunc("/gateway/runtime/audits", h.GetGatewayRuntimeAudits).Methods("GET")
	api.HandleFunc("/gateway/northbound/sync-identity", h.SyncGatewayIdentityToNorthbound).Methods("POST")
}
