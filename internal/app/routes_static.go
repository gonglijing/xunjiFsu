package app

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/httpapi"
	"log/slog"
)

func buildRouter(h *httpapi.AuthHandler, apiDeps *apiRouteDeps, authManager *auth.JWTManager) *http.ServeMux {
	r := http.NewServeMux()

	staticDir := resolveStaticDir()
	registerStaticRoutes(r, staticDir)
	registerAPIRoutes(r, apiDeps, authManager)
	registerPageRoutes(r, h, authManager)
	registerHealthRoutes(r)

	return r
}

func resolveStaticDir() http.Dir {
	workDir, err := os.Getwd()
	if err != nil {
		slog.Warn("Failed to get working directory, using relative static path", "error", err)
		return http.Dir(filepath.Join("ui", "static"))
	}
	return http.Dir(filepath.Join(workDir, "ui", "static"))
}

func registerStaticRoutes(r *http.ServeMux, staticDir http.Dir) {
	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(staticDir)))
	r.Handle("/ui/static/", http.StripPrefix("/ui/static/", http.FileServer(staticDir)))
}
