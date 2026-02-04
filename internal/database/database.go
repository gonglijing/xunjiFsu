package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/pwdutil"

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

// 数据同步相关
var dataSyncMu sync.Mutex
var dataSyncTicker *time.Ticker
var dataSyncStop chan struct{}

// 数据清理相关
var retentionTicker *time.Ticker
var retentionStop chan struct{}

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

	var err error
	ParamDB, err = sql.Open("sqlite", path)
	if err != nil {
		return fmt.Errorf("failed to open param database: %w", err)
	}

	// 配置连接池
	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConns)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConns)
	ParamDB.SetMaxOpenConns(maxOpen)
	ParamDB.SetMaxIdleConns(maxIdle)
	ParamDB.SetConnMaxLifetime(ConnMaxLifetime)

	if err := ParamDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping param database: %w", err)
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

	var err error
	DataDB, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open data database: %w", err)
	}

	// 配置连接池
	maxOpen := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConns)
	maxIdle := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConns)
	DataDB.SetMaxOpenConns(maxOpen)
	DataDB.SetMaxIdleConns(maxIdle)
	DataDB.SetConnMaxLifetime(ConnMaxLifetime)

	if err := DataDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping data database: %w", err)
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

// StartDataSync 启动数据同步任务（内存 -> 磁盘批量写入）
func StartDataSync() {
	dataSyncStop = make(chan struct{})
	dataSyncTicker = time.NewTicker(SyncInterval)

	go func() {
		log.Printf("Data sync started (interval: %v, batch_trigger: %d)", SyncInterval, SyncBatchTrigger)
		for {
			select {
			case <-dataSyncTicker.C:
				if err := syncDataToDisk(); err != nil {
					log.Printf("Failed to sync data to disk: %v", err)
				}
			case <-dataSyncStop:
				log.Println("Data sync stopped")
				return
			}
		}
	}()
}

// TriggerSyncIfNeeded 检查是否需要触发同步
// 返回true表示已触发同步
func TriggerSyncIfNeeded() bool {
	var count int
	DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count)
	if count >= SyncBatchTrigger {
		log.Printf("Triggering sync due to data count: %d", count)
		go syncDataToDisk()
		return true
	}
	return false
}

// SyncDataToDisk 手动触发数据同步（公开函数，供优雅关闭调用）
func SyncDataToDisk() error {
	return syncDataToDisk()
}

// StopDataSync 停止数据同步任务
func StopDataSync() {
	if dataSyncTicker != nil {
		dataSyncTicker.Stop()
	}
	if dataSyncStop != nil {
		close(dataSyncStop)
	}
}

// StartRetentionCleanup 启动定期历史数据清理
func StartRetentionCleanup(interval time.Duration) {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	retentionStop = make(chan struct{})
	retentionTicker = time.NewTicker(interval)

	go func() {
		log.Printf("Retention cleanup started (interval: %v)", interval)
		for {
			select {
			case <-retentionTicker.C:
				if _, err := CleanupOldDataByConfig(); err != nil {
					log.Printf("Retention cleanup error: %v", err)
				}
			case <-retentionStop:
				log.Println("Retention cleanup stopped")
				return
			}
		}
	}()
}

// StopRetentionCleanup 停止历史数据清理任务
func StopRetentionCleanup() {
	if retentionTicker != nil {
		retentionTicker.Stop()
	}
	if retentionStop != nil {
		close(retentionStop)
	}
}

// syncDataToDisk 将内存数据批量同步到磁盘
func syncDataToDisk() error {
	dataSyncMu.Lock()
	defer dataSyncMu.Unlock()

	log.Println("Syncing data to disk...")

	// 1. 创建临时数据库文件
	tempFile := dataDBFile + ".tmp"
	diskDB, err := sql.Open("sqlite", tempFile)
	if err != nil {
		return fmt.Errorf("failed to open temp database: %w", err)
	}
	defer diskDB.Close()

	// 2. 复制 schema
	schema, err := DataDB.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}
	defer schema.Close()

	for schema.Next() {
		var sql string
		if err := schema.Scan(&sql); err != nil {
			return err
		}
		if _, err := diskDB.Exec(sql); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// 3. 批量复制数据点
	points, err := DataDB.Query("SELECT device_id, device_name, field_name, value, value_type, collected_at FROM data_points ORDER BY collected_at")
	if err != nil {
		return fmt.Errorf("failed to query data points: %w", err)
	}
	defer points.Close()

	// 开启事务批量插入
	tx, err := diskDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO data_points 
		(id, device_id, device_name, field_name, value, value_type, collected_at) 
		VALUES (NULL, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for points.Next() {
		var deviceID int64
		var deviceName, fieldName, value, valueType string
		var collectedAt time.Time
		if err := points.Scan(&deviceID, &deviceName, &fieldName, &value, &valueType, &collectedAt); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := stmt.Exec(deviceID, deviceName, fieldName, value, valueType, collectedAt); err != nil {
			tx.Rollback()
			return err
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 4. 原子替换文件
	if err := os.Rename(tempFile, dataDBFile); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("Data synced to disk: %d points", count)
	return nil
}

// restoreDataFromFile 尝试从备份文件恢复数据
// 注意：由于使用内存数据库，完整恢复比较复杂
// 如果备份文件存在，记录日志但不影响主程序运行
// 实时数据会在系统运行后自动重新采集
func restoreDataFromFile(filename string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	// 记录有备份文件存在，但不尝试恢复
	// 数据会在运行时自动重新采集
	log.Printf("Backup file exists (%s), real-time data will be collected on startup", filename)
	return nil
}

// InitParamSchema 初始化配置数据库schema
func InitParamSchema() error {
	migration, err := os.ReadFile("migrations/002_param_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read param migration: %w", err)
	}

	_, err = ParamDB.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("failed to execute param migration: %w", err)
	}

	return nil
}

// InitStoragePolicyTable 创建存储策略表
func InitStoragePolicyTable() error {
	// 不做向前兼容，直接使用当前结构重建
	if _, err := ParamDB.Exec(`DROP TABLE IF EXISTS storage_policies`); err != nil {
		return err
	}
	_, err := ParamDB.Exec(`CREATE TABLE storage_policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		product_key TEXT,
		device_key TEXT,
		storage_days INTEGER DEFAULT 30,
		enabled INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

// InitDataSchema 初始化历史数据数据库schema
func InitDataSchema() error {
	migration, err := os.ReadFile("migrations/003_data_schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read data migration: %w", err)
	}

	_, err = DataDB.Exec(string(migration))
	if err != nil {
		return fmt.Errorf("failed to execute data migration: %w", err)
	}

	// 执行索引迁移
	if err := initDataIndexes(); err != nil {
		log.Printf("Warning: failed to create data indexes: %v", err)
	}

	// 确保 alarm_logs 表存在
	if err := ensureAlarmLogsTable(); err != nil {
		return err
	}

	return nil
}

// initDataIndexes 创建数据表索引
func initDataIndexes() error {
	indexes, err := os.ReadFile("migrations/004_indexes.sql")
	if err != nil {
		return fmt.Errorf("failed to read index migration: %w", err)
	}

	// 分割并执行每个索引创建语句
	indexSQL := string(indexes)
	// 移除注释
	lines := strings.Split(indexSQL, "\n")
	var statements []string
	var current strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, current.String())
			current.Reset()
		}
	}

	for _, stmt := range statements {
		_, err := DataDB.Exec(stmt)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	log.Println("Data indexes initialized")
	return nil
}

// ensureAlarmLogsTable 创建 alarm_logs 表（若缺失）
func ensureAlarmLogsTable() error {
	_, err := ParamDB.Exec(`CREATE TABLE IF NOT EXISTS alarm_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER,
		threshold_id INTEGER,
		field_name TEXT,
		actual_value REAL,
		threshold_value REAL,
		operator TEXT,
		severity TEXT,
		message TEXT,
		triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		acknowledged INTEGER DEFAULT 0,
		acknowledged_by TEXT,
		acknowledged_at TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("failed to ensure alarm_logs table: %w", err)
	}
	return nil
}

// InitDefaultData 初始化默认数据
func InitDefaultData() error {
	var count int
	err := ParamDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		// 表可能不存在或为空，返回错误
		log.Printf("Warning: Failed to query users count: %v, trying to create default user", err)
		// 尝试直接创建用户
		_, err := ParamDB.Exec(
			"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
			"admin", pwdutil.Hash("123456"), "admin",
		)
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}
		log.Println("Created default admin user")
		return nil
	}

	if count == 0 {
		_, err := ParamDB.Exec(
			"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
			"admin", pwdutil.Hash("123456"), "admin",
		)
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}
		log.Println("Created default admin user")
	}

	return nil
}

// ==================== 用户操作 (param.db - 直接写) ====================

// CreateUser 创建用户
func CreateUser(user *models.User) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
		user.Username, user.Password, user.Role,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetUserByUsername 根据用户名获取用户
func GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := ParamDB.QueryRow(
		"SELECT id, username, password, role, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID 根据ID获取用户
func GetUserByID(id int64) (*models.User, error) {
	user := &models.User{}
	err := ParamDB.QueryRow(
		"SELECT id, username, password, role, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetAllUsers 获取所有用户
func GetAllUsers() ([]*models.User, error) {
	rows, err := ParamDB.Query(
		"SELECT id, username, password, role, created_at, updated_at FROM users ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// UpdateUser 更新用户
func UpdateUser(user *models.User) error {
	_, err := ParamDB.Exec(
		"UPDATE users SET username = ?, password = ?, role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		user.Username, user.Password, user.Role, user.ID,
	)
	return err
}

// DeleteUser 删除用户
func DeleteUser(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

// ==================== 驱动操作 (param.db - 直接写) ====================

// CreateDriver 创建驱动
func CreateDriver(driver *models.Driver) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO drivers (name, file_path, description, version, config_schema, enabled) VALUES (?, ?, ?, ?, ?, ?)",
		driver.Name, driver.FilePath, driver.Description, driver.Version, driver.ConfigSchema, driver.Enabled,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetDriverByID 根据ID获取驱动
func GetDriverByID(id int64) (*models.Driver, error) {
	driver := &models.Driver{}
	err := ParamDB.QueryRow(
		"SELECT id, name, file_path, description, version, config_schema, enabled, created_at, updated_at FROM drivers WHERE id = ?",
		id,
	).Scan(&driver.ID, &driver.Name, &driver.FilePath, &driver.Description, &driver.Version, &driver.ConfigSchema,
		&driver.Enabled, &driver.CreatedAt, &driver.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return driver, nil
}

// GetAllDrivers 获取所有驱动
func GetAllDrivers() ([]*models.Driver, error) {
	rows, err := ParamDB.Query(
		"SELECT id, name, file_path, description, version, config_schema, enabled, created_at, updated_at FROM drivers ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drivers []*models.Driver
	for rows.Next() {
		driver := &models.Driver{}
		if err := rows.Scan(&driver.ID, &driver.Name, &driver.FilePath, &driver.Description, &driver.Version, &driver.ConfigSchema,
			&driver.Enabled, &driver.CreatedAt, &driver.UpdatedAt); err != nil {
			return nil, err
		}
		drivers = append(drivers, driver)
	}
	return drivers, nil
}

// UpdateDriver 更新驱动
func UpdateDriver(driver *models.Driver) error {
	_, err := ParamDB.Exec(
		"UPDATE drivers SET name = ?, file_path = ?, description = ?, version = ?, config_schema = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		driver.Name, driver.FilePath, driver.Description, driver.Version, driver.ConfigSchema, driver.Enabled, driver.ID,
	)
	return err
}

// DeleteDriver 删除驱动
func DeleteDriver(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM drivers WHERE id = ?", id)
	return err
}

// GetDriverByName 根据名称获取驱动
func GetDriverByName(name string) (*models.Driver, error) {
	driver := &models.Driver{}
	err := ParamDB.QueryRow(
		"SELECT id, name, file_path, description, version, config_schema, enabled, created_at, updated_at FROM drivers WHERE name = ?",
		name,
	).Scan(&driver.ID, &driver.Name, &driver.FilePath, &driver.Description, &driver.Version, &driver.ConfigSchema,
		&driver.Enabled, &driver.CreatedAt, &driver.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return driver, nil
}

// UpsertDriverFile 保存或忽略重复的驱动记录
func UpsertDriverFile(name, path string) error {
	_, err := ParamDB.Exec(
		`INSERT OR IGNORE INTO drivers (name, file_path, description, version, config_schema, enabled) 
		 VALUES (?, ?, '', '', '', 1)`, name, path)
	return err
}

// ==================== 北向配置操作 (param.db - 直接写) ====================

// CreateNorthboundConfig 创建北向配置
func CreateNorthboundConfig(config *models.NorthboundConfig) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO northbound_configs (name, type, enabled, config, upload_interval) VALUES (?, ?, ?, ?, ?)",
		config.Name, config.Type, config.Enabled, config.Config, config.UploadInterval,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetNorthboundConfigByID 根据ID获取北向配置
func GetNorthboundConfigByID(id int64) (*models.NorthboundConfig, error) {
	config := &models.NorthboundConfig{}
	err := ParamDB.QueryRow(
		"SELECT id, name, type, enabled, config, upload_interval, created_at, updated_at FROM northbound_configs WHERE id = ?",
		id,
	).Scan(&config.ID, &config.Name, &config.Type, &config.Enabled, &config.Config, &config.UploadInterval,
		&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetAllNorthboundConfigs 获取所有北向配置
func GetAllNorthboundConfigs() ([]*models.NorthboundConfig, error) {
	rows, err := ParamDB.Query(
		"SELECT id, name, type, enabled, config, upload_interval, created_at, updated_at FROM northbound_configs ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.NorthboundConfig
	for rows.Next() {
		config := &models.NorthboundConfig{}
		if err := rows.Scan(&config.ID, &config.Name, &config.Type, &config.Enabled, &config.Config, &config.UploadInterval,
			&config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// GetEnabledNorthboundConfigs 获取所有启用的北向配置
func GetEnabledNorthboundConfigs() ([]*models.NorthboundConfig, error) {
	rows, err := ParamDB.Query(
		"SELECT id, name, type, enabled, config, upload_interval, created_at, updated_at FROM northbound_configs WHERE enabled = 1 ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.NorthboundConfig
	for rows.Next() {
		config := &models.NorthboundConfig{}
		if err := rows.Scan(&config.ID, &config.Name, &config.Type, &config.Enabled, &config.Config, &config.UploadInterval,
			&config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// UpdateNorthboundConfig 更新北向配置
func UpdateNorthboundConfig(config *models.NorthboundConfig) error {
	_, err := ParamDB.Exec(
		"UPDATE northbound_configs SET name = ?, type = ?, enabled = ?, config = ?, upload_interval = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		config.Name, config.Type, config.Enabled, config.Config, config.UploadInterval, config.ID,
	)
	return err
}

// UpdateNorthboundEnabled 更新北向使能状态
func UpdateNorthboundEnabled(id int64, enabled int) error {
	_, err := ParamDB.Exec("UPDATE northbound_configs SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", enabled, id)
	return err
}

// DeleteNorthboundConfig 删除北向配置
func DeleteNorthboundConfig(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM northbound_configs WHERE id = ?", id)
	return err
}

// ==================== 阈值操作 (param.db - 直接写) ====================

// CreateThreshold 创建阈值
func CreateThreshold(threshold *models.Threshold) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO thresholds (device_id, field_name, operator, value, severity, enabled, message) VALUES (?, ?, ?, ?, ?, ?, ?)",
		threshold.DeviceID, threshold.FieldName, threshold.Operator, threshold.Value, threshold.Severity, threshold.Enabled, threshold.Message,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetThresholdByID 根据ID获取阈值
func GetThresholdByID(id int64) (*models.Threshold, error) {
	threshold := &models.Threshold{}
	err := ParamDB.QueryRow(
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds WHERE id = ?",
		id,
	).Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
		&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return threshold, nil
}

// GetThresholdsByDeviceID 根据设备ID获取阈值
func GetThresholdsByDeviceID(deviceID int64) ([]*models.Threshold, error) {
	rows, err := ParamDB.Query(
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds WHERE device_id = ?",
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var thresholds []*models.Threshold
	for rows.Next() {
		threshold := &models.Threshold{}
		if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
			&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
			return nil, err
		}
		thresholds = append(thresholds, threshold)
	}
	return thresholds, nil
}

// GetEnabledThresholdsByDeviceID 根据设备ID获取启用的阈值
func GetEnabledThresholdsByDeviceID(deviceID int64) ([]*models.Threshold, error) {
	rows, err := ParamDB.Query(
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds WHERE device_id = ? AND enabled = 1",
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var thresholds []*models.Threshold
	for rows.Next() {
		threshold := &models.Threshold{}
		if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
			&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
			return nil, err
		}
		thresholds = append(thresholds, threshold)
	}
	return thresholds, nil
}

// GetAllThresholds 获取所有阈值
func GetAllThresholds() ([]*models.Threshold, error) {
	rows, err := ParamDB.Query(
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var thresholds []*models.Threshold
	for rows.Next() {
		threshold := &models.Threshold{}
		if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
			&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
			return nil, err
		}
		thresholds = append(thresholds, threshold)
	}
	return thresholds, nil
}

// UpdateThreshold 更新阈值
func UpdateThreshold(threshold *models.Threshold) error {
	_, err := ParamDB.Exec(
		"UPDATE thresholds SET device_id = ?, field_name = ?, operator = ?, value = ?, severity = ?, enabled = ?, message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		threshold.DeviceID, threshold.FieldName, threshold.Operator, threshold.Value, threshold.Severity, threshold.Enabled, threshold.Message, threshold.ID,
	)
	return err
}

// DeleteThreshold 删除阈值
func DeleteThreshold(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM thresholds WHERE id = ?", id)
	return err
}

// ==================== 报警日志操作 (param.db - 直接写) ====================

// CreateAlarmLog 创建报警日志
func CreateAlarmLog(log *models.AlarmLog) (int64, error) {
	result, err := ParamDB.Exec(
		`INSERT INTO alarm_logs (device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.DeviceID, log.ThresholdID, log.FieldName, log.ActualValue, log.ThresholdValue, log.Operator, log.Severity, log.Message,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetAlarmLogsByDeviceID 根据设备ID获取报警日志
func GetAlarmLogsByDeviceID(deviceID int64, limit int) ([]*models.AlarmLog, error) {
	rows, err := ParamDB.Query(
		`SELECT id, device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, triggered_at, acknowledged, acknowledged_by, acknowledged_at 
		FROM alarm_logs WHERE device_id = ? ORDER BY triggered_at DESC LIMIT ?`,
		deviceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AlarmLog
	for rows.Next() {
		log := &models.AlarmLog{}
		if err := rows.Scan(&log.ID, &log.DeviceID, &log.ThresholdID, &log.FieldName, &log.ActualValue, &log.ThresholdValue,
			&log.Operator, &log.Severity, &log.Message, &log.TriggeredAt, &log.Acknowledged, &log.AcknowledgedBy, &log.AcknowledgedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetRecentAlarmLogs 获取最近的报警日志
func GetRecentAlarmLogs(limit int) ([]*models.AlarmLog, error) {
	rows, err := ParamDB.Query(
		`SELECT id, device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, triggered_at, acknowledged, acknowledged_by, acknowledged_at 
		FROM alarm_logs ORDER BY triggered_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.AlarmLog
	for rows.Next() {
		log := &models.AlarmLog{}
		if err := rows.Scan(&log.ID, &log.DeviceID, &log.ThresholdID, &log.FieldName, &log.ActualValue, &log.ThresholdValue,
			&log.Operator, &log.Severity, &log.Message, &log.TriggeredAt, &log.Acknowledged, &log.AcknowledgedBy, &log.AcknowledgedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// AcknowledgeAlarmLog 确认报警日志
func AcknowledgeAlarmLog(id int64, acknowledgedBy string) error {
	now := time.Now()
	_, err := ParamDB.Exec(
		"UPDATE alarm_logs SET acknowledged = 1, acknowledged_by = ?, acknowledged_at = ? WHERE id = ?",
		acknowledgedBy, now, id,
	)
	return err
}

// ==================== 存储配置操作 (param.db - 直接写) ====================

// StorageConfig 存储配置模型（沿用存量表名 storage_policies）
type StorageConfig struct {
	ID          int64     `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	ProductKey  string    `json:"product_key" db:"product_key"`
	DeviceKey   string    `json:"device_key" db:"device_key"`
	StorageDays int       `json:"storage_days" db:"storage_days"`
	Enabled     int       `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateStorageConfig 创建存储配置
func CreateStorageConfig(config *StorageConfig) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO storage_policies (name, product_key, device_key, storage_days, enabled) VALUES (?, ?, ?, ?, ?)",
		config.Name, config.ProductKey, config.DeviceKey, config.StorageDays, config.Enabled,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetAllStorageConfigs 获取所有存储配置
func GetAllStorageConfigs() ([]*StorageConfig, error) {
	rows, err := ParamDB.Query(
		"SELECT id, name, product_key, device_key, storage_days, enabled, created_at, updated_at FROM storage_policies ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*StorageConfig
	for rows.Next() {
		config := &StorageConfig{}
		if err := rows.Scan(&config.ID, &config.Name, &config.ProductKey, &config.DeviceKey, &config.StorageDays, &config.Enabled,
			&config.CreatedAt, &config.UpdatedAt); err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, nil
}

// GetStorageConfigByID 根据ID获取存储配置
func GetStorageConfigByID(id int64) (*StorageConfig, error) {
	config := &StorageConfig{}
	err := ParamDB.QueryRow(
		"SELECT id, name, product_key, device_key, storage_days, enabled, created_at, updated_at FROM storage_policies WHERE id = ?",
		id,
	).Scan(&config.ID, &config.Name, &config.ProductKey, &config.DeviceKey, &config.StorageDays, &config.Enabled,
		&config.CreatedAt, &config.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateStorageConfig 更新存储配置
func UpdateStorageConfig(config *StorageConfig) error {
	_, err := ParamDB.Exec(
		"UPDATE storage_policies SET name = ?, product_key = ?, device_key = ?, storage_days = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		config.Name, config.ProductKey, config.DeviceKey, config.StorageDays, config.Enabled, config.ID,
	)
	return err
}

// DeleteStorageConfig 删除存储配置
func DeleteStorageConfig(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM storage_policies WHERE id = ?", id)
	return err
}

// ==================== 实时数据缓存操作 (param.db) ====================

// SaveDataCache 保存实时数据缓存（内存）
func SaveDataCache(deviceID int64, deviceName, fieldName, value, valueType string) error {
	_, err := ParamDB.Exec(
		`INSERT OR REPLACE INTO data_cache (device_id, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		deviceID, fieldName, value, valueType,
	)
	if err != nil {
		return err
	}

	// 检查并清理过量的缓存条目
	enforceDataCacheLimit()
	return nil
}

// enforceDataCacheLimit 强制执行缓存大小限制
func enforceDataCacheLimit() {
	var count int
	ParamDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&count)
	if count > MaxDataCache {
		ParamDB.Exec("DELETE FROM data_cache WHERE id IN (SELECT id FROM data_cache ORDER BY collected_at ASC LIMIT ?)", count-MaxDataCache)
		log.Printf("Cleaned up data cache, removed %d entries", count-MaxDataCache)
	}
}

// GetDataCacheByDeviceID 根据设备ID获取数据缓存（从内存）
func GetDataCacheByDeviceID(deviceID int64) ([]*models.DataCache, error) {
	rows, err := ParamDB.Query(
		"SELECT id, device_id, field_name, value, value_type, collected_at FROM data_cache WHERE device_id = ?",
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cache []*models.DataCache
	for rows.Next() {
		item := &models.DataCache{}
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.FieldName, &item.Value, &item.ValueType, &item.CollectedAt); err != nil {
			return nil, err
		}
		cache = append(cache, item)
	}
	return cache, nil
}

// GetAllDataCache 获取所有数据缓存（从内存）
func GetAllDataCache() ([]*models.DataCache, error) {
	rows, err := ParamDB.Query(
		"SELECT id, device_id, field_name, value, value_type, collected_at FROM data_cache",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cache []*models.DataCache
	for rows.Next() {
		item := &models.DataCache{}
		if err := rows.Scan(&item.ID, &item.DeviceID, &item.FieldName, &item.Value, &item.ValueType, &item.CollectedAt); err != nil {
			return nil, err
		}
		cache = append(cache, item)
	}
	return cache, nil
}

// ==================== 历史数据点操作 (data.db - 内存暂存) ====================

// DataPoint 历史数据点
type DataPoint struct {
	ID          int64     `json:"id"`
	DeviceID    int64     `json:"device_id"`
	DeviceName  string    `json:"device_name"`
	FieldName   string    `json:"field_name"`
	Value       string    `json:"value"`
	ValueType   string    `json:"value_type"`
	CollectedAt time.Time `json:"collected_at"`
}

// SaveDataPoint 保存历史数据点（内存暂存）
func SaveDataPoint(deviceID int64, deviceName, fieldName, value, valueType string) error {
	_, err := DataDB.Exec(
		`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		deviceID, deviceName, fieldName, value, valueType,
	)
	if err != nil {
		return err
	}

	// 检查并清理过量的数据点
	enforceDataPointsLimit()
	return nil
}

// DataPointEntry 单个数据点条目
type DataPointEntry struct {
	DeviceID    int64
	DeviceName  string
	FieldName   string
	Value       string
	ValueType   string
	CollectedAt time.Time
}

// BatchSaveDataPoints 批量保存历史数据点（提高写入性能）
func BatchSaveDataPoints(entries []DataPointEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// 使用事务批量插入
	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO data_points
		(device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, entry := range entries {
		collectedAt := entry.CollectedAt
		if collectedAt.IsZero() {
			collectedAt = time.Now()
		}
		if _, err := stmt.Exec(entry.DeviceID, entry.DeviceName, entry.FieldName,
			entry.Value, entry.ValueType, collectedAt); err != nil {
			return fmt.Errorf("failed to insert data point: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 检查并清理过量的数据点
	enforceDataPointsLimit()

	// 检查是否需要触发同步
	TriggerSyncIfNeeded()

	return nil
}

// enforceDataPointsLimit 强制执行数据点大小限制
func enforceDataPointsLimit() {
	var count int
	DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count)
	if count > MaxDataPoints {
		DataDB.Exec("DELETE FROM data_points WHERE id IN (SELECT id FROM data_points ORDER BY collected_at ASC LIMIT ?)", count-MaxDataPoints)
		log.Printf("Cleaned up data points, removed %d old entries", count-MaxDataPoints)
	}
}

// GetDataPointsByDeviceAndTime 根据设备ID和时间范围获取历史数据（从内存）
func GetDataPointsByDeviceAndTime(deviceID int64, startTime, endTime time.Time) ([]*DataPoint, error) {
	rows, err := DataDB.Query(
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ? AND collected_at >= ? AND collected_at <= ? 
		ORDER BY collected_at DESC`,
		deviceID, startTime, endTime,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []*DataPoint
	for rows.Next() {
		point := &DataPoint{}
		if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

// GetDataPointsByDevice 根据设备ID获取历史数据（从内存）
func GetDataPointsByDevice(deviceID int64, limit int) ([]*DataPoint, error) {
	rows, err := DataDB.Query(
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ? ORDER BY collected_at DESC LIMIT ?`,
		deviceID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []*DataPoint
	for rows.Next() {
		point := &DataPoint{}
		if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

// GetLatestDataPoints 获取最新的历史数据点（从内存）
func GetLatestDataPoints(limit int) ([]*DataPoint, error) {
	rows, err := DataDB.Query(
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points ORDER BY collected_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []*DataPoint
	for rows.Next() {
		point := &DataPoint{}
		if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, nil
}

// CleanupOldData 清理过期数据（从内存和磁盘）
func CleanupOldData() (int64, error) {
	// 清理内存中的数据，默认 30 天，实际以策略为准
	result, err := DataDB.Exec(
		`DELETE FROM data_points WHERE collected_at < datetime('now', '-30 days')`,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// InsertCollectData 将采集数据写入缓存与历史库
func InsertCollectData(data *models.CollectData) error {
	if data == nil {
		return fmt.Errorf("collect data is nil")
	}

	entries := make([]DataPointEntry, 0, len(data.Fields))
	for field, value := range data.Fields {
		if err := SaveDataCache(data.DeviceID, data.DeviceName, field, value, "string"); err != nil {
			log.Printf("SaveDataCache error: %v", err)
		}
		entries = append(entries, DataPointEntry{
			DeviceID:    data.DeviceID,
			DeviceName:  data.DeviceName,
			FieldName:   field,
			Value:       value,
			ValueType:   "string",
			CollectedAt: data.Timestamp,
		})
	}
	return BatchSaveDataPoints(entries)
}

// CleanupOldDataByConfig 根据存储配置清理过期数据
func CleanupOldDataByConfig() (int64, error) {
	configs, err := GetAllStorageConfigs()
	if err != nil {
		return 0, err
	}

	var (
		totalDeleted int64
		globalDays   = retentionDaysDefault
		protectedIDs = make(map[int64]struct{}) // 具备专属策略的设备，避免全局策略覆盖
	)

	for _, config := range configs {
		if config.Enabled != 1 {
			continue
		}

		// 设备专属策略：按 product_key/device_key 过滤
		if config.ProductKey != "" || config.DeviceKey != "" {
			ids, err := findDeviceIDs(config.ProductKey, config.DeviceKey)
			if err != nil {
				return totalDeleted, err
			}
			if len(ids) == 0 {
				continue
			}
			for _, id := range ids {
				protectedIDs[id] = struct{}{}
			}
			days := config.StorageDays
			if days <= 0 {
				days = DefaultRetentionDays
			}
			placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
			args := make([]interface{}, 0, len(ids)+1)
			args = append(args, fmt.Sprintf("-%d days", days))
			for _, id := range ids {
				args = append(args, id)
			}
			query := fmt.Sprintf(`DELETE FROM data_points WHERE collected_at < datetime('now', ?) AND device_id IN (%s)`, placeholders)
			result, err := DataDB.Exec(query, args...)
			if err != nil {
				return totalDeleted, err
			}
			deleted, _ := result.RowsAffected()
			totalDeleted += deleted
			continue
		}

		// 全局策略
		if config.StorageDays > 0 {
			globalDays = config.StorageDays
		}
	}

	// 全局策略（或默认值），排除已受专属策略保护的设备
	idsToExclude := make([]int64, 0, len(protectedIDs))
	for id := range protectedIDs {
		idsToExclude = append(idsToExclude, id)
	}

	args := []interface{}{fmt.Sprintf("-%d days", globalDays)}
	query := `DELETE FROM data_points WHERE collected_at < datetime('now', ?)`
	if len(idsToExclude) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(idsToExclude)), ",")
		query += fmt.Sprintf(" AND device_id NOT IN (%s)", placeholders)
		for _, id := range idsToExclude {
			args = append(args, id)
		}
	}

	result, err := DataDB.Exec(query, args...)
	if err != nil {
		return totalDeleted, err
	}
	deleted, _ := result.RowsAffected()
	totalDeleted += deleted

	return totalDeleted, nil
}

// findDeviceIDs 按 product_key/device_key 查找设备ID
func findDeviceIDs(productKey, deviceKey string) ([]int64, error) {
	query := "SELECT id FROM devices WHERE 1=1"
	args := []interface{}{}
	if productKey != "" {
		query += " AND product_key = ?"
		args = append(args, productKey)
	}
	if deviceKey != "" {
		query += " AND device_key = ?"
		args = append(args, deviceKey)
	}

	rows, err := ParamDB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// CleanupData 清理过期数据（根据时间戳）
func CleanupData(before string) (int64, error) {
	result, err := DataDB.Exec(
		"DELETE FROM data_points WHERE collected_at < ?",
		before,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
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
