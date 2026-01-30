package main

import (
	"crypto/rand"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gogw/internal/auth"
	"gogw/internal/collector"
	"gogw/internal/database"
	"gogw/internal/driver"
	"gogw/internal/handlers"
	"gogw/internal/models"
	"gogw/internal/northbound"
	"gogw/internal/resource"
)

var (
	dbPath     string
	listenAddr string
	secretKey  []byte
)

func init() {
	// 解析命令行参数
	flag.StringVar(&dbPath, "db", "gogw.db", "数据库路径")
	flag.StringVar(&listenAddr, "addr", ":8080", "监听地址")
	flag.Parse()

	// 生成随机密钥
	secretKey = make([]byte, 32)
	_, err := rand.Read(secretKey)
	if err != nil {
		log.Fatalf("Failed to generate secret key: %v", err)
	}
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

func main() {
	// 初始化数据库
	// param.db: 持久化文件（配置操作频率低，直接写磁盘）
	// data.db: 内存模式 + 批量同步（采集数据高频写入）
	log.Println("Initializing param database (persistent mode)...")
	if err := database.InitParamDB(); err != nil {
		log.Fatalf("Failed to initialize param database: %v", err)
	}
	defer database.ParamDB.Close()

	log.Println("Initializing data database (memory mode + batch sync)...")
	if err := database.InitDataDB(); err != nil {
		log.Fatalf("Failed to initialize data database: %v", err)
	}
	defer database.DataDB.Close()

	// 初始化数据库schema
	log.Println("Initializing param database schema...")
	if err := database.InitParamSchema(); err != nil {
		log.Fatalf("Failed to initialize param schema: %v", err)
	}

	log.Println("Initializing data database schema...")
	if err := database.InitDataSchema(); err != nil {
		log.Fatalf("Failed to initialize data schema: %v", err)
	}

	// 初始化默认数据
	log.Println("Initializing default data...")
	if err := database.InitDefaultData(); err != nil {
		log.Fatalf("Failed to initialize default data: %v", err)
	}

	// 启动数据同步任务（内存 -> 磁盘批量写入）
	log.Println("Starting data sync task...")
	database.StartDataSync()

	// 初始化组件
	log.Println("Initializing components...")

	// 资源管理器
	resourceMgr := resource.NewResourceManagerImpl()

	// 驱动管理器
	driverManager := driver.NewDriverManager()
	driverExecutor := driver.NewDriverExecutor(driverManager)
	
	// 设置资源管理器到驱动执行器（用于资源访问锁）
	driverExecutor.SetResourceManager(resourceMgr)

	// 北向管理器
	northboundMgr := northbound.NewNorthboundManager()

	// 加载数据库中的北向配置
	log.Println("Loading northbound configs from database...")
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		log.Printf("Warning: Failed to load northbound configs: %v", err)
	} else {
		for _, config := range configs {
			if config.Enabled == 1 {
				registerNorthboundAdapter(northboundMgr, config)
				log.Printf("Loaded northbound config: %s (%s)", config.Name, config.Type)
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

	// 创建路由器
	r := mux.NewRouter()

	// 静态文件服务
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// 页面路由
	r.HandleFunc("/login", h.Login).Methods("GET")
	r.HandleFunc("/login", h.LoginPost).Methods("POST")
	r.HandleFunc("/logout", h.Logout).Methods("GET")
	r.Handle("/", sessionManager.RequireAuth(http.HandlerFunc(h.Dashboard))).Methods("GET")
	r.Handle("/realtime", sessionManager.RequireAuth(http.HandlerFunc(h.RealTime))).Methods("GET")
	r.Handle("/history", sessionManager.RequireAuth(http.HandlerFunc(h.History))).Methods("GET")

	// API路由 - 状态
	r.HandleFunc("/api/status", h.GetStatus).Methods("GET")

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

	// API路由 - 设备
	r.HandleFunc("/api/devices", h.GetDevices).Methods("GET")
	r.HandleFunc("/api/devices", h.CreateDevice).Methods("POST")
	r.HandleFunc("/api/devices/{id}", h.UpdateDevice).Methods("PUT")
	r.HandleFunc("/api/devices/{id}", h.DeleteDevice).Methods("DELETE")

	// API路由 - 北向配置
	r.HandleFunc("/api/northbound", h.GetNorthboundConfigs).Methods("GET")
	r.HandleFunc("/api/northbound", h.CreateNorthboundConfig).Methods("POST")
	r.HandleFunc("/api/northbound/{id}", h.UpdateNorthboundConfig).Methods("PUT")
	r.HandleFunc("/api/northbound/{id}", h.DeleteNorthboundConfig).Methods("DELETE")

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

	// CORS中间件
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins([]string{"*"}),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// 日志中间件
	loggingHandler := gorillaHandlers.LoggingHandler(os.Stdout, r)

	// 启动采集器
	if err := collect.Start(); err != nil {
		log.Printf("Warning: Failed to start collector: %v", err)
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down...")
		
		// 停止数据同步
		database.StopDataSync()
		
		// 最后一次同步数据到磁盘
		log.Println("Final sync to disk...")
		database.SyncDataToDisk()
		
		// 停止采集器
		collect.Stop()
		
		os.Exit(0)
	}()

	// 启动服务器
	log.Printf("Starting server on %s...", listenAddr)
	if err := http.ListenAndServe(listenAddr, corsHandler(loggingHandler)); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
