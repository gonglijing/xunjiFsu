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

	r.PathPrefix("/").
		Handler(authManager.RequireAuth(http.HandlerFunc(h.SPA))).
		Methods("GET").
		MatcherFunc(func(req *http.Request, _ *mux.RouteMatch) bool {
			path := req.URL.Path
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
