package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"gogw/internal/auth"
	"gogw/internal/collector"
	"gogw/internal/database"
	"gogw/internal/driver"
	"gogw/internal/models"
	"gogw/internal/northbound"
	"gogw/internal/resource"
)

// Handler Web处理器
type Handler struct {
	sessionManager *auth.SessionManager
	collector      *collector.Collector
	driverManager  *driver.DriverManager
	resourceMgr    *resource.ResourceManagerImpl
	northboundMgr  *northbound.NorthboundManager
}

// NewHandler 创建处理器
func NewHandler(
	sessionManager *auth.SessionManager,
	collector *collector.Collector,
	driverManager *driver.DriverManager,
	resourceMgr *resource.ResourceManagerImpl,
	northboundMgr *northbound.NorthboundManager,
) *Handler {
	return &Handler{
		sessionManager: sessionManager,
		collector:      collector,
		driverManager:  driverManager,
		resourceMgr:    resourceMgr,
		northboundMgr:  northboundMgr,
	}
}

// ==================== 认证相关 ====================

// Login GET显示登录页面
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/pages/login.html")
}

// LoginPost 处理登录
func (h *Handler) LoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	username := r.PostFormValue("username")
	password := r.PostFormValue("password")

	if err := h.sessionManager.Login(w, r, username, password); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout 登出
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessionManager.Logout(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// ==================== 仪表盘 ====================

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
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ==================== 资源管理 ====================

// GetResources 获取所有资源
func (h *Handler) GetResources(w http.ResponseWriter, r *http.Request) {
	resources, err := database.GetAllResources()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resources)
}

// CreateResource 创建资源
func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var resource models.Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateResource(&resource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resource.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resource)
}

// UpdateResource 更新资源
func (h *Handler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var resource models.Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resource.ID = id
	if err := database.UpdateResource(&resource); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteResource 删除资源
func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := database.DeleteResource(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// OpenResource 打开资源
func (h *Handler) OpenResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	resource, err := database.GetResourceByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.resourceMgr.OpenResource(resource); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CloseResource 关闭资源
func (h *Handler) CloseResource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	resource, err := database.GetResourceByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.resourceMgr.CloseResource(id, resource.Type); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ==================== 驱动管理 ====================

// GetDrivers 获取所有驱动
func (h *Handler) GetDrivers(w http.ResponseWriter, r *http.Request) {
	drivers, err := database.GetAllDrivers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drivers)
}

// CreateDriver 创建驱动
func (h *Handler) CreateDriver(w http.ResponseWriter, r *http.Request) {
	var driver models.Driver
	if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateDriver(&driver)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	driver.ID = id

	// 加载驱动
	if err := h.driverManager.LoadDriver(&driver); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(driver)
}

// UpdateDriver 更新驱动
func (h *Handler) UpdateDriver(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var driver models.Driver
	if err := json.NewDecoder(r.Body).Decode(&driver); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	driver.ID = id
	if err := database.UpdateDriver(&driver); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 如果驱动已加载，重新加载
	if h.driverManager.IsLoaded(id) {
		h.driverManager.UnloadDriver(id)
		h.driverManager.LoadDriver(&driver)
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteDriver 删除驱动
func (h *Handler) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 卸载驱动
	if h.driverManager.IsLoaded(id) {
		h.driverManager.UnloadDriver(id)
	}

	if err := database.DeleteDriver(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ==================== 设备管理 ====================

// GetDevices 获取所有设备
func (h *Handler) GetDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := database.GetAllDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// CreateDevice 创建设备
func (h *Handler) CreateDevice(w http.ResponseWriter, r *http.Request) {
	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateDevice(&device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	device.ID = id

	// 添加到采集器
	if device.Enabled == 1 {
		h.collector.AddDevice(&device)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(device)
}

// UpdateDevice 更新设备
func (h *Handler) UpdateDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	device.ID = id
	if err := database.UpdateDevice(&device); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 更新采集器
	h.collector.UpdateDevice(&device)

	w.WriteHeader(http.StatusOK)
}

// DeleteDevice 删除设备
func (h *Handler) DeleteDevice(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 从采集器移除
	h.collector.RemoveDevice(id)

	if err := database.DeleteDevice(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ==================== 北向配置管理 ====================

// NorthboundConfigResponse 北向配置响应（包含解析的配置字段）
type NorthboundConfigResponse struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Enabled        int    `json:"enabled"`
	UploadInterval int    `json:"upload_interval"`
	ProductKey     string `json:"productKey,omitempty"`
	DeviceKey      string `json:"deviceKey,omitempty"`
	ServerURL      string `json:"serverUrl,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// GetNorthboundConfigs 获取所有北向配置
func (h *Handler) GetNorthboundConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转换为响应结构，解析配置字段
	response := make([]NorthboundConfigResponse, 0, len(configs))
	for _, config := range configs {
		resp := NorthboundConfigResponse{
			ID:             config.ID,
			Name:           config.Name,
			Type:           config.Type,
			Enabled:        config.Enabled,
			UploadInterval: config.UploadInterval,
			CreatedAt:      config.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:      config.UpdatedAt.Format("2006-01-02 15:04:05"),
		}

		// 解析配置JSON
		if config.Type == "xunji" {
			var xunjiConfig models.XunJiConfig
			if err := json.Unmarshal([]byte(config.Config), &xunjiConfig); err == nil {
				resp.ProductKey = xunjiConfig.ProductKey
				resp.DeviceKey = xunjiConfig.DeviceKey
				resp.ServerURL = xunjiConfig.ServerURL
			}
		}

		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateNorthboundConfig 创建北向配置
func (h *Handler) CreateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	var config models.NorthboundConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateNorthboundConfig(&config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config.ID = id

	// 如果启用，注册适配器
	if config.Enabled == 1 {
		h.registerNorthboundAdapter(&config)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateNorthboundConfig 更新北向配置
func (h *Handler) UpdateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var config models.NorthboundConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.ID = id
	if err := database.UpdateNorthboundConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 重新注册适配器
	h.northboundMgr.UnregisterAdapter(config.Name)
	if config.Enabled == 1 {
		h.registerNorthboundAdapter(&config)
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteNorthboundConfig 删除北向配置
func (h *Handler) DeleteNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 注销适配器
	h.northboundMgr.UnregisterAdapter(config.Name)

	if err := database.DeleteNorthboundConfig(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// registerNorthboundAdapter 注册北向适配器
func (h *Handler) registerNorthboundAdapter(config *models.NorthboundConfig) {
	var adapter northbound.Northbound

	switch config.Type {
	case "xunji":
		adapter = northbound.NewXunJiAdapter()
	case "http":
		adapter = northbound.NewHTTPAdapter()
	case "mqtt":
		adapter = northbound.NewMQTTAdapter()
	default:
		return
	}

	if err := adapter.Initialize(config.Config); err != nil {
		return
	}

	h.northboundMgr.RegisterAdapter(config.Name, adapter)
}

// ==================== 阈值管理 ====================

// GetThresholds 获取所有阈值
func (h *Handler) GetThresholds(w http.ResponseWriter, r *http.Request) {
	thresholds, err := database.GetAllThresholds()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(thresholds)
}

// CreateThreshold 创建阈值
func (h *Handler) CreateThreshold(w http.ResponseWriter, r *http.Request) {
	var threshold models.Threshold
	if err := json.NewDecoder(r.Body).Decode(&threshold); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateThreshold(&threshold)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	threshold.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(threshold)
}

// UpdateThreshold 更新阈值
func (h *Handler) UpdateThreshold(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var threshold models.Threshold
	if err := json.NewDecoder(r.Body).Decode(&threshold); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	threshold.ID = id
	if err := database.UpdateThreshold(&threshold); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteThreshold 删除阈值
func (h *Handler) DeleteThreshold(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := database.DeleteThreshold(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ==================== 报警日志 ====================

// GetAlarmLogs 获取报警日志
func (h *Handler) GetAlarmLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := database.GetRecentAlarmLogs(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// AcknowledgeAlarm 确认报警
func (h *Handler) AcknowledgeAlarm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	session, _ := h.sessionManager.GetSession(r)
	var acknowledgedBy string
	if session != nil {
		acknowledgedBy = session.Username
	}

	if err := database.AcknowledgeAlarmLog(id, acknowledgedBy); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ==================== 数据缓存 ====================

// GetDataCache 获取所有数据缓存
func (h *Handler) GetDataCache(w http.ResponseWriter, r *http.Request) {
	cache, err := database.GetAllDataCache()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cache)
}

// GetDataCacheByDeviceID 根据设备ID获取数据缓存
func (h *Handler) GetDataCacheByDeviceID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	cache, err := database.GetDataCacheByDeviceID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cache)
}

// ==================== 历史数据 ====================

// GetDataPoints 获取历史数据点
func (h *Handler) GetDataPoints(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	// 获取查询参数
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // 默认limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	points, err := database.GetDataPointsByDevice(id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}

// GetLatestDataPoints 获取最新历史数据点
func (h *Handler) GetLatestDataPoints(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // 默认limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	points, err := database.GetLatestDataPoints(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}

// ==================== 存储配置 ====================

// GetStorageConfigs 获取所有存储配置
func (h *Handler) GetStorageConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllStorageConfigs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// CreateStorageConfig 创建存储配置
func (h *Handler) CreateStorageConfig(w http.ResponseWriter, r *http.Request) {
	var config database.StorageConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateStorageConfig(&config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	config.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateStorageConfig 更新存储配置
func (h *Handler) UpdateStorageConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var config database.StorageConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config.ID = id
	if err := database.UpdateStorageConfig(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteStorageConfig 删除存储配置
func (h *Handler) DeleteStorageConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := database.DeleteStorageConfig(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CleanupData 清理过期数据
func (h *Handler) CleanupData(w http.ResponseWriter, r *http.Request) {
	deleted, err := database.CleanupOldData()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": deleted,
	})
}

// ==================== 采集控制 ====================

// StartCollector 启动采集器
func (h *Handler) StartCollector(w http.ResponseWriter, r *http.Request) {
	if err := h.collector.Start(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 添加所有启用的设备
	devices, _ := database.GetAllDevices()
	for _, device := range devices {
		if device.Enabled == 1 {
			h.collector.AddDevice(device)
		}
	}

	w.WriteHeader(http.StatusOK)
}

// StopCollector 停止采集器
func (h *Handler) StopCollector(w http.ResponseWriter, r *http.Request) {
	if err := h.collector.Stop(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ==================== 用户管理 ====================

// GetUsers 获取所有用户
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := database.GetAllUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// CreateUser 创建用户
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := database.CreateUser(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// UpdateUser 更新用户
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user.ID = id
	if err := database.UpdateUser(&user); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := database.DeleteUser(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session, _ := h.sessionManager.GetSession(r)
	if session == nil {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	if err := auth.ChangePassword(session.UserID, req.OldPassword, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
