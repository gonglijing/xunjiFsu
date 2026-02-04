package database

import (
	"database/sql"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"time"
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
		`SELECT id, device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, triggered_at, acknowledged, acknowledged_by, acknowledged_at 
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
		`SELECT id, device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, triggered_at, acknowledged, acknowledged_by, acknowledged_at 
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
