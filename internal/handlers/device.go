package handlers

import (
	"fmt"
	"net/http"

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

	// HTMX 请求，返回 HTML 片段
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		if err := tmpl.ExecuteTemplate(w, "devices.html", map[string]interface{}{"Devices": devices}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	WriteSuccess(w, devices)
}

// CreateDevice 创建设备
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := ParseRequest(r, &device); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	id, err := database.CreateDevice(&device)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	device.ID = id

	// 如果启用了采集，自动添加到采集器
	if device.Enabled == 1 {
		if err := h.collector.AddDevice(&device); err != nil {
			WriteServerError(w, err.Error())
			return
		}
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

	// 处理使能状态变化
	if device.Enabled == 1 {
		h.collector.AddDevice(&device)
	} else {
		h.collector.RemoveDevice(device.ID)
	}

	WriteSuccess(w, device)
}

// DeleteDevice 删除设备
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	// 先获取设备信息，删除时从采集器移除
	device, err := database.GetDeviceByID(id)
	if err == nil {
		h.collector.RemoveDevice(device.ID)
	}

	if err := database.DeleteDevice(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
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

	// 更新采集器
	if newState == 1 {
		h.collector.AddDevice(device)
	} else {
		h.collector.RemoveDevice(device.ID)
	}

	WriteSuccess(w, map[string]interface{}{
		"enabled": newState,
		"message": "Device enabled"[:7] + "disabled"[(newState*7):],
	})
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
		ResourceType: "serial",
		Config:       config,
		DeviceConfig: device.DeviceConfig,
	}

	// 调用驱动的指定函数
	result, err := h.driverManager.ExecuteDriver(*device.DriverID, req.Function, ctx)
	if err != nil {
		WriteServerError(w, fmt.Sprintf("Failed to execute %s: %v", req.Function, err))
		return
	}

	WriteSuccess(w, result)
}
