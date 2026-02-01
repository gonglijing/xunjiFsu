package database

import (
	"database/sql"
	"log"
	"sync"
	"time"
)

// DBHealthChecker 数据库连接健康检查器
type DBHealthChecker struct {
	mu           sync.RWMutex
	paramHealthy bool
	dataHealthy  bool
	lastCheck    time.Time
	checkInterval time.Duration
	stopChan     chan struct{}
}

// NewDBHealthChecker 创建健康检查器
func NewDBHealthChecker(checkInterval time.Duration) *DBHealthChecker {
	return &DBHealthChecker{
		paramHealthy: false,
		dataHealthy:  false,
		checkInterval: checkInterval,
		stopChan:     make(chan struct{}),
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
	close(c.stopChan)
	log.Println("Database health checker stopped")
}

// check 执行健康检查
func (c *DBHealthChecker) check() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// 检查ParamDB
	if err := ParamDB.Ping(); err != nil {
		log.Printf("ParamDB health check failed: %v", err)
		c.paramHealthy = false
	} else {
		c.paramHealthy = true
	}

	// 检查DataDB
	if err := DataDB.Ping(); err != nil {
		log.Printf("DataDB health check failed: %v", err)
		c.dataHealthy = false
	} else {
		c.dataHealthy = true
	}

	c.lastCheck = now
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
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"healthy":       c.paramHealthy && c.dataHealthy,
		"param_healthy": c.paramHealthy,
		"data_healthy":  c.dataHealthy,
		"last_check":    c.lastCheck,
	}
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
	if healthChecker == nil {
		// 如果没有启动健康检查，直接ping
		if err := ParamDB.Ping(); err != nil {
			return false
		}
		if err := DataDB.Ping(); err != nil {
			return false
		}
		return true
	}
	return healthChecker.IsHealthy()
}

// GetDBStatus 获取数据库状态
func GetDBStatus() map[string]interface{} {
	if healthChecker == nil {
		return map[string]interface{}{
			"healthy":       IsDBHealthy(),
			"param_healthy": ParamDB.Ping() == nil,
			"data_healthy":  DataDB.Ping() == nil,
			"last_check":    time.Now(),
		}
	}
	return healthChecker.GetStatus()
}

// ConnectionStats 连接统计
type ConnectionStats struct {
	ParamDBOpen    int `json:"param_db_open"`
	ParamDBIdle    int `json:"param_db_idle"`
	DataDBOpen     int `json:"data_db_open"`
	DataDBIdle     int `json:"data_db_idle"`
	MaxOpenConns   int `json:"max_open_conns"`
	MaxIdleConns   int `json:"max_idle_conns"`
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

	// 尝试重新连接ParamDB
	if err := ParamDB.Ping(); err != nil {
		log.Printf("Reconnecting ParamDB...")
		ParamDB.Close()
		ParamDB, err = sql.Open("sqlite", "param.db")
		if err != nil {
			return err
		}
		ParamDB.SetMaxOpenConns(DefaultMaxOpenConns)
		ParamDB.SetMaxIdleConns(DefaultMaxIdleConns)
		ParamDB.SetConnMaxLifetime(ConnMaxLifetime)
	}

	// 尝试重新连接DataDB
	if err := DataDB.Ping(); err != nil {
		log.Printf("Reconnecting DataDB...")
		DataDB.Close()
		DataDB, err = sql.Open("sqlite", ":memory:")
		if err != nil {
			return err
		}
		DataDB.SetMaxOpenConns(DefaultMaxOpenConns)
		DataDB.SetMaxIdleConns(DefaultMaxIdleConns)
		DataDB.SetConnMaxLifetime(ConnMaxLifetime)
	}

	log.Println("Database connections recovered")
	return nil
}
