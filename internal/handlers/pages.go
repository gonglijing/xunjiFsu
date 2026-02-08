package handlers

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// ==================== 页面渲染 ====================

// Dashboard 仪表盘页面
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	h.SPA(w, r)
}

// RealTime 实时数据页面
func (h *Handler) RealTime(w http.ResponseWriter, r *http.Request) {
	h.SPA(w, r)
}

// History 历史数据页面
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	h.SPA(w, r)
}

// SPA 统一入口，返回最小 HTML，由前端 Preact 接管路由
func (h *Handler) SPA(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html><html><head><meta charset="utf-8"><title>HuShu智能网关</title><link rel="stylesheet" href="/static/style.css"><script defer src="/static/dist/main.js"></script></head><body><div id="app-root"></div></body></html>`))
}

// StatusData 状态数据结构
type StatusData struct {
	CollectorRunning bool            `json:"collector_running"`
	Devices          DeviceStats     `json:"devices"`
	Northbound       NorthboundStats `json:"northbound"`
	Alarms           AlarmStats      `json:"alarms"`
	Drivers          DriverStats     `json:"drivers"`
	Timestamp        time.Time       `json:"timestamp"`
}

// DeviceStats 设备统计
type DeviceStats struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
}

// NorthboundStats 北向统计
type NorthboundStats struct {
	Total   int `json:"total"`
	Enabled int `json:"enabled"`
}

// AlarmStats 报警统计
type AlarmStats struct {
	Total   int `json:"total"`
	Unacked int `json:"unacked"`
	Today   int `json:"today"`
}

// DriverStats 驱动统计
type DriverStats struct {
	Total int `json:"total"`
}

// GetStatus 获取系统状态
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	devices, _ := database.GetAllDevices()
	configs, _ := database.GetAllNorthboundConfigs()
	alarms, _ := database.GetRecentAlarmLogs(1000)
	drivers := h.driverManager.ListDrivers()
	now := time.Now()

	status := StatusData{
		CollectorRunning: h.collector.IsRunning(),
		Devices:          summarizeDeviceStats(devices),
		Northbound:       summarizeNorthboundStats(configs),
		Alarms:           summarizeAlarmStats(alarms, now),
		Drivers: DriverStats{
			Total: len(drivers),
		},
		Timestamp: now,
	}

	WriteSuccess(w, status)
}

// StartCollector 启动采集器
func (h *Handler) StartCollector(w http.ResponseWriter, r *http.Request) {
	if err := h.collector.Start(); err != nil {
		writeServerErrorWithLog(w, apiErrStartCollectorFailed, err)
		return
	}
	WriteSuccess(w, map[string]string{"status": "started"})
}

// StopCollector 停止采集器
func (h *Handler) StopCollector(w http.ResponseWriter, r *http.Request) {
	h.collector.Stop()
	WriteSuccess(w, map[string]string{"status": "stopped"})
}

// 模板渲染与解析已移除（前端接管）
