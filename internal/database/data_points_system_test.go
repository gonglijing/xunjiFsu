package database

import (
	"path/filepath"
	"testing"

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
