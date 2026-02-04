package database

import (
	"database/sql"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

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
	return queryList[*models.Threshold](ParamDB,
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds WHERE device_id = ?",
		[]any{deviceID},
		func(rows *sql.Rows) (*models.Threshold, error) {
			threshold := &models.Threshold{}
			if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
				&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
				return nil, err
			}
			return threshold, nil
		},
	)
}

// GetEnabledThresholdsByDeviceID 根据设备ID获取启用的阈值
func GetEnabledThresholdsByDeviceID(deviceID int64) ([]*models.Threshold, error) {
	return queryList[*models.Threshold](ParamDB,
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds WHERE device_id = ? AND enabled = 1",
		[]any{deviceID},
		func(rows *sql.Rows) (*models.Threshold, error) {
			threshold := &models.Threshold{}
			if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
				&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
				return nil, err
			}
			return threshold, nil
		},
	)
}

// GetAllThresholds 获取所有阈值
func GetAllThresholds() ([]*models.Threshold, error) {
	return queryList[*models.Threshold](ParamDB,
		"SELECT id, device_id, field_name, operator, value, severity, enabled, message, created_at, updated_at FROM thresholds ORDER BY id",
		nil,
		func(rows *sql.Rows) (*models.Threshold, error) {
			threshold := &models.Threshold{}
			if err := rows.Scan(&threshold.ID, &threshold.DeviceID, &threshold.FieldName, &threshold.Operator, &threshold.Value, &threshold.Severity,
				&threshold.Enabled, &threshold.Message, &threshold.CreatedAt, &threshold.UpdatedAt); err != nil {
				return nil, err
			}
			return threshold, nil
		},
	)
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
