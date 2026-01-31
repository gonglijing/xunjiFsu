package handlers

import (
	"net/http"
)

// ==================== 页面路由 ====================

// Dashboard 仪表盘页面
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/pages/dashboard.html")
}

// RealTime 实时数据页面
func (h *Handler) RealTime(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/pages/realtime.html")
}

// History 历史数据页面
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/pages/history.html")
}

// GetStatus 获取系统状态
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"collector_running": h.collector.IsRunning(),
		"loaded_drivers":    len(h.driverManager.ListDrivers()),
		"resource_count":    h.resourceMgr.GetResourceCount(),
		"northbound_count":  h.northboundMgr.GetAdapterCount(),
	}
	WriteSuccess(w, status)
}

// StartCollector 启动采集器
func (h *Handler) StartCollector(w http.ResponseWriter, r *http.Request) {
	if err := h.collector.Start(); err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, map[string]string{"status": "started"})
}

// StopCollector 停止采集器
func (h *Handler) StopCollector(w http.ResponseWriter, r *http.Request) {
	h.collector.Stop()
	WriteSuccess(w, map[string]string{"status": "stopped"})
}
