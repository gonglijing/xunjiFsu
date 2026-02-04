package database

import (
	"database/sql"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

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
	return queryList[*models.NorthboundConfig](ParamDB,
		"SELECT id, name, type, enabled, config, upload_interval, created_at, updated_at FROM northbound_configs ORDER BY id",
		nil,
		func(rows *sql.Rows) (*models.NorthboundConfig, error) {
			config := &models.NorthboundConfig{}
			if err := rows.Scan(&config.ID, &config.Name, &config.Type, &config.Enabled, &config.Config, &config.UploadInterval,
				&config.CreatedAt, &config.UpdatedAt); err != nil {
				return nil, err
			}
			return config, nil
		},
	)
}

// GetEnabledNorthboundConfigs 获取所有启用的北向配置
func GetEnabledNorthboundConfigs() ([]*models.NorthboundConfig, error) {
	return queryList[*models.NorthboundConfig](ParamDB,
		"SELECT id, name, type, enabled, config, upload_interval, created_at, updated_at FROM northbound_configs WHERE enabled = 1 ORDER BY id",
		nil,
		func(rows *sql.Rows) (*models.NorthboundConfig, error) {
			config := &models.NorthboundConfig{}
			if err := rows.Scan(&config.ID, &config.Name, &config.Type, &config.Enabled, &config.Config, &config.UploadInterval,
				&config.CreatedAt, &config.UpdatedAt); err != nil {
				return nil, err
			}
			return config, nil
		},
	)
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
