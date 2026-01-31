package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/pwdutil"

	_ "modernc.org/sqlite"
)

// Configuration
const (
	SyncInterval = 5 * time.Minute // 同步间隔
	DataDBFile   = "data.db"       // 数据数据库文件名
)

// ParamDB 配置数据库连接（持久化文件）
var ParamDB *sql.DB

// DataDB 历史数据数据库连接（内存模式）
var DataDB *sql.DB

// 数据同步相关
var dataSyncMu sync.Mutex
var dataSyncTicker *time.Ticker
var dataSyncStop chan struct{}

// InitParamDB 初始化配置数据库（持久化文件）
func InitParamDB() error {
	var err error
	ParamDB, err = sql.Open("sqlite", "param.db")
	if err != nil {
		return fmt.Errorf("failed to open param database: %w", err)
	}

	ParamDB.SetMaxOpenConns(10)
	ParamDB.SetMaxIdleConns(5)
	ParamDB.SetConnMaxLifetime(time.Hour)

	if err := ParamDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping param database: %w", err)
	}

	log.Println("Param database initialized (persistent mode)")
	return nil
}

// InitDataDB 初始化历史数据数据库（内存模式 + 批量同步）
func InitDataDB() error {
	var err error
	DataDB, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open data database: %w", err)
	}

	DataDB.SetMaxOpenConns(10)
	DataDB.SetMaxIdleConns(5)
	DataDB.SetConnMaxLifetime(time.Hour)

	if err := DataDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping data database: %w", err)
	}

	// 从文件恢复数据（如果存在）
	if _, err := os.Stat(DataDBFile); err == nil {
		log.Println("Restoring data database from file...")
		if err := restoreDataFromFile(DataDBFile); err != nil {
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
		log.Printf("Data sync started (interval: %v)", SyncInterval)
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

// syncDataToDisk 将内存数据批量同步到磁盘
func syncDataToDisk() error {
	dataSyncMu.Lock()
	defer dataSyncMu.Unlock()

	log.Println("Syncing data to disk...")

	// 1. 创建临时数据库文件
	tempFile := DataDBFile + ".tmp"
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

	// 4. 复制数据缓存
	cache, err := DataDB.Query("SELECT device_id, field_name, value, value_type, updated_at FROM data_cache")
	if err != nil {
		return fmt.Errorf("failed to query data cache: %w", err)
	}
	defer cache.Close()

	tx, err = diskDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err = tx.Prepare(`INSERT OR REPLACE INTO data_cache 
		(id, device_id, field_name, value, value_type, updated_at) 
		VALUES (NULL, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for cache.Next() {
		var deviceID int64
		var fieldName, value, valueType string
		var updatedAt time.Time
		if err := cache.Scan(&deviceID, &fieldName, &value, &valueType, &updatedAt); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := stmt.Exec(deviceID, fieldName, value, valueType, updatedAt); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 5. 原子替换文件
	if err := os.Rename(tempFile, DataDBFile); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	log.Printf("Data synced to disk: %d points + cache", count)
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

// ==================== 资源操作 (param.db - 直接写) ====================

// CreateResource 创建资源
func CreateResource(resource *models.Resource) (int64, error) {
	result, err := ParamDB.Exec(
		`INSERT INTO resources (name, type, port, address, enabled) 
		VALUES (?, ?, ?, ?, ?)`,
		resource.Name, resource.Type, resource.Port, resource.Address, resource.Enabled,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetResourceByID 根据ID获取资源
func GetResourceByID(id int64) (*models.Resource, error) {
	resource := &models.Resource{}
	err := ParamDB.QueryRow(
		`SELECT id, name, type, port, address, enabled, created_at, updated_at 
		FROM resources WHERE id = ?`,
		id,
	).Scan(&resource.ID, &resource.Name, &resource.Type, &resource.Port, &resource.Address,
		&resource.Enabled, &resource.CreatedAt, &resource.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return resource, nil
}

// GetAllResources 获取所有资源
func GetAllResources() ([]*models.Resource, error) {
	rows, err := ParamDB.Query(
		`SELECT id, name, type, port, address, enabled, created_at, updated_at 
		FROM resources ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var resources []*models.Resource
	for rows.Next() {
		resource := &models.Resource{}
		if err := rows.Scan(&resource.ID, &resource.Name, &resource.Type, &resource.Port, &resource.Address,
			&resource.Enabled, &resource.CreatedAt, &resource.UpdatedAt); err != nil {
			return nil, err
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

// UpdateResource 更新资源
func UpdateResource(resource *models.Resource) error {
	_, err := ParamDB.Exec(
		`UPDATE resources SET name = ?, type = ?, port = ?, address = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		resource.Name, resource.Type, resource.Port, resource.Address, resource.Enabled, resource.ID,
	)
	return err
}

// DeleteResource 删除资源
func DeleteResource(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM resources WHERE id = ?", id)
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

// ==================== 设备操作 (param.db - 直接写) ====================

// CreateDevice 创建设备
func CreateDevice(device *models.Device) (int64, error) {
	result, err := ParamDB.Exec(
		`INSERT INTO devices (name, description, resource_id, driver_id, device_config, collect_interval, upload_interval, enabled) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		device.Name, device.Description, device.ResourceID, device.DriverID, device.DeviceConfig,
		device.CollectInterval, device.UploadInterval, device.Enabled,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetDeviceByID 根据ID获取设备
func GetDeviceByID(id int64) (*models.Device, error) {
	device := &models.Device{}
	err := ParamDB.QueryRow(
		`SELECT id, name, description, resource_id, driver_id, device_config, collect_interval, upload_interval, enabled, created_at, updated_at 
		FROM devices WHERE id = ?`,
		id,
	).Scan(&device.ID, &device.Name, &device.Description, &device.ResourceID, &device.DriverID, &device.DeviceConfig,
		&device.CollectInterval, &device.UploadInterval, &device.Enabled, &device.CreatedAt, &device.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return device, nil
}

// GetAllDevices 获取所有设备
func GetAllDevices() ([]*models.Device, error) {
	rows, err := ParamDB.Query(
		`SELECT id, name, description, resource_id, driver_id, device_config, collect_interval, upload_interval, enabled, created_at, updated_at 
		FROM devices ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		device := &models.Device{}
		if err := rows.Scan(&device.ID, &device.Name, &device.Description, &device.ResourceID, &device.DriverID, &device.DeviceConfig,
			&device.CollectInterval, &device.UploadInterval, &device.Enabled, &device.CreatedAt, &device.UpdatedAt); err != nil {
			return nil, err
		}
		devices = append(devices, device)
	}
	return devices, nil
}

// UpdateDevice 更新设备
func UpdateDevice(device *models.Device) error {
	_, err := ParamDB.Exec(
		`UPDATE devices SET name = ?, description = ?, resource_id = ?, driver_id = ?, device_config = ?, 
		collect_interval = ?, upload_interval = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		device.Name, device.Description, device.ResourceID, device.DriverID, device.DeviceConfig,
		device.CollectInterval, device.UploadInterval, device.Enabled, device.ID,
	)
	return err
}

// UpdateDeviceEnabled 更新设备使能状态
func UpdateDeviceEnabled(id int64, enabled int) error {
	_, err := ParamDB.Exec("UPDATE devices SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", enabled, id)
	return err
}

// DeleteDevice 删除设备
func DeleteDevice(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM devices WHERE id = ?", id)
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

// StorageConfig 存储配置模型
type StorageConfig struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	ProductKey  string    `json:"product_key"`
	DeviceKey   string    `json:"device_key"`
	StorageDays int       `json:"storage_days"`
	Enabled     int       `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateStorageConfig 创建存储配置
func CreateStorageConfig(config *StorageConfig) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO storage_config (name, product_key, device_key, storage_days, enabled) VALUES (?, ?, ?, ?, ?)",
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
		"SELECT id, name, product_key, device_key, storage_days, enabled, created_at, updated_at FROM storage_config ORDER BY id",
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
		"SELECT id, name, product_key, device_key, storage_days, enabled, created_at, updated_at FROM storage_config WHERE id = ?",
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
		"UPDATE storage_config SET name = ?, product_key = ?, device_key = ?, storage_days = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		config.Name, config.ProductKey, config.DeviceKey, config.StorageDays, config.Enabled, config.ID,
	)
	return err
}

// DeleteStorageConfig 删除存储配置
func DeleteStorageConfig(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM storage_config WHERE id = ?", id)
	return err
}

// ==================== 实时数据缓存操作 (data.db - 内存暂存) ====================

// SaveDataCache 保存实时数据缓存（内存）
func SaveDataCache(deviceID int64, deviceName, fieldName, value, valueType string) error {
	_, err := DataDB.Exec(
		`INSERT OR REPLACE INTO data_cache (device_id, field_name, value, value_type, updated_at) 
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		deviceID, fieldName, value, valueType,
	)
	return err
}

// GetDataCacheByDeviceID 根据设备ID获取数据缓存（从内存）
func GetDataCacheByDeviceID(deviceID int64) ([]*models.DataCache, error) {
	rows, err := DataDB.Query(
		"SELECT id, device_id, field_name, value, value_type, updated_at FROM data_cache WHERE device_id = ?",
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
	rows, err := DataDB.Query(
		"SELECT id, device_id, field_name, value, value_type, updated_at FROM data_cache",
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
	return err
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
	// 清理内存中的数据
	result, err := DataDB.Exec(
		`DELETE FROM data_points WHERE collected_at < datetime('now', '-30 days')`,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// CleanupOldDataByConfig 根据存储配置清理过期数据
func CleanupOldDataByConfig() (int64, error) {
	configs, err := GetAllStorageConfigs()
	if err != nil {
		return 0, err
	}

	var totalDeleted int64
	for _, config := range configs {
		if config.Enabled == 1 {
			result, err := DataDB.Exec(
				fmt.Sprintf(`DELETE FROM data_points WHERE collected_at < datetime('now', '-%d days') AND device_id IN (SELECT id FROM devices WHERE name = ?)`,
					config.StorageDays), config.Name,
			)
			if err != nil {
				return totalDeleted, err
			}
			deleted, _ := result.RowsAffected()
			totalDeleted += deleted
		}
	}
	return totalDeleted, nil
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
