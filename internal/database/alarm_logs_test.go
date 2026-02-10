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

func insertAlarmLogRow(t *testing.T, acknowledgedBy any, acknowledgedAt any) int64 {
	t.Helper()

	result, err := ParamDB.Exec(
		`INSERT INTO alarm_logs (device_id, threshold_id, field_name, actual_value, threshold_value, operator, severity, message, acknowledged, acknowledged_by, acknowledged_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		1, nil, "temperature", 101.1, 99.9, ">", "high", "too hot", 0, acknowledgedBy, acknowledgedAt,
	)
	if err != nil {
		t.Fatalf("insert alarm log: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func countAlarmLogs(t *testing.T) int {
	t.Helper()
	var count int
	if err := ParamDB.QueryRow("SELECT COUNT(*) FROM alarm_logs").Scan(&count); err != nil {
		t.Fatalf("count alarm logs: %v", err)
	}
	return count
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

func TestDeleteAlarmLog(t *testing.T) {
	setupAlarmLogsTestDB(t)
	id := insertAlarmLogRow(t, nil, nil)

	if err := DeleteAlarmLog(id); err != nil {
		t.Fatalf("DeleteAlarmLog: %v", err)
	}
	if got := countAlarmLogs(t); got != 0 {
		t.Fatalf("expected 0 logs after deleting one, got %d", got)
	}
}

func TestDeleteAlarmLogsByIDs(t *testing.T) {
	setupAlarmLogsTestDB(t)
	id1 := insertAlarmLogRow(t, nil, nil)
	_ = insertAlarmLogRow(t, nil, nil)
	id3 := insertAlarmLogRow(t, nil, nil)

	deleted, err := DeleteAlarmLogsByIDs([]int64{id1, id3})
	if err != nil {
		t.Fatalf("DeleteAlarmLogsByIDs: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected deleted=2, got %d", deleted)
	}
	if got := countAlarmLogs(t); got != 1 {
		t.Fatalf("expected 1 log remaining after batch delete, got %d", got)
	}
}

func TestClearAlarmLogs(t *testing.T) {
	setupAlarmLogsTestDB(t)
	insertAlarmLogRow(t, nil, nil)
	insertAlarmLogRow(t, nil, nil)

	deleted, err := ClearAlarmLogs()
	if err != nil {
		t.Fatalf("ClearAlarmLogs: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected deleted=2 when clearing, got %d", deleted)
	}
	if got := countAlarmLogs(t); got != 0 {
		t.Fatalf("expected 0 logs after clear, got %d", got)
	}
}
