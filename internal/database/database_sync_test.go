package database

import (
	"path/filepath"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestSyncDataToDisk(t *testing.T) {
	tmpDir := t.TempDir()
	dataDBFile = filepath.Join(tmpDir, "data.db")

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

	_, _ = DataDB.Exec("PRAGMA foreign_keys = OFF")

	_, err = DataDB.Exec(`CREATE TABLE data_points (
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
		t.Fatalf("create data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-1 hours'))`, 1, "dev1", "temp", "20", "string")
	if err != nil {
		t.Fatalf("insert data point 1: %v", err)
	}
	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-30 minutes'))`, 2, "dev2", "hum", "55", "string")
	if err != nil {
		t.Fatalf("insert data point 2: %v", err)
	}

	if err := syncDataToDisk(); err != nil {
		t.Fatalf("syncDataToDisk: %v", err)
	}

	var memCount int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&memCount); err != nil {
		t.Fatalf("count memory data points: %v", err)
	}
	if memCount != 0 {
		t.Fatalf("expected memory data points to be cleared, got %d", memCount)
	}

	diskDB, err := openSQLite(dataDBFile, 1, 1)
	if err != nil {
		t.Fatalf("open disk db: %v", err)
	}
	defer diskDB.Close()

	var count int
	if err := diskDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&count); err != nil {
		t.Fatalf("count disk data points: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 data points, got %d", count)
	}
}

func TestSyncDataToDisk_NormalizesSystemDeviceName(t *testing.T) {
	tmpDir := t.TempDir()
	dataDBFile = filepath.Join(tmpDir, "data.db")

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

	_, _ = DataDB.Exec("PRAGMA foreign_keys = OFF")

	_, err = DataDB.Exec(`CREATE TABLE data_points (
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
		t.Fatalf("create data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-1 minutes'))`, models.SystemStatsDeviceID, "-1", "cpu_usage", "20", "string")
	if err != nil {
		t.Fatalf("insert system point: %v", err)
	}

	if err := syncDataToDisk(); err != nil {
		t.Fatalf("syncDataToDisk: %v", err)
	}

	diskDB, err := openSQLite(dataDBFile, 1, 1)
	if err != nil {
		t.Fatalf("open disk db: %v", err)
	}
	defer diskDB.Close()

	var got string
	if err := diskDB.QueryRow(`SELECT device_name FROM data_points WHERE device_id = ? AND field_name = ?`, models.SystemStatsDeviceID, "cpu_usage").Scan(&got); err != nil {
		t.Fatalf("query disk system point: %v", err)
	}
	if got != models.SystemStatsDeviceName {
		t.Fatalf("disk device_name = %q, want %q", got, models.SystemStatsDeviceName)
	}
}

func TestCleanupOldDataByGatewayRetention(t *testing.T) {
	gatewayColumnsEnsured = false
	if ParamDB != nil {
		_ = ParamDB.Close()
	}
	if DataDB != nil {
		_ = DataDB.Close()
	}

	tmpDir := t.TempDir()
	paramDBFile = filepath.Join(tmpDir, "param.db")
	dataDBFile = filepath.Join(tmpDir, "data.db")

	var err error
	ParamDB, err = openSQLite(paramDBFile, 1, 1)
	if err != nil {
		t.Fatalf("open param db: %v", err)
	}
	t.Cleanup(func() {
		_ = ParamDB.Close()
	})

	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}
	t.Cleanup(func() {
		_ = DataDB.Close()
	})

	_, err = ParamDB.Exec(`CREATE TABLE gateway_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_key TEXT NOT NULL,
		device_key TEXT NOT NULL,
		gateway_name TEXT DEFAULT 'HuShu智能网关',
		data_retention_days INTEGER DEFAULT 30,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create gateway_config: %v", err)
	}

	_, err = ParamDB.Exec(`INSERT INTO gateway_config (product_key, device_key, gateway_name, data_retention_days)
		VALUES ('', '', 'HuShu智能网关', 1)`)
	if err != nil {
		t.Fatalf("insert gateway config: %v", err)
	}

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
		t.Fatalf("create memory data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-2 days'))`, 1, "dev1", "temp", "20", "string")
	if err != nil {
		t.Fatalf("insert memory old point: %v", err)
	}
	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-12 hours'))`, 1, "dev1", "temp", "21", "string")
	if err != nil {
		t.Fatalf("insert memory fresh point: %v", err)
	}

	diskDB, err := openSQLite(dataDBFile, 1, 1)
	if err != nil {
		t.Fatalf("open disk db: %v", err)
	}
	defer diskDB.Close()

	if err := ensureDiskDataSchema(diskDB); err != nil {
		t.Fatalf("ensure disk schema: %v", err)
	}

	_, err = diskDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-3 days'))`, 2, "dev2", "hum", "45", "string")
	if err != nil {
		t.Fatalf("insert disk old point: %v", err)
	}
	_, err = diskDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-2 hours'))`, 2, "dev2", "hum", "47", "string")
	if err != nil {
		t.Fatalf("insert disk fresh point: %v", err)
	}

	deleted, err := CleanupOldDataByGatewayRetention()
	if err != nil {
		t.Fatalf("CleanupOldDataByGatewayRetention: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deleted rows(total memory+disk), got %d", deleted)
	}

	var memCount int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&memCount); err != nil {
		t.Fatalf("count memory points: %v", err)
	}
	if memCount != 1 {
		t.Fatalf("expected 1 memory point remain, got %d", memCount)
	}

	var diskCount int
	if err := diskDB.QueryRow("SELECT COUNT(*) FROM data_points").Scan(&diskCount); err != nil {
		t.Fatalf("count disk points: %v", err)
	}
	if diskCount != 1 {
		t.Fatalf("expected 1 disk point remain, got %d", diskCount)
	}
}
