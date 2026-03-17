package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/database"
)

func TestParseOptionalDuration(t *testing.T) {
	if _, ok, err := parseOptionalDuration(""); err != nil || ok {
		t.Fatalf("empty should return (ok=false, err=nil), got ok=%v err=%v", ok, err)
	}

	if got, ok, err := parseOptionalDuration("150ms"); err != nil || !ok || got != 150*time.Millisecond {
		t.Fatalf("parse 150ms failed, got (%v, %v, %v)", got, ok, err)
	}

	if _, _, err := parseOptionalDuration("bad"); err == nil {
		t.Fatalf("expected parse error for invalid duration")
	}

	if _, _, err := parseOptionalDuration("0s"); err == nil {
		t.Fatalf("expected parse error for non-positive duration")
	}
}

func TestApplyGatewayRuntimeConfig_NegativeRetries(t *testing.T) {
	negative := -1
	h := &Handler{}
	h.appConfig = &config.Config{}

	_, err := h.applyGatewayRuntimeConfig(&gatewayRuntimeConfig{DriverSerialOpenRetries: &negative})
	if err == nil {
		t.Fatalf("expected error for negative driver_serial_open_retries")
	}

	_, err = h.applyGatewayRuntimeConfig(&gatewayRuntimeConfig{DriverTCPDialRetries: &negative})
	if err == nil {
		t.Fatalf("expected error for negative driver_tcp_dial_retries")
	}
}

func TestApplyGatewayDurationConfigChange(t *testing.T) {
	changes := make(map[string]runtimeConfigChange)
	target := 5 * time.Second

	if err := applyGatewayDurationConfigChange(changes, "driver_tcp_read_timeout", "8s", &target); err != nil {
		t.Fatalf("applyGatewayDurationConfigChange returned error: %v", err)
	}
	if target != 8*time.Second {
		t.Fatalf("target = %v, want 8s", target)
	}
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}
}

func TestApplyGatewayRetryConfigChange(t *testing.T) {
	changes := make(map[string]runtimeConfigChange)
	target := 1
	value := 3

	if err := applyGatewayRetryConfigChange(changes, "driver_tcp_dial_retries", &value, &target); err != nil {
		t.Fatalf("applyGatewayRetryConfigChange returned error: %v", err)
	}
	if target != 3 {
		t.Fatalf("target = %d, want 3", target)
	}

	negative := -1
	if err := applyGatewayRetryConfigChange(changes, "driver_tcp_dial_retries", &negative, &target); err == nil {
		t.Fatal("expected negative retry error")
	}
}

func TestRecordRuntimeConfigChange(t *testing.T) {
	changes := make(map[string]runtimeConfigChange)

	recordRuntimeConfigChange(changes, "x", "1s", "1s")
	if len(changes) != 0 {
		t.Fatalf("expected same from/to not recorded")
	}

	recordRuntimeConfigChange(changes, "x", "1s", "2s")
	if len(changes) != 1 {
		t.Fatalf("expected one change recorded")
	}
}

func TestBuildRuntimeAuditViews(t *testing.T) {
	rawJSON, _ := json.Marshal(map[string]runtimeConfigChange{
		"driver_tcp_dial_retries": {From: 0, To: 2},
	})

	views := buildRuntimeAuditViews([]*database.RuntimeConfigAudit{
		{
			ID:               1,
			OperatorUsername: "admin",
			Changes:          string(rawJSON),
		},
		{
			ID:      2,
			Changes: "not-json",
		},
	})

	if len(views) != 2 {
		t.Fatalf("expected 2 views, got %d", len(views))
	}
	if views[0].Changes == nil || len(views[0].Changes) != 1 {
		t.Fatalf("expected parsed structured changes")
	}
	if views[1].ChangesRaw == "" {
		t.Fatalf("expected raw changes fallback for invalid json")
	}
}

func TestParseGatewayConfigPayload(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/gateway/config", strings.NewReader(`{"gateway_name":" ","data_retention_days":0}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cfg, ok := parseGatewayConfigPayload(w, req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.GatewayName != defaultGatewayName {
		t.Fatalf("cfg.GatewayName = %q, want %q", cfg.GatewayName, defaultGatewayName)
	}
	if cfg.DataRetentionDays != database.DefaultRetentionDays {
		t.Fatalf("cfg.DataRetentionDays = %d, want %d", cfg.DataRetentionDays, database.DefaultRetentionDays)
	}
}
