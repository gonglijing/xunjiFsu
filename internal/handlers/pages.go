package handlers

import (
	"net/http"
	"html/template"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

var tmpl *template.Template

// ==================== 页面渲染 ====================

// Dashboard 仪表盘页面
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "仪表盘",
		"Active": "dashboard",
	}
	renderTemplate(w, "dashboard.html", data)
}

// RealTime 实时数据页面
func (h *Handler) RealTime(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "实时数据",
		"Active": "realtime",
	}
	renderTemplate(w, "realtime.html", data)
}

// History 历史数据页面
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "历史数据",
		"Active": "history",
	}
	renderTemplate(w, "history.html", data)
}

// StatusData 状态数据结构
type StatusData struct {
	CollectorRunning bool            `json:"collector_running"`
	Devices          DeviceStats     `json:"devices"`
	Northbound       NorthboundStats `json:"northbound"`
	Alarms           AlarmStats      `json:"alarms"`
	Resources        ResourceStats   `json:"resources"`
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
	Total      int `json:"total"`
	Unacked    int `json:"unacked"`
	Today      int `json:"today"`
}

// ResourceStats 资源统计
type ResourceStats struct {
	Total int `json:"total"`
}

// DriverStats 驱动统计
type DriverStats struct {
	Total int `json:"total"`
}

// GetStatus 获取系统状态
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	// 获取设备统计
	devices, _ := database.GetAllDevices()
	deviceTotal := len(devices)
	deviceEnabled := 0
	for _, d := range devices {
		if d.Enabled == 1 {
			deviceEnabled++
		}
	}

	// 获取北向配置统计
	configs, _ := database.GetAllNorthboundConfigs()
	northboundTotal := len(configs)
	northboundEnabled := 0
	for _, c := range configs {
		if c.Enabled == 1 {
			northboundEnabled++
		}
	}

	// 获取报警统计
	alarms, _ := database.GetRecentAlarmLogs(1000)
	alarmTotal := len(alarms)
	alarmUnacked := 0
	alarmToday := 0
	today := time.Now().Truncate(24 * time.Hour)
	for _, a := range alarms {
		if a.Acknowledged == 0 {
			alarmUnacked++
		}
		if a.TriggeredAt.After(today) {
			alarmToday++
		}
	}

	// 资源统计
	resourceCount := h.resourceMgr.GetResourceCount()

	// 驱动统计
	drivers := h.driverManager.ListDrivers()
	driverTotal := len(drivers)

	status := StatusData{
		CollectorRunning: h.collector.IsRunning(),
		Devices: DeviceStats{
			Total:   deviceTotal,
			Enabled: deviceEnabled,
		},
		Northbound: NorthboundStats{
			Total:   northboundTotal,
			Enabled: northboundEnabled,
		},
		Alarms: AlarmStats{
			Total:   alarmTotal,
			Unacked: alarmUnacked,
			Today:   alarmToday,
		},
		Resources: ResourceStats{
			Total: resourceCount,
		},
		Drivers: DriverStats{
			Total: driverTotal,
		},
		Timestamp: time.Now(),
	}

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "status-cards.html", status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
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

// renderTemplate 渲染模板
func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// InitTemplates 初始化模板
func InitTemplates(pattern string) error {
	var err error
	tmpl, err = template.ParseGlob(pattern + "/*.html")
	if err != nil {
		return err
	}
	return nil
}
