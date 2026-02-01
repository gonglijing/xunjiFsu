package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/auth"
)

// ==================== 认证相关 ====================

// Login GET显示登录页面
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>Login</title><link rel="stylesheet" href="/static/style.css"></head><body style="max-width:420px;margin:80px auto;color:#f1f5f9;"><h2>HuShu 智能网关</h2><form method="POST" action="/login" class="form" style="margin-top:24px;"><div class="form-group"><label class="form-label">用户名</label><input class="form-input" name="username" value="admin" /></div><div class="form-group"><label class="form-label">密码</label><input class="form-input" name="password" type="password" value="123456" /></div><button class="btn btn-primary" style="width:100%;">登录</button></form></body></html>`))
}

// LoginPost 处理登录
func (h *Handler) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteBadRequest(w, "Invalid request")
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if err := h.sessionManager.Login(w, r, username, password); err != nil {
		WriteUnauthorized(w, "Invalid credentials")
		return
	}

	// 设置 HX-Redirect 头，让 HTMX 自动处理重定向
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// Logout 登出
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessionManager.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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

	session, _ := h.sessionManager.GetSession(r)
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
