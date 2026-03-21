package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

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
	collectDataCacheBatchSize   = 100
	collectDataHistoryBatchSize = 100
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

func appendDataPointStringValuesSQL(dst []byte, batchSize int) []byte {
	for i := 0; i < batchSize; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = append(dst, "(?, ?, ?, ?, 'string', ?)"...)
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

func buildDataPointStringBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 96+batchSize*40)
	sqlText = append(sqlText, "INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendDataPointStringValuesSQL(sqlText, batchSize)
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

func buildLatestDataPointStringBatchSQL(batchSize int) string {
	if batchSize <= 0 {
		return ""
	}
	sqlText := make([]byte, 0, 96+batchSize*42)
	sqlText = append(sqlText, "INSERT OR REPLACE INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES "...)
	sqlText = appendCollectDataHistoryStringValuesSQL(sqlText, batchSize)
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

func appendDataPointStringArg(dst []any, entry DataPointEntry, collectedAt time.Time) []any {
	return append(dst,
		entry.DeviceID,
		normalizeDeviceName(entry.DeviceID, entry.DeviceName),
		entry.FieldName,
		entry.Value,
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

func appendLatestDataPointStringArg(dst []any, entry DataPointEntry) []any {
	return append(dst,
		entry.DeviceID,
		normalizeDeviceName(entry.DeviceID, entry.DeviceName),
		entry.FieldName,
		entry.Value,
	)
}

func appendDataCacheEntryArg(dst []any, entry DataPointEntry) []any {
	return append(dst, entry.DeviceID, entry.FieldName, entry.Value, normalizedCollectDataValueType(entry.ValueType))
}

func appendDataCacheEntryStringArg(dst []any, entry DataPointEntry) []any {
	return append(dst, entry.DeviceID, entry.FieldName, entry.Value)
}

func normalizedCollectDataValueType(valueType string) string {
	if valueType == "" {
		return collectDataValueTypeString
	}
	return valueType
}

func allEntriesUseDefaultCollectDataValueType(entries []DataPointEntry) bool {
	for _, entry := range entries {
		if entry.ValueType != "" && entry.ValueType != collectDataValueTypeString {
			return false
		}
	}
	return true
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

func (s *collectDataCacheShape) release() {
	if s == nil {
		return
	}
	putCollectDataFieldNameSet(s.pointFieldNames)
	s.pointFieldNames = nil
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

func getCollectDataFieldNameSet() map[string]struct{} {
	names, _ := collectDataFieldNameSetPool.Get().(map[string]struct{})
	if names == nil {
		return make(map[string]struct{}, collectDataCacheBatchSize)
	}
	return names
}

func putCollectDataFieldNameSet(names map[string]struct{}) {
	if names == nil {
		return
	}
	clear(names)
	collectDataFieldNameSetPool.Put(names)
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
			names = getCollectDataFieldNameSet()
		}
		names[name] = struct{}{}
	}
	if len(names) == 0 {
		putCollectDataFieldNameSet(names)
		return nil
	}
	return names
}

func countCollectDataCacheRows(data *models.CollectData) int {
	shape := buildCollectDataCacheShape(data)
	defer shape.release()
	return shape.rows
}

func appendCollectDataCacheArgsForData(dst []any, data *models.CollectData) []any {
	if data == nil {
		return dst
	}
	if len(data.Points) == 0 {
		for field, value := range data.Fields {
			dst = appendCollectDataCacheStringArg(dst, data.DeviceID, field, value)
		}
		return dst
	}
	shape := buildCollectDataCacheShape(data)
	defer shape.release()
	return appendCollectDataCacheArgsForDataWithShape(dst, data, shape)
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

func insertCollectDataCacheChunkedDirect(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	args := getCollectDataArgs(collectDataCacheBatchSize * 3)
	defer putCollectDataArgs(args)

	batchRows := 0
	flush := func() error {
		if batchRows == 0 {
			return nil
		}
		stmt, err := dataCacheExecStmtCache.get(DataDB, collectDataCacheStringBatchSQLCache.get(batchRows))
		if err != nil {
			return fmt.Errorf("failed to prepare data cache batch statement: %w", err)
		}
		if _, err := stmt.Exec(args...); err != nil {
			return fmt.Errorf("failed to upsert data cache batch: %w", err)
		}
		args = args[:0]
		batchRows = 0
		return nil
	}

	appendField := func(field, value string) error {
		args = appendCollectDataCacheStringArg(args, data.DeviceID, field, value)
		batchRows++
		if batchRows < collectDataCacheBatchSize {
			return nil
		}
		return flush()
	}

	if len(data.Points) == 0 {
		for field, value := range data.Fields {
			if err := appendField(field, value); err != nil {
				return err
			}
		}
		return flush()
	}

	if len(data.Fields) == 0 {
		for _, point := range data.Points {
			field := trimDataPointFieldName(point.FieldName)
			if field == "" {
				continue
			}
			if err := appendField(field, models.CollectPointValueString(point.Value)); err != nil {
				return err
			}
		}
		return flush()
	}

	shape := buildCollectDataCacheShape(data)
	defer shape.release()
	if !shape.allFieldsOverridden {
		for field, value := range data.Fields {
			if _, overridden := shape.pointFieldNames[field]; overridden {
				continue
			}
			if err := appendField(field, value); err != nil {
				return err
			}
		}
	}
	for i, point := range data.Points {
		field := normalizedCollectPointFieldName(data.Points, shape.normalizedPointFields, i)
		if field == "" {
			continue
		}
		if err := appendField(field, models.CollectPointValueString(point.Value)); err != nil {
			return err
		}
	}
	return flush()
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
	defaultValueType := allEntriesUseDefaultCollectDataValueType(entries)
	if len(entries) <= collectDataHistoryBatchSize {
		if err := batchSaveDataPointsDirect(entries, defaultValueType); err != nil {
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

	sqlForBatch := dataPointBatchSQLCache.get
	if defaultValueType {
		sqlForBatch = dataPointStringBatchSQLCache.get
	}
	stmtCache := newCollectDataStmtCache(tx, sqlForBatch)
	defer stmtCache.close()
	argsPerEntry := 6
	if defaultValueType {
		argsPerEntry = 5
	}
	args := getCollectDataArgs(collectDataHistoryBatchSize * argsPerEntry)
	defer putCollectDataArgs(args)
	batchCount := 0
	now := time.Now()

	for _, entry := range entries {
		collectedAt := entry.CollectedAt
		if collectedAt.IsZero() {
			collectedAt = now
		}
		if defaultValueType {
			args = appendDataPointStringArg(args, entry, collectedAt)
		} else {
			args = appendDataPointArg(args, entry, collectedAt)
		}
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

func batchSaveDataPointsDirect(entries []DataPointEntry, defaultValueType bool) error {
	argsPerEntry := 6
	if defaultValueType {
		argsPerEntry = 5
	}
	args := getCollectDataArgs(len(entries) * argsPerEntry)
	defer putCollectDataArgs(args)

	now := time.Now()
	for _, entry := range entries {
		collectedAt := entry.CollectedAt
		if collectedAt.IsZero() {
			collectedAt = now
		}
		if defaultValueType {
			args = appendDataPointStringArg(args, entry, collectedAt)
		} else {
			args = appendDataPointArg(args, entry, collectedAt)
		}
	}

	sqlCache := dataPointBatchSQLCache
	if defaultValueType {
		sqlCache = dataPointStringBatchSQLCache
	}
	stmt, err := dataCacheExecStmtCache.get(DataDB, sqlCache.get(len(entries)))
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
