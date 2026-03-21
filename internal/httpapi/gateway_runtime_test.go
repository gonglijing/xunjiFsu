package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func TestBuildRuntimeAuditViews(t *testing.T) {
	rawJSON, _ := json.Marshal(map[string]service.RuntimeConfigChange{
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

func TestRuntimeConfigAuditActor(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/gateway/runtime", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	req.Header.Set("X-Forwarded-For", "10.0.0.8")

	actor := runtimeConfigAuditActor(req)
	if actor != (service.RuntimeConfigActor{Username: "unknown", SourceIP: "10.0.0.8"}) {
		t.Fatalf("actor = %#v", actor)
	}
}

func TestParseGatewayConfigRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/gateway/config", strings.NewReader(`{"gateway_name":" ","data_retention_days":0}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	cfg, ok := parseGatewayConfigRequest(w, req)
	if !ok {
		t.Fatal("expected ok=true, got false")
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.GatewayName != service.DefaultGatewayName {
		t.Fatalf("cfg.GatewayName = %q, want %q", cfg.GatewayName, service.DefaultGatewayName)
	}
	if cfg.DataRetentionDays != database.DefaultRetentionDays {
		t.Fatalf("cfg.DataRetentionDays = %d, want %d", cfg.DataRetentionDays, database.DefaultRetentionDays)
	}
}
