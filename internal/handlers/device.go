package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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

	// 资源映射（用于展示 resource 名称/路径）
	resources, _ := database.ListResources()
	resourceMap := make(map[int64]*models.Resource)
	for _, res := range resources {
		resourceMap[res.ID] = res
	}

	// 为每个设备填充驱动名称
	for _, device := range devices {
		if device.DriverID != nil {
			// 从文件名获取驱动名称
			device.DriverName = h.getDriverNameByID(*device.DriverID)
		}
		if device.ResourceID != nil {
			if res, ok := resourceMap[*device.ResourceID]; ok {
				device.ResourceName = res.Name
				device.ResourceType = res.Type
				device.ResourcePath = res.Path
			}
		}
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

	WriteSuccess(w, map[string]interface{}{
		"enabled": newState,
	})
}

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
	// 补充设备信息
	config["device_address"] = device.DeviceAddress
	if req.Function == "" || req.Function == "collect" {
		config["func_name"] = "read"
	} else {
		config["func_name"] = req.Function
	}

	// 创建驱动上下文
	resourceID := int64(0)
	if device.ResourceID != nil {
		resourceID = *device.ResourceID
	}
	resourceType := device.ResourceType
	if resourceType == "" {
		resourceType = device.DriverType
	}
	ctx := &driver.DriverContext{
		DeviceID:     device.ID,
		DeviceName:   device.Name,
		ResourceID:   resourceID,
		ResourceType: resourceType,
		Config:       config,
		DeviceConfig: "",
	}

	// 调用驱动的指定函数（collect -> handle）
	funcName := req.Function
	if funcName == "collect" || funcName == "" {
		funcName = "handle"
	}

	result, err := h.driverManager.ExecuteDriver(*device.DriverID, funcName, ctx)
	if err != nil {
		WriteServerError(w, fmt.Sprintf("Failed to execute %s: %v", req.Function, err))
		return
	}

	WriteSuccess(w, result)
}

// GetDeviceWritables 返回设备驱动声明的可写寄存器元数据（来自 Driver.ConfigSchema.writable）
func (h *Handler) GetDeviceWritables(w http.ResponseWriter, r *http.Request) {
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
	if device.DriverID == nil {
		WriteSuccess(w, []interface{}{})
		return
	}

	driverModel, err := database.GetDriverByID(*device.DriverID)
	if err != nil {
		WriteServerError(w, "driver not found")
		return
	}

	var cfg struct {
		Writable []interface{} `json:"writable"`
	}
	if driverModel.ConfigSchema != "" {
		_ = json.Unmarshal([]byte(driverModel.ConfigSchema), &cfg)
	}

	WriteSuccess(w, cfg.Writable)
}
