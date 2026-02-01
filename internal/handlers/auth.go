package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/auth"
)

// ==================== 认证相关 ====================

// Login GET显示登录页面
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "login.html", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
