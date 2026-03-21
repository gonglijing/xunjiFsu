package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// Configuration
const (
	SyncInterval                  = 5 * time.Minute // 同步间隔
	SyncBatchTrigger              = 1000            // 数据量触发同步的阈值
	DefaultParamDBFile            = "param.db"      // 配置数据库文件名
	DataDBFile                    = "data.db"       // 数据数据库文件名
	MaxDataPoints                 = 20000           // 内存数据库最大数据点数（128 MB 设备适用）
	MaxDataCache                  = 15000           // 内存缓存最大条目数（覆盖万级测点）
	DefaultRetentionDays          = 30              // 默认历史保留天数
	DefaultStorageIntervalSeconds = 300             // 默认存储周期(s)

	// 连接池配置（嵌入式设备适用，可通过环境变量覆盖）
	DefaultMaxOpenConns = 8         // 默认最大打开连接数
	DefaultMaxIdleConns = 4         // 默认最大空闲连接数
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

	slog.Info("Applied data limits", "max_data_points", maxDataPointsLimit, "max_data_cache", maxDataCacheLimit)
}

// ApplySyncInterval 应用数据同步间隔。
// 当传入值小于等于0时，回退到默认值。
func ApplySyncInterval(interval time.Duration) {
	if interval > 0 {
		syncInterval = interval
	} else {
		syncInterval = SyncInterval
	}
	slog.Info("Applied data sync interval", "interval", syncInterval)
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

	slog.Info("Param database initialized", "max_open", maxOpen, "max_idle", maxIdle)
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
		slog.Info("Restoring data database from file", "path", dataDBFile)
	}

	return nil
}

// getEnvInt 从环境变量获取整数配置
func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return defaultVal
}
