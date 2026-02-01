package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 设备管理 ====================

// GetDevices 获取所有设备
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 获取驱动列表并创建映射
	drivers := h.getAvailableDrivers()
	driverMap := make(map[int64]string)
	for _, d := range drivers {
		driverMap[0] = d.Name // 使用名称作为标识
	}

	// 为每个设备填充驱动名称
	for _, device := range devices {
		if device.DriverID != nil {
			// 从文件名获取驱动名称
			device.DriverName = h.getDriverNameByID(*device.DriverID)
		}
	}

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "devices.html", map[string]interface{}{"Devices": devices, "Drivers": drivers}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	WriteSuccess(w, devices)
}

// getDriverNameByID 根据ID获取驱动名称（从文件名）
func (h *Handler) getDriverNameByID(driverID int64) string {
	// 从数据库获取驱动信息
	drv, err := database.GetDriverByID(driverID)
	if err != nil {
		return fmt.Sprintf("驱动 #%d", driverID)
	}
	// 从 file_path 提取文件名作为驱动名称
	if drv.FilePath != "" {
		name := filepath.Base(drv.FilePath)
		return strings.TrimSuffix(name, ".wasm")
	}
	return drv.Name
}

// getAvailableDrivers 获取可用的驱动列表
func (h *Handler) getAvailableDrivers() []*models.Driver {
	drivers := []*models.Driver{}

	// 从 drivers 目录扫描 .wasm 文件
	driversDir := "drivers"
	entries, err := os.ReadDir(driversDir)
	if err != nil {
		return drivers
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".wasm") {
			driver := &models.Driver{
				Name:     strings.TrimSuffix(entry.Name(), ".wasm"),
				FilePath: filepath.Join(driversDir, entry.Name()),
			}
			drivers = append(drivers, driver)
		}
	}
	return drivers
}

// CreateDevice 创建设备
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, "Invalid request body: "+err.Error())
		return
	}

	id, err := database.CreateDevice(&device)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	device.ID = id

	// 注意：设备状态由采集器定时管理，不再直接调用采集

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		drivers := h.getAvailableDrivers()
		if err := tmpl.ExecuteTemplate(w, "devices.html", map[string]interface{}{"Devices": []*models.Device{&device}, "Drivers": drivers}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	WriteCreated(w, device)
}

// UpdateDevice 更新设备
func (h *Handler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	device.ID = id
	if err := database.UpdateDevice(&device); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 注意：设备状态由采集器定时管理，不再直接调用采集

	WriteSuccess(w, device)
}

// DeleteDevice 删除设备
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	// 注意：设备状态由采集器定时管理，删除时不需要手动移除

	if err := database.DeleteDevice(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 返回空响应，HTMX 会移除该行
	w.WriteHeader(http.StatusNoContent)
}

// ToggleDeviceEnable 切换设备使能状态
func (h *Handler) ToggleDeviceEnable(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFound(w, "Device not found")
		return
	}

	// 切换状态
	newState := 0
	if device.Enabled == 0 {
		newState = 1
	}

	if err := database.UpdateDeviceEnabled(id, newState); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 注意：设备状态由采集器定时管理，不再直接调用采集

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		device.Enabled = newState
		renderDeviceRow(w, device)
		return
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": newState,
	})
}

// renderDeviceRow 渲染设备行 HTML
func renderDeviceRow(w http.ResponseWriter, device *models.Device) {
	statusText := "停止采集"
	statusClass := "btn btn-success"
	badgeClass := "badge badge-stopped"
	
	if device.Enabled == 1 {
		statusText = "采集中"
		statusClass = "btn btn-danger"
		badgeClass = "badge badge-running"
	}

	driverName := device.DriverName
	if driverName == "" && device.DriverID != nil {
		driverName = fmt.Sprintf("驱动 #%d", *device.DriverID)
	}

	fmt.Fprintf(w, `<tr id="device-row-%d">
		<td>%d</td>
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td>%dms</td>
		<td>%s</td>
		<td><span class="%s">%s</span></td>
		<td>
			<button hx-post="/api/devices/%d/toggle" hx-target="#device-row-%d" hx-swap="outerHTML" class="%s" style="padding: 4px 8px;">%s</button>
			<button hx-delete="/api/devices/%d" hx-target="#device-row-%d" hx-swap="outerHTML" hx-confirm="确定删除?" class="btn btn-danger" style="padding: 4px 8px;">删除</button>
		</td>
	</tr>`,
		device.ID,
		device.ID,
		device.Name,
		device.DriverType,
		device.DeviceAddress,
		driverName,
		device.CollectInterval,
		statusText,
		badgeClass, statusText,
		device.ID, device.ID, statusClass, statusText,
		device.ID, device.ID,
	)
}

// ExecuteDriverFunction 执行驱动的任意函数
//
// 请求体:
//
//	{
//	    "function": "change_address",  // 函数名
//	    "params": {                    // 参数，将作为 config 传递给驱动
//	        "old_addr": 1,
//	        "new_addr": 2
//	    }
//	}
func (h *Handler) ExecuteDriverFunction(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	// 解析请求体
	var req struct {
		Function string                 `json:"function"`
		Params   map[string]interface{} `json:"params"`
	}
	if err := ParseRequest(r, &req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Function == "" {
		WriteBadRequest(w, "Function name is required")
		return
	}

	device, err := database.GetDeviceByID(id)
	if err != nil {
		WriteNotFound(w, "Device not found")
		return
	}

	if device.DriverID == nil {
		WriteBadRequest(w, "Device has no driver")
		return
	}

	// 将 params 转换为 config (map[string]string)
	config := make(map[string]string)
	for k, v := range req.Params {
		switch val := v.(type) {
		case string:
			config[k] = val
		case float64:
			config[k] = fmt.Sprintf("%.0f", val) // JSON number 解析为 float64
		case int:
			config[k] = fmt.Sprintf("%d", val)
		case bool:
			config[k] = fmt.Sprintf("%t", val)
		default:
			config[k] = fmt.Sprintf("%v", v)
		}
	}

	// 创建驱动上下文
	ctx := &driver.DriverContext{
		DeviceID:     device.ID,
		DeviceName:   device.Name,
		ResourceID:   0,
		ResourceType: device.DriverType,
		Config:       config,
		DeviceConfig: "",
	}

	// 调用驱动的指定函数
	result, err := h.driverManager.ExecuteDriver(*device.DriverID, req.Function, ctx)
	if err != nil {
		WriteServerError(w, fmt.Sprintf("Failed to execute %s: %v", req.Function, err))
		return
	}

	WriteSuccess(w, result)
}
