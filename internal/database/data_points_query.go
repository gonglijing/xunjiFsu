package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

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

	dataDiskDBMu.Lock()
	defer dataDiskDBMu.Unlock()

	if dataDiskDB != nil && dataDiskDBPath == dataDBFile {
		return dataDiskDB, nil
	}
	if dataDiskDB != nil {
		_ = dataDiskDB.Close()
		dataDiskDB = nil
		dataDiskDBPath = ""
	}
	if _, err := os.Stat(dataDBFile); err != nil {
		return nil, err
	}

	db, err := openSQLite(dataDiskDSN(dataDBFile), 1, 1)
	if err != nil {
		return nil, err
	}
	dataDiskDB = db
	dataDiskDBPath = dataDBFile
	return dataDiskDB, nil
}

func closeCachedDataDiskDBForPath(path string) {
	dataDiskDBMu.Lock()
	defer dataDiskDBMu.Unlock()
	if dataDiskDB == nil {
		return
	}
	if path != "" && dataDiskDBPath != path {
		return
	}
	_ = dataDiskDB.Close()
	dataDiskDB = nil
	dataDiskDBPath = ""
}

func getDiskDataPointsByDevice(deviceID int64, limit int, before time.Time) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}

	return queryDataPointsByDevice(db, deviceID, limit, before)
}

func getDiskLatestDataPoints(limit int, before time.Time) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}

	return queryLatestDataPoints(db, limit, before)
}

func getDiskDataPointsByDeviceFieldAndTime(deviceID int64, fieldName string, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}

	return queryDataPointsByDeviceFieldAndTime(db, deviceID, fieldName, startTime, endTime, limit)
}

func getDiskDataPointsByDeviceAndTime(deviceID int64, startTime, endTime time.Time, limit int) ([]*DataPoint, error) {
	db, err := openDataDiskDB()
	if err != nil {
		return nil, err
	}

	return queryDataPointsByDeviceAndTime(db, deviceID, startTime, endTime, limit)
}

// enforceDataPointsLimit 强制执行数据点大小限制
func enforceDataPointsLimit() {
	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		slog.Error("Failed to count data points", "error", err)
		return
	}
	if count > maxDataPointsLimit {
		if _, err := DataDB.Exec("DELETE FROM data_points WHERE id IN (SELECT id FROM data_points ORDER BY collected_at ASC LIMIT ?)", count-maxDataPointsLimit); err != nil {
			slog.Error("Failed to cleanup data points", "error", err)
			return
		}
		slog.Info("Cleaned up data points", "removed", count-maxDataPointsLimit)
	}
}

func maybeEnforceDataPointsLimit() {
	if maxDataPointsLimit <= 0 {
		return
	}

	writes := atomic.AddUint64(&dataPointsCleanupCounter, 1)
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&dataPointsLastCleanupNS)
	minIntervalNS := int64(dataPointsCleanupMinInterval)
	if minIntervalNS < 0 {
		minIntervalNS = 0
	}

	if dataPointsCleanupEveryWrites > 0 && writes%dataPointsCleanupEveryWrites != 0 {
		if last == 0 {
			return
		}
		now = time.Now().UnixNano()
		if now-last < minIntervalNS {
			return
		}
		if !atomic.CompareAndSwapInt64(&dataPointsLastCleanupNS, last, now) {
			return
		}
		enforceDataPointsLimit()
		return
	}

	now = time.Now().UnixNano()
	if !atomic.CompareAndSwapInt64(&dataPointsLastCleanupNS, last, now) {
		return
	}

	enforceDataPointsLimit()
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
	memPoints, err := queryDataPointsByDeviceAndTime(DataDB, deviceID, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(memPoints) >= limit {
		return memPoints[:limit], nil
	}
	if dataDBFile == "" {
		return memPoints, nil
	}

	diskPoints, err := getDiskDataPointsByDeviceAndTime(deviceID, startTime, endTime, limit)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("Failed to read data points by time from disk", "error", err)
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
	memPoints, err := queryDataPointsByDeviceFieldAndTime(DataDB, deviceID, fieldName, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(memPoints) >= limit {
		return memPoints[:limit], nil
	}
	if dataDBFile == "" {
		return memPoints, nil
	}

	diskPoints, err := getDiskDataPointsByDeviceFieldAndTime(deviceID, fieldName, startTime, endTime, limit)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("Failed to read data points by field from disk", "error", err)
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
	memPoints, err := queryDataPointsByDevice(DataDB, deviceID, limit, time.Time{})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(memPoints) >= limit {
		return memPoints[:limit], nil
	}
	if dataDBFile == "" {
		return memPoints, nil
	}

	diskPoints, err := getDiskDataPointsByDevice(deviceID, limit, oldestCollectedAt(memPoints))
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("Failed to read data points from disk", "error", err)
		}
		return memPoints, nil
	}
	return mergeDataPoints(memPoints, diskPoints, limit), nil
}

// GetLatestDataPoints 获取最新的历史数据点（内存 + 磁盘）
func GetLatestDataPoints(limit int) ([]*DataPoint, error) {
	memPoints, err := queryLatestDataPoints(DataDB, limit, time.Time{})
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(memPoints) >= limit {
		return memPoints[:limit], nil
	}
	if dataDBFile == "" {
		return memPoints, nil
	}

	diskPoints, err := getDiskLatestDataPoints(limit, oldestCollectedAt(memPoints))
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("Failed to read latest data points from disk", "error", err)
		}
		return memPoints, nil
	}
	return mergeDataPoints(memPoints, diskPoints, limit), nil
}

// DeleteHistoryDataByPoint 删除指定测点的全部历史数据（内存 + 磁盘）
func DeleteHistoryDataByPoint(deviceID int64, fieldName string) (int64, error) {
	fieldName = strings.TrimSpace(fieldName)
	if deviceID == 0 {
		return 0, fmt.Errorf("invalid device_id")
	}
	if fieldName == "" {
		return 0, fmt.Errorf("field_name is required")
	}

	result, err := DataDB.Exec(
		`DELETE FROM data_points WHERE device_id = ? AND field_name = ?`,
		deviceID, fieldName,
	)
	if err != nil {
		return 0, err
	}

	memDeleted, _ := result.RowsAffected()

	diskDeleted, err := deleteHistoryDataByPointOnDisk(deviceID, fieldName)
	if err != nil {
		return memDeleted, err
	}

	return memDeleted + diskDeleted, nil
}

func deleteHistoryDataByPointOnDisk(deviceID int64, fieldName string) (int64, error) {
	if dataDBFile == "" {
		return 0, nil
	}
	if _, err := os.Stat(dataDBFile); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	diskDB, err := openSQLite(dataDBFile, 1, 1)
	if err != nil {
		return 0, err
	}
	defer diskDB.Close()

	if err := ensureDiskDataSchema(diskDB); err != nil {
		return 0, err
	}

	result, err := diskDB.Exec(
		`DELETE FROM data_points WHERE device_id = ? AND field_name = ?`,
		deviceID, fieldName,
	)
	if err != nil {
		return 0, err
	}

	deleted, _ := result.RowsAffected()
	return deleted, nil
}

// LatestDeviceData 单个设备的最新数据
type LatestDeviceData struct {
	DeviceID    int64
	DeviceName  string
	Fields      map[string]string
	CollectedAt time.Time
}

type latestDeviceFieldKey struct {
	deviceID  int64
	fieldName string
}

type latestDeviceFieldRow struct {
	DeviceID    int64
	DeviceName  string
	FieldName   string
	Value       string
	CollectedAt time.Time
}

func queryRowsWithCachedStmt(db *sql.DB, cache *dbStmtCache, query string, args ...any) (*sql.Rows, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if query == "" {
		return nil, fmt.Errorf("query is empty")
	}
	if cache == nil {
		return db.Query(query, args...)
	}
	stmt, err := cache.get(db, query)
	if err != nil {
		return nil, err
	}
	if stmt == nil {
		return nil, fmt.Errorf("stmt is nil")
	}
	return stmt.Query(args...)
}

const latestDeviceFieldRowsFromCacheQuery = `SELECT device_id, field_name, value, collected_at
	FROM data_cache
	ORDER BY device_id ASC, collected_at DESC`

const latestDeviceFieldRowsFromHistoryQuery = `SELECT p.device_id, p.device_name, p.field_name, p.value, p.collected_at
	FROM data_points p
	INNER JOIN (
		SELECT device_id, field_name, MAX(collected_at) AS max_collected_at
		FROM data_points
		GROUP BY device_id, field_name
	) latest
	ON p.device_id = latest.device_id
		AND p.field_name = latest.field_name
		AND p.collected_at = latest.max_collected_at
	ORDER BY p.device_id ASC, p.collected_at DESC`

const latestDeviceFieldsInitialCap = 24

func appendLatestDeviceFieldRow(item *LatestDeviceData, fieldTimes map[latestDeviceFieldKey]int64, row latestDeviceFieldRow) {
	if item == nil || row.DeviceID == 0 || row.FieldName == "" {
		return
	}
	if fieldTimes != nil {
		key := latestDeviceFieldKey{deviceID: row.DeviceID, fieldName: row.FieldName}
		collectedAtNS := row.CollectedAt.UnixNano()
		if existingAtNS, ok := fieldTimes[key]; ok && existingAtNS > collectedAtNS {
			return
		}
		fieldTimes[key] = collectedAtNS
	}
	if item.DeviceName == "" {
		item.DeviceName = row.DeviceName
	}
	item.Fields[row.FieldName] = row.Value
	if row.CollectedAt.After(item.CollectedAt) {
		item.CollectedAt = row.CollectedAt
	}
}

func appendLatestDeviceFieldRowsFromHistory(db *sql.DB, result []*LatestDeviceData, deviceIndex map[int64]int, fieldTimes map[latestDeviceFieldKey]int64) ([]*LatestDeviceData, error) {
	rows, err := queryRowsWithCachedStmt(db, &latestDeviceFieldQueryStmtCache, latestDeviceFieldRowsFromHistoryQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if fieldTimes == nil && len(result) == 0 {
		var currentDeviceID int64
		var currentItem *LatestDeviceData
		for rows.Next() {
			var item latestDeviceFieldRow
			if err := rows.Scan(&item.DeviceID, &item.DeviceName, &item.FieldName, &item.Value, &item.CollectedAt); err != nil {
				return nil, err
			}
			item.FieldName = trimDataPointFieldName(item.FieldName)
			if item.DeviceID == 0 || item.FieldName == "" {
				continue
			}
			if currentItem == nil || currentDeviceID != item.DeviceID {
				currentDeviceID = item.DeviceID
				currentItem = &LatestDeviceData{
					DeviceID:   item.DeviceID,
					DeviceName: item.DeviceName,
					Fields:     make(map[string]string, latestDeviceFieldsInitialCap),
				}
				result = append(result, currentItem)
			}
			appendLatestDeviceFieldRow(currentItem, nil, item)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}

	if deviceIndex == nil {
		deviceIndex = make(map[int64]int, len(result)+64)
		for i, item := range result {
			if item == nil {
				continue
			}
			deviceIndex[item.DeviceID] = i
		}
	}

	var currentDeviceID int64
	var currentItem *LatestDeviceData
	for rows.Next() {
		var item latestDeviceFieldRow
		if err := rows.Scan(&item.DeviceID, &item.DeviceName, &item.FieldName, &item.Value, &item.CollectedAt); err != nil {
			return nil, err
		}
		item.FieldName = trimDataPointFieldName(item.FieldName)
		if item.DeviceID == 0 || item.FieldName == "" {
			continue
		}
		if currentItem == nil || currentDeviceID != item.DeviceID {
			idx, exists := deviceIndex[item.DeviceID]
			if !exists {
				idx = len(result)
				deviceIndex[item.DeviceID] = idx
				result = append(result, &LatestDeviceData{
					DeviceID:   item.DeviceID,
					DeviceName: item.DeviceName,
					Fields:     make(map[string]string, latestDeviceFieldsInitialCap),
				})
			}
			currentDeviceID = item.DeviceID
			currentItem = result[idx]
		}
		appendLatestDeviceFieldRow(currentItem, fieldTimes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func appendLatestDeviceFieldRowsFromCache(db *sql.DB, result []*LatestDeviceData, deviceIndex map[int64]int, fieldTimes map[latestDeviceFieldKey]int64) ([]*LatestDeviceData, error) {
	rows, err := queryRowsWithCachedStmt(db, &latestDeviceFieldQueryStmtCache, latestDeviceFieldRowsFromCacheQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if fieldTimes == nil && len(result) == 0 {
		var currentDeviceID int64
		var currentItem *LatestDeviceData
		for rows.Next() {
			var deviceID int64
			var fieldName, value string
			var collectedAt time.Time
			if err := rows.Scan(&deviceID, &fieldName, &value, &collectedAt); err != nil {
				return nil, err
			}
			fieldName = trimDataPointFieldName(fieldName)
			if deviceID == 0 || fieldName == "" {
				continue
			}
			if currentItem == nil || currentDeviceID != deviceID {
				currentDeviceID = deviceID
				currentItem = &LatestDeviceData{
					DeviceID: deviceID,
					Fields:   make(map[string]string, latestDeviceFieldsInitialCap),
				}
				result = append(result, currentItem)
			}
			currentItem.Fields[fieldName] = value
			if collectedAt.After(currentItem.CollectedAt) {
				currentItem.CollectedAt = collectedAt
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}

	if deviceIndex == nil {
		deviceIndex = make(map[int64]int, len(result)+64)
		for i, item := range result {
			if item == nil {
				continue
			}
			deviceIndex[item.DeviceID] = i
		}
	}

	var currentDeviceID int64
	var currentItem *LatestDeviceData
	for rows.Next() {
		var item latestDeviceFieldRow
		if err := rows.Scan(&item.DeviceID, &item.FieldName, &item.Value, &item.CollectedAt); err != nil {
			return nil, err
		}
		item.FieldName = trimDataPointFieldName(item.FieldName)
		if item.DeviceID == 0 || item.FieldName == "" {
			continue
		}
		if currentItem == nil || currentDeviceID != item.DeviceID {
			idx, exists := deviceIndex[item.DeviceID]
			if !exists {
				idx = len(result)
				deviceIndex[item.DeviceID] = idx
				result = append(result, &LatestDeviceData{
					DeviceID: item.DeviceID,
					Fields:   make(map[string]string, latestDeviceFieldsInitialCap),
				})
			}
			currentDeviceID = item.DeviceID
			currentItem = result[idx]
		}
		appendLatestDeviceFieldRow(currentItem, fieldTimes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func fillLatestDeviceNamesFromParamDB(items []*LatestDeviceData) {
	if len(items) == 0 || ParamDB == nil {
		return
	}

	missingIDs := make([]int64, 0, len(items))
	itemByID := make(map[int64]*LatestDeviceData, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if item.DeviceID == models.SystemStatsDeviceID {
			item.DeviceName = models.SystemStatsDeviceName
			continue
		}
		if strings.TrimSpace(item.DeviceName) != "" || item.DeviceID == 0 {
			continue
		}
		if _, exists := itemByID[item.DeviceID]; exists {
			continue
		}
		itemByID[item.DeviceID] = item
		missingIDs = append(missingIDs, item.DeviceID)
	}
	if len(missingIDs) == 0 {
		return
	}

	query := strings.Builder{}
	query.Grow(len(missingIDs)*3 + 48)
	query.WriteString("SELECT id, name FROM devices WHERE id IN (")
	args := make([]any, 0, len(missingIDs))
	for i, id := range missingIDs {
		if i > 0 {
			query.WriteByte(',')
		}
		query.WriteByte('?')
		args = append(args, id)
	}
	query.WriteByte(')')

	rows, err := ParamDB.Query(query.String(), args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var deviceID int64
		var deviceName string
		if err := rows.Scan(&deviceID, &deviceName); err != nil {
			return
		}
		if item := itemByID[deviceID]; item != nil {
			item.DeviceName = strings.TrimSpace(deviceName)
		}
	}
}

// GetAllDevicesLatestData 获取所有设备的最新数据（每个设备每个字段仅保留最新值）
func GetAllDevicesLatestData() ([]*LatestDeviceData, error) {
	var fieldTimes map[latestDeviceFieldKey]int64
	if dataDBFile != "" {
		fieldTimes = make(map[latestDeviceFieldKey]int64, 256)
	}
	result, err := appendLatestDeviceFieldRowsFromCache(DataDB, make([]*LatestDeviceData, 0, 64), nil, fieldTimes)
	if err != nil {
		result, err = appendLatestDeviceFieldRowsFromHistory(DataDB, make([]*LatestDeviceData, 0, 64), nil, fieldTimes)
		if err != nil {
			return nil, err
		}
	} else if len(result) == 0 {
		result, err = appendLatestDeviceFieldRowsFromHistory(DataDB, result, nil, fieldTimes)
		if err != nil {
			return nil, err
		}
	}
	if dataDBFile == "" {
		fillLatestDeviceNamesFromParamDB(result)
		return result, nil
	}

	diskDB, err := openDataDiskDB()
	if err != nil {
		fillLatestDeviceNamesFromParamDB(result)
		if !os.IsNotExist(err) {
			slog.Warn("Failed to read latest device fields from disk", "error", err)
		}
		return result, nil
	}

	deviceIndex := make(map[int64]int, len(result)+64)
	for i, item := range result {
		if item == nil {
			continue
		}
		deviceIndex[item.DeviceID] = i
	}

	result, err = appendLatestDeviceFieldRowsFromHistory(diskDB, result, deviceIndex, fieldTimes)
	if err != nil {
		slog.Warn("Failed to query latest device fields from disk", "error", err)
		fillLatestDeviceNamesFromParamDB(result)
		return result, nil
	}

	fillLatestDeviceNamesFromParamDB(result)
	return result, nil
}

// InsertCollectData 将采集数据写入缓存与历史库
func InsertCollectData(data *models.CollectData) error {
	return InsertCollectDataWithOptions(data, true)
}

func insertCollectDataWithOptionsTx(tx *sql.Tx, data *models.CollectData, storeHistory bool, cacheStmtCache, historyStmtCache *collectDataStmtCache) (int, error) {
	if data == nil {
		return 0, fmt.Errorf("collect data is nil")
	}
	if len(data.Fields) == 0 && len(data.Points) == 0 {
		return 0, nil
	}

	deviceName := ""
	if storeHistory {
		deviceName = normalizeDeviceName(data.DeviceID, data.DeviceName)
	}

	batchSize := collectDataCacheBatchSize
	if storeHistory && collectDataHistoryBatchSize < batchSize {
		batchSize = collectDataHistoryBatchSize
	}

	cacheArgs := getCollectDataArgs(batchSize * 3)
	defer putCollectDataArgs(cacheArgs)
	var historyArgs []any
	if storeHistory {
		historyArgs = getCollectDataArgs(batchSize * 4)
		defer putCollectDataArgs(historyArgs)
	}
	batchCount := 0
	historyCount := 0
	if len(data.Points) == 0 {
		for field, value := range data.Fields {
			cacheArgs = appendCollectDataCacheStringArg(cacheArgs, data.DeviceID, field, value)
			if storeHistory {
				historyArgs = appendCollectDataHistoryStringArg(historyArgs, data.DeviceID, deviceName, field, value)
				historyCount++
			}
			batchCount++
			if batchCount < batchSize {
				continue
			}
			if err := flushCollectDataArgs(cacheStmtCache, historyStmtCache, batchCount, cacheArgs, historyArgs, storeHistory); err != nil {
				return 0, err
			}
			cacheArgs = cacheArgs[:0]
			if storeHistory {
				historyArgs = historyArgs[:0]
			}
			batchCount = 0
		}
		if batchCount > 0 {
			if err := flushCollectDataArgs(cacheStmtCache, historyStmtCache, batchCount, cacheArgs, historyArgs, storeHistory); err != nil {
				return 0, err
			}
		}
		return historyCount, nil
	}
	normalizedPointFields, _ := normalizeCollectPointFieldNames(data.Points)
	pointFieldNames := collectOverriddenPointFieldNames(data.Fields, data.Points, normalizedPointFields)
	defer putCollectDataFieldNameSet(pointFieldNames)
	allFieldsOverridden := len(data.Fields) > 0 && len(pointFieldNames) == len(data.Fields)

	if !allFieldsOverridden {
		for field, value := range data.Fields {
			if _, overridden := pointFieldNames[field]; overridden {
				continue
			}
			cacheArgs = appendCollectDataCacheStringArg(cacheArgs, data.DeviceID, field, value)
			if storeHistory {
				historyArgs = appendCollectDataHistoryStringArg(historyArgs, data.DeviceID, deviceName, field, value)
				historyCount++
			}
			batchCount++
			if batchCount < batchSize {
				continue
			}
			if err := flushCollectDataArgs(cacheStmtCache, historyStmtCache, batchCount, cacheArgs, historyArgs, storeHistory); err != nil {
				return 0, err
			}
			cacheArgs = cacheArgs[:0]
			if storeHistory {
				historyArgs = historyArgs[:0]
			}
			batchCount = 0
		}
	}
	for i, point := range data.Points {
		field := normalizedCollectPointFieldName(data.Points, normalizedPointFields, i)
		if field == "" {
			continue
		}
		value := models.CollectPointValueString(point.Value)
		cacheArgs = appendCollectDataCacheStringArg(cacheArgs, data.DeviceID, field, value)
		if storeHistory {
			historyArgs = appendCollectDataHistoryStringArg(historyArgs, data.DeviceID, deviceName, field, value)
			historyCount++
		}
		batchCount++
		if batchCount < batchSize {
			continue
		}
		if err := flushCollectDataArgs(cacheStmtCache, historyStmtCache, batchCount, cacheArgs, historyArgs, storeHistory); err != nil {
			return 0, err
		}
		cacheArgs = cacheArgs[:0]
		if storeHistory {
			historyArgs = historyArgs[:0]
		}
		batchCount = 0
	}
	if batchCount > 0 {
		if err := flushCollectDataArgs(cacheStmtCache, historyStmtCache, batchCount, cacheArgs, historyArgs, storeHistory); err != nil {
			return 0, err
		}
	}

	return historyCount, nil
}

func trimDataPointFieldName(s string) string {
	if s == "" {
		return ""
	}
	start := 0
	end := len(s)
	for start < end && isASCIIDataPointSpace(s[start]) {
		start++
	}
	if start == end {
		return ""
	}
	for end > start && isASCIIDataPointSpace(s[end-1]) {
		end--
	}
	if start == 0 && end == len(s) {
		if s[0] < utf8.RuneSelf && s[len(s)-1] < utf8.RuneSelf {
			return s
		}
		return strings.TrimSpace(s)
	}
	if s[start] < utf8.RuneSelf && s[end-1] < utf8.RuneSelf {
		return s[start:end]
	}
	return strings.TrimSpace(s)
}

func isASCIIDataPointSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

// SaveLatestDataPoint 保存最新数据点（使用 upsert，只保留最新值）
func SaveLatestDataPoint(deviceID int64, deviceName, fieldName, value string) error {
	deviceName = normalizeDeviceName(deviceID, deviceName)
	stmt, err := dataCacheExecStmtCache.get(DataDB, latestDataPointSingleSQL)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(deviceID, deviceName, fieldName, value, collectDataValueTypeString)
	return err
}

// BatchSaveLatestDataPoints 批量保存最新数据点（使用 upsert）
func BatchSaveLatestDataPoints(entries []DataPointEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if allEntriesUseDefaultCollectDataValueType(entries) {
		if len(entries) <= collectDataHistoryBatchSize {
			return batchSaveLatestDataPointsStringDirect(entries)
		}
		return batchSaveLatestDataPointsStringChunkedDirect(entries)
	}
	if len(entries) <= collectDataHistoryBatchSize {
		return batchSaveLatestDataPointsDirect(entries)
	}
	return batchSaveLatestDataPointsChunkedDirect(entries)
}

func batchSaveLatestDataPointsDirect(entries []DataPointEntry) error {
	args := getCollectDataArgs(len(entries) * 5)
	defer putCollectDataArgs(args)

	for _, entry := range entries {
		args = appendLatestDataPointArg(args, entry)
	}

	stmt, err := dataCacheExecStmtCache.get(DataDB, latestDataPointBatchSQLCache.get(len(entries)))
	if err != nil {
		return fmt.Errorf("failed to prepare latest data point batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data point batch: %w", err)
	}
	return nil
}

func batchSaveLatestDataPointsStringDirect(entries []DataPointEntry) error {
	args := getCollectDataArgs(len(entries) * 4)
	defer putCollectDataArgs(args)

	for _, entry := range entries {
		args = appendLatestDataPointStringArg(args, entry)
	}

	stmt, err := dataCacheExecStmtCache.get(DataDB, latestDataPointStringBatchSQLCache.get(len(entries)))
	if err != nil {
		return fmt.Errorf("failed to prepare latest data point batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data point batch: %w", err)
	}
	return nil
}

func batchSaveLatestDataPointsChunkedDirect(entries []DataPointEntry) error {
	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin latest data point transaction: %w", err)
	}
	defer tx.Rollback()

	stmtCache := newCollectDataStmtCache(tx, latestDataPointBatchSQLCache.get)
	defer stmtCache.close()

	args := getCollectDataArgs(collectDataHistoryBatchSize * 5)
	defer putCollectDataArgs(args)

	batchCount := 0
	for _, entry := range entries {
		args = appendLatestDataPointArg(args, entry)
		batchCount++
		if batchCount < collectDataHistoryBatchSize {
			continue
		}
		stmt, err := stmtCache.get(batchCount)
		if err != nil {
			return fmt.Errorf("failed to prepare latest data point batch statement: %w", err)
		}
		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("failed to upsert data point batch: %w", err)
		}
		args = args[:0]
		batchCount = 0
	}
	if batchCount == 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit latest data point transaction: %w", err)
		}
		return nil
	}
	stmt, err := stmtCache.get(batchCount)
	if err != nil {
		return fmt.Errorf("failed to prepare latest data point batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data point batch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit latest data point transaction: %w", err)
	}
	return nil
}

func batchSaveLatestDataPointsStringChunkedDirect(entries []DataPointEntry) error {
	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin latest data point transaction: %w", err)
	}
	defer tx.Rollback()

	stmtCache := newCollectDataStmtCache(tx, latestDataPointStringBatchSQLCache.get)
	defer stmtCache.close()

	args := getCollectDataArgs(collectDataHistoryBatchSize * 4)
	defer putCollectDataArgs(args)

	batchCount := 0
	for _, entry := range entries {
		args = appendLatestDataPointStringArg(args, entry)
		batchCount++
		if batchCount < collectDataHistoryBatchSize {
			continue
		}
		stmt, err := stmtCache.get(batchCount)
		if err != nil {
			return fmt.Errorf("failed to prepare latest data point batch statement: %w", err)
		}
		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("failed to upsert data point batch: %w", err)
		}
		args = args[:0]
		batchCount = 0
	}
	if batchCount == 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit latest data point transaction: %w", err)
		}
		return nil
	}
	stmt, err := stmtCache.get(batchCount)
	if err != nil {
		return fmt.Errorf("failed to prepare latest data point batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data point batch: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit latest data point transaction: %w", err)
	}
	return nil
}

// InsertCollectDataWithOptions 写入实时缓存，并按需写入历史数据。
func InsertCollectDataWithOptions(data *models.CollectData, storeHistory bool) error {
	if data == nil {
		return fmt.Errorf("collect data is nil")
	}
	if len(data.Fields) == 0 && len(data.Points) == 0 {
		return nil
	}
	if !storeHistory {
		rowCount := countCollectDataCacheRows(data)
		if rowCount <= collectDataCacheBatchSize {
			if err := insertCollectDataCacheDirect(data); err != nil {
				return err
			}
		} else {
			if err := insertCollectDataCacheChunkedDirect(data); err != nil {
				return err
			}
		}
		maybeEnforceDataCacheLimit()
		return nil
	}

	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin collect data transaction: %w", err)
	}
	defer tx.Rollback()
	cacheStmtCache := newCollectDataStmtCache(tx, collectDataCacheStringBatchSQLCache.get)
	defer cacheStmtCache.close()
	var historyStmtCache *collectDataStmtCache
	if storeHistory {
		historyStmtCache = newCollectDataStmtCache(tx, collectDataHistoryStringBatchSQLCache.get)
		defer historyStmtCache.close()
	}
	historyCount, err := insertCollectDataWithOptionsTx(tx, data, storeHistory, cacheStmtCache, historyStmtCache)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit collect data transaction: %w", err)
	}

	maybeEnforceDataCacheLimit()
	if storeHistory && historyCount > 0 {
		noteHistoryRowsWritten(historyCount)
		maybeEnforceDataPointsLimit()
		TriggerSyncIfNeeded()
	}

	return nil
}
