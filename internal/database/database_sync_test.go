package database

import (
	"path/filepath"
	"testing"
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

func TestCleanupOldDataByConfig(t *testing.T) {
	if ParamDB != nil {
		_ = ParamDB.Close()
	}
	if DataDB != nil {
		_ = DataDB.Close()
	}

	tmpDir := t.TempDir()
	paramDBFile = filepath.Join(tmpDir, "param.db")

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

	_, err = ParamDB.Exec(`CREATE TABLE devices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_key TEXT,
		device_key TEXT
	)`)
	if err != nil {
		t.Fatalf("create devices: %v", err)
	}
	_, err = ParamDB.Exec(`CREATE TABLE storage_policies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		product_key TEXT,
		device_key TEXT,
		storage_days INTEGER DEFAULT 30,
		enabled INTEGER DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create storage_policies: %v", err)
	}

	res, err := ParamDB.Exec(`INSERT INTO devices (product_key, device_key) VALUES (?, ?)`, "p1", "d1")
	if err != nil {
		t.Fatalf("insert device1: %v", err)
	}
	device1ID, _ := res.LastInsertId()
	res, err = ParamDB.Exec(`INSERT INTO devices (product_key, device_key) VALUES (?, ?)`, "p2", "d2")
	if err != nil {
		t.Fatalf("insert device2: %v", err)
	}
	device2ID, _ := res.LastInsertId()

	_, err = ParamDB.Exec(`INSERT INTO storage_policies (name, product_key, device_key, storage_days, enabled)
		VALUES (?, ?, ?, ?, ?)`, "device1", "p1", "d1", 1, 1)
	if err != nil {
		t.Fatalf("insert device policy: %v", err)
	}
	_, err = ParamDB.Exec(`INSERT INTO storage_policies (name, product_key, device_key, storage_days, enabled)
		VALUES (?, ?, ?, ?, ?)`, "global", "", "", 10, 1)
	if err != nil {
		t.Fatalf("insert global policy: %v", err)
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
		t.Fatalf("create data_points: %v", err)
	}

	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-2 days'))`, device1ID, "dev1", "temp", "20", "string")
	if err != nil {
		t.Fatalf("insert data point device1: %v", err)
	}
	_, err = DataDB.Exec(`INSERT INTO data_points (device_id, device_name, field_name, value, value_type, collected_at)
		VALUES (?, ?, ?, ?, ?, datetime('now', '-2 days'))`, device2ID, "dev2", "temp", "21", "string")
	if err != nil {
		t.Fatalf("insert data point device2: %v", err)
	}

	deleted, err := CleanupOldDataByConfig()
	if err != nil {
		t.Fatalf("CleanupOldDataByConfig: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted row, got %d", deleted)
	}

	var count1 int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ?", device1ID).Scan(&count1); err != nil {
		t.Fatalf("count device1 data points: %v", err)
	}
	if count1 != 0 {
		t.Fatalf("expected device1 data points to be deleted, got %d", count1)
	}

	var count2 int
	if err := DataDB.QueryRow("SELECT COUNT(*) FROM data_points WHERE device_id = ?", device2ID).Scan(&count2); err != nil {
		t.Fatalf("count device2 data points: %v", err)
	}
	if count2 != 1 {
		t.Fatalf("expected device2 data points to remain, got %d", count2)
	}
}
