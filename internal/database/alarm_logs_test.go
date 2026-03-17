package database

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
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

type stubAlarmLogScanner struct {
	values []any
	err    error
}

func (s stubAlarmLogScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}

	for i := range dest {
		switch out := dest[i].(type) {
		case *int64:
			*out = s.values[i].(int64)
		case **int64:
			if s.values[i] == nil {
				*out = nil
				continue
			}
			value := s.values[i].(int64)
			*out = &value
		case *string:
			*out = s.values[i].(string)
		case *float64:
			*out = s.values[i].(float64)
		case *int:
			*out = s.values[i].(int)
		case *time.Time:
			*out = s.values[i].(time.Time)
		case **time.Time:
			if s.values[i] == nil {
				*out = nil
				continue
			}
			value := s.values[i].(time.Time)
			*out = &value
		}
	}

	return nil
}

func TestScanAlarmLog(t *testing.T) {
	now := time.Now()
	thresholdID := int64(7)
	acknowledgedAt := now.Add(time.Minute)
	log := &models.AlarmLog{}
	scanner := stubAlarmLogScanner{
		values: []any{
			int64(1), int64(2), thresholdID, "temperature", 101.1, 99.9, ">", "high", "too hot",
			now, 1, "admin", acknowledgedAt,
		},
	}

	err := scanAlarmLog(scanner, log)
	if err != nil {
		t.Fatalf("scanAlarmLog returned error: %v", err)
	}
	if log.ID != 1 || log.DeviceID != 2 || log.FieldName != "temperature" {
		t.Fatalf("unexpected alarm log core fields: %+v", log)
	}
	if log.ThresholdID == nil || *log.ThresholdID != thresholdID {
		t.Fatalf("unexpected threshold id: %+v", log.ThresholdID)
	}
	if log.AcknowledgedAt == nil || !log.AcknowledgedAt.Equal(acknowledgedAt) {
		t.Fatalf("unexpected acknowledged_at: %+v", log.AcknowledgedAt)
	}
}

func TestScanAlarmLog_AllowsNilOptionalFields(t *testing.T) {
	now := time.Now()
	log := &models.AlarmLog{}
	scanner := stubAlarmLogScanner{
		values: []any{
			int64(1), int64(2), nil, "temperature", 101.1, 99.9, ">", "high", "too hot",
			now, 0, "", nil,
		},
	}

	err := scanAlarmLog(scanner, log)
	if err != nil {
		t.Fatalf("scanAlarmLog returned error: %v", err)
	}
	if log.ThresholdID != nil || log.AcknowledgedAt != nil {
		t.Fatalf("expected nil optional fields, got threshold_id=%v acknowledged_at=%v", log.ThresholdID, log.AcknowledgedAt)
	}
}

func TestScanAlarmLog_Error(t *testing.T) {
	err := scanAlarmLog(stubAlarmLogScanner{err: errors.New("scan failed")}, &models.AlarmLog{})
	if err == nil {
		t.Fatal("expected scanAlarmLog error")
	}
}
