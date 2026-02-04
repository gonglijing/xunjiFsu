package database

import (
	"database/sql"
	"fmt"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"log"
	"time"
)

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
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		log.Printf("Failed to count data points: %v", err)
		return
	}
	if count > maxDataPointsLimit {
		if _, err := DataDB.Exec("DELETE FROM data_points WHERE id IN (SELECT id FROM data_points ORDER BY collected_at ASC LIMIT ?)", count-maxDataPointsLimit); err != nil {
			log.Printf("Failed to cleanup data points: %v", err)
			return
		}
		log.Printf("Cleaned up data points, removed %d old entries", count-maxDataPointsLimit)
	}
}

// GetDataPointsByDeviceAndTime 根据设备ID和时间范围获取历史数据（从内存）
func GetDataPointsByDeviceAndTime(deviceID int64, startTime, endTime time.Time) ([]*DataPoint, error) {
	return queryList[*DataPoint](DataDB,
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ? AND collected_at >= ? AND collected_at <= ? 
		ORDER BY collected_at DESC`,
		[]any{deviceID, startTime, endTime},
		func(rows *sql.Rows) (*DataPoint, error) {
			point := &DataPoint{}
			if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
				return nil, err
			}
			return point, nil
		},
	)
}

// GetDataPointsByDevice 根据设备ID获取历史数据（从内存）
func GetDataPointsByDevice(deviceID int64, limit int) ([]*DataPoint, error) {
	return queryList[*DataPoint](DataDB,
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ? ORDER BY collected_at DESC LIMIT ?`,
		[]any{deviceID, limit},
		func(rows *sql.Rows) (*DataPoint, error) {
			point := &DataPoint{}
			if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
				return nil, err
			}
			return point, nil
		},
	)
}

// GetLatestDataPoints 获取最新的历史数据点（从内存）
func GetLatestDataPoints(limit int) ([]*DataPoint, error) {
	return queryList[*DataPoint](DataDB,
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points ORDER BY collected_at DESC LIMIT ?`,
		[]any{limit},
		func(rows *sql.Rows) (*DataPoint, error) {
			point := &DataPoint{}
			if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
				return nil, err
			}
			return point, nil
		},
	)
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
