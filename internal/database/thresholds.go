package database

import (
	"database/sql"
	"fmt"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 阈值操作 (param.db - 直接写) ====================

const DefaultAlarmRepeatIntervalSeconds = 60

func ensureThresholdColumns() error {
	hasShielded, err := columnExists(ParamDB, "thresholds", "shielded")
	if err != nil {
		return err
	}
	if !hasShielded {
		if _, err := ParamDB.Exec("ALTER TABLE thresholds ADD COLUMN shielded INTEGER DEFAULT 0"); err != nil {
			return err
		}
	}

	if _, err := ParamDB.Exec("UPDATE thresholds SET shielded = 0 WHERE shielded IS NULL"); err != nil {
		return err
	}

	return nil
}

func ensureGatewayAlarmRepeatIntervalColumn() error {
	if err := InitGatewayConfigTable(); err != nil {
		return err
	}

	hasColumn, err := columnExists(ParamDB, "gateway_config", "alarm_repeat_interval_seconds")
	if err != nil {
		return err
	}
	if !hasColumn {
		if _, err := ParamDB.Exec("ALTER TABLE gateway_config ADD COLUMN alarm_repeat_interval_seconds INTEGER DEFAULT 60"); err != nil {
			return err
		}
	}

	if _, err := ParamDB.Exec("UPDATE gateway_config SET alarm_repeat_interval_seconds = ? WHERE alarm_repeat_interval_seconds IS NULL OR alarm_repeat_interval_seconds <= 0", DefaultAlarmRepeatIntervalSeconds); err != nil {
		return err
	}

	return nil
}

// GetAlarmRepeatIntervalSeconds 获取全局报警重复上报间隔（秒）
func GetAlarmRepeatIntervalSeconds() (int, error) {
	if err := ensureGatewayAlarmRepeatIntervalColumn(); err != nil {
		return DefaultAlarmRepeatIntervalSeconds, err
	}

	var seconds int
	err := ParamDB.QueryRow(
		"SELECT COALESCE(alarm_repeat_interval_seconds, ?) FROM gateway_config ORDER BY id LIMIT 1",
		DefaultAlarmRepeatIntervalSeconds,
	).Scan(&seconds)
	if err != nil {
		return DefaultAlarmRepeatIntervalSeconds, err
	}
	if seconds <= 0 {
		seconds = DefaultAlarmRepeatIntervalSeconds
	}
	return seconds, nil
}

// UpdateAlarmRepeatIntervalSeconds 更新全局报警重复上报间隔（秒）
func UpdateAlarmRepeatIntervalSeconds(seconds int) error {
	if seconds <= 0 {
		return fmt.Errorf("alarm repeat interval must be > 0")
	}
	if err := ensureGatewayAlarmRepeatIntervalColumn(); err != nil {
		return err
	}

	cfg, err := GetGatewayConfig()
	if err != nil {
		return err
	}

	_, err = ParamDB.Exec(
		"UPDATE gateway_config SET alarm_repeat_interval_seconds = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		seconds, cfg.ID,
	)
	return err
}

// CreateThreshold 创建阈值
func CreateThreshold(threshold *models.Threshold) (int64, error) {
	if err := ensureThresholdColumns(); err != nil {
		return 0, err
	}

	result, err := ParamDB.Exec(
		"INSERT INTO thresholds (device_id, field_name, operator, value, severity, shielded, message) VALUES (?, ?, ?, ?, ?, ?, ?)",
		threshold.DeviceID, threshold.FieldName, threshold.Operator, threshold.Value, threshold.Severity, threshold.Shielded, threshold.Message,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetThresholdByID 根据ID获取阈值
func GetThresholdByID(id int64) (*models.Threshold, error) {
	if err := ensureThresholdColumns(); err != nil {
		return nil, err
	}

	threshold := &models.Threshold{}
	err := ParamDB.QueryRow(
		"SELECT id, device_id, field_name, operator, value, severity, COALESCE(shielded, 0), message, created_at, updated_at FROM thresholds WHERE id = ?",
		id,
	).Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
		&threshold.Shielded, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return threshold, nil
}

// GetThresholdsByDeviceID 根据设备ID获取阈值
func GetThresholdsByDeviceID(deviceID int64) ([]*models.Threshold, error) {
	if err := ensureThresholdColumns(); err != nil {
		return nil, err
	}

	return queryList[*models.Threshold](ParamDB,
		"SELECT id, device_id, field_name, operator, value, severity, COALESCE(shielded, 0), message, created_at, updated_at FROM thresholds WHERE device_id = ?",
		[]any{deviceID},
		func(rows *sql.Rows) (*models.Threshold, error) {
			threshold := &models.Threshold{}
			if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
				&threshold.Shielded, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
				return nil, err
			}
			return threshold, nil
		},
	)
}

// GetAllThresholds 获取所有阈值
func GetAllThresholds() ([]*models.Threshold, error) {
	if err := ensureThresholdColumns(); err != nil {
		return nil, err
	}

	return queryList[*models.Threshold](ParamDB,
		"SELECT id, device_id, field_name, operator, value, severity, COALESCE(shielded, 0), message, created_at, updated_at FROM thresholds ORDER BY id",
		nil,
		func(rows *sql.Rows) (*models.Threshold, error) {
			threshold := &models.Threshold{}
			if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
				&threshold.Shielded, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
				return nil, err
			}
			return threshold, nil
		},
	)
}

// UpdateThreshold 更新阈值
func UpdateThreshold(threshold *models.Threshold) error {
	if err := ensureThresholdColumns(); err != nil {
		return err
	}

	_, err := ParamDB.Exec(
		"UPDATE thresholds SET device_id = ?, field_name = ?, operator = ?, value = ?, severity = ?, shielded = ?, message = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		threshold.DeviceID, threshold.FieldName, threshold.Operator, threshold.Value, threshold.Severity, threshold.Shielded, threshold.Message, threshold.ID,
	)
	return err
}

// DeleteThreshold 删除阈值
func DeleteThreshold(id int64) error {
	if err := ensureThresholdColumns(); err != nil {
		return err
	}

	_, err := ParamDB.Exec("DELETE FROM thresholds WHERE id = ?", id)
	return err
}
