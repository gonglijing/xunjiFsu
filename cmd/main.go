package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/graceful"
	"github.com/gonglijing/xunjiFsu/internal/handlers"
	"github.com/gonglijing/xunjiFsu/internal/logger"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/resource"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var cfg *config.Config

func init() {
	// 加载配置
	var err error
	cfg, err = config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化结构化日志
	level := logger.ParseLevel(cfg.LogLevel)
	logger.SetLevel(level)
	logger.SetJSONOutput(cfg.LogJSON)

	// 解析命令行参数（会覆盖环境变量配置）
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "数据库路径")
	flag.StringVar(&cfg.ListenAddr, "addr", cfg.ListenAddr, "监听地址")
	flag.Parse()
}

// loadOrGenerateSecretKey 加载或生成固定的会话密钥
func loadOrGenerateSecretKey() []byte {
	// 1. 优先从环境变量读取
	if key := os.Getenv("SESSION_SECRET"); key != "" {
		h := sha256.Sum256([]byte(key))
		return h[:]
	}

	// 2. 从 config 目录读取
	configDir := "config"
	keyFile := filepath.Join(configDir, "session_secret.key")
	if data, err := os.ReadFile(keyFile); err == nil {
		key := string(data)
		if len(key) >= 32 {
			h := sha256.Sum256([]byte(key))
			return h[:]
		}
	}

	// 3. 确保 config 目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Warn("Failed to create config directory", "error", err)
	}

	// 4. 生成新密钥并保存到 config 目录
	newKey := make([]byte, 32)
	_, err := rand.Read(newKey)
	if err != nil {
		logger.Fatal("Failed to generate secret key", err)
	}

	// 保存到文件
	if err := os.WriteFile(keyFile, newKey, 0600); err != nil {
		logger.Warn("Failed to save session secret key", "error", err)
	} else {
		logger.Info("Generated new session secret key")
	}

	h := sha256.Sum256(newKey)
	return h[:]
}

// registerNorthboundAdapter 注册北向适配器
func registerNorthboundAdapter(northboundMgr *northbound.NorthboundManager, config *models.NorthboundConfig) {
	var adapter northbound.Northbound

	switch config.Type {
	case "xunji":
		adapter = northbound.NewXunJiAdapter()
	case "http":
		adapter = northbound.NewHTTPAdapter()
	case "mqtt":
		adapter = northbound.NewMQTTAdapter()
	default:
		log.Printf("Unknown northbound type: %s", config.Type)
		return
	}

	if err := adapter.Initialize(config.Config); err != nil {
		log.Printf("Failed to initialize northbound adapter %s: %v", config.Name, err)
		return
	}

	northboundMgr.RegisterAdapter(config.Name, adapter)
}

// startNorthboundUploadTask 启动北向定期上传任务
func startNorthboundUploadTask(northboundMgr *northbound.NorthboundManager, collect *collector.Collector) {
	configs, err := database.GetEnabledNorthboundConfigs()
	if err != nil {
		logger.Warn("Failed to get enabled northbound configs", "error", err)
		return
	}

	for _, config := range configs {
		go func(config *models.NorthboundConfig) {
			logger.Info("Starting upload task for northbound",
				"name", config.Name,
				"interval", config.UploadInterval)

			ticker := time.NewTicker(time.Duration(config.UploadInterval) * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				updatedConfig, err := database.GetNorthboundConfigByID(config.ID)
				if err != nil || updatedConfig.Enabled != 1 {
					logger.Info("Northbound config disabled, stopping upload task", "name", config.Name)
					return
				}

				logger.Info("Uploading data to northbound", "name", config.Name)
			}
		}(config)
	}
}

func main() {
	secretKey := loadOrGenerateSecretKey()

	// 初始化数据库
	logger.Info("Initializing param database (persistent mode)...")
	if err := database.InitParamDB(); err != nil {
		logger.Fatal("Failed to initialize param database", err)
	}
	defer database.ParamDB.Close()

	logger.Info("Initializing data database (memory mode + batch sync)...")
	if err := database.InitDataDB(); err != nil {
		logger.Fatal("Failed to initialize data database", err)
	}
	defer database.DataDB.Close()

	// 初始化数据库schema
	logger.Info("Initializing param database schema...")
	if err := database.InitParamSchema(); err != nil {
		logger.Fatal("Failed to initialize param schema", err)
	}

	logger.Info("Initializing data database schema...")
	if err := database.InitDataSchema(); err != nil {
		logger.Fatal("Failed to initialize data schema", err)
	}

	// 初始化默认数据
	logger.Info("Initializing default data...")
	if err := database.InitDefaultData(); err != nil {
		logger.Fatal("Failed to initialize default data", err)
	}

	// 启动数据同步任务（内存 -> 磁盘批量写入）
	logger.Info("Starting data sync task...")
	database.StartDataSync()

	// 启动阈值缓存服务（如果启用）
	if cfg.ThresholdCacheEnabled {
		logger.Info("Starting threshold cache...")
		collector.StartThresholdCache()
	}

	// 初始化组件
	logger.Info("Initializing components...")

	// 资源管理器
	resourceMgr := resource.NewResourceManagerImpl()

	// 驱动管理器
	driverManager := driver.NewDriverManager()
	driverExecutor := driver.NewDriverExecutor(driverManager)
	driverExecutor.SetResourceManager(resourceMgr)

	// 北向管理器
	northboundMgr := northbound.NewNorthboundManager()

	// 加载数据库中的北向配置
	logger.Info("Loading northbound configs from database...")
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		logger.Warn("Failed to load northbound configs", "error", err)
	} else {
		for _, config := range configs {
			if config.Enabled == 1 {
				registerNorthboundAdapter(northboundMgr, config)
				logger.Info("Loaded northbound config", "name", config.Name, "type", config.Type)
			}
		}
	}

	// 采集器
	collect := collector.NewCollector(driverExecutor, northboundMgr)

	// 会话管理器
	sessionManager := auth.NewSessionManager(secretKey)

	// 创建处理器
	h := handlers.NewHandler(
		sessionManager,
		collect,
		driverManager,
		resourceMgr,
		northboundMgr,
	)

	// 初始化模板
	logger.Info("Initializing templates...")
	if err := handlers.InitTemplates("templates"); err != nil {
		logger.Warn("Failed to initialize templates", "error", err)
	}

	// 创建路由器
	r := mux.NewRouter()

	// 获取工作目录，确保静态文件路径正确
	workDir, _ := os.Getwd()
	staticDir := http.Dir(filepath.Join(workDir, "web", "static"))

	// 静态文件服务 - 支持 /static/ 和 /web/static/ 两种路径
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(staticDir)))
	r.PathPrefix("/web/static/").Handler(http.StripPrefix("/web/static/", http.FileServer(staticDir)))

	// 页面路由
	r.HandleFunc("/login", h.Login).Methods("GET")
	r.HandleFunc("/login", h.LoginPost).Methods("POST")
	r.HandleFunc("/logout", h.Logout).Methods("GET")
	r.Handle("/", sessionManager.RequireAuth(http.HandlerFunc(h.Dashboard))).Methods("GET")
	r.Handle("/realtime", sessionManager.RequireAuth(http.HandlerFunc(h.RealTime))).Methods("GET")
	r.Handle("/history", sessionManager.RequireAuth(http.HandlerFunc(h.History))).Methods("GET")

	// API路由 - 状态
	r.HandleFunc("/api/status", h.GetStatus).Methods("GET")

	// 健康检查路由 (无需认证)
	r.HandleFunc("/health", handlers.Health).Methods("GET")
	r.HandleFunc("/ready", handlers.Readiness).Methods("GET")
	r.HandleFunc("/live", handlers.Liveness).Methods("GET")

	// 指标监控路由 (无需认证)
	r.HandleFunc("/metrics", handlers.Metrics).Methods("GET")

	// API路由 - 采集控制
	r.HandleFunc("/api/collector/start", h.StartCollector).Methods("POST")
	r.HandleFunc("/api/collector/stop", h.StopCollector).Methods("POST")

	// API路由 - 资源
	r.HandleFunc("/api/resources", h.GetResources).Methods("GET")
	r.HandleFunc("/api/resources", h.CreateResource).Methods("POST")
	r.HandleFunc("/api/resources/{id}", h.UpdateResource).Methods("PUT")
	r.HandleFunc("/api/resources/{id}", h.DeleteResource).Methods("DELETE")
	r.HandleFunc("/api/resources/{id}/open", h.OpenResource).Methods("POST")
	r.HandleFunc("/api/resources/{id}/close", h.CloseResource).Methods("POST")

	// API路由 - 驱动
	r.HandleFunc("/api/drivers", h.GetDrivers).Methods("GET")
	r.HandleFunc("/api/drivers", h.CreateDriver).Methods("POST")
	r.HandleFunc("/api/drivers/{id}", h.UpdateDriver).Methods("PUT")
	r.HandleFunc("/api/drivers/{id}", h.DeleteDriver).Methods("DELETE")
	r.HandleFunc("/api/drivers/{id}/download", h.DownloadDriver).Methods("GET")
	r.HandleFunc("/api/drivers/upload", h.UploadDriverFile).Methods("POST")
	r.HandleFunc("/api/drivers/files", h.ListDriverFiles).Methods("GET")

	// API路由 - 设备
	r.HandleFunc("/api/devices", h.GetDevices).Methods("GET")
	r.HandleFunc("/api/devices", h.CreateDevice).Methods("POST")
	r.HandleFunc("/api/devices/{id}", h.UpdateDevice).Methods("PUT")
	r.HandleFunc("/api/devices/{id}", h.DeleteDevice).Methods("DELETE")
	r.HandleFunc("/api/devices/{id}/toggle", h.ToggleDeviceEnable).Methods("POST")
	r.HandleFunc("/api/devices/{id}/execute", h.ExecuteDriverFunction).Methods("POST")

	// API路由 - 北向配置
	r.HandleFunc("/api/northbound", h.GetNorthboundConfigs).Methods("GET")
	r.HandleFunc("/api/northbound", h.CreateNorthboundConfig).Methods("POST")
	r.HandleFunc("/api/northbound/{id}", h.UpdateNorthboundConfig).Methods("PUT")
	r.HandleFunc("/api/northbound/{id}", h.DeleteNorthboundConfig).Methods("DELETE")
	r.HandleFunc("/api/northbound/{id}/toggle", h.ToggleNorthboundEnable).Methods("POST")

	// API路由 - 阈值
	r.HandleFunc("/api/thresholds", h.GetThresholds).Methods("GET")
	r.HandleFunc("/api/thresholds", h.CreateThreshold).Methods("POST")
	r.HandleFunc("/api/thresholds/{id}", h.UpdateThreshold).Methods("PUT")
	r.HandleFunc("/api/thresholds/{id}", h.DeleteThreshold).Methods("DELETE")

	// API路由 - 报警日志
	r.HandleFunc("/api/alarms", h.GetAlarmLogs).Methods("GET")
	r.HandleFunc("/api/alarms/{id}/acknowledge", h.AcknowledgeAlarm).Methods("POST")

	// API路由 - 数据缓存
	r.HandleFunc("/api/data", h.GetDataCache).Methods("GET")
	r.HandleFunc("/api/data/cache/{id}", h.GetDataCacheByDeviceID).Methods("GET")

	// API路由 - 历史数据
	r.HandleFunc("/api/data/points/{id}", h.GetDataPoints).Methods("GET")
	r.HandleFunc("/api/data/points", h.GetLatestDataPoints).Methods("GET")
	r.HandleFunc("/api/data/history", h.GetHistoryData).Methods("GET")

	// API路由 - 存储配置
	r.HandleFunc("/api/storage", h.GetStorageConfigs).Methods("GET")
	r.HandleFunc("/api/storage", h.CreateStorageConfig).Methods("POST")
	r.HandleFunc("/api/storage/{id}", h.UpdateStorageConfig).Methods("PUT")
	r.HandleFunc("/api/storage/{id}", h.DeleteStorageConfig).Methods("DELETE")
	r.HandleFunc("/api/storage/cleanup", h.CleanupData).Methods("POST")

	// API路由 - 用户管理
	r.HandleFunc("/api/users", h.GetUsers).Methods("GET")
	r.HandleFunc("/api/users", h.CreateUser).Methods("POST")
	r.HandleFunc("/api/users/{id}", h.UpdateUser).Methods("PUT")
	r.HandleFunc("/api/users/{id}", h.DeleteUser).Methods("DELETE")
	r.HandleFunc("/api/users/password", h.ChangePassword).Methods("PUT")

	// 中间件栈
	// CORS中间件
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins(cfg.GetAllowedOrigins()),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
		gorillaHandlers.AllowCredentials(),
	)

	// 日志中间件
	loggingHandler := gorillaHandlers.LoggingHandler(os.Stdout, r)

	// Gzip压缩中间件
	gzipHandler := handlers.GzipMiddleware(loggingHandler)

	// 超时中间件配置
	timeoutConfig := handlers.DefaultTimeoutConfig()
	timeoutConfig.ReadTimeout = cfg.HTTPReadTimeout
	timeoutConfig.WriteTimeout = cfg.HTTPWriteTimeout
	timeoutConfig.IdleTimeout = cfg.HTTPIdleTimeout

	// 构建最终处理器链
	finalHandler := corsHandler(gzipHandler)
	finalHandler = handlers.TimeoutMiddleware(timeoutConfig)(finalHandler)

	// 启动采集器
	if err := collect.Start(); err != nil {
		logger.Warn("Failed to start collector", "error", err)
	}

	// 自动加载使能的设备到采集器
	logger.Info("Loading enabled devices...")
	devices, err := database.GetAllDevices()
	if err != nil {
		logger.Warn("Failed to load devices", "error", err)
	} else {
		enabledCount := 0
		for _, device := range devices {
			if device.Enabled == 1 {
				if err := collect.AddDevice(device); err != nil {
					logger.Warn("Failed to add device", "name", device.Name, "error", err)
				} else {
					logger.Info("Device added to collector", "name", device.Name, "interval", device.CollectInterval)
					enabledCount++
				}
			}
		}
		logger.Info("Loaded enabled devices to collector", "count", enabledCount)
	}

	// 自动注册使能的北向配置
	logger.Info("Loading enabled northbound configs...")
	northboundConfigs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		logger.Warn("Failed to load northbound configs", "error", err)
	} else {
		enabledCount := 0
		for _, config := range northboundConfigs {
			if config.Enabled == 1 {
				registerNorthboundAdapter(northboundMgr, config)
				logger.Info("Northbound config registered",
					"name", config.Name,
					"type", config.Type,
					"upload_interval", config.UploadInterval)
				enabledCount++
			}
		}
		logger.Info("Loaded enabled northbound configs", "count", enabledCount)
	}

	// 启动采集任务协程
	go collect.Run()

	// 启动北向上传任务
	go startNorthboundUploadTask(northboundMgr, collect)

	// 优雅关闭管理器
	gracefulMgr := graceful.NewGracefulShutdown(30 * time.Second)

	// 注册关闭函数
	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping collector...")
		collect.Stop()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Stopping data sync...")
		database.StopDataSync()
		return nil
	})

	gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
		logger.Info("Final sync to disk...")
		database.SyncDataToDisk()
		return nil
	})

	if cfg.ThresholdCacheEnabled {
		gracefulMgr.AddShutdownFunc(func(ctx context.Context) error {
			logger.Info("Stopping threshold cache...")
			collector.StopThresholdCache()
			return nil
		})
	}

	// 启动优雅关闭监听
	gracefulMgr.Start()

	// 启动服务器
	logger.Info("Starting server", "addr", cfg.ListenAddr)

	server := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      finalHandler,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	gracefulMgr.SetHTTPServer(server)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Server error", err)
	}

	// 等待优雅关闭完成
	gracefulMgr.Wait()
}
