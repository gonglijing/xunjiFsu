package app

import (
	"net/http"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/httpapi"
	"github.com/gonglijing/xunjiFsu/internal/platform/auth"
)

var spaBlockedPrefixes = []string{
	"/api",
	"/static/",
	"/ui/static/",
}

var spaBlockedPaths = map[string]struct{}{
	"/health":  {},
	"/ready":   {},
	"/live":    {},
	"/metrics": {},
	"/login":   {},
	"/logout":  {},
}

func registerPageRoutes(r *http.ServeMux, h *httpapi.AuthHandler, authManager *auth.JWTManager) {
	r.HandleFunc("GET /login", h.Login)
	r.HandleFunc("POST /login", h.LoginPost)
	r.HandleFunc("GET /logout", h.Logout)

	r.Handle("/", authManager.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !shouldServeSPA(req) {
			http.NotFound(w, req)
			return
		}
		h.SPA(w, req)
	})))
}

func shouldServeSPA(req *http.Request) bool {
	if req == nil || req.Method != http.MethodGet {
		return false
	}

	path := req.URL.Path
	for _, prefix := range spaBlockedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	_, blocked := spaBlockedPaths[path]
	return !blocked
}
