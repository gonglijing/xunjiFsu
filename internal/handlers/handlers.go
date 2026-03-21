package handlers

import (
	"github.com/gonglijing/xunjiFsu/internal/auth"
)

// Handler Web处理器
type Handler struct {
	authManager *auth.JWTManager
}

// NewHandler 创建处理器
func NewHandler(authManager *auth.JWTManager) *Handler {
	return &Handler{
		authManager: authManager,
	}
}
