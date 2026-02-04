package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// Configuration
const (
	SyncInterval         = 5 * time.Minute // 同步间隔
	SyncBatchTrigger     = 1000            // 数据量触发同步的阈值
	DefaultParamDBFile   = "param.db"      // 配置数据库文件名
	DataDBFile           = "data.db"       // 数据数据库文件名
	MaxDataPoints        = 100000          // 内存数据库最大数据点数
	MaxDataCache         = 10000           // 内存缓存最大条目数
	DefaultRetentionDays = 30              // 默认历史保留天数

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
var retentionDaysDefault = 30 // fallback if no storage policy
var maxDataPointsLimit = MaxDataPoints
var maxDataCacheLimit = MaxDataCache
var syncBatchTrigger = SyncBatchTrigger
var syncDataToDiskFn = syncDataToDisk

// StoragePolicy 数据保留策略
type StoragePolicy struct {
	ID         int64     `json:"id" db:"id"`
	ProductKey string    `json:"product_key" db:"product_key"`
	DeviceKey  string    `json:"device_key" db:"device_key"`
	Retention  int       `json:"retention" db:"retention"` // days
	Enabled    int       `json:"enabled" db:"enabled"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
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
