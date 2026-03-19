package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

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
	latestDataPointBatchSQLCache          = newCollectDataBatchSQLCache(buildLatestDataPointBatchSQL)
	dataPointQueryStmtCache               dbStmtCache
	latestDeviceFieldQueryStmtCache       dbStmtCache
	collectDataArgsPool                   = sync.Pool{
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

// DataPointEntry 单个数据点条目
type DataPointEntry struct {
	DeviceID    int64
	DeviceName  string
	FieldName   string
	Value       string
	ValueType   string
	CollectedAt time.Time
}

type collectDataBatchSQLCache struct {
	mu      sync.RWMutex
	sqlByN  map[int]string
	builder func(int) string
}

const (
	collectDataCacheBatchSize   = 200
	collectDataHistoryBatchSize = 150
)

func newCollectDataBatchSQLCache(builder func(int) string) *collectDataBatchSQLCache {
	return &collectDataBatchSQLCache{
		sqlByN:  make(map[int]string, collectDataCacheBatchSize),
		builder: builder,
	}
}

func (c *collectDataBatchSQLCache) get(batchSize int) string {
	if c == nil || batchSize <= 0 {
		return ""
	}

	c.mu.RLock()
	sqlText, ok := c.sqlByN[batchSize]
	c.mu.RUnlock()
	if ok {
		return sqlText
	}

	sqlText = c.builder(batchSize)

	c.mu.Lock()
	if cached, exists := c.sqlByN[batchSize]; exists {
		c.mu.Unlock()
		return cached
	}
	c.sqlByN[batchSize] = sqlText
	c.mu.Unlock()

	return sqlText
}

func appendCollectDataCacheValuesSQL(dst []byte, batchSize int) []byte {
	for i := 0; i < batchSize; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, "(?, ?, ?, ?, CURRENT_TIMESTAMP)"...)
	}
	return dst
}

func appendCollectDataCacheStringValuesSQL(dst []byte, batchSize int) []byte {
	for i := 0; i < batchSize; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, "(?, ?, ?, 'string', CURRENT_TIMESTAMP)"...)
	}
	return dst
}

func appendCollectDataHistoryValuesSQL(dst []byte, batchSize int) []byte {
	for i := 0; i < batchSize; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, "(?, ?, ?, ?, ?, CURRENT_TIMESTAMP)"...)
	}
	return dst
}

func appendCollectDataHistoryStringValuesSQL(dst []byte, batchSize int) []byte {
	for i := 0; i < batchSize; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, "(?, ?, ?, ?, 'string', CURRENT_TIMESTAMP)"...)
	}
	return dst
}

func appendDataPointValuesSQL(dst []byte, batchSize int) []byte {
	for i := 0; i < batchSize; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, "(?, ?, ?, ?, ?, ?)"...)
	}
	return dst
}

func buildCollectDataCacheBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 128+batchSize*28)
	sqlText = append(sqlText, "INSERT INTO data_cache (device_id, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendCollectDataCacheValuesSQL(sqlText, batchSize)
	sqlText = append(sqlText, ` ON CONFLICT(device_id, field_name) DO UPDATE SET value = excluded.value, value_type = excluded.value_type, collected_at = CURRENT_TIMESTAMP`...)
	return string(sqlText)
}

func buildCollectDataCacheStringBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 128+batchSize*38)
	sqlText = append(sqlText, "INSERT INTO data_cache (device_id, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendCollectDataCacheStringValuesSQL(sqlText, batchSize)
	sqlText = append(sqlText, ` ON CONFLICT(device_id, field_name) DO UPDATE SET value = excluded.value, value_type = 'string', collected_at = CURRENT_TIMESTAMP`...)
	return string(sqlText)
}

func buildCollectDataHistoryBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 96+batchSize*32)
	sqlText = append(sqlText, "INSERT OR REPLACE INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendCollectDataHistoryValuesSQL(sqlText, batchSize)
	return string(sqlText)
}

func buildCollectDataHistoryStringBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 96+batchSize*42)
	sqlText = append(sqlText, "INSERT OR REPLACE INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendCollectDataHistoryStringValuesSQL(sqlText, batchSize)
	return string(sqlText)
}

func buildDataPointBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 96+batchSize*30)
	sqlText = append(sqlText, "INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendDataPointValuesSQL(sqlText, batchSize)
	return string(sqlText)
}

func buildLatestDataPointBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 96+batchSize*32)
	sqlText = append(sqlText, "INSERT OR REPLACE INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendCollectDataHistoryValuesSQL(sqlText, batchSize)
	return string(sqlText)
}

func appendCollectDataCacheArg(dst []any, deviceID int64, field, value string) []any {
	return append(dst, deviceID, field, value, collectDataValueTypeString)
}

func appendCollectDataCacheStringArg(dst []any, deviceID int64, field, value string) []any {
	return append(dst, deviceID, field, value)
}

func appendCollectDataHistoryArg(dst []any, deviceID int64, deviceName, field, value string) []any {
	return append(dst, deviceID, deviceName, field, value, collectDataValueTypeString)
}

func appendCollectDataHistoryStringArg(dst []any, deviceID int64, deviceName, field, value string) []any {
	return append(dst, deviceID, deviceName, field, value)
}

func appendDataPointArg(dst []any, entry DataPointEntry, collectedAt time.Time) []any {
	valueType := normalizedCollectDataValueType(entry.ValueType)
	return append(dst,
		entry.DeviceID,
		normalizeDeviceName(entry.DeviceID, entry.DeviceName),
		entry.FieldName,
		entry.Value,
		valueType,
		collectedAt,
	)
}

func appendLatestDataPointArg(dst []any, entry DataPointEntry) []any {
	return append(dst,
		entry.DeviceID,
		normalizeDeviceName(entry.DeviceID, entry.DeviceName),
		entry.FieldName,
		entry.Value,
		normalizedCollectDataValueType(entry.ValueType),
	)
}

func appendDataCacheEntryArg(dst []any, entry DataPointEntry) []any {
	return append(dst, entry.DeviceID, entry.FieldName, entry.Value, normalizedCollectDataValueType(entry.ValueType))
}

func normalizedCollectDataValueType(valueType string) string {
	if valueType == "" {
		return collectDataValueTypeString
	}
	return valueType
}

func normalizeCollectPointFieldNames(points []models.CollectPoint) ([]string, int) {
	if len(points) == 0 {
		return nil, 0
	}
	var names []string
	validCount := 0
	for i, point := range points {
		name := trimDataPointFieldName(point.FieldName)
		if name == "" {
			if names == nil {
				names = make([]string, len(points))
				for j := 0; j < i; j++ {
					names[j] = points[j].FieldName
				}
			}
			continue
		}
		if names != nil {
			names[i] = name
		} else if name != point.FieldName {
			names = make([]string, len(points))
			for j := 0; j < i; j++ {
				names[j] = points[j].FieldName
			}
			names[i] = name
		}
		validCount++
	}
	return names, validCount
}

func normalizedCollectPointFieldName(points []models.CollectPoint, normalizedPointFields []string, index int) string {
	if normalizedPointFields != nil {
		return normalizedPointFields[index]
	}
	return points[index].FieldName
}

type collectDataCacheShape struct {
	normalizedPointFields []string
	pointFieldNames       map[string]struct{}
	allFieldsOverridden   bool
	rows                  int
}

func buildCollectDataCacheShape(data *models.CollectData) collectDataCacheShape {
	if data == nil {
		return collectDataCacheShape{}
	}
	normalizedPointFields, validPointCount := normalizeCollectPointFieldNames(data.Points)
	pointFieldNames := collectOverriddenPointFieldNames(data.Fields, data.Points, normalizedPointFields)
	rows := validPointCount
	allFieldsOverridden := len(data.Fields) > 0 && len(pointFieldNames) == len(data.Fields)
	if !allFieldsOverridden {
		for field := range data.Fields {
			if _, overridden := pointFieldNames[field]; overridden {
				continue
			}
			rows++
		}
	}
	return collectDataCacheShape{
		normalizedPointFields: normalizedPointFields,
		pointFieldNames:       pointFieldNames,
		allFieldsOverridden:   allFieldsOverridden,
		rows:                  rows,
	}
}

func collectOverriddenPointFieldNames(fields map[string]string, points []models.CollectPoint, normalizedPointFields []string) map[string]struct{} {
	if len(fields) == 0 || len(points) == 0 {
		return nil
	}
	var names map[string]struct{}
	for i := range points {
		name := normalizedCollectPointFieldName(points, normalizedPointFields, i)
		if name == "" {
			continue
		}
		if _, exists := fields[name]; !exists {
			continue
		}
		if names == nil {
			names = make(map[string]struct{}, len(points))
		}
		names[name] = struct{}{}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

func countCollectDataCacheRows(data *models.CollectData) int {
	return buildCollectDataCacheShape(data).rows
}

func appendCollectDataCacheArgsForData(dst []any, data *models.CollectData) []any {
	return appendCollectDataCacheArgsForDataWithShape(dst, data, buildCollectDataCacheShape(data))
}

func appendCollectDataCacheArgsForDataWithShape(dst []any, data *models.CollectData, shape collectDataCacheShape) []any {
	if data == nil {
		return dst
	}
	if !shape.allFieldsOverridden {
		for field, value := range data.Fields {
			if _, overridden := shape.pointFieldNames[field]; overridden {
				continue
			}
			dst = appendCollectDataCacheStringArg(dst, data.DeviceID, field, value)
		}
	}
	for i, point := range data.Points {
		field := normalizedCollectPointFieldName(data.Points, shape.normalizedPointFields, i)
		if field == "" {
			continue
		}
		dst = appendCollectDataCacheStringArg(dst, data.DeviceID, field, models.CollectPointValueString(point.Value))
	}
	return dst
}

func getCollectDataArgs(capHint int) []any {
	args, _ := collectDataArgsPool.Get().([]any)
	if capHint <= 0 {
		return args[:0]
	}
	if cap(args) < capHint {
		return make([]any, 0, capHint)
	}
	return args[:0]
}

func putCollectDataArgs(args []any) {
	if args == nil {
		return
	}
	clear(args)
	collectDataArgsPool.Put(args[:0])
}

func flushCollectDataArgs(cacheStmtCache, historyStmtCache *collectDataStmtCache, batchCount int, cacheArgs, historyArgs []any, storeHistory bool) error {
	if batchCount <= 0 {
		return nil
	}
	if err := executeCollectDataCacheBatchWithArgs(cacheStmtCache, batchCount, cacheArgs); err != nil {
		return err
	}
	if storeHistory {
		if err := executeCollectDataHistoryBatchWithArgs(historyStmtCache, batchCount, historyArgs); err != nil {
			return err
		}
	}
	return nil
}

func insertCollectDataCacheDirect(data *models.CollectData) error {
	if data == nil {
		return nil
	}
	if len(data.Fields) == 1 && len(data.Points) == 0 {
		for field, value := range data.Fields {
			stmt, err := dataCacheExecStmtCache.get(DataDB, collectDataCacheSingleStringSQL)
			if err != nil {
				return fmt.Errorf("failed to prepare data cache statement: %w", err)
			}
			_, err = stmt.Exec(data.DeviceID, field, value)
			if err != nil {
				return fmt.Errorf("failed to upsert data cache batch: %w", err)
			}
			return nil
		}
	}

	if len(data.Fields) == 0 {
		validPoints := 0
		var singleField string
		var singleValue string
		for _, point := range data.Points {
			field := trimDataPointFieldName(point.FieldName)
			if field == "" {
				continue
			}
			validPoints++
			if validPoints == 1 {
				singleField = field
				singleValue = models.CollectPointValueString(point.Value)
			}
			if validPoints > 1 {
				break
			}
		}
		if validPoints == 0 {
			return nil
		}
		if validPoints == 1 {
			stmt, err := dataCacheExecStmtCache.get(DataDB, collectDataCacheSingleStringSQL)
			if err != nil {
				return fmt.Errorf("failed to prepare data cache statement: %w", err)
			}
			_, err = stmt.Exec(data.DeviceID, singleField, singleValue)
			if err != nil {
				return fmt.Errorf("failed to upsert data cache batch: %w", err)
			}
			return nil
		}
	}

	argsCap := len(data.Fields) + len(data.Points)
	if argsCap <= 0 {
		return nil
	}
	args := getCollectDataArgs(argsCap * 3)
	defer putCollectDataArgs(args)
	args = appendCollectDataCacheArgsForData(args, data)
	if len(args) == 0 {
		return nil
	}

	stmt, err := dataCacheExecStmtCache.get(DataDB, collectDataCacheStringBatchSQLCache.get(len(args)/3))
	if err != nil {
		return fmt.Errorf("failed to prepare data cache batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data cache batch: %w", err)
	}
	return nil
}

type collectDataStmtCache struct {
	tx     *sql.Tx
	sqlFor func(int) string
	stmts  map[int]*sql.Stmt
}

func newCollectDataStmtCache(tx *sql.Tx, sqlFor func(int) string) *collectDataStmtCache {
	return &collectDataStmtCache{
		tx:     tx,
		sqlFor: sqlFor,
	}
}

func (c *collectDataStmtCache) get(batchSize int) (*sql.Stmt, error) {
	if c == nil || c.tx == nil || batchSize <= 0 {
		return nil, nil
	}
	if c.stmts == nil {
		c.stmts = make(map[int]*sql.Stmt, 2)
	}
	if stmt, ok := c.stmts[batchSize]; ok {
		return stmt, nil
	}
	stmt, err := c.tx.Prepare(c.sqlFor(batchSize))
	if err != nil {
		return nil, err
	}
	c.stmts[batchSize] = stmt
	return stmt, nil
}

func (c *collectDataStmtCache) close() {
	if c == nil {
		return
	}
	for size, stmt := range c.stmts {
		if stmt != nil {
			_ = stmt.Close()
		}
		delete(c.stmts, size)
	}
}

func executeCollectDataCacheBatchWithArgs(stmtCache *collectDataStmtCache, batchSize int, args []any) error {
	if batchSize <= 0 {
		return nil
	}
	stmt, err := stmtCache.get(batchSize)
	if err != nil {
		return fmt.Errorf("failed to prepare data cache batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data cache batch: %w", err)
	}
	return nil
}

func executeCollectDataHistoryBatchWithArgs(stmtCache *collectDataStmtCache, batchSize int, args []any) error {
	if batchSize <= 0 {
		return nil
	}
	stmt, err := stmtCache.get(batchSize)
	if err != nil {
		return fmt.Errorf("failed to prepare data point batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to upsert data point batch: %w", err)
	}
	return nil
}

// BatchSaveDataPoints 批量保存历史数据点（提高写入性能）
func BatchSaveDataPoints(entries []DataPointEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if len(entries) <= collectDataHistoryBatchSize {
		if err := batchSaveDataPointsDirect(entries); err != nil {
			return err
		}
		noteHistoryRowsWritten(len(entries))
		maybeEnforceDataPointsLimit()
		TriggerSyncIfNeeded()
		return nil
	}

	// 使用事务批量插入
	tx, err := DataDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmtCache := newCollectDataStmtCache(tx, dataPointBatchSQLCache.get)
	defer stmtCache.close()
	args := getCollectDataArgs(collectDataHistoryBatchSize * 6)
	defer putCollectDataArgs(args)
	batchCount := 0
	now := time.Now()

	for _, entry := range entries {
		collectedAt := entry.CollectedAt
		if collectedAt.IsZero() {
			collectedAt = now
		}
		args = appendDataPointArg(args, entry, collectedAt)
		batchCount++
		if batchCount < collectDataHistoryBatchSize {
			continue
		}
		if err := executeCollectDataHistoryBatchWithArgs(stmtCache, batchCount, args); err != nil {
			return fmt.Errorf("failed to insert data point: %w", err)
		}
		args = args[:0]
		batchCount = 0
	}

	if batchCount > 0 {
		if err := executeCollectDataHistoryBatchWithArgs(stmtCache, batchCount, args); err != nil {
			return fmt.Errorf("failed to insert data point: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 节流检查并清理过量的数据点，避免每次写入都 count(*)
	noteHistoryRowsWritten(len(entries))
	maybeEnforceDataPointsLimit()

	// 检查是否需要触发同步
	TriggerSyncIfNeeded()

	return nil
}

func batchSaveDataPointsDirect(entries []DataPointEntry) error {
	args := getCollectDataArgs(len(entries) * 6)
	defer putCollectDataArgs(args)

	now := time.Now()
	for _, entry := range entries {
		collectedAt := entry.CollectedAt
		if collectedAt.IsZero() {
			collectedAt = now
		}
		args = appendDataPointArg(args, entry, collectedAt)
	}

	stmt, err := dataCacheExecStmtCache.get(DataDB, dataPointBatchSQLCache.get(len(entries)))
	if err != nil {
		return fmt.Errorf("failed to prepare data point batch statement: %w", err)
	}
	if _, err := stmt.Exec(args...); err != nil {
		return fmt.Errorf("failed to insert data point batch: %w", err)
	}
	return nil
}

type dataPointKey struct {
	deviceID    int64
	fieldName   string
	collectedAt int64
}

func mergeDataPoints(primary, secondary []*DataPoint, limit int) []*DataPoint {
	if limit <= 0 {
		limit = len(primary) + len(secondary)
	}

	if len(secondary) == 0 {
		if len(primary) > limit {
			return primary[:limit]
		}
		return primary
	}
	if len(primary) == 0 {
		if len(secondary) > limit {
			return secondary[:limit]
		}
		return secondary
	}
	if dataPointsDoNotOverlap(primary, secondary) {
		if len(primary) >= limit {
			return primary[:limit]
		}
		remaining := limit - len(primary)
		if remaining > len(secondary) {
			remaining = len(secondary)
		}
		result := make([]*DataPoint, 0, len(primary)+remaining)
		result = append(result, primary...)
		result = append(result, secondary[:remaining]...)
		return result
	}

	maxSeen := len(primary) + len(secondary)
	if maxSeen > limit {
		maxSeen = limit
	}
	if maxSeen <= 0 {
		maxSeen = 1
	}

	result := make([]*DataPoint, 0, limit)
	var seen map[dataPointKey]struct{}

	i, j := 0, 0
	for len(result) < limit && (i < len(primary) || j < len(secondary)) {
		var candidate *DataPoint
		takePrimary := j >= len(secondary)
		if !takePrimary && i < len(primary) {
			candidate = primary[i]
			other := secondary[j]
			if other == nil || (candidate != nil && !candidate.CollectedAt.Before(other.CollectedAt)) {
				takePrimary = true
			}
		}
		if takePrimary {
			candidate = primary[i]
			i++
		} else {
			candidate = secondary[j]
			j++
		}

		if candidate == nil {
			continue
		}

		if seen == nil {
			duplicate := false
			for _, existing := range result {
				if existing == nil {
					continue
				}
				if existing.DeviceID == candidate.DeviceID &&
					existing.FieldName == candidate.FieldName &&
					existing.CollectedAt.Equal(candidate.CollectedAt) {
					duplicate = true
					break
				}
			}
			if duplicate {
				seen = make(map[dataPointKey]struct{}, maxSeen)
				for _, existing := range result {
					if existing == nil {
						continue
					}
					seen[dataPointKey{
						deviceID:    existing.DeviceID,
						fieldName:   existing.FieldName,
						collectedAt: existing.CollectedAt.UnixNano(),
					}] = struct{}{}
				}
			} else {
				result = append(result, candidate)
				continue
			}
		}

		key := dataPointKey{
			deviceID:    candidate.DeviceID,
			fieldName:   candidate.FieldName,
			collectedAt: candidate.CollectedAt.UnixNano(),
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}

	return result
}

func dataPointsDoNotOverlap(primary, secondary []*DataPoint) bool {
	if len(primary) == 0 || len(secondary) == 0 {
		return true
	}
	oldestPrimary := primary[len(primary)-1]
	newestSecondary := secondary[0]
	if oldestPrimary == nil || newestSecondary == nil {
		return false
	}
	return !newestSecondary.CollectedAt.After(oldestPrimary.CollectedAt)
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
			log.Printf("Failed to read data points from disk: %v", err)
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
			log.Printf("Failed to read latest data points from disk: %v", err)
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
			log.Printf("Failed to read latest device fields from disk: %v", err)
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
		log.Printf("Failed to query latest device fields from disk: %v", err)
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
	normalizedPointFields, _ := normalizeCollectPointFieldNames(data.Points)
	pointFieldNames := collectOverriddenPointFieldNames(data.Fields, data.Points, normalizedPointFields)
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

// InsertCollectDataWithOptions 写入实时缓存，并按需写入历史数据。
func InsertCollectDataWithOptions(data *models.CollectData, storeHistory bool) error {
	if data == nil {
		return fmt.Errorf("collect data is nil")
	}
	if len(data.Fields) == 0 && len(data.Points) == 0 {
		return nil
	}
	if !storeHistory && ((len(data.Fields) > 0 && len(data.Fields) <= collectDataCacheBatchSize) || (len(data.Fields) == 0 && len(data.Points) <= collectDataCacheBatchSize)) {
		if err := insertCollectDataCacheDirect(data); err != nil {
			return err
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
