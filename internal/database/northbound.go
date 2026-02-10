package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 北向配置操作 (param.db - 直接写) ====================

// CreateNorthboundConfig 创建北向配置
func CreateNorthboundConfig(config *models.NorthboundConfig) (int64, error) {
	if err := ensureNorthboundConfigColumns(); err != nil {
		return 0, err
	}
	result, err := ParamDB.Exec(
		`INSERT INTO northbound_configs (
			name, type, enabled, upload_interval,
			server_url, port, path, username, password, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config, config
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		config.Name, config.Type, config.Enabled, config.UploadInterval,
		config.ServerURL, config.Port, config.Path, config.Username, config.Password, config.ClientID,
		config.Topic, config.AlarmTopic,
		config.QOS, config.Retain, config.KeepAlive, config.Timeout,
		config.ProductKey, config.DeviceKey,
		config.ExtConfig, config.Config,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetNorthboundConfigByID 根据ID获取北向配置
func GetNorthboundConfigByID(id int64) (*models.NorthboundConfig, error) {
	if err := ensureNorthboundConfigColumns(); err != nil {
		return nil, err
	}
	config := &models.NorthboundConfig{}
	err := ParamDB.QueryRow(
		`SELECT id, name, type, enabled, upload_interval,
			server_url, port, path, username, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config, config,
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
		&config.ExtConfig, &config.Config,
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
	if err := ensureNorthboundConfigColumns(); err != nil {
		return nil, err
	}
	return queryList[*models.NorthboundConfig](ParamDB,
		`SELECT id, name, type, enabled, upload_interval,
			server_url, port, path, username, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config, config,
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
				&config.ExtConfig, &config.Config,
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
	if err := ensureNorthboundConfigColumns(); err != nil {
		return nil, err
	}
	return queryList[*models.NorthboundConfig](ParamDB,
		`SELECT id, name, type, enabled, upload_interval,
			server_url, port, path, username, client_id,
			topic, alarm_topic,
			qos, retain, keep_alive, timeout,
			product_key, device_key,
			ext_config, config,
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
				&config.ExtConfig, &config.Config,
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
	if err := ensureNorthboundConfigColumns(); err != nil {
		return err
	}
	_, err := ParamDB.Exec(
		`UPDATE northbound_configs SET
			name = ?, type = ?, enabled = ?, upload_interval = ?,
			server_url = ?, port = ?, path = ?, username = ?, password = ?, client_id = ?,
			topic = ?, alarm_topic = ?,
			qos = ?, retain = ?, keep_alive = ?, timeout = ?,
			product_key = ?, device_key = ?,
			ext_config = ?, config = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		config.Name, config.Type, config.Enabled, config.UploadInterval,
		config.ServerURL, config.Port, config.Path, config.Username, config.Password, config.ClientID,
		config.Topic, config.AlarmTopic,
		config.QOS, config.Retain, config.KeepAlive, config.Timeout,
		config.ProductKey, config.DeviceKey,
		config.ExtConfig, config.Config,
		config.ID,
	)
	return err
}

// UpdateNorthboundEnabled 更新北向使能状态
func UpdateNorthboundEnabled(id int64, enabled int) error {
	if err := ensureNorthboundConfigColumns(); err != nil {
		return err
	}
	_, err := ParamDB.Exec(
		"UPDATE northbound_configs SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		enabled, id,
	)
	return err
}

// UpdateNorthboundConnected 更新北向连接状态
func UpdateNorthboundConnected(id int64, connected bool) error {
	if err := ensureNorthboundConfigColumns(); err != nil {
		return err
	}
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
	if err := ensureNorthboundConfigColumns(); err != nil {
		return err
	}
	_, err := ParamDB.Exec("DELETE FROM northbound_configs WHERE id = ?", id)
	return err
}

var northboundColumnsEnsured bool

func ensureNorthboundConfigColumns() error {
	if northboundColumnsEnsured {
		return nil
	}

	rows, err := ParamDB.Query("PRAGMA table_info(northbound_configs)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasColumn := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		hasColumn[strings.ToLower(strings.TrimSpace(name))] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	columns := []struct {
		name    string
		typeDef string
	}{
		{name: "server_url", typeDef: "TEXT"},
		{name: "port", typeDef: "INTEGER DEFAULT 0"},
		{name: "path", typeDef: "TEXT"},
		{name: "username", typeDef: "TEXT"},
		{name: "password", typeDef: "TEXT"},
		{name: "client_id", typeDef: "TEXT"},
		{name: "topic", typeDef: "TEXT"},
		{name: "alarm_topic", typeDef: "TEXT"},
		{name: "qos", typeDef: "INTEGER DEFAULT 0"},
		{name: "retain", typeDef: "INTEGER DEFAULT 0"},
		{name: "keep_alive", typeDef: "INTEGER DEFAULT 60"},
		{name: "timeout", typeDef: "INTEGER DEFAULT 30"},
		{name: "product_key", typeDef: "TEXT"},
		{name: "device_key", typeDef: "TEXT"},
		{name: "ext_config", typeDef: "TEXT"},
		{name: "connected", typeDef: "INTEGER DEFAULT 0"},
		{name: "last_connected_at", typeDef: "DATETIME"},
	}

	for _, col := range columns {
		if hasColumn[col.name] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE northbound_configs ADD COLUMN %s %s", col.name, col.typeDef)
		if _, err := ParamDB.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := ParamDB.Exec(`UPDATE northbound_configs SET type = 'sagoo' WHERE LOWER(TRIM(type)) = 'xunji'`); err != nil {
		return err
	}

	northboundColumnsEnsured = true
	return nil
}

// getCurrentTime 获取当前时间
func getCurrentTime() time.Time {
	return time.Now()
}
