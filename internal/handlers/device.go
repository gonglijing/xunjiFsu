package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
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
