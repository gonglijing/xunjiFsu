package database

import (
	"path/filepath"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func setupThresholdsTestDB(t *testing.T) {
	t.Helper()
	thresholdColumnsEnsureState.mu.Lock()
	thresholdColumnsEnsureState.ensuredDB = nil
	thresholdColumnsEnsureState.mu.Unlock()
	gatewayAlarmRepeatEnsureState.mu.Lock()
	gatewayAlarmRepeatEnsureState.ensuredDB = nil
	gatewayAlarmRepeatEnsureState.mu.Unlock()

	originalParamDB := ParamDB
	t.Cleanup(func() {
		thresholdColumnsEnsureState.mu.Lock()
		thresholdColumnsEnsureState.ensuredDB = nil
		thresholdColumnsEnsureState.mu.Unlock()
		gatewayAlarmRepeatEnsureState.mu.Lock()
		gatewayAlarmRepeatEnsureState.ensuredDB = nil
		gatewayAlarmRepeatEnsureState.mu.Unlock()
		if ParamDB != nil {
			_ = ParamDB.Close()
		}
		ParamDB = originalParamDB
	})

	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	tmpDir := t.TempDir()
	var err error
	ParamDB, err = openSQLite(filepath.Join(tmpDir, "param.db"), 1, 1)
	if err != nil {
		t.Fatalf("open param db: %v", err)
	}

	_, err = ParamDB.Exec(`CREATE TABLE thresholds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		operator TEXT NOT NULL,
		value REAL NOT NULL,
		severity TEXT DEFAULT 'warning',
		enabled INTEGER DEFAULT 1,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create thresholds table: %v", err)
	}

	_, err = ParamDB.Exec(`CREATE TABLE gateway_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_key TEXT NOT NULL,
		device_key TEXT NOT NULL,
		gateway_name TEXT DEFAULT 'HuShu智能网关',
		data_retention_days INTEGER DEFAULT 30,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create gateway_config table: %v", err)
	}
	_, err = ParamDB.Exec(`INSERT INTO gateway_config (product_key, device_key, gateway_name, data_retention_days) VALUES ('', '', 'gw', 30)`)
	if err != nil {
		t.Fatalf("insert gateway config row: %v", err)
	}
}

func TestEnsureThresholdColumns_RecheckAfterDBSwitch(t *testing.T) {
	setupThresholdsTestDB(t)

	if err := ensureThresholdColumns(); err != nil {
		t.Fatalf("ensureThresholdColumns first call: %v", err)
	}

	if err := ensureThresholdColumns(); err != nil {
		t.Fatalf("ensureThresholdColumns second call(idempotent): %v", err)
	}

	tmpDir := t.TempDir()
	newDB, err := openSQLite(filepath.Join(tmpDir, "param_new.db"), 1, 1)
	if err != nil {
		t.Fatalf("open new param db: %v", err)
	}
	defer func() { _ = newDB.Close() }()

	_, err = newDB.Exec(`CREATE TABLE thresholds (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_id INTEGER NOT NULL,
		field_name TEXT NOT NULL,
		operator TEXT NOT NULL,
		value REAL NOT NULL,
		severity TEXT DEFAULT 'warning',
		enabled INTEGER DEFAULT 1,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create thresholds in new db: %v", err)
	}

	original := ParamDB
	ParamDB = newDB
	defer func() { ParamDB = original }()

	if err := ensureThresholdColumns(); err != nil {
		t.Fatalf("ensureThresholdColumns after db switch: %v", err)
	}

	hasEnabled, err := columnExists(ParamDB, "thresholds", "enabled")
	if err != nil {
		t.Fatalf("columnExists enabled after switch: %v", err)
	}
	if hasEnabled {
		t.Fatalf("enabled column should be dropped after db switch")
	}
}

func TestThresholdCRUD_WithShieldedColumn(t *testing.T) {
	setupThresholdsTestDB(t)

	id, err := CreateThreshold(&models.Threshold{
		DeviceID:  1,
		FieldName: "temperature",
		Operator:  ">",
		Value:     40,
		Severity:  "warning",
		Shielded:  1,
		Message:   "too hot",
	})
	if err != nil {
		t.Fatalf("CreateThreshold: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected id > 0")
	}

	threshold, err := GetThresholdByID(id)
	if err != nil {
		t.Fatalf("GetThresholdByID: %v", err)
	}
	if threshold.Shielded != 1 {
		t.Fatalf("expected shielded=1, got %d", threshold.Shielded)
	}

	threshold.Shielded = 0
	if err := UpdateThreshold(threshold); err != nil {
		t.Fatalf("UpdateThreshold: %v", err)
	}

	updated, err := GetThresholdByID(id)
	if err != nil {
		t.Fatalf("GetThresholdByID(updated): %v", err)
	}
	if updated.Shielded != 0 {
		t.Fatalf("expected shielded=0 after update, got %d", updated.Shielded)
	}
}

func TestAlarmRepeatIntervalSettings(t *testing.T) {
	setupThresholdsTestDB(t)

	seconds, err := GetAlarmRepeatIntervalSeconds()
	if err != nil {
		t.Fatalf("GetAlarmRepeatIntervalSeconds: %v", err)
	}
	if seconds != DefaultAlarmRepeatIntervalSeconds {
		t.Fatalf("expected default repeat interval %d, got %d", DefaultAlarmRepeatIntervalSeconds, seconds)
	}

	if err := UpdateAlarmRepeatIntervalSeconds(180); err != nil {
		t.Fatalf("UpdateAlarmRepeatIntervalSeconds: %v", err)
	}
	seconds, err = GetAlarmRepeatIntervalSeconds()
	if err != nil {
		t.Fatalf("GetAlarmRepeatIntervalSeconds(after update): %v", err)
	}
	if seconds != 180 {
		t.Fatalf("expected repeat interval 180, got %d", seconds)
	}

	if err := UpdateAlarmRepeatIntervalSeconds(0); err == nil {
		t.Fatalf("expected error when setting non-positive repeat interval")
	}
}
