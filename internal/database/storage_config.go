package database

import (
	"database/sql"
	"time"
)

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
	return queryList[*StorageConfig](ParamDB,
		"SELECT id, name, product_key, device_key, storage_days, enabled, created_at, updated_at FROM storage_policies ORDER BY id",
		nil,
		func(rows *sql.Rows) (*StorageConfig, error) {
			config := &StorageConfig{}
			if err := rows.Scan(&config.ID, &config.Name, &config.ProductKey, &config.DeviceKey, &config.StorageDays, &config.Enabled,
				&config.CreatedAt, &config.UpdatedAt); err != nil {
				return nil, err
			}
			return config, nil
		},
	)
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
