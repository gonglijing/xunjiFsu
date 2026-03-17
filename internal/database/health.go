package database

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// DBHealthChecker 数据库连接健康检查器
type DBHealthChecker struct {
	mu            sync.RWMutex
	paramHealthy  bool
	dataHealthy   bool
	lastCheck     time.Time
	checkInterval time.Duration
	stopChan      chan struct{}
	stopOnce      sync.Once
}

type dbHealthStatus struct {
	paramHealthy bool
	dataHealthy  bool
	lastCheck    time.Time
}

// NewDBHealthChecker 创建健康检查器
func NewDBHealthChecker(checkInterval time.Duration) *DBHealthChecker {
	return &DBHealthChecker{
		paramHealthy:  false,
		dataHealthy:   false,
		checkInterval: checkInterval,
		stopChan:      make(chan struct{}),
	}
}

// Start 启动健康检查
func (c *DBHealthChecker) Start() {
	// 立即检查一次
	c.check()

	go func() {
		ticker := time.NewTicker(c.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-c.stopChan:
				return
			case <-ticker.C:
				c.check()
			}
		}
	}()

	log.Printf("Database health checker started (interval: %v)", c.checkInterval)
}

// Stop 停止健康检查
func (c *DBHealthChecker) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
	log.Println("Database health checker stopped")
}

// check 执行健康检查
func (c *DBHealthChecker) check() {
	status := currentDBHealthStatus(time.Now())

	c.mu.Lock()
	c.paramHealthy = status.paramHealthy
	c.dataHealthy = status.dataHealthy
	c.lastCheck = status.lastCheck
	c.mu.Unlock()
}

// IsHealthy 获取整体健康状态
func (c *DBHealthChecker) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.paramHealthy && c.dataHealthy
}

// GetStatus 获取详细状态
func (c *DBHealthChecker) GetStatus() map[string]interface{} {
	c.mu.RLock()
	status := dbHealthStatus{
		paramHealthy: c.paramHealthy,
		dataHealthy:  c.dataHealthy,
		lastCheck:    c.lastCheck,
	}
	c.mu.RUnlock()

	return status.toMap()
}

// Global health checker instance
var healthChecker *DBHealthChecker

// StartDBHealthChecker 全局启动健康检查
func StartDBHealthChecker(interval time.Duration) {
	healthChecker = NewDBHealthChecker(interval)
	healthChecker.Start()
}

// StopDBHealthChecker 全局停止健康检查
func StopDBHealthChecker() {
	if healthChecker != nil {
		healthChecker.Stop()
	}
}

// IsDBHealthy 检查数据库是否健康
func IsDBHealthy() bool {
	if healthChecker != nil {
		return healthChecker.IsHealthy()
	}
	return currentDBHealthStatus(time.Now()).healthy()
}

// GetDBStatus 获取数据库状态
func GetDBStatus() map[string]interface{} {
	if healthChecker != nil {
		return healthChecker.GetStatus()
	}
	return currentDBHealthStatus(time.Now()).toMap()
}

// ConnectionStats 连接统计
type ConnectionStats struct {
	ParamDBOpen  int `json:"param_db_open"`
	ParamDBIdle  int `json:"param_db_idle"`
	DataDBOpen   int `json:"data_db_open"`
	DataDBIdle   int `json:"data_db_idle"`
	MaxOpenConns int `json:"max_open_conns"`
	MaxIdleConns int `json:"max_idle_conns"`
}

// GetConnectionStats 获取连接统计
func GetConnectionStats() ConnectionStats {
	paramStats := ParamDB.Stats()
	dataStats := DataDB.Stats()

	return ConnectionStats{
		ParamDBOpen:  paramStats.OpenConnections,
		ParamDBIdle:  paramStats.Idle,
		DataDBOpen:   dataStats.OpenConnections,
		DataDBIdle:   dataStats.Idle,
		MaxOpenConns: paramStats.MaxOpenConnections,
		MaxIdleConns: DefaultMaxIdleConns, // 无法从stats获取，使用配置值
	}
}

// RecoverConnection 尝试恢复连接
func RecoverConnection() error {
	log.Println("Attempting to recover database connections...")

	if err := recoverParamDB(); err != nil {
		return err
	}
	if err := recoverDataDB(); err != nil {
		return err
	}

	log.Println("Database connections recovered")
	return nil
}

func currentDBHealthStatus(now time.Time) dbHealthStatus {
	return dbHealthStatus{
		paramHealthy: pingDatabase("ParamDB", ParamDB),
		dataHealthy:  pingDatabase("DataDB", DataDB),
		lastCheck:    now,
	}
}

func pingDatabase(name string, db *sql.DB) bool {
	if db == nil {
		log.Printf("%s health check failed: database is nil", name)
		return false
	}
	if err := db.Ping(); err != nil {
		log.Printf("%s health check failed: %v", name, err)
		return false
	}
	return true
}

func (s dbHealthStatus) healthy() bool {
	return s.paramHealthy && s.dataHealthy
}

func (s dbHealthStatus) toMap() map[string]interface{} {
	return map[string]interface{}{
		"healthy":       s.healthy(),
		"param_healthy": s.paramHealthy,
		"data_healthy":  s.dataHealthy,
		"last_check":    s.lastCheck,
	}
}

func recoverParamDB() error {
	if pingDatabase("ParamDB", ParamDB) {
		return nil
	}

	log.Printf("Reconnecting ParamDB...")
	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConns)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConns)

	var err error
	ParamDB, err = openSQLite(paramDBFile, maxOpen, maxIdle)
	return err
}

func recoverDataDB() error {
	if pingDatabase("DataDB", DataDB) {
		return nil
	}

	log.Printf("Reconnecting DataDB...")
	if DataDB != nil {
		_ = DataDB.Close()
	}

	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConns)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConns)

	var err error
	DataDB, err = openSQLite(":memory:", maxOpen, maxIdle)
	if err != nil {
		return err
	}
	_, _ = DataDB.Exec("PRAGMA foreign_keys = OFF")
	return nil
}
