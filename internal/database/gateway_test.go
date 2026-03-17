package database

import (
	"errors"
	"path/filepath"
	"testing"
)

func setupGatewayTestDB(t *testing.T) {
	t.Helper()

	originalParamDB := ParamDB
	originalGatewayColumnsEnsured := gatewayColumnsEnsured
	t.Cleanup(func() {
		gatewayColumnsEnsured = originalGatewayColumnsEnsured
		if ParamDB != nil {
			_ = ParamDB.Close()
		}
		ParamDB = originalParamDB
	})

	if ParamDB != nil {
		_ = ParamDB.Close()
	}

	gatewayColumnsEnsured = false

	var err error
	ParamDB, err = openSQLite(filepath.Join(t.TempDir(), "param.db"), 1, 1)
	if err != nil {
		t.Fatalf("open param db: %v", err)
	}
}

type stubGatewayConfigScanner struct {
	values []any
	err    error
}

func (s stubGatewayConfigScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}

	for i := range dest {
		switch out := dest[i].(type) {
		case *int64:
			*out = s.values[i].(int64)
		case *string:
			*out = s.values[i].(string)
		case *int:
			*out = s.values[i].(int)
		}
	}

	return nil
}

func TestInitGatewayConfigTable_CreatesDefaultConfig(t *testing.T) {
	setupGatewayTestDB(t)

	if err := InitGatewayConfigTable(); err != nil {
		t.Fatalf("InitGatewayConfigTable returned error: %v", err)
	}

	cfg, err := GetGatewayConfig()
	if err != nil {
		t.Fatalf("GetGatewayConfig returned error: %v", err)
	}
	if cfg.GatewayName != defaultGatewayConfigName {
		t.Fatalf("cfg.GatewayName = %q, want %q", cfg.GatewayName, defaultGatewayConfigName)
	}
	if cfg.DataRetentionDays != DefaultRetentionDays {
		t.Fatalf("cfg.DataRetentionDays = %d, want %d", cfg.DataRetentionDays, DefaultRetentionDays)
	}
}

func TestNormalizeGatewayConfig(t *testing.T) {
	cfg := &GatewayConfig{GatewayName: " ", DataRetentionDays: 0}

	normalizeGatewayConfig(cfg)

	if cfg.GatewayName != defaultGatewayConfigName {
		t.Fatalf("cfg.GatewayName = %q, want %q", cfg.GatewayName, defaultGatewayConfigName)
	}
	if cfg.DataRetentionDays != DefaultRetentionDays {
		t.Fatalf("cfg.DataRetentionDays = %d, want %d", cfg.DataRetentionDays, DefaultRetentionDays)
	}
}

func TestScanGatewayConfig(t *testing.T) {
	cfg := &GatewayConfig{}
	scanner := stubGatewayConfigScanner{
		values: []any{int64(1), "pk", "dk", "gw", 15, "2026-03-18 10:00:00"},
	}

	err := scanGatewayConfig(scanner, cfg)
	if err != nil {
		t.Fatalf("scanGatewayConfig returned error: %v", err)
	}
	if cfg.ID != 1 || cfg.ProductKey != "pk" || cfg.DeviceKey != "dk" {
		t.Fatalf("unexpected gateway identity: %+v", cfg)
	}
	if cfg.GatewayName != "gw" || cfg.DataRetentionDays != 15 {
		t.Fatalf("unexpected gateway config: %+v", cfg)
	}
}

func TestScanGatewayConfig_Error(t *testing.T) {
	err := scanGatewayConfig(stubGatewayConfigScanner{err: errors.New("scan failed")}, &GatewayConfig{})
	if err == nil {
		t.Fatal("expected scanGatewayConfig error")
	}
}

func TestResolveTargetGatewayConfigID_UsesCurrentConfigWhenEmpty(t *testing.T) {
	setupGatewayTestDB(t)

	if err := InitGatewayConfigTable(); err != nil {
		t.Fatalf("InitGatewayConfigTable returned error: %v", err)
	}

	cfg, err := GetGatewayConfig()
	if err != nil {
		t.Fatalf("GetGatewayConfig returned error: %v", err)
	}

	got, err := resolveTargetGatewayConfigID(0)
	if err != nil {
		t.Fatalf("resolveTargetGatewayConfigID returned error: %v", err)
	}
	if got != cfg.ID {
		t.Fatalf("resolveTargetGatewayConfigID() = %d, want %d", got, cfg.ID)
	}
}
