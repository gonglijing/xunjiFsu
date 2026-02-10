package database

import (
	"path/filepath"
	"testing"
)

func setupResourceTestDB(t *testing.T, createSQL string) {
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

	if _, err := ParamDB.Exec(createSQL); err != nil {
		t.Fatalf("create resources table: %v", err)
	}
}

func TestInitResourceTable_CleansLegacyColumns(t *testing.T) {
	setupResourceTestDB(t, `CREATE TABLE resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		type TEXT NOT NULL CHECK(type IN ('serial', 'di', 'do')),
		port TEXT,
		address INTEGER DEFAULT 1,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	if _, err := ParamDB.Exec(`INSERT INTO resources (name, type, port, address, enabled) VALUES (?, ?, ?, ?, ?)`,
		"COM1", "serial", "/dev/ttyUSB0", 1, 1,
	); err != nil {
		t.Fatalf("insert legacy resource: %v", err)
	}

	if err := InitResourceTable(); err != nil {
		t.Fatalf("InitResourceTable: %v", err)
	}

	hasPort, err := columnExists(ParamDB, "resources", "port")
	if err != nil {
		t.Fatalf("columnExists port: %v", err)
	}
	hasAddress, err := columnExists(ParamDB, "resources", "address")
	if err != nil {
		t.Fatalf("columnExists address: %v", err)
	}
	hasPath, err := columnExists(ParamDB, "resources", "path")
	if err != nil {
		t.Fatalf("columnExists path: %v", err)
	}

	if hasPort {
		t.Fatalf("expected legacy column port to be removed")
	}
	if hasAddress {
		t.Fatalf("expected legacy column address to be removed")
	}
	if !hasPath {
		t.Fatalf("expected path column to exist")
	}

	var gotPath string
	if err := ParamDB.QueryRow(`SELECT path FROM resources WHERE name = ?`, "COM1").Scan(&gotPath); err != nil {
		t.Fatalf("query path: %v", err)
	}
	if gotPath != "/dev/ttyUSB0" {
		t.Fatalf("path=%q, want %q", gotPath, "/dev/ttyUSB0")
	}
}

func TestInitResourceTable_NormalizesLegacyTypeWhenCleaning(t *testing.T) {
	setupResourceTestDB(t, `CREATE TABLE resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		type TEXT NOT NULL CHECK(type IN ('serial', 'net', 'network', 'tcp', 'di', 'do')),
		port TEXT,
		address INTEGER DEFAULT 1,
		path TEXT,
		enabled INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	if _, err := ParamDB.Exec(`INSERT INTO resources (name, type, path, enabled) VALUES (?, ?, ?, ?)`,
		"NET1", "tcp", "192.168.1.10:502", 1,
	); err != nil {
		t.Fatalf("insert legacy net resource: %v", err)
	}

	if err := InitResourceTable(); err != nil {
		t.Fatalf("InitResourceTable: %v", err)
	}

	var gotType string
	if err := ParamDB.QueryRow(`SELECT type FROM resources WHERE name = ?`, "NET1").Scan(&gotType); err != nil {
		t.Fatalf("query type: %v", err)
	}
	if gotType != "net" {
		t.Fatalf("type=%q, want %q", gotType, "net")
	}
}
