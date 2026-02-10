package database

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestApplyRuntimeLimits(t *testing.T) {
	oldPoints := maxDataPointsLimit
	oldCache := maxDataCacheLimit
	defer func() {
		maxDataPointsLimit = oldPoints
		maxDataCacheLimit = oldCache
	}()

	ApplyRuntimeLimits(1234, 567)
	if maxDataPointsLimit != 1234 {
		t.Fatalf("maxDataPointsLimit = %d, want 1234", maxDataPointsLimit)
	}
	if maxDataCacheLimit != 567 {
		t.Fatalf("maxDataCacheLimit = %d, want 567", maxDataCacheLimit)
	}

	ApplyRuntimeLimits(0, -1)
	if maxDataPointsLimit != MaxDataPoints {
		t.Fatalf("maxDataPointsLimit = %d, want default %d", maxDataPointsLimit, MaxDataPoints)
	}
	if maxDataCacheLimit != MaxDataCache {
		t.Fatalf("maxDataCacheLimit = %d, want default %d", maxDataCacheLimit, MaxDataCache)
	}
}

func TestApplySyncInterval(t *testing.T) {
	old := syncInterval
	defer func() { syncInterval = old }()

	ApplySyncInterval(30 * time.Second)
	if syncInterval != 30*time.Second {
		t.Fatalf("syncInterval = %v, want 30s", syncInterval)
	}

	ApplySyncInterval(0)
	if syncInterval != SyncInterval {
		t.Fatalf("syncInterval = %v, want default %v", syncInterval, SyncInterval)
	}
}

func TestTriggerSyncIfNeeded(t *testing.T) {
	oldTrigger := syncBatchTrigger
	oldFn := syncDataToDiskFn
	defer func() {
		syncBatchTrigger = oldTrigger
		syncDataToDiskFn = oldFn
	}()

	syncBatchTrigger = 2
	done := make(chan struct{})
	syncDataToDiskFn = func() error {
		close(done)
		return nil
	}

	if DataDB != nil {
		_ = DataDB.Close()
	}
	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	t.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = DataDB.Exec(`CREATE TABLE data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-1 seconds'))`, 1, "dev1", "f1", "1", "string")
	if err != nil {
		t.Fatalf("insert data point 1: %v", err)
	}

	if TriggerSyncIfNeeded() {
		t.Fatalf("expected trigger to be false below threshold")
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now'))`, 1, "dev1", "f2", "2", "string")
	if err != nil {
		t.Fatalf("insert data point 2: %v", err)
	}

	if !TriggerSyncIfNeeded() {
		t.Fatalf("expected trigger to be true at threshold")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("sync function was not called")
	}
}

func TestEnforceDataCacheLimit(t *testing.T) {
	oldLimit := maxDataCacheLimit
	maxDataCacheLimit = 3
	defer func() {
		maxDataCacheLimit = oldLimit
	}()

	if DataDB != nil {
		_ = DataDB.Close()
	}
	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	t.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = DataDB.Exec(`CREATE TABLE data_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		t.Fatalf("create data_cache: %v", err)
	}

	tx, err := DataDB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO data_cache (device_id, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, datetime('now', ?))`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("prepare insert: %v", err)
	}
	for i := 1; i <= 5; i++ {
		shift := fmt.Sprintf("-%d seconds", 6-i)
		if _, err := stmt.Exec(1, fmt.Sprintf("f%d", i), fmt.Sprintf("v%d", i), "string", shift); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			t.Fatalf("insert data_cache %d: %v", i, err)
		}
	}
	_ = stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	enforceDataCacheLimit()

	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&count); err != nil {
		t.Fatalf("count data_cache: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 rows after cleanup, got %d", count)
	}

	rows, err := DataDB.Query("SELECT field_name FROM data_cache ORDER BY collected_at ASC")
	if err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	defer rows.Close()
	var fields []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan field_name: %v", err)
		}
		fields = append(fields, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	expected := []string{"f3", "f4", "f5"}
	if len(fields) != len(expected) {
		t.Fatalf("expected fields %v, got %v", expected, fields)
	}
	for i, name := range expected {
		if fields[i] != name {
			t.Fatalf("expected fields %v, got %v", expected, fields)
		}
	}
}

func TestEnforceDataPointsLimit(t *testing.T) {
	oldLimit := maxDataPointsLimit
	maxDataPointsLimit = 3
	defer func() {
		maxDataPointsLimit = oldLimit
	}()

	if DataDB != nil {
		_ = DataDB.Close()
	}
	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	t.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = DataDB.Exec(`CREATE TABLE data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create data_points: %v", err)
	}

	tx, err := DataDB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', ?))`)
	if err != nil {
		_ = tx.Rollback()
		t.Fatalf("prepare insert: %v", err)
	}
	for i := 1; i <= 5; i++ {
		shift := fmt.Sprintf("-%d seconds", 6-i)
		if _, err := stmt.Exec(1, "dev1", fmt.Sprintf("f%d", i), fmt.Sprintf("v%d", i), "string", shift); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			t.Fatalf("insert data_points %d: %v", i, err)
		}
	}
	_ = stmt.Close()
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	enforceDataPointsLimit()

	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		t.Fatalf("count data_points: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 rows after cleanup, got %d", count)
	}

	rows, err := DataDB.Query("SELECT field_name FROM data_points ORDER BY collected_at ASC")
	if err != nil {
		t.Fatalf("query remaining: %v", err)
	}
	defer rows.Close()
	var fields []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan field_name: %v", err)
		}
		fields = append(fields, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	expected := []string{"f3", "f4", "f5"}
	if len(fields) != len(expected) {
		t.Fatalf("expected fields %v, got %v", expected, fields)
	}
	for i, name := range expected {
		if fields[i] != name {
			t.Fatalf("expected fields %v, got %v", expected, fields)
		}
	}
}

func TestSaveDataCache_ThrottledCleanup(t *testing.T) {
	oldLimit := maxDataCacheLimit
	oldEvery := dataCacheCleanupEveryWrites
	oldInterval := dataCacheCleanupMinInterval
	defer func() {
		maxDataCacheLimit = oldLimit
		dataCacheCleanupEveryWrites = oldEvery
		dataCacheCleanupMinInterval = oldInterval
		atomic.StoreUint64(&dataCacheCleanupCounter, 0)
		atomic.StoreInt64(&dataCacheLastCleanupNS, 0)
	}()

	maxDataCacheLimit = 3
	dataCacheCleanupEveryWrites = 1000
	dataCacheCleanupMinInterval = time.Hour
	atomic.StoreUint64(&dataCacheCleanupCounter, 0)
	atomic.StoreInt64(&dataCacheLastCleanupNS, 0)

	if DataDB != nil {
		_ = DataDB.Close()
	}
	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	t.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = DataDB.Exec(`CREATE TABLE data_cache (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		t.Fatalf("create data_cache: %v", err)
	}

	for i := 1; i <= 5; i++ {
		if err := SaveDataCache(1, "dev1", fmt.Sprintf("f%d", i), fmt.Sprintf("v%d", i), "string"); err != nil {
			t.Fatalf("SaveDataCache %d: %v", i, err)
		}
	}

	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&count); err != nil {
		t.Fatalf("count data_cache before forced cleanup: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected throttled mode to keep 5 rows before cleanup, got %d", count)
	}

	dataCacheCleanupEveryWrites = 1
	dataCacheCleanupMinInterval = 0
	if err := SaveDataCache(1, "dev1", "f6", "v6", "string"); err != nil {
		t.Fatalf("SaveDataCache force cleanup: %v", err)
	}

	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_cache").Scan(&count); err != nil {
		t.Fatalf("count data_cache after forced cleanup: %v", err)
	}
	if count > maxDataCacheLimit {
		t.Fatalf("expected count <= %d after cleanup, got %d", maxDataCacheLimit, count)
	}
}

func TestSaveDataPoint_ThrottledCleanup(t *testing.T) {
	oldLimit := maxDataPointsLimit
	oldEvery := dataPointsCleanupEveryWrites
	oldInterval := dataPointsCleanupMinInterval
	defer func() {
		maxDataPointsLimit = oldLimit
		dataPointsCleanupEveryWrites = oldEvery
		dataPointsCleanupMinInterval = oldInterval
		atomic.StoreUint64(&dataPointsCleanupCounter, 0)
		atomic.StoreInt64(&dataPointsLastCleanupNS, 0)
	}()

	maxDataPointsLimit = 3
	dataPointsCleanupEveryWrites = 1000
	dataPointsCleanupMinInterval = time.Hour
	atomic.StoreUint64(&dataPointsCleanupCounter, 0)
	atomic.StoreInt64(&dataPointsLastCleanupNS, 0)

	if DataDB != nil {
		_ = DataDB.Close()
	}
	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	t.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = DataDB.Exec(`CREATE TABLE data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create data_points: %v", err)
	}

	for i := 1; i <= 5; i++ {
		if err := SaveDataPoint(1, "dev1", fmt.Sprintf("f%d", i), fmt.Sprintf("v%d", i), "string"); err != nil {
			t.Fatalf("SaveDataPoint %d: %v", i, err)
		}
	}

	var count int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		t.Fatalf("count data_points before forced cleanup: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected throttled mode to keep 5 rows before cleanup, got %d", count)
	}

	dataPointsCleanupEveryWrites = 1
	dataPointsCleanupMinInterval = 0
	if err := SaveDataPoint(1, "dev1", "f6", "v6", "string"); err != nil {
		t.Fatalf("SaveDataPoint force cleanup: %v", err)
	}

	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		t.Fatalf("count data_points after forced cleanup: %v", err)
	}
	if count > maxDataPointsLimit {
		t.Fatalf("expected count <= %d after cleanup, got %d", maxDataPointsLimit, count)
	}
}
