package database

import (
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
