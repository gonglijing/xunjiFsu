package handlers

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNormalizeGatewayConfigInput_DefaultsNameAndRetention(t *testing.T) {
	cfg := &models.GatewayConfig{
		GatewayName:       "   ",
		DataRetentionDays: 0,
	}

	normalizeGatewayConfigInput(cfg)

	if cfg.GatewayName != defaultGatewayName {
		t.Fatalf("GatewayName = %q, want %q", cfg.GatewayName, defaultGatewayName)
	}
	if cfg.DataRetentionDays != database.DefaultRetentionDays {
		t.Fatalf("DataRetentionDays = %d, want %d", cfg.DataRetentionDays, database.DefaultRetentionDays)
	}
}

func TestNormalizeGatewayConfigInput_PreservesExplicitValues(t *testing.T) {
	cfg := &models.GatewayConfig{
		GatewayName:       "  Demo Gateway  ",
		DataRetentionDays: 15,
	}

	normalizeGatewayConfigInput(cfg)

	if cfg.GatewayName != "Demo Gateway" {
		t.Fatalf("GatewayName = %q, want %q", cfg.GatewayName, "Demo Gateway")
	}
	if cfg.DataRetentionDays != 15 {
		t.Fatalf("DataRetentionDays = %d, want 15", cfg.DataRetentionDays)
	}
}

func TestBuildDatabaseGatewayConfig(t *testing.T) {
	cfg := &models.GatewayConfig{
		ID:                7,
		GatewayName:       "Demo Gateway",
		DataRetentionDays: 21,
	}

	got := buildDatabaseGatewayConfig(cfg)
	if got == nil {
		t.Fatal("buildDatabaseGatewayConfig() = nil, want non-nil")
	}
	if got.ID != 7 {
		t.Fatalf("ID = %d, want 7", got.ID)
	}
	if got.GatewayName != "Demo Gateway" {
		t.Fatalf("GatewayName = %q, want %q", got.GatewayName, "Demo Gateway")
	}
	if got.DataRetentionDays != 21 {
		t.Fatalf("DataRetentionDays = %d, want 21", got.DataRetentionDays)
	}
}

func TestBuildDatabaseGatewayConfig_Nil(t *testing.T) {
	if got := buildDatabaseGatewayConfig(nil); got != nil {
		t.Fatalf("buildDatabaseGatewayConfig(nil) = %#v, want nil", got)
	}
}
