package app

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gonglijing/xunjiFsu/internal/logger"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func buildRouter(h *handlers.Handler, sessionManager *auth.SessionManager) *mux.Router {
	r := mux.NewRouter()

	staticDir := resolveStaticDir()
	registerStaticRoutes(r, staticDir)
	registerPageRoutes(r, h, sessionManager)
	registerAPIRoutes(r, h)
	registerHealthRoutes(r)

	return r
}

func buildHandlerChain(cfg *config.Config, router *mux.Router) http.Handler {
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins(cfg.GetAllowedOrigins()),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		gorillaHandlers.AllowCredentials(),
	)

	loggingHandler := gorillaHandlers.LoggingHandler(os.Stdout, router)
	gzipHandler := handlers.GzipMiddleware(loggingHandler)

	timeoutConfig := handlers.DefaultTimeoutConfig()
	timeoutConfig.ReadTimeout = cfg.HTTPReadTimeout
	timeoutConfig.WriteTimeout = cfg.HTTPWriteTimeout
	timeoutConfig.IdleTimeout = cfg.HTTPIdleTimeout

	finalHandler := corsHandler(gzipHandler)
	return handlers.TimeoutMiddleware(timeoutConfig)(finalHandler)
}

func resolveStaticDir() http.Dir {
	workDir, err := os.Getwd()
	if err != nil {
		logger.Warn("Failed to get working directory, using relative static path", "error", err)
		return http.Dir(filepath.Join("web", "static"))
	}
	return http.Dir(filepath.Join(workDir, "web", "static"))
}

func registerStaticRoutes(r *mux.Router, staticDir http.Dir) {
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(staticDir)))
	r.PathPrefix("/web/static/").Handler(http.StripPrefix("/web/static/", http.FileServer(staticDir)))
}

func registerPageRoutes(r *mux.Router, h *handlers.Handler, sessionManager *auth.SessionManager) {
	r.HandleFunc("/login", h.Login).Methods("GET")
	r.HandleFunc("/login", h.LoginPost).Methods("POST")
	r.HandleFunc("/logout", h.Logout).Methods("GET")
	// SPA 入口，所有非 API/静态 GET 请求交给前端路由
	r.PathPrefix("/").Handler(sessionManager.RequireAuth(http.HandlerFunc(h.SPA))).Methods("GET")
}

func registerHealthRoutes(r *mux.Router) {
	r.HandleFunc("/health", handlers.Health).Methods("GET")
	r.HandleFunc("/ready", handlers.Readiness).Methods("GET")
	r.HandleFunc("/live", handlers.Liveness).Methods("GET")
	r.HandleFunc("/metrics", handlers.Metrics).Methods("GET")
}

func registerAPIRoutes(r *mux.Router, h *handlers.Handler) {
	r.HandleFunc("/api/status", h.GetStatus).Methods("GET")

	r.HandleFunc("/api/collector/start", h.StartCollector).Methods("POST")
	r.HandleFunc("/api/collector/stop", h.StopCollector).Methods("POST")

	r.HandleFunc("/api/drivers", h.GetDrivers).Methods("GET")
	r.HandleFunc("/api/drivers", h.CreateDriver).Methods("POST")
	r.HandleFunc("/api/drivers/{id}", h.UpdateDriver).Methods("PUT")
	r.HandleFunc("/api/drivers/{id}", h.DeleteDriver).Methods("DELETE")
	r.HandleFunc("/api/drivers/{id}/download", h.DownloadDriver).Methods("GET")
	r.HandleFunc("/api/drivers/upload", h.UploadDriverFile).Methods("POST")
	r.HandleFunc("/api/drivers/files", h.ListDriverFiles).Methods("GET")

	r.HandleFunc("/api/devices", h.GetDevices).Methods("GET")
	r.HandleFunc("/api/devices", h.CreateDevice).Methods("POST")
	r.HandleFunc("/api/devices/{id}", h.UpdateDevice).Methods("PUT")
	r.HandleFunc("/api/devices/{id}", h.DeleteDevice).Methods("DELETE")
	r.HandleFunc("/api/devices/{id}/toggle", h.ToggleDeviceEnable).Methods("POST")
	r.HandleFunc("/api/devices/{id}/execute", h.ExecuteDriverFunction).Methods("POST")

	r.HandleFunc("/api/northbound", h.GetNorthboundConfigs).Methods("GET")
	r.HandleFunc("/api/northbound", h.CreateNorthboundConfig).Methods("POST")
	r.HandleFunc("/api/northbound/{id}", h.UpdateNorthboundConfig).Methods("PUT")
	r.HandleFunc("/api/northbound/{id}", h.DeleteNorthboundConfig).Methods("DELETE")
	r.HandleFunc("/api/northbound/{id}/toggle", h.ToggleNorthboundEnable).Methods("POST")

	r.HandleFunc("/api/thresholds", h.GetThresholds).Methods("GET")
	r.HandleFunc("/api/thresholds", h.CreateThreshold).Methods("POST")
	r.HandleFunc("/api/thresholds/{id}", h.UpdateThreshold).Methods("PUT")
	r.HandleFunc("/api/thresholds/{id}", h.DeleteThreshold).Methods("DELETE")

	r.HandleFunc("/api/alarms", h.GetAlarmLogs).Methods("GET")
	r.HandleFunc("/api/alarms/{id}/acknowledge", h.AcknowledgeAlarm).Methods("POST")

	r.HandleFunc("/api/data", h.GetDataCache).Methods("GET")
	r.HandleFunc("/api/data/cache/{id}", h.GetDataCacheByDeviceID).Methods("GET")

	r.HandleFunc("/api/data/points/{id}", h.GetDataPoints).Methods("GET")
	r.HandleFunc("/api/data/points", h.GetLatestDataPoints).Methods("GET")
	r.HandleFunc("/api/data/history", h.GetHistoryData).Methods("GET")

	r.HandleFunc("/api/storage", h.GetStorageConfigs).Methods("GET")
	r.HandleFunc("/api/storage", h.CreateStorageConfig).Methods("POST")
	r.HandleFunc("/api/storage/{id}", h.UpdateStorageConfig).Methods("PUT")
	r.HandleFunc("/api/storage/{id}", h.DeleteStorageConfig).Methods("DELETE")
	r.HandleFunc("/api/storage/cleanup", h.CleanupData).Methods("POST")
	r.HandleFunc("/api/storage/run", h.CleanupDataByPolicy).Methods("POST")

	r.HandleFunc("/api/users", h.GetUsers).Methods("GET")
	r.HandleFunc("/api/users", h.CreateUser).Methods("POST")
	r.HandleFunc("/api/users/{id}", h.UpdateUser).Methods("PUT")
	r.HandleFunc("/api/users/{id}", h.DeleteUser).Methods("DELETE")
	r.HandleFunc("/api/users/password", h.ChangePassword).Methods("PUT")

	r.HandleFunc("/api/resources", h.GetResources).Methods("GET")
	r.HandleFunc("/api/resources", h.CreateResource).Methods("POST")
	r.HandleFunc("/api/resources/{id}", h.UpdateResource).Methods("PUT")
	r.HandleFunc("/api/resources/{id}", h.DeleteResource).Methods("DELETE")
	r.HandleFunc("/api/resources/{id}/toggle", h.ToggleResource).Methods("POST")
}
