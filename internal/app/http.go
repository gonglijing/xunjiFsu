package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gonglijing/xunjiFsu/internal/logger"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func buildRouter(h *handlers.Handler, authManager *auth.JWTManager) *mux.Router {
	r := mux.NewRouter()

	staticDir := resolveStaticDir()
	registerStaticRoutes(r, staticDir)
	registerAPIRoutes(r, h, authManager)
	registerPageRoutes(r, h, authManager)
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

	loggingHandler := gorillaHandlers.LoggingHandler(logger.Output(), router)
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
		return http.Dir(filepath.Join("ui", "static"))
	}
	return http.Dir(filepath.Join(workDir, "ui", "static"))
}

func registerStaticRoutes(r *mux.Router, staticDir http.Dir) {
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(staticDir)))
	r.PathPrefix("/ui/static/").Handler(http.StripPrefix("/ui/static/", http.FileServer(staticDir)))
}

func registerPageRoutes(r *mux.Router, h *handlers.Handler, authManager *auth.JWTManager) {
	r.HandleFunc("/login", h.Login).Methods("GET")
	r.HandleFunc("/login", h.LoginPost).Methods("POST")
	r.HandleFunc("/logout", h.Logout).Methods("GET")
	// SPA 入口，所有非 API/静态 GET 请求交给前端路由
	r.PathPrefix("/").
		Handler(authManager.RequireAuth(http.HandlerFunc(h.SPA))).
		Methods("GET").
		MatcherFunc(func(req *http.Request, _ *mux.RouteMatch) bool {
			path := req.URL.Path
			// 排除 API 和静态资源
			if strings.HasPrefix(path, "/api") ||
				strings.HasPrefix(path, "/static/") ||
				strings.HasPrefix(path, "/ui/static/") {
				return false
			}
			return true
		})
}

func registerHealthRoutes(r *mux.Router) {
	r.HandleFunc("/health", handlers.Health).Methods("GET")
	r.HandleFunc("/ready", handlers.Readiness).Methods("GET")
	r.HandleFunc("/live", handlers.Liveness).Methods("GET")
	r.HandleFunc("/metrics", handlers.Metrics).Methods("GET")
}

func registerAPIRoutes(r *mux.Router, h *handlers.Handler, authManager *auth.JWTManager) {
	// 所有 /api 路径统一做鉴权
	api := r.PathPrefix("/api").Subrouter()
	api.Use(authManager.RequireAuth)

	api.HandleFunc("/status", h.GetStatus).Methods("GET")

	api.HandleFunc("/collector/start", h.StartCollector).Methods("POST")
	api.HandleFunc("/collector/stop", h.StopCollector).Methods("POST")

	api.HandleFunc("/drivers", h.GetDrivers).Methods("GET")
	api.HandleFunc("/drivers/files", h.ListDriverFiles).Methods("GET")
	api.HandleFunc("/drivers", h.CreateDriver).Methods("POST")
	api.HandleFunc("/drivers/{id}", h.UpdateDriver).Methods("PUT")
	api.HandleFunc("/drivers/{id}", h.DeleteDriver).Methods("DELETE")
	api.HandleFunc("/drivers/{id}/download", h.DownloadDriver).Methods("GET")
	api.HandleFunc("/drivers/upload", h.UploadDriverFile).Methods("POST")

	api.HandleFunc("/devices", h.GetDevices).Methods("GET")
	api.HandleFunc("/devices", h.CreateDevice).Methods("POST")
	api.HandleFunc("/devices/{id}", h.UpdateDevice).Methods("PUT")
	api.HandleFunc("/devices/{id}", h.DeleteDevice).Methods("DELETE")
	api.HandleFunc("/devices/{id}/toggle", h.ToggleDeviceEnable).Methods("POST")
	api.HandleFunc("/devices/{id}/execute", h.ExecuteDriverFunction).Methods("POST")
	api.HandleFunc("/devices/{id}/writables", h.GetDeviceWritables).Methods("GET")

	api.HandleFunc("/northbound", h.GetNorthboundConfigs).Methods("GET")
	api.HandleFunc("/northbound", h.CreateNorthboundConfig).Methods("POST")
	api.HandleFunc("/northbound/{id}", h.UpdateNorthboundConfig).Methods("PUT")
	api.HandleFunc("/northbound/{id}", h.DeleteNorthboundConfig).Methods("DELETE")
	api.HandleFunc("/northbound/{id}/toggle", h.ToggleNorthboundEnable).Methods("POST")

	api.HandleFunc("/thresholds", h.GetThresholds).Methods("GET")
	api.HandleFunc("/thresholds", h.CreateThreshold).Methods("POST")
	api.HandleFunc("/thresholds/{id}", h.UpdateThreshold).Methods("PUT")
	api.HandleFunc("/thresholds/{id}", h.DeleteThreshold).Methods("DELETE")

	api.HandleFunc("/alarms", h.GetAlarmLogs).Methods("GET")
	api.HandleFunc("/alarms/{id}/acknowledge", h.AcknowledgeAlarm).Methods("POST")

	api.HandleFunc("/data", h.GetDataCache).Methods("GET")
	api.HandleFunc("/data/cache/{id}", h.GetDataCacheByDeviceID).Methods("GET")

	api.HandleFunc("/data/history", h.GetHistoryData).Methods("GET")

	api.HandleFunc("/storage", h.GetStorageConfigs).Methods("GET")
	api.HandleFunc("/storage", h.CreateStorageConfig).Methods("POST")
	api.HandleFunc("/storage/{id}", h.UpdateStorageConfig).Methods("PUT")
	api.HandleFunc("/storage/{id}", h.DeleteStorageConfig).Methods("DELETE")
	api.HandleFunc("/storage/cleanup", h.CleanupData).Methods("POST")
	api.HandleFunc("/storage/run", h.CleanupDataByPolicy).Methods("POST")

	api.HandleFunc("/users", h.GetUsers).Methods("GET")
	api.HandleFunc("/users", h.CreateUser).Methods("POST")
	api.HandleFunc("/users/{id}", h.UpdateUser).Methods("PUT")
	api.HandleFunc("/users/{id}", h.DeleteUser).Methods("DELETE")
	api.HandleFunc("/users/password", h.ChangePassword).Methods("PUT")

	api.HandleFunc("/resources", h.GetResources).Methods("GET")
	api.HandleFunc("/resources", h.CreateResource).Methods("POST")
	api.HandleFunc("/resources/{id}", h.UpdateResource).Methods("PUT")
	api.HandleFunc("/resources/{id}", h.DeleteResource).Methods("DELETE")
	api.HandleFunc("/resources/{id}/toggle", h.ToggleResource).Methods("POST")

	// 网关配置
	api.HandleFunc("/gateway/config", h.GetGatewayConfig).Methods("GET")
	api.HandleFunc("/gateway/config", h.UpdateGatewayConfig).Methods("PUT")
}
