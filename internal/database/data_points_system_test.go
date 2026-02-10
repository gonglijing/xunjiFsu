package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func prepareDataPointsTestDB(t *testing.T) {
	t.Helper()

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
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		t.Fatalf("create data_points: %v", err)
	}

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
}

func TestSaveLatestDataPoint_NormalizesSystemDeviceName(t *testing.T) {
	prepareDataPointsTestDB(t)

	if err := SaveLatestDataPoint(models.SystemStatsDeviceID, "-1", "cpu_usage", "12.3"); err != nil {
		t.Fatalf("SaveLatestDataPoint() error = %v", err)
	}

	var got string
	if err := DataDB.QueryRow(`SELECT device_name FROM data_points WHERE device_id = ? AND field_name = ?`, models.SystemStatsDeviceID, "cpu_usage").Scan(&got); err != nil {
		t.Fatalf("query device_name: %v", err)
	}
	if got != models.SystemStatsDeviceName {
		t.Fatalf("device_name = %q, want %q", got, models.SystemStatsDeviceName)
	}
}

func TestGetAllDevicesLatestData_MergeMemoryAndDisk(t *testing.T) {
	prepareDataPointsTestDB(t)

	oldDataDBFile := dataDBFile
	t.Cleanup(func() {
		dataDBFile = oldDataDBFile
	})

	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "data.db")
	diskDB, err := openSQLite(diskPath, 1, 1)
	if err != nil {
		t.Fatalf("open disk db: %v", err)
	}
	defer func() { _ = diskDB.Close() }()

	_, err = diskDB.Exec(`CREATE TABLE data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name)
	)`)
	if err != nil {
		t.Fatalf("create disk data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES
		(1, 'dev-1', 'humidity', '55', 'string', datetime('now', '-10 seconds')),
		(2, 'dev-2', 'temperature', '20', 'string', datetime('now', '-5 seconds'))`)
	if err != nil {
		t.Fatalf("insert mem rows: %v", err)
	}

	_, err = diskDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES
		(1, 'dev-1', 'humidity', '66', 'string', datetime('now', '-1 seconds')),
		(1, 'dev-1', 'temperature', '18', 'string', datetime('now', '-2 seconds')),
		(3, 'dev-3', 'pressure', '100', 'string', datetime('now', '-1 seconds'))`)
	if err != nil {
		t.Fatalf("insert disk rows: %v", err)
	}

	dataDBFile = diskPath

	items, err := GetAllDevicesLatestData()
	if err != nil {
		t.Fatalf("GetAllDevicesLatestData: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 devices, got %d", len(items))
	}

	byID := make(map[int64]*LatestDeviceData, len(items))
	for _, item := range items {
		byID[item.DeviceID] = item
	}

	if got := byID[1].Fields["humidity"]; got != "66" {
		t.Fatalf("device1 humidity expected 66 from newer disk row, got %q", got)
	}
	if got := byID[1].Fields["temperature"]; got != "18" {
		t.Fatalf("device1 temperature expected 18, got %q", got)
	}
	if got := byID[2].Fields["temperature"]; got != "20" {
		t.Fatalf("device2 temperature expected 20 from mem row, got %q", got)
	}
	if got := byID[3].Fields["pressure"]; got != "100" {
		t.Fatalf("device3 pressure expected 100 from disk row, got %q", got)
	}
}

func TestGetAllDevicesLatestData_FallbackWhenDiskMissing(t *testing.T) {
	prepareDataPointsTestDB(t)

	oldDataDBFile := dataDBFile
	t.Cleanup(func() {
		dataDBFile = oldDataDBFile
	})

	_, err := DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (9, 'dev-9', 'humidity', '77', 'string', datetime('now'))`)
	if err != nil {
		t.Fatalf("insert mem rows: %v", err)
	}

	dataDBFile = filepath.Join(t.TempDir(), "missing.db")

	items, err := GetAllDevicesLatestData()
	if err != nil {
		t.Fatalf("GetAllDevicesLatestData: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 device from memory fallback, got %d", len(items))
	}
	if got := items[0].Fields["humidity"]; got != "77" {
		t.Fatalf("expected humidity=77, got %q", got)
	}
}

func TestBatchSaveDataCacheEntries_UpsertLatestValue(t *testing.T) {
	prepareDataPointsTestDB(t)

	entries := []DataPointEntry{
		{DeviceID: 1, FieldName: "humidity", Value: "50", ValueType: "string"},
		{DeviceID: 1, FieldName: "temperature", Value: "20", ValueType: "string"},
		{DeviceID: 1, FieldName: "humidity", Value: "55", ValueType: "string"},
	}
	if err := BatchSaveDataCacheEntries(entries); err != nil {
		t.Fatalf("BatchSaveDataCacheEntries: %v", err)
	}

	rows, err := GetDataCacheByDeviceID(1)
	if err != nil {
		t.Fatalf("GetDataCacheByDeviceID: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 cache rows, got %d", len(rows))
	}

	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.FieldName] = row.Value
	}
	if got := values["humidity"]; got != "55" {
		t.Fatalf("expected humidity latest value 55, got %q", got)
	}
	if got := values["temperature"]; got != "20" {
		t.Fatalf("expected temperature value 20, got %q", got)
	}
}

func TestInsertCollectDataWithOptions_StoreHistoryFlag(t *testing.T) {
	prepareDataPointsTestDB(t)

	collectedAt := time.Now()
	data := &models.CollectData{
		DeviceID:   2,
		DeviceName: "dev-2",
		Timestamp:  collectedAt,
		Fields: map[string]string{
			"humidity":    "66",
			"temperature": "18",
		},
	}

	if err := InsertCollectDataWithOptions(data, false); err != nil {
		t.Fatalf("InsertCollectDataWithOptions(storeHistory=false): %v", err)
	}

	var cacheCount int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_cache WHERE device_id = ?", data.DeviceID).Scan(&cacheCount); err != nil {
		t.Fatalf("count data_cache: %v", err)
	}
	if cacheCount != 2 {
		t.Fatalf("expected cache count 2, got %d", cacheCount)
	}

	var historyCount int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ?", data.DeviceID).Scan(&historyCount); err != nil {
		t.Fatalf("count data_points: %v", err)
	}
	if historyCount != 0 {
		t.Fatalf("expected history count 0 when storeHistory=false, got %d", historyCount)
	}

	if err := InsertCollectDataWithOptions(data, true); err != nil {
		t.Fatalf("InsertCollectDataWithOptions(storeHistory=true): %v", err)
	}
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ?", data.DeviceID).Scan(&historyCount); err != nil {
		t.Fatalf("count data_points after history write: %v", err)
	}
	if historyCount != 2 {
		t.Fatalf("expected history count 2 when storeHistory=true, got %d", historyCount)
	}
}

func TestDeleteHistoryDataByPoint_MemoryAndDisk(t *testing.T) {
	prepareDataPointsTestDB(t)

	oldDataDBFile := dataDBFile
	t.Cleanup(func() {
		dataDBFile = oldDataDBFile
	})

	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "data.db")
	diskDB, err := openSQLite(diskPath, 1, 1)
	if err != nil {
		t.Fatalf("open disk db: %v", err)
	}
	defer func() { _ = diskDB.Close() }()

	_, err = diskDB.Exec(`CREATE TABLE data_points (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		device_name TEXT NOT NULL,
		field_name TEXT NOT NULL,
		value TEXT NOT NULL,
		value_type TEXT DEFAULT 'string',
		collected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(device_id, field_name, collected_at)
	)`)
	if err != nil {
		t.Fatalf("create disk data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES
		(1, 'dev-1', 'temperature', '20', 'string', datetime('now', '-10 seconds')),
		(1, 'dev-1', 'humidity', '60', 'string', datetime('now', '-9 seconds'))`)
	if err != nil {
		t.Fatalf("insert mem rows: %v", err)
	}

	_, err = diskDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at) VALUES
		(1, 'dev-1', 'temperature', '19', 'string', datetime('now', '-7 seconds')),
		(1, 'dev-1', 'humidity', '58', 'string', datetime('now', '-6 seconds'))`)
	if err != nil {
		t.Fatalf("insert disk rows: %v", err)
	}

	dataDBFile = diskPath

	deleted, err := DeleteHistoryDataByPoint(1, "temperature")
	if err != nil {
		t.Fatalf("DeleteHistoryDataByPoint: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	var memTempCount int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ? AND field_name = ?", 1, "temperature").Scan(&memTempCount); err != nil {
		t.Fatalf("count memory temperature rows: %v", err)
	}
	if memTempCount != 0 {
		t.Fatalf("memory temperature rows = %d, want 0", memTempCount)
	}

	var memHumidityCount int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ? AND field_name = ?", 1, "humidity").Scan(&memHumidityCount); err != nil {
		t.Fatalf("count memory humidity rows: %v", err)
	}
	if memHumidityCount != 1 {
		t.Fatalf("memory humidity rows = %d, want 1", memHumidityCount)
	}

	var diskTempCount int
	if err := diskDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ? AND field_name = ?", 1, "temperature").Scan(&diskTempCount); err != nil {
		t.Fatalf("count disk temperature rows: %v", err)
	}
	if diskTempCount != 0 {
		t.Fatalf("disk temperature rows = %d, want 0", diskTempCount)
	}

	var diskHumidityCount int
	if err := diskDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ? AND field_name = ?", 1, "humidity").Scan(&diskHumidityCount); err != nil {
		t.Fatalf("count disk humidity rows: %v", err)
	}
	if diskHumidityCount != 1 {
		t.Fatalf("disk humidity rows = %d, want 1", diskHumidityCount)
	}
}
