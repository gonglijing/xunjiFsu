package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/auth"
)

// ==================== 认证相关 ====================

// Login GET显示登录页面
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 返回 SPA 外壳，由前端渲染登录页
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>HuShu 网关登录</title><link rel="stylesheet" href="/static/style.css"><script defer src="/static/dist/main.js"></script></head><body><div id="app-root"></div></body></html>`))
}

// LoginPost 处理登录
func (h *Handler) LoginPost(w http.ResponseWriter, r *http.Request) {
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

	// JSON API 返回 token，同时 cookie 已设置
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"token":"` + token + `"}`))
}

// Logout 登出
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authManager.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	session, _ := h.authManager.GetSession(r)
	if session == nil {
		WriteUnauthorized(w, "not authenticated")
		return
	}

	if err := auth.ChangePassword(session.UserID, req.OldPassword, req.NewPassword); err != nil {
		WriteBadRequest(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
}
