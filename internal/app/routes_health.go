package app

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/httpapi"
)

func registerHealthRoutes(r *http.ServeMux) {
	r.HandleFunc("GET /health", httpapi.Health)
	r.HandleFunc("GET /ready", httpapi.Readiness)
	r.HandleFunc("GET /live", httpapi.Liveness)
	r.HandleFunc("GET /metrics", httpapi.Metrics)
}
