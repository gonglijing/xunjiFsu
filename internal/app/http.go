package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gonglijing/xunjiFsu/internal/logger"

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
	allowedOrigins := cfg.GetAllowedOrigins()
	loggingHandler := requestLoggingMiddleware(router)
	gzipHandler := handlers.GzipMiddleware(loggingHandler)
	corsHandler := corsMiddleware(allowedOrigins)(gzipHandler)

	timeoutConfig := handlers.DefaultTimeoutConfig()
	timeoutConfig.ReadTimeout = cfg.HTTPReadTimeout
	timeoutConfig.WriteTimeout = cfg.HTTPWriteTimeout
	timeoutConfig.IdleTimeout = cfg.HTTPIdleTimeout

	return handlers.TimeoutMiddleware(timeoutConfig)(corsHandler)
}

func requestLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		logger.Printf("%s %s %d %d %v", r.Method, r.URL.RequestURI(), rw.statusCode, rw.bytes, time.Since(start))
	})
}

func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	allowSet := make(map[string]struct{}, len(origins))
	allowAll := false
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowAll = true
			continue
		}
		allowSet[trimmed] = struct{}{}
	}

	allowMethods := "GET, POST, PUT, DELETE, OPTIONS"
	allowHeaders := "Content-Type, Authorization"

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if allowedOrigin(origin, allowSet, allowAll) {
				header := w.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Methods", allowMethods)
				header.Set("Access-Control-Allow-Headers", allowHeaders)
				header.Set("Access-Control-Allow-Credentials", "true")
				header.Set("Vary", appendVaryHeader(header.Get("Vary"), "Origin"))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func allowedOrigin(origin string, allowSet map[string]struct{}, allowAll bool) bool {
	if origin == "" {
		return false
	}
	if allowAll {
		return true
	}
	_, ok := allowSet[origin]
	return ok
}

func appendVaryHeader(existing string, value string) string {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return existing
	}
	if existing == "" {
		return trimmedValue
	}
	for _, part := range strings.Split(existing, ",") {
		if strings.EqualFold(strings.TrimSpace(part), trimmedValue) {
			return existing
		}
	}
	return existing + ", " + trimmedValue
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(p []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytes += n
	return n, err
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
