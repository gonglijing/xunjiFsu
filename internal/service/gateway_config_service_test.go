package service

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNormalizeGatewayConfigInput_DefaultsNameAndRetention(t *testing.T) {
	cfg := &models.GatewayConfig{
		GatewayName:       "   ",
		DataRetentionDays: 0,
	}

	NormalizeGatewayConfigInput(cfg)

	if cfg.GatewayName != DefaultGatewayName {
		t.Fatalf("GatewayName = %q, want %q", cfg.GatewayName, DefaultGatewayName)
	}
}

func TestNormalizeGatewayConfigInput_PreservesExplicitValues(t *testing.T) {
	cfg := &models.GatewayConfig{
		GatewayName:       "  Demo Gateway  ",
		DataRetentionDays: 15,
	}

	NormalizeGatewayConfigInput(cfg)

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

	got := BuildDatabaseGatewayConfig(cfg)
	if got == nil {
		t.Fatal("BuildDatabaseGatewayConfig() = nil, want non-nil")
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
	if got := BuildDatabaseGatewayConfig(nil); got != nil {
		t.Fatalf("BuildDatabaseGatewayConfig(nil) = %#v, want nil", got)
	}
}
