package collector

import (
	"database/sql"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func setupCollectorAlarmBehaviorTestDB(t *testing.T) *sql.DB {
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
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		operator TEXT NOT NULL,
		value REAL NOT NULL,
		severity TEXT DEFAULT 'warning',
		enabled INTEGER DEFAULT 1,
		shielded INTEGER DEFAULT 0,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create thresholds table failed: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE alarm_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER,
		threshold_id INTEGER,
		field_name TEXT,
		actual_value REAL,
		threshold_value REAL,
		operator TEXT,
		severity TEXT,
		message TEXT,
		triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		acknowledged INTEGER DEFAULT 0,
		acknowledged_by TEXT,
		acknowledged_at TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create alarm_logs table failed: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE gateway_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_key TEXT NOT NULL,
		device_key TEXT NOT NULL,
		gateway_name TEXT DEFAULT 'gw',
		data_retention_days INTEGER DEFAULT 30,
		alarm_repeat_interval_seconds INTEGER DEFAULT 60,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create gateway_config table failed: %v", err)
	}

	_, err = db.Exec(`INSERT INTO gateway_config (product_key, device_key, gateway_name, data_retention_days, alarm_repeat_interval_seconds)
		VALUES ('', '', 'gw', 30, 60)`)
	if err != nil {
		t.Fatalf("insert gateway_config row failed: %v", err)
	}

	_, err = db.Exec(`INSERT INTO devices (
		id, name, description, product_key, device_key, driver_type, serial_port, baud_rate, data_bits, stop_bits, parity,
		ip_address, port_num, device_address, collect_interval, storage_interval, timeout, driver_id, enabled, resource_id
	) VALUES (1, 'd1', '', '', '', 'modbus_rtu', '', 9600, 8, 1, 'N', '', 0, '1', 1000, 300, 1000, NULL, 1, NULL)`)
	if err != nil {
		t.Fatalf("insert device failed: %v", err)
	}

	return db
}

func countAlarmLogsFromDB(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM alarm_logs`).Scan(&count); err != nil {
		t.Fatalf("count alarm logs: %v", err)
	}
	return count
}

func TestCheckThresholds_ShieldAndRepeatInterval(t *testing.T) {
	oldDB := database.ParamDB
	db := setupCollectorAlarmBehaviorTestDB(t)
	database.ParamDB = db
	t.Cleanup(func() {
		database.ParamDB = oldDB
		_ = db.Close()
	})

	InvalidateAllCache()
	clearAlarmStateForDevice(1)
	InvalidateAlarmRepeatIntervalCache()

	_, err := database.CreateThreshold(&models.Threshold{
		DeviceID:  1,
		FieldName: "temp",
		Operator:  ">",
		Value:     30,
		Severity:  "warning",
		Enabled:   1,
		Shielded:  1,
		Message:   "high temp",
	})
	if err != nil {
		t.Fatalf("CreateThreshold: %v", err)
	}

	collector := NewCollector(nil, nil)
	device := &models.Device{ID: 1, Name: "d1"}
	data := &models.CollectData{DeviceID: 1, Fields: map[string]string{"temp": "35"}}

	if err := collector.checkThresholds(device, data); err != nil {
		t.Fatalf("checkThresholds shielded: %v", err)
	}
	if got := countAlarmLogsFromDB(t, db); got != 0 {
		t.Fatalf("expected 0 alarms when threshold shielded, got %d", got)
	}

	all, err := database.GetAllThresholds()
	if err != nil || len(all) != 1 {
		t.Fatalf("GetAllThresholds err=%v len=%d", err, len(all))
	}
	all[0].Shielded = 0
	if err := database.UpdateThreshold(all[0]); err != nil {
		t.Fatalf("UpdateThreshold(unshield): %v", err)
	}
	if err := database.UpdateAlarmRepeatIntervalSeconds(3600); err != nil {
		t.Fatalf("UpdateAlarmRepeatIntervalSeconds: %v", err)
	}

	InvalidateDeviceCache(1)
	InvalidateAlarmRepeatIntervalCache()

	if err := collector.checkThresholds(device, data); err != nil {
		t.Fatalf("checkThresholds first emit: %v", err)
	}
	if err := collector.checkThresholds(device, data); err != nil {
		t.Fatalf("checkThresholds second emit: %v", err)
	}

	if got := countAlarmLogsFromDB(t, db); got != 1 {
		t.Fatalf("expected 1 alarm with repeat suppression, got %d", got)
	}
}
