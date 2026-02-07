package database

import (
	"database/sql"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 北向配置操作 (param.db - 直接写) ====================

// CreateNorthboundConfig 创建北向配置
func CreateNorthboundConfig(config *models.NorthboundConfig) (int64, error) {
	result, err := ParamDB.Exec(
		`INSERT INTO northbound_configs (
			name, type, enabled, upload_interval,
			server_url, port, path, username, password, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		config.Name, config.Type, config.Enabled, config.UploadInterval,
		config.ServerURL, config.Port, config.Path, config.Username, config.Password, config.ClientID,
		config.Topic, config.AlarmTopic,
		config.QOS, config.Retain, config.KeepAlive, config.Timeout,
		config.ProductKey, config.DeviceKey,
		config.ExtConfig,
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
		`SELECT id, name, type, enabled, upload_interval,
			server_url, port, path, username, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config,
			connected, last_connected_at,
			created_at, updated_at
		FROM northbound_configs WHERE id = ?`,
		id,
	).Scan(
		&config.ID, &config.Name, &config.Type, &config.Enabled, &config.UploadInterval,
		&config.ServerURL, &config.Port, &config.Path, &config.Username, &config.ClientID,
		&config.Topic, &config.AlarmTopic,
		&config.QOS, &config.Retain, &config.KeepAlive, &config.Timeout,
		&config.ProductKey, &config.DeviceKey,
		&config.ExtConfig,
		&config.Connected, &config.LastConnectedAt,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetAllNorthboundConfigs 获取所有北向配置
func GetAllNorthboundConfigs() ([]*models.NorthboundConfig, error) {
	return queryList[*models.NorthboundConfig](ParamDB,
		`SELECT id, name, type, enabled, upload_interval,
			server_url, port, path, username, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config,
			connected, last_connected_at,
			created_at, updated_at
		FROM northbound_configs ORDER BY id`,
		nil,
		func(rows *sql.Rows) (*models.NorthboundConfig, error) {
			config := &models.NorthboundConfig{}
			if err := rows.Scan(
				&config.ID, &config.Name, &config.Type, &config.Enabled, &config.UploadInterval,
				&config.ServerURL, &config.Port, &config.Path, &config.Username, &config.ClientID,
				&config.Topic, &config.AlarmTopic,
				&config.QOS, &config.Retain, &config.KeepAlive, &config.Timeout,
				&config.ProductKey, &config.DeviceKey,
				&config.ExtConfig,
				&config.Connected, &config.LastConnectedAt,
				&config.CreatedAt, &config.UpdatedAt,
			); err != nil {
				return nil, err
			}
			return config, nil
		},
	)
}

// GetEnabledNorthboundConfigs 获取所有启用的北向配置
func GetEnabledNorthboundConfigs() ([]*models.NorthboundConfig, error) {
	return queryList[*models.NorthboundConfig](ParamDB,
		`SELECT id, name, type, enabled, upload_interval,
			server_url, port, path, username, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config,
			connected, last_connected_at,
			created_at, updated_at
		FROM northbound_configs WHERE enabled = 1 ORDER BY id`,
		nil,
		func(rows *sql.Rows) (*models.NorthboundConfig, error) {
			config := &models.NorthboundConfig{}
			if err := rows.Scan(
				&config.ID, &config.Name, &config.Type, &config.Enabled, &config.UploadInterval,
				&config.ServerURL, &config.Port, &config.Path, &config.Username, &config.ClientID,
				&config.Topic, &config.AlarmTopic,
				&config.QOS, &config.Retain, &config.KeepAlive, &config.Timeout,
				&config.ProductKey, &config.DeviceKey,
				&config.ExtConfig,
				&config.Connected, &config.LastConnectedAt,
				&config.CreatedAt, &config.UpdatedAt,
			); err != nil {
				return nil, err
			}
			return config, nil
		},
	)
}

// UpdateNorthboundConfig 更新北向配置
func UpdateNorthboundConfig(config *models.NorthboundConfig) error {
	_, err := ParamDB.Exec(
		`UPDATE northbound_configs SET
			name = ?, type = ?, enabled = ?, upload_interval = ?,
			server_url = ?, port = ?, path = ?, username = ?, password = ?, client_id = ?,
			topic = ?, alarm_topic = ?,
			qos = ?, retain = ?, keep_alive = ?, timeout = ?,
			product_key = ?, device_key = ?,
			ext_config = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		config.Name, config.Type, config.Enabled, config.UploadInterval,
		config.ServerURL, config.Port, config.Path, config.Username, config.Password, config.ClientID,
		config.Topic, config.AlarmTopic,
		config.QOS, config.Retain, config.KeepAlive, config.Timeout,
		config.ProductKey, config.DeviceKey,
		config.ExtConfig,
		config.ID,
	)
	return err
}

// UpdateNorthboundEnabled 更新北向使能状态
func UpdateNorthboundEnabled(id int64, enabled int) error {
	_, err := ParamDB.Exec(
		"UPDATE northbound_configs SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		enabled, id,
	)
	return err
}

// UpdateNorthboundConnected 更新北向连接状态
func UpdateNorthboundConnected(id int64, connected bool) error {
	var lastConnected interface{}
	if connected {
		now := getCurrentTime()
		lastConnected = &now
	}
	_, err := ParamDB.Exec(
		"UPDATE northbound_configs SET connected = ?, last_connected_at = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		connected, lastConnected, id,
	)
	return err
}

// DeleteNorthboundConfig 删除北向配置
func DeleteNorthboundConfig(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM northbound_configs WHERE id = ?", id)
	return err
}

// getCurrentTime 获取当前时间
func getCurrentTime() time.Time {
	return time.Now()
}
