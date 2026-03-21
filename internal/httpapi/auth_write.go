package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/auth"
)

// AuthHandler 认证与页面处理器
type AuthHandler struct {
	authManager *auth.JWTManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(authManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{authManager: authManager}
}

// Login GET 显示登录页面（SPA 外壳）
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>HuShu 网关登录</title><link rel="stylesheet" href="/static/style.css"><script defer src="/static/dist/main.js"></script></head><body><div id="app-root"></div></body></html>`))
}

// LoginPost 处理登录
func (h *AuthHandler) LoginPost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := ParseRequest(r, &req); err != nil || req.Username == "" {
		_ = r.ParseForm()
		req.Username = r.PostFormValue("username")
		req.Password = r.PostFormValue("password")
	}

	token, err := h.authManager.Login(w, r, req.Username, req.Password)
	if err != nil {
		WriteUnauthorized(w, "Invalid credentials")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"token":"` + token + `"}`))
}

// Logout 登出
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authManager.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// SPA 单页应用入口，返回最小 HTML，由前端接管路由
func (h *AuthHandler) SPA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>HuShu智能网关</title><link rel="stylesheet" href="/static/style.css"><script defer src="/static/dist/main.js"></script></head><body><div id="app-root"></div></body></html>`))
}
