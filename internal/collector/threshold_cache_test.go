package collector

import (
	"database/sql"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func setupThresholdCacheTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db failed: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE devices (
		id INTEGER PRIMARY KEY,
		name TEXT,
		description TEXT,
		product_key TEXT,
		device_key TEXT,
		driver_type TEXT,
		serial_port TEXT,
		baud_rate INTEGER,
		data_bits INTEGER,
		stop_bits INTEGER,
		parity TEXT,
		ip_address TEXT,
		port_num INTEGER,
		device_address TEXT,
		collect_interval INTEGER,
		storage_interval INTEGER,
		timeout INTEGER,
		driver_id INTEGER,
		enabled INTEGER,
		resource_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create devices table failed: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE thresholds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER,
		field_name TEXT,
		operator TEXT,
		value REAL,
		severity TEXT,
		enabled INTEGER,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create thresholds table failed: %v", err)
	}

	return db
}

func resetThresholdCache() {
	cache.mu.Lock()
	cache.thresholds = make(map[int64][]*models.Threshold)
	cache.rules = make(map[int64][]thresholdEvalRule)
	cache.lastRefresh = time.Time{}
	cache.mu.Unlock()
}

func TestGetDeviceThresholds_RefreshesStaleCache(t *testing.T) {
	oldDB := database.ParamDB
	db := setupThresholdCacheTestDB(t)
	database.ParamDB = db
	defer func() {
		StopThresholdCache()
		resetThresholdCache()
		database.ParamDB = oldDB
		_ = db.Close()
	}()

	_, err := db.Exec(`INSERT INTO devices (
		id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity,
		ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id
	) VALUES (1, 'd1', '', '', '', 'modbus_rtu', '', 9600, 8, 1, 'N', '', 0, '1', 1000, 300, 1000, NULL, 1, NULL)`)
	if err != nil {
		t.Fatalf("insert device failed: %v", err)
	}

	_, err = db.Exec(`INSERT INTO thresholds (device_id, field_name, operator, value, severity, enabled, message)
		VALUES (1, 'humidity', '>', 50, 'warning', 1, '湿度高')`)
	if err != nil {
		t.Fatalf("insert threshold failed: %v", err)
	}

	resetThresholdCache()
	cache.mu.Lock()
	cache.thresholds[1] = []*models.Threshold{}
	cache.lastRefresh = time.Now().Add(-3 * cache.interval)
	cache.mu.Unlock()

	thresholds, _, err := getThresholdCacheEntry(1)
	if err != nil {
		t.Fatalf("getThresholdCacheEntry failed: %v", err)
	}
	if len(thresholds) != 1 {
		t.Fatalf("expected 1 threshold after stale refresh, got %d", len(thresholds))
	}
	if thresholds[0].FieldName != "humidity" {
		t.Fatalf("expected field_name humidity, got %q", thresholds[0].FieldName)
	}
}

func TestThresholdCache_RefreshRebuildsAndDropsStaleEntries(t *testing.T) {
	oldDB := database.ParamDB
	db := setupThresholdCacheTestDB(t)
	database.ParamDB = db
	defer func() {
		StopThresholdCache()
		resetThresholdCache()
		database.ParamDB = oldDB
		_ = db.Close()
	}()

	_, err := db.Exec(`INSERT INTO devices (
		id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity,
		ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id
	) VALUES
	(1, 'd1', '', '', '', 'modbus_rtu', '', 9600, 8, 1, 'N', '', 0, '1', 1000, 300, 1000, NULL, 1, NULL),
	(2, 'd2', '', '', '', 'modbus_rtu', '', 9600, 8, 1, 'N', '', 0, '2', 1000, 300, 1000, NULL, 1, NULL)`)
	if err != nil {
		t.Fatalf("insert devices failed: %v", err)
	}

	_, err = db.Exec(`INSERT INTO thresholds (device_id, field_name, operator, value, severity, enabled, message)
		VALUES
		(1, 'humidity', '>', 50, 'warning', 1, '湿度高'),
		(999, 'temperature', '>', 60, 'warning', 1, '孤立阈值')`)
	if err != nil {
		t.Fatalf("insert thresholds failed: %v", err)
	}

	resetThresholdCache()
	cache.mu.Lock()
	cache.thresholds[12345] = []*models.Threshold{{DeviceID: 12345, FieldName: "stale"}}
	cache.mu.Unlock()

	cache.Refresh()

	cache.mu.RLock()
	defer cache.mu.RUnlock()

	if _, ok := cache.thresholds[12345]; ok {
		t.Fatalf("expected stale device cache to be removed")
	}
	if got := len(cache.thresholds[1]); got != 1 {
		t.Fatalf("expected device 1 threshold count 1, got %d", got)
	}
	if _, ok := cache.thresholds[2]; ok {
		t.Fatalf("expected device 2 with no thresholds to be absent from cache")
	}
}

func TestThresholdCache_StartStopIdempotent(t *testing.T) {
	oldDB := database.ParamDB
	db := setupThresholdCacheTestDB(t)
	database.ParamDB = db
	defer func() {
		StopThresholdCache()
		resetThresholdCache()
		database.ParamDB = oldDB
		_ = db.Close()
	}()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("start/stop should be idempotent, panic: %v", recovered)
		}
	}()

	cache.mu.Lock()
	prevInterval := cache.interval
	cache.interval = 10 * time.Millisecond
	cache.mu.Unlock()
	defer func() {
		cache.mu.Lock()
		cache.interval = prevInterval
		cache.mu.Unlock()
	}()

	StartThresholdCache()
	StartThresholdCache()
	time.Sleep(25 * time.Millisecond)
	StopThresholdCache()
	StopThresholdCache()
}

func TestGetDeviceThresholds_DoesNotCacheEmptyMiss(t *testing.T) {
	oldDB := database.ParamDB
	db := setupThresholdCacheTestDB(t)
	database.ParamDB = db
	defer func() {
		StopThresholdCache()
		resetThresholdCache()
		database.ParamDB = oldDB
		_ = db.Close()
	}()

	_, err := db.Exec(`INSERT INTO devices (
		id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity,
		ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id
	) VALUES (10, 'd10', '', '', '', 'modbus_rtu', '', 9600, 8, 1, 'N', '', 0, '10', 1000, 300, 1000, NULL, 1, NULL)`)
	if err != nil {
		t.Fatalf("insert device failed: %v", err)
	}

	resetThresholdCache()

	thresholds, _, err := getThresholdCacheEntry(10)
	if err != nil {
		t.Fatalf("getThresholdCacheEntry first call failed: %v", err)
	}
	if len(thresholds) != 0 {
		t.Fatalf("expected 0 thresholds, got %d", len(thresholds))
	}

	cache.mu.RLock()
	_, existsAfterMiss := cache.thresholds[10]
	cache.mu.RUnlock()
	if existsAfterMiss {
		t.Fatalf("empty threshold miss should not be cached")
	}

	_, err = db.Exec(`INSERT INTO thresholds (device_id, field_name, operator, value, severity, message)
		VALUES (10, 'humidity', '>', 50, 'warning', '湿度高')`)
	if err != nil {
		t.Fatalf("insert threshold failed: %v", err)
	}

	thresholds, _, err = getThresholdCacheEntry(10)
	if err != nil {
		t.Fatalf("getThresholdCacheEntry second call failed: %v", err)
	}
	if len(thresholds) != 1 {
		t.Fatalf("expected 1 threshold after insert, got %d", len(thresholds))
	}
}

func TestGetDeviceThresholdRules_RebuildsMissingRulesCache(t *testing.T) {
	resetThresholdCache()

	cache.mu.Lock()
	cache.thresholds[99] = []*models.Threshold{
		{ID: 1, DeviceID: 99, FieldName: " temp ", Operator: ">", Value: 30},
	}
	cache.lastRefresh = time.Now()
	cache.mu.Unlock()

	rules, err := getDeviceThresholdRules(99)
	if err != nil {
		t.Fatalf("getDeviceThresholdRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rebuilt rule, got %d", len(rules))
	}
	if rules[0].fieldName != "temp" {
		t.Fatalf("expected trimmed field name temp, got %q", rules[0].fieldName)
	}
	if rules[0].normalizedFieldName != "temp" {
		t.Fatalf("expected normalized field name temp, got %q", rules[0].normalizedFieldName)
	}

	cache.mu.RLock()
	defer cache.mu.RUnlock()
	if len(cache.rules[99]) != 1 {
		t.Fatalf("expected rules cache rebuilt for device 99")
	}
}
