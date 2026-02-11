package database

import (
	"os"
	"path/filepath"
	"testing"
)

func switchToRepoRoot(t *testing.T) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	candidates := []string{
		".",
		"..",
		"../..",
		"../../..",
	}

	for _, candidate := range candidates {
		migrationPath := filepath.Join(candidate, "migrations", "002_param_schema.sql")
		if _, err := os.Stat(migrationPath); err == nil {
			if err := os.Chdir(candidate); err != nil {
				t.Fatalf("chdir to %s: %v", candidate, err)
			}
			t.Cleanup(func() {
				_ = os.Chdir(originalWD)
			})
			return
		}
	}

	t.Fatalf("could not locate repository root with migrations directory from %s", originalWD)
}

func TestInitParamSchema_DropsUnusedStoragePoliciesTable(t *testing.T) {
	switchToRepoRoot(t)

	oldParam := ParamDB
	t.Cleanup(func() {
		if ParamDB != nil {
			_ = ParamDB.Close()
		}
		ParamDB = oldParam
	})

	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	tmp := t.TempDir()
	var err error
	ParamDB, err = openSQLite(filepath.Join(tmp, "param.db"), 1, 1)
	if err != nil {
		t.Fatalf("open param db: %v", err)
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

	if err := InitParamSchema(); err != nil {
		t.Fatalf("InitParamSchema: %v", err)
	}

	var count int
	if err := ParamDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='storage_policies'`).Scan(&count); err != nil {
		t.Fatalf("check storage_policies existence: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected storage_policies to be dropped, still exists")
	}
}

func TestInitDataSchema_DropsUnusedStorageConfigTable(t *testing.T) {
	switchToRepoRoot(t)

	oldData := DataDB
	oldParam := ParamDB
	t.Cleanup(func() {
		if DataDB != nil {
			_ = DataDB.Close()
		}
		DataDB = oldData
		if ParamDB != nil {
			_ = ParamDB.Close()
		}
		ParamDB = oldParam
	})

	if DataDB != nil {
		_ = DataDB.Close()
	}
	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}

	_, err = DataDB.Exec(`CREATE TABLE storage_config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		product_key TEXT NOT NULL,
		device_key TEXT NOT NULL,
		storage_days INTEGER DEFAULT 30,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create storage_config: %v", err)
	}

	ParamDB, err = openSQLite(filepath.Join(t.TempDir(), "param.db"), 1, 1)
	if err != nil {
		t.Fatalf("open temp param db: %v", err)
	}

	if err := InitDataSchema(); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	var count int
	if err := DataDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='storage_config'`).Scan(&count); err != nil {
		t.Fatalf("check storage_config existence: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected storage_config to be dropped, still exists")
	}
}

func TestInitDataSchema_CreatesAlarmLogIndexesOnParamDB(t *testing.T) {
	switchToRepoRoot(t)

	oldData := DataDB
	oldParam := ParamDB
	t.Cleanup(func() {
		if DataDB != nil {
			_ = DataDB.Close()
		}
		DataDB = oldData
		if ParamDB != nil {
			_ = ParamDB.Close()
		}
		ParamDB = oldParam
	})

	if DataDB != nil {
		_ = DataDB.Close()
	}
	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	var err error
	DataDB, err = openSQLite(":memory:", 1, 1)
	if err != nil {
		t.Fatalf("open data db: %v", err)
	}

	ParamDB, err = openSQLite(filepath.Join(t.TempDir(), "param.db"), 1, 1)
	if err != nil {
		t.Fatalf("open param db: %v", err)
	}

	if err := InitParamSchema(); err != nil {
		t.Fatalf("InitParamSchema: %v", err)
	}
	if err := InitDataSchema(); err != nil {
		t.Fatalf("InitDataSchema: %v", err)
	}

	var count int
	if err := ParamDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_alarm_logs_device_time'`).Scan(&count); err != nil {
		t.Fatalf("check idx_alarm_logs_device_time: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected idx_alarm_logs_device_time on param db, got %d", count)
	}

	if err := ParamDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_alarm_logs_unacked'`).Scan(&count); err != nil {
		t.Fatalf("check idx_alarm_logs_unacked: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected idx_alarm_logs_unacked on param db, got %d", count)
	}
}
