package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 历史数据点操作 (data.db - 内存 + 磁盘) ====================

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

func scanDataPoint(rows *sql.Rows) (*DataPoint, error) {
	point := &DataPoint{}
	if err := rows.Scan(&point.ID, &point.DeviceID, &point.DeviceName, &point.FieldName, &point.Value, &point.ValueType, &point.CollectedAt); err != nil {
		return nil, err
	}
	return point, nil
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

type dataPointKey struct {
	deviceID    int64
	fieldName   string
	collectedAt int64
}

func mergeDataPoints(primary, secondary []*DataPoint, limit int) []*DataPoint {
	if len(secondary) == 0 {
		if limit > 0 && len(primary) > limit {
			return primary[:limit]
		}
		return primary
	}

	combined := make([]*DataPoint, 0, len(primary)+len(secondary))
	combined = append(combined, primary...)
	combined = append(combined, secondary...)

	sort.Slice(combined, func(i, j int) bool {
		return combined[i].CollectedAt.After(combined[j].CollectedAt)
	})

	seen := make(map[dataPointKey]struct{}, len(combined))
	result := make([]*DataPoint, 0, len(combined))
	for _, point := range combined {
		if point == nil {
			continue
		}
		key := dataPointKey{
			deviceID:    point.DeviceID,
			fieldName:   point.FieldName,
			collectedAt: point.CollectedAt.UnixNano(),
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, point)
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result
}

func oldestCollectedAt(points []*DataPoint) time.Time {
	if len(points) == 0 {
		return time.Time{}
	}
	return points[len(points)-1].CollectedAt
}

func dataDiskDSN(path string) string {
	if strings.HasPrefix(path, "file:") {
		if strings.Contains(path, "mode=") {
			return path
		}
		sep := "?"
		if strings.Contains(path, "?") {
			sep = "&"
		}
		return path + sep + "mode=ro"
	}
	return "file:" + filepath.ToSlash(path) + "?mode=ro"
}

func openDataDiskDB() (*sql.DB, error) {
	if dataDBFile == "" {
		return nil, fmt.Errorf("data db path is empty")
	}
	if _, err := os.Stat(dataDBFile); err != nil {
		return nil, err
	}
	return openSQLite(dataDiskDSN(dataDBFile), 1, 1)
}

func getDiskDataPointsByDevice(deviceID int64, limit int, before time.Time) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, device_id, device_name, field_name, value, value_type, collected_at
		FROM data_points WHERE device_id = ?`
	args := []any{deviceID}
	if !before.IsZero() {
		query += " AND collected_at < ?"
		args = append(args, before)
	}
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)

	return queryList[*DataPoint](db, query, args, scanDataPoint)
}

func getDiskLatestDataPoints(limit int, before time.Time) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, device_id, device_name, field_name, value, value_type, collected_at FROM data_points`
	args := []any{}
	if !before.IsZero() {
		query += " WHERE collected_at < ?"
		args = append(args, before)
	}
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)

	return queryList[*DataPoint](db, query, args, scanDataPoint)
}

func getDiskDataPointsByDeviceFieldAndTime(deviceID int64, fieldName string, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, device_id, device_name, field_name, value, value_type, collected_at
		FROM data_points WHERE device_id = ? AND field_name = ?`
	args := []any{deviceID, fieldName}
	if !startTime.IsZero() {
		query += " AND collected_at >= ?"
		args = append(args, formatSQLiteTime(startTime))
	}
	if !endTime.IsZero() {
		query += " AND collected_at <= ?"
		args = append(args, formatSQLiteTime(endTime))
	}
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)

	return queryList[*DataPoint](db, query, args, scanDataPoint)
}

func getDiskDataPointsByDeviceAndTime(deviceID int64, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `SELECT id, device_id, device_name, field_name, value, value_type, collected_at
		FROM data_points WHERE device_id = ?`
	args := []any{deviceID}
	if !startTime.IsZero() {
		query += " AND collected_at >= ?"
		args = append(args, formatSQLiteTime(startTime))
	}
	if !endTime.IsZero() {
		query += " AND collected_at <= ?"
		args = append(args, formatSQLiteTime(endTime))
	}
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)

	return queryList[*DataPoint](db, query, args, scanDataPoint)
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

// GetDataPointsByDeviceAndTime 根据设备ID和时间范围获取历史数据（内存 + 磁盘）
func GetDataPointsByDeviceAndTime(deviceID int64, startTime, endTime time.Time) ([]*DataPoint, error) {
	return GetDataPointsByDeviceAndTimeLimit(deviceID, startTime, endTime, 2000)
}

// GetDataPointsByDeviceAndTimeLimit 根据设备ID和时间范围获取历史数据（内存 + 磁盘）
func GetDataPointsByDeviceAndTimeLimit(deviceID int64, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	if limit <= 0 {
		limit = 2000
	}
	query := `SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ?`
	args := []any{deviceID}
	if !startTime.IsZero() {
		query += " AND collected_at >= ?"
		args = append(args, formatSQLiteTime(startTime))
	}
	if !endTime.IsZero() {
		query += " AND collected_at <= ?"
		args = append(args, formatSQLiteTime(endTime))
	}
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)

	memPoints, err := queryList[*DataPoint](DataDB, query, args, scanDataPoint)
	if err != nil {
		return nil, err
	}

	diskPoints, err := getDiskDataPointsByDeviceAndTime(deviceID, startTime, endTime, limit)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read data points by time from disk: %v", err)
		}
		return memPoints, nil
	}
	return mergeDataPoints(memPoints, diskPoints, limit), nil
}

// GetDataPointsByDeviceFieldAndTime 根据设备ID/字段/时间范围获取历史数据（内存 + 磁盘）
func GetDataPointsByDeviceFieldAndTime(deviceID int64, fieldName string, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	if limit <= 0 {
		limit = 2000
	}
	query := `SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ? AND field_name = ?`
	args := []any{deviceID, fieldName}
	if !startTime.IsZero() {
		query += " AND collected_at >= ?"
		args = append(args, formatSQLiteTime(startTime))
	}
	if !endTime.IsZero() {
		query += " AND collected_at <= ?"
		args = append(args, formatSQLiteTime(endTime))
	}
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)

	memPoints, err := queryList[*DataPoint](DataDB, query, args, scanDataPoint)
	if err != nil {
		return nil, err
	}

	diskPoints, err := getDiskDataPointsByDeviceFieldAndTime(deviceID, fieldName, startTime, endTime, limit)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read data points by field from disk: %v", err)
		}
		return memPoints, nil
	}
	return mergeDataPoints(memPoints, diskPoints, limit), nil
}

func formatSQLiteTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// GetDataPointsByDevice 根据设备ID获取历史数据（内存 + 磁盘）
func GetDataPointsByDevice(deviceID int64, limit int) ([]*DataPoint, error) {
	memPoints, err := queryList[*DataPoint](DataDB,
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points WHERE device_id = ? ORDER BY collected_at DESC LIMIT ?`,
		[]any{deviceID, limit},
		scanDataPoint,
	)
	if err != nil {
		return nil, err
	}

	diskPoints, err := getDiskDataPointsByDevice(deviceID, limit, oldestCollectedAt(memPoints))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read data points from disk: %v", err)
		}
		return memPoints, nil
	}
	return mergeDataPoints(memPoints, diskPoints, limit), nil
}

// GetLatestDataPoints 获取最新的历史数据点（内存 + 磁盘）
func GetLatestDataPoints(limit int) ([]*DataPoint, error) {
	memPoints, err := queryList[*DataPoint](DataDB,
		`SELECT id, device_id, device_name, field_name, value, value_type, collected_at 
		FROM data_points ORDER BY collected_at DESC LIMIT ?`,
		[]any{limit},
		scanDataPoint,
	)
	if err != nil {
		return nil, err
	}

	diskPoints, err := getDiskLatestDataPoints(limit, oldestCollectedAt(memPoints))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read latest data points from disk: %v", err)
		}
		return memPoints, nil
	}
	return mergeDataPoints(memPoints, diskPoints, limit), nil
}

// InsertCollectData 将采集数据写入缓存与历史库
func InsertCollectData(data *models.CollectData) error {
	return InsertCollectDataWithOptions(data, true)
}

// InsertCollectDataWithOptions 写入缓存，并可选写入历史
func InsertCollectDataWithOptions(data *models.CollectData, storeHistory bool) error {
	if data == nil {
		return fmt.Errorf("collect data is nil")
	}

	entries := make([]DataPointEntry, 0, len(data.Fields))
	for field, value := range data.Fields {
		if err := SaveDataCache(data.DeviceID, data.DeviceName, field, value, "string"); err != nil {
			log.Printf("SaveDataCache error: %v", err)
		}
		if !storeHistory {
			continue
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
	if !storeHistory {
		return nil
	}
	return BatchSaveDataPoints(entries)
}
