package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func setupAlarmLogsTestDB(t *testing.T) {
	t.Helper()

	originalParamDB := ParamDB
	t.Cleanup(func() {
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

	_, err = ParamDB.Exec(`CREATE TABLE alarm_logs (
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
		t.Fatalf("create alarm_logs table: %v", err)
	}
}

func insertAlarmLogRow(t *testing.T, acknowledgedBy any, acknowledgedAt any) {
	t.Helper()

	_, err := ParamDB.Exec(
		`INSERT INTO alarm_logs (device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, acknowledged, acknowledged_by, acknowledged_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		1, nil, "temperature", 101.1, 99.9, ">", "high", "too hot", 0, acknowledgedBy, acknowledgedAt,
	)
	if err != nil {
		t.Fatalf("insert alarm log: %v", err)
	}
}

func TestGetRecentAlarmLogs_AllowsNullAcknowledgedBy(t *testing.T) {
	setupAlarmLogsTestDB(t)
	insertAlarmLogRow(t, nil, nil)

	logs, err := GetRecentAlarmLogs(10)
	if err != nil {
		t.Fatalf("GetRecentAlarmLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 alarm log, got %d", len(logs))
	}
	if logs[0].AcknowledgedBy != "" {
		t.Fatalf("expected empty acknowledged_by when NULL, got %q", logs[0].AcknowledgedBy)
	}
	if logs[0].AcknowledgedAt != nil {
		t.Fatalf("expected nil acknowledged_at, got %v", logs[0].AcknowledgedAt)
	}
}

func TestGetAlarmLogsByDeviceID_AllowsNullAcknowledgedBy(t *testing.T) {
	setupAlarmLogsTestDB(t)
	insertAlarmLogRow(t, sql.NullString{String: "", Valid: false}, nil)

	logs, err := GetAlarmLogsByDeviceID(1, 10)
	if err != nil {
		t.Fatalf("GetAlarmLogsByDeviceID: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 alarm log, got %d", len(logs))
	}
	if logs[0].AcknowledgedBy != "" {
		t.Fatalf("expected empty acknowledged_by when NULL, got %q", logs[0].AcknowledgedBy)
	}
}
