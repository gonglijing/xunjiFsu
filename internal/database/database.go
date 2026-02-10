package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

// Configuration
const (
	SyncInterval                  = 5 * time.Minute // 同步间隔
	SyncBatchTrigger              = 1000            // 数据量触发同步的阈值
	DefaultParamDBFile            = "param.db"      // 配置数据库文件名
	DataDBFile                    = "data.db"       // 数据数据库文件名
	MaxDataPoints                 = 100000          // 内存数据库最大数据点数
	MaxDataCache                  = 10000           // 内存缓存最大条目数
	DefaultRetentionDays          = 30              // 默认历史保留天数
	DefaultStorageIntervalSeconds = 300             // 默认存储周期(s)

	// 连接池配置（可调整）
	DefaultMaxOpenConns = 25        // 默认最大打开连接数
	DefaultMaxIdleConns = 10        // 默认最大空闲连接数
	ConnMaxLifetime     = time.Hour // 连接最大生命周期
)

// ParamDB 配置数据库连接（持久化文件）
var ParamDB *sql.DB

// DataDB 历史数据数据库连接（内存模式）
var DataDB *sql.DB

var paramDBFile = DefaultParamDBFile
var dataDBFile = DataDBFile
var maxDataPointsLimit = MaxDataPoints
var maxDataCacheLimit = MaxDataCache
var syncInterval = SyncInterval
var syncBatchTrigger = SyncBatchTrigger
var syncDataToDiskFn = syncDataToDisk

// ApplyRuntimeLimits 应用运行时内存数据限制。
// 当传入值小于等于0时，会回退到默认值，避免限制失效导致内存持续增长。
func ApplyRuntimeLimits(maxDataPoints, maxDataCache int) {
	if maxDataPoints > 0 {
		maxDataPointsLimit = maxDataPoints
	} else {
		maxDataPointsLimit = MaxDataPoints
	}

	if maxDataCache > 0 {
		maxDataCacheLimit = maxDataCache
	} else {
		maxDataCacheLimit = MaxDataCache
	}

	log.Printf("Applied data limits (max_data_points=%d, max_data_cache=%d)", maxDataPointsLimit, maxDataCacheLimit)
}

// ApplySyncInterval 应用数据同步间隔。
// 当传入值小于等于0时，回退到默认值。
func ApplySyncInterval(interval time.Duration) {
	if interval > 0 {
		syncInterval = interval
	} else {
		syncInterval = SyncInterval
	}
	log.Printf("Applied data sync interval: %v", syncInterval)
}

// InitParamDB 初始化配置数据库（持久化文件）
func InitParamDB() error {
	return InitParamDBWithPath(DefaultParamDBFile)
}

// InitParamDBWithPath 初始化配置数据库并指定路径
func InitParamDBWithPath(path string) error {
	if path == "" {
		path = DefaultParamDBFile
	}
	paramDBFile = path

	// 配置连接池
	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConns)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConns)
	var err error
	ParamDB, err = openSQLite(path, maxOpen, maxIdle)
	if err != nil {
		return fmt.Errorf("failed to open param database: %w", err)
	}

	log.Printf("Param database initialized (max_open=%d, max_idle=%d)", maxOpen, maxIdle)
	return nil
}

// InitDataDB 初始化历史数据数据库（内存模式 + 批量同步）
func InitDataDB() error {
	return InitDataDBWithPath(DataDBFile)
}

// InitDataDBWithPath 初始化历史数据数据库并指定落盘路径
func InitDataDBWithPath(path string) error {
	if path == "" {
		path = DataDBFile
	}
	dataDBFile = path

	// 配置连接池
	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConns)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConns)
	var err error
	DataDB, err = openSQLite(":memory:", maxOpen, maxIdle)
	if err != nil {
		return fmt.Errorf("failed to open data database: %w", err)
	}
	// data.db 仅做历史存储，不依赖 devices 表，禁用外键以避免跨库引用错误
	_, _ = DataDB.Exec("PRAGMA foreign_keys = OFF")

	// 从文件恢复数据（如果存在）
	if _, err := os.Stat(dataDBFile); err == nil {
		log.Println("Restoring data database from file...")
		if err := restoreDataFromFile(dataDBFile); err != nil {
			log.Printf("Warning: failed to restore data database: %v", err)
		}
	}

	return nil
}
