package app

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
)

func registerAPIRoutes(r *http.ServeMux, h *handlers.Handler, authManager *auth.JWTManager) {
	api := http.NewServeMux()

	api.HandleFunc("GET /status", h.GetStatus)

	registerCollectorRoutes(api, h)
	registerDriverRoutes(api, h)
	registerDeviceRoutes(api, h)
	registerNorthboundRoutes(api, h)
	registerThresholdRoutes(api, h)
	registerAlarmRoutes(api, h)
	registerDataRoutes(api, h)
	registerUserRoutes(api, h)
	registerResourceRoutes(api, h)
	registerGatewayRoutes(api, h)
	registerDebugRoutes(api, h)

	r.Handle("/api/", authManager.RequireAuth(http.StripPrefix("/api", api)))
}

func registerCollectorRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("POST /collector/start", h.StartCollector)
	api.HandleFunc("POST /collector/stop", h.StopCollector)
}

func registerDriverRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /drivers", h.GetDrivers)
	api.HandleFunc("GET /drivers/runtime", h.GetDriverRuntimeList)
	api.HandleFunc("GET /drivers/files", h.ListDriverFiles)
	api.HandleFunc("POST /drivers", h.CreateDriver)
	api.HandleFunc("PUT /drivers/{id}", h.UpdateDriver)
	api.HandleFunc("DELETE /drivers/{id}", h.DeleteDriver)
	api.HandleFunc("GET /drivers/{id}/runtime", h.GetDriverRuntime)
	api.HandleFunc("POST /drivers/{id}/reload", h.ReloadDriver)
	api.HandleFunc("GET /drivers/{id}/download", h.DownloadDriver)
	api.HandleFunc("POST /drivers/upload", h.UploadDriverFile)
}

func registerDeviceRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /devices", h.GetDevices)
	api.HandleFunc("POST /devices", h.CreateDevice)
	api.HandleFunc("PUT /devices/{id}", h.UpdateDevice)
	api.HandleFunc("DELETE /devices/{id}", h.DeleteDevice)
	api.HandleFunc("POST /devices/{id}/toggle", h.ToggleDeviceEnable)
	api.HandleFunc("POST /devices/{id}/execute", h.ExecuteDriverFunction)
	api.HandleFunc("GET /devices/{id}/writables", h.GetDeviceWritables)
}

func registerNorthboundRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /northbound", h.GetNorthboundConfigs)
	api.HandleFunc("GET /northbound/status", h.GetNorthboundStatus)
	api.HandleFunc("GET /northbound/schema", h.GetNorthboundSchema)
	api.HandleFunc("POST /northbound", h.CreateNorthboundConfig)
	api.HandleFunc("PUT /northbound/{id}", h.UpdateNorthboundConfig)
	api.HandleFunc("DELETE /northbound/{id}", h.DeleteNorthboundConfig)
	api.HandleFunc("POST /northbound/{id}/toggle", h.ToggleNorthboundEnable)
	api.HandleFunc("POST /northbound/{id}/reload", h.ReloadNorthboundConfig)
}

func registerThresholdRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /thresholds", h.GetThresholds)
	api.HandleFunc("POST /thresholds", h.CreateThreshold)
	api.HandleFunc("GET /thresholds/repeat-interval", h.GetAlarmRepeatInterval)
	api.HandleFunc("POST /thresholds/repeat-interval", h.UpdateAlarmRepeatInterval)
	api.HandleFunc("PUT /thresholds/{id}", h.UpdateThreshold)
	api.HandleFunc("DELETE /thresholds/{id}", h.DeleteThreshold)
}

func registerAlarmRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /alarms", h.GetAlarmLogs)
	api.HandleFunc("DELETE /alarms", h.ClearAlarms)
	api.HandleFunc("POST /alarms/batch-delete", h.BatchDeleteAlarms)
	api.HandleFunc("DELETE /alarms/{id}", h.DeleteAlarm)
	api.HandleFunc("POST /alarms/{id}/acknowledge", h.AcknowledgeAlarm)
}

func registerDataRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /data", h.GetDataCache)
	api.HandleFunc("GET /data/cache/{id}", h.GetDataCacheByDeviceID)
	api.HandleFunc("GET /data/history", h.GetHistoryData)
	api.HandleFunc("DELETE /data/history", h.ClearHistoryData)
}

func registerUserRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /users", h.GetUsers)
	api.HandleFunc("POST /users", h.CreateUser)
	api.HandleFunc("PUT /users/{id}", h.UpdateUser)
	api.HandleFunc("DELETE /users/{id}", h.DeleteUser)
	api.HandleFunc("PUT /users/password", h.ChangePassword)
}

func registerResourceRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /resources", h.GetResources)
	api.HandleFunc("POST /resources", h.CreateResource)
	api.HandleFunc("PUT /resources/{id}", h.UpdateResource)
	api.HandleFunc("DELETE /resources/{id}", h.DeleteResource)
	api.HandleFunc("POST /resources/{id}/toggle", h.ToggleResource)
}

func registerGatewayRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("GET /gateway/config", h.GetGatewayConfig)
	api.HandleFunc("PUT /gateway/config", h.UpdateGatewayConfig)
	api.HandleFunc("GET /gateway/runtime", h.GetGatewayRuntimeConfig)
	api.HandleFunc("PUT /gateway/runtime", h.UpdateGatewayRuntimeConfig)
	api.HandleFunc("GET /gateway/runtime/audits", h.GetGatewayRuntimeAudits)
}

func registerDebugRoutes(api *http.ServeMux, h *handlers.Handler) {
	api.HandleFunc("POST /debug/modbus/serial", h.DebugModbusSerial)
	api.HandleFunc("POST /debug/modbus/tcp", h.DebugModbusTCP)
}
