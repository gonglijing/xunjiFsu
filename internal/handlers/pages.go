package handlers

import "net/http"

// ==================== 页面渲染 ====================

// SPA 统一入口，返回最小 HTML，由前端 Preact 接管路由
func (h *Handler) SPA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>HuShu智能网关</title><link rel="stylesheet" href="/static/style.css"><script defer src="/static/dist/main.js"></script></head><body><div id="app-root"></div></body></html>`))
}

// 模板渲染与解析已移除（前端接管）
