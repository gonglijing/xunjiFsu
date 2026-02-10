package database

import (
	"database/sql"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

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
	return queryList[*models.AlarmLog](ParamDB,
		`SELECT id, device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, triggered_at, acknowledged, COALESCE(acknowledged_by, ''), acknowledged_at
		FROM alarm_logs WHERE device_id = ? ORDER BY triggered_at DESC LIMIT ?`,
		[]any{deviceID, limit},
		func(rows *sql.Rows) (*models.AlarmLog, error) {
			log := &models.AlarmLog{}
			if err := rows.Scan(&log.ID, &log.DeviceID, &log.ThresholdID, &log.FieldName, &log.ActualValue, &log.ThresholdValue,
				&log.Operator, &log.Severity, &log.Message, &log.TriggeredAt, &log.Acknowledged, &log.AcknowledgedBy, &log.AcknowledgedAt); err != nil {
				return nil, err
			}
			return log, nil
		},
	)
}

// GetRecentAlarmLogs 获取最近的报警日志
func GetRecentAlarmLogs(limit int) ([]*models.AlarmLog, error) {
	return queryList[*models.AlarmLog](ParamDB,
		`SELECT id, device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, triggered_at, acknowledged, COALESCE(acknowledged_by, ''), acknowledged_at
		FROM alarm_logs ORDER BY triggered_at DESC LIMIT ?`,
		[]any{limit},
		func(rows *sql.Rows) (*models.AlarmLog, error) {
			log := &models.AlarmLog{}
			if err := rows.Scan(&log.ID, &log.DeviceID, &log.ThresholdID, &log.FieldName, &log.ActualValue, &log.ThresholdValue,
				&log.Operator, &log.Severity, &log.Message, &log.TriggeredAt, &log.Acknowledged, &log.AcknowledgedBy, &log.AcknowledgedAt); err != nil {
				return nil, err
			}
			return log, nil
		},
	)
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

// DeleteAlarmLog 删除单条报警日志
func DeleteAlarmLog(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM alarm_logs WHERE id = ?", id)
	return err
}

// DeleteAlarmLogsByIDs 批量删除报警日志
func DeleteAlarmLogsByIDs(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	result, err := ParamDB.Exec("DELETE FROM alarm_logs WHERE id IN ("+placeholders+")", args...)
	if err != nil {
		return 0, err
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return deleted, nil
}

// ClearAlarmLogs 清空报警日志
func ClearAlarmLogs() (int64, error) {
	result, err := ParamDB.Exec("DELETE FROM alarm_logs")
	if err != nil {
		return 0, err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return deleted, nil
}
