package database

import (
	"path/filepath"
	"testing"
)

func TestRuntimeConfigAuditCRUD(t *testing.T) {
	if ParamDB != nil {
		_ = ParamDB.Close()
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

	if err := InitRuntimeConfigAuditTable(); err != nil {
		t.Fatalf("InitRuntimeConfigAuditTable: %v", err)
	}

	id, err := CreateRuntimeConfigAudit(&RuntimeConfigAudit{
		OperatorUserID:   7,
		OperatorUsername: "admin",
		SourceIP:         "127.0.0.1",
		Changes:          `{"x":{"from":"1s","to":"2s"}}`,
	})
	if err != nil {
		t.Fatalf("CreateRuntimeConfigAudit: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected id > 0")
	}

	items, err := ListRuntimeConfigAudits(10)
	if err != nil {
		t.Fatalf("ListRuntimeConfigAudits: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one audit item")
	}
	if items[0].OperatorUsername != "admin" {
		t.Fatalf("unexpected operator username: %s", items[0].OperatorUsername)
	}
}
