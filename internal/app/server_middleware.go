package app

import (
	"net/http"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/httpapi"
	"github.com/gonglijing/xunjiFsu/internal/logger"
)

const (
	corsAllowMethods = "GET, POST, PUT, DELETE, OPTIONS"
	corsAllowHeaders = "Content-Type, Authorization"
)

func buildHandlerChain(cfg *config.Config, router *http.ServeMux) http.Handler {
	allowedOrigins := cfg.GetAllowedOrigins()
	loggingHandler := requestLoggingMiddleware(router)
	gzipHandler := httpapi.GzipMiddleware(loggingHandler)
	corsHandler := corsMiddleware(allowedOrigins)(gzipHandler)

	timeoutConfig := httpapi.DefaultTimeoutConfig()
	timeoutConfig.ReadTimeout = cfg.HTTPReadTimeout
	timeoutConfig.WriteTimeout = cfg.HTTPWriteTimeout
	timeoutConfig.IdleTimeout = cfg.HTTPIdleTimeout

	return httpapi.TimeoutMiddleware(timeoutConfig)(corsHandler)
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
	allowSet, allowAll := buildAllowedOriginSet(origins)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if isAllowedOrigin(origin, allowSet, allowAll) {
				header := w.Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Methods", corsAllowMethods)
				header.Set("Access-Control-Allow-Headers", corsAllowHeaders)
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

func buildAllowedOriginSet(origins []string) (map[string]struct{}, bool) {
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

	return allowSet, allowAll
}

func isAllowedOrigin(origin string, allowSet map[string]struct{}, allowAll bool) bool {
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
