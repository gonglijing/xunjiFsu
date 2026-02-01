package handlers

import (
	"bytes"
	"net/http"
	"html/template"
	"path/filepath"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

var tmpl *template.Template

// ==================== 页面渲染 ====================

// Dashboard 仪表盘页面
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":           "仪表盘",
		"Active":          "dashboard",
		"ContentTemplate": "dashboard-content",
	}
	renderTemplate(w, "dashboard.html", "dashboard-content", data)
}

// RealTime 实时数据页面
func (h *Handler) RealTime(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":           "实时数据",
		"Active":          "realtime",
		"ContentTemplate": "realtime-content",
	}
	renderTemplate(w, "realtime.html", "realtime-content", data)
}

// History 历史数据页面
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":           "历史数据",
		"Active":          "history",
		"ContentTemplate": "history-content",
	}
	renderTemplate(w, "history.html", "history-content", data)
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
	Total      int `json:"total"`
	Unacked    int `json:"unacked"`
	Today      int `json:"today"`
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

// renderTemplate 渲染模板（支持继承 base.html）
func renderTemplate(w http.ResponseWriter, page string, name string, data interface{}) {
	// 先将 data 转换为 map
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		dataMap = make(map[string]interface{})
	}

	// 检查是否存在 base 模板，如果存在则先渲染内容，再渲染 base
	if tmpl.Lookup("base") != nil {
		// 渲染内容模板到缓冲区
		var contentBuf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&contentBuf, name, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// 将渲染后的内容添加到数据中（使用 template.HTML 避免转义）
		dataMap["Content"] = template.HTML(contentBuf.String())
		// 渲染 base 模板
		if err := tmpl.ExecuteTemplate(w, "base", dataMap); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	// 否则直接渲染模板
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// InitTemplates 初始化模板
func InitTemplates(pattern string) error {
	var err error
	
	// 使用 template.ParseFiles 加载所有模板
	files, err := filepath.Glob(pattern + "/*.html")
	if err != nil {
		return err
	}
	
	// 添加 fragments 目录中的模板
	fragmentFiles, err := filepath.Glob(pattern + "/fragments/*.html")
	if err != nil {
		return err
	}
	
	files = append(files, fragmentFiles...)
	
	tmpl, err = template.ParseFiles(files...)
	if err != nil {
		return err
	}
	
	return nil
}
