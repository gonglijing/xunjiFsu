package app

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/handlers"
)

func registerHealthRoutes(r *http.ServeMux) {
	r.HandleFunc("GET /health", handlers.Health)
	r.HandleFunc("GET /ready", handlers.Readiness)
	r.HandleFunc("GET /live", handlers.Liveness)
	r.HandleFunc("GET /metrics", handlers.Metrics)
}
