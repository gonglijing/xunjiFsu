package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestBuildHealthStatus_UsesRuntimeSnapshot(t *testing.T) {
	originalStartTime := appStartTime
	appStartTime = time.Date(2026, 3, 18, 11, 59, 30, 0, time.UTC)
	defer func() { appStartTime = originalStartTime }()

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := buildHealthStatus(now)

	if !status.Timestamp.Equal(now) {
		t.Fatalf("Timestamp = %#v, want %#v", status.Timestamp, now)
	}
	if status.Uptime != "30s" {
		t.Fatalf("Uptime = %q, want %q", status.Uptime, "30s")
	}
	if status.Checks == nil {
		t.Fatal("Checks = nil, want initialized map")
	}
	if status.System.GoVersion == "" {
		t.Fatal("GoVersion = empty, want runtime version")
	}
	if status.System.Goroutines <= 0 {
		t.Fatalf("Goroutines = %d, want positive", status.System.Goroutines)
	}
}

func TestResolveOverallHealthStatus_PreservesExistingStatus(t *testing.T) {
	checks := map[string]Check{
		"database": {Status: "fail", Message: "db down"},
	}
	if got := resolveOverallHealthStatus(checks, "degraded"); got != "degraded" {
		t.Fatalf("resolveOverallHealthStatus() = %q, want degraded", got)
	}
}

func TestResolveOverallHealthStatus_DetectsCheckFailures(t *testing.T) {
	checks := map[string]Check{
		"database":    {Status: "pass", Message: "Connected"},
		"data_points": {Status: "fail", Message: "query failed"},
	}
	if got := resolveOverallHealthStatus(checks, ""); got != "degraded" {
		t.Fatalf("resolveOverallHealthStatus() = %q, want degraded", got)
	}
}

func TestResolveOverallHealthStatus_DefaultsHealthy(t *testing.T) {
	checks := map[string]Check{
		"database":    {Status: "pass", Message: "Connected"},
		"data_points": {Status: "pass", Message: "10"},
	}
	if got := resolveOverallHealthStatus(checks, ""); got != "healthy" {
		t.Fatalf("resolveOverallHealthStatus() = %q, want healthy", got)
	}
}

func TestHealthHTTPStatus(t *testing.T) {
	if got := healthHTTPStatus("healthy"); got != http.StatusOK {
		t.Fatalf("healthHTTPStatus(healthy) = %d, want %d", got, http.StatusOK)
	}
	if got := healthHTTPStatus("degraded"); got != http.StatusOK {
		t.Fatalf("healthHTTPStatus(degraded) = %d, want %d", got, http.StatusOK)
	}
	if got := healthHTTPStatus("unhealthy"); got != http.StatusServiceUnavailable {
		t.Fatalf("healthHTTPStatus(unhealthy) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}
