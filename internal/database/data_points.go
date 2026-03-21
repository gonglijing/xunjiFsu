package database

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
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

var (
	dataPointsCleanupCounter uint64
	dataPointsLastCleanupNS  int64
	dataDiskDBMu             sync.Mutex
	dataDiskDB               *sql.DB
	dataDiskDBPath           string

	// 允许在测试中覆盖
	dataPointsCleanupEveryWrites uint64        = 128
	dataPointsCleanupMinInterval time.Duration = 2 * time.Second

	collectDataCacheBatchSQLCache         = newCollectDataBatchSQLCache(buildCollectDataCacheBatchSQL)
	collectDataCacheStringBatchSQLCache   = newCollectDataBatchSQLCache(buildCollectDataCacheStringBatchSQL)
	collectDataHistoryBatchSQLCache       = newCollectDataBatchSQLCache(buildCollectDataHistoryBatchSQL)
	collectDataHistoryStringBatchSQLCache = newCollectDataBatchSQLCache(buildCollectDataHistoryStringBatchSQL)
	dataPointBatchSQLCache                = newCollectDataBatchSQLCache(buildDataPointBatchSQL)
	dataPointStringBatchSQLCache          = newCollectDataBatchSQLCache(buildDataPointStringBatchSQL)
	latestDataPointBatchSQLCache          = newCollectDataBatchSQLCache(buildLatestDataPointBatchSQL)
	latestDataPointStringBatchSQLCache    = newCollectDataBatchSQLCache(buildLatestDataPointStringBatchSQL)
	dataPointQueryStmtCache               dbStmtCache
	latestDeviceFieldQueryStmtCache       dbStmtCache
	collectDataFieldNameSetPool           = sync.Pool{
		New: func() any {
			return make(map[string]struct{}, collectDataCacheBatchSize)
		},
	}
	collectDataArgsPool = sync.Pool{
		New: func() any {
			return make([]any, 0, collectDataCacheBatchSize*5)
		},
	}
)

const collectDataValueTypeString = "string"
const selectDataPointFields = `SELECT id, device_id, device_name, field_name, value, value_type, collected_at FROM data_points`
const selectDataPointFieldsLatestLimit = selectDataPointFields + " ORDER BY collected_at DESC LIMIT ?"
const selectDataPointFieldsByDeviceLimit = selectDataPointFields + " WHERE device_id = ? ORDER BY collected_at DESC LIMIT ?"
const dataPointSingleSQL = `INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
const latestDataPointSingleSQL = `INSERT OR REPLACE INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
	VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`
const collectDataCacheSingleSQL = `INSERT INTO data_cache (device_id, field_name, value, value_type, collected_at)
	VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(device_id, field_name) DO UPDATE SET
		value = excluded.value,
		value_type = excluded.value_type,
		collected_at = CURRENT_TIMESTAMP`
const collectDataCacheSingleStringSQL = `INSERT INTO data_cache (device_id, field_name, value, value_type, collected_at)
	VALUES (?, ?, ?, 'string', CURRENT_TIMESTAMP)
	ON CONFLICT(device_id, field_name) DO UPDATE SET
		value = excluded.value,
		value_type = 'string',
		collected_at = CURRENT_TIMESTAMP`

func normalizeDeviceName(deviceID int64, deviceName string) string {
	if deviceID == models.SystemStatsDeviceID {
		return models.SystemStatsDeviceName
	}
	return strings.TrimSpace(deviceName)
}

// SaveDataPoint 保存历史数据点（内存暂存）
func SaveDataPoint(deviceID int64, deviceName, fieldName, value, valueType string) error {
	deviceName = normalizeDeviceName(deviceID, deviceName)
	valueType = normalizedCollectDataValueType(valueType)
	stmt, err := dataCacheExecStmtCache.get(DataDB, dataPointSingleSQL)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(deviceID, deviceName, fieldName, value, valueType)
	if err != nil {
		return err
	}

	// 节流检查并清理过量的数据点，避免每次写入都 count(*)
	noteHistoryRowsWritten(1)
	maybeEnforceDataPointsLimit()
	TriggerSyncIfNeeded()
	return nil
}

type dataPointScanner interface {
	Scan(dest ...any) error
}

func scanDataPoint(scanner dataPointScanner, point *DataPoint) error {
	return scanner.Scan(
		&point.ID,
		&point.DeviceID,
		&point.DeviceName,
		&point.FieldName,
		&point.Value,
		&point.ValueType,
		&point.CollectedAt,
	)
}

func listDataPointsLimit(db *sql.DB, query string, limit int, args ...any) ([]*DataPoint, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if limit < 0 {
		limit = 0
	}
	points := make([]DataPoint, 0, limit)
	list := make([]*DataPoint, 0, limit)
	for rows.Next() {
		points = append(points, DataPoint{})
		point := &points[len(points)-1]
		if err := scanDataPoint(rows, point); err != nil {
			return nil, err
		}
		list = append(list, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func queryDataPointsByDevice(db *sql.DB, deviceID int64, limit int, before time.Time) ([]*DataPoint, error) {
	if before.IsZero() {
		stmt, err := dataPointQueryStmtCache.get(db, selectDataPointFieldsByDeviceLimit)
		if err != nil {
			return nil, err
		}
		return listDataPointsStmtLimit(stmt, limit, deviceID, limit)
	}
	query := selectDataPointFields + " WHERE device_id = ?"
	args := []any{deviceID}
	query += " AND collected_at < ?"
	args = append(args, before)
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)
	return listDataPointsLimit(db, query, limit, args...)
}

func queryLatestDataPoints(db *sql.DB, limit int, before time.Time) ([]*DataPoint, error) {
	if before.IsZero() {
		stmt, err := dataPointQueryStmtCache.get(db, selectDataPointFieldsLatestLimit)
		if err != nil {
			return nil, err
		}
		return listDataPointsStmtLimit(stmt, limit, limit)
	}
	query := selectDataPointFields
	args := make([]any, 0, 2)
	query += " WHERE collected_at < ?"
	args = append(args, before)
	query += " ORDER BY collected_at DESC LIMIT ?"
	args = append(args, limit)
	return listDataPointsLimit(db, query, limit, args...)
}

func queryDataPointsByDeviceAndTime(db *sql.DB, deviceID int64, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	query := selectDataPointFields + " WHERE device_id = ?"
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
	return listDataPointsLimit(db, query, limit, args...)
}

func queryDataPointsByDeviceFieldAndTime(db *sql.DB, deviceID int64, fieldName string, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	query := selectDataPointFields + " WHERE device_id = ? AND field_name = ?"
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
	return listDataPointsLimit(db, query, limit, args...)
}

func listDataPointsStmtLimit(stmt *sql.Stmt, limit int, args ...any) ([]*DataPoint, error) {
	if stmt == nil {
		return nil, fmt.Errorf("stmt is nil")
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if limit < 0 {
		limit = 0
	}
	points := make([]DataPoint, 0, limit)
	list := make([]*DataPoint, 0, limit)
	for rows.Next() {
		points = append(points, DataPoint{})
		point := &points[len(points)-1]
		if err := scanDataPoint(rows, point); err != nil {
			return nil, err
		}
		list = append(list, point)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

