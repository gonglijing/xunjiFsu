package database

import (
	"testing"
	"time"
)

func TestDBHealthStatusToMap(t *testing.T) {
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := dbHealthStatus{
		paramHealthy: true,
		dataHealthy:  false,
		lastCheck:    now,
	}

	got := status.toMap()

	if got["healthy"] != false {
		t.Fatalf("healthy = %#v, want false", got["healthy"])
	}
	if got["param_healthy"] != true {
		t.Fatalf("param_healthy = %#v, want true", got["param_healthy"])
	}
	if got["data_healthy"] != false {
		t.Fatalf("data_healthy = %#v, want false", got["data_healthy"])
	}
	if got["last_check"] != now {
		t.Fatalf("last_check = %#v, want %#v", got["last_check"], now)
	}
}

func TestPingDatabase_NilDB(t *testing.T) {
	if pingDatabase("UnitDB", nil) {
		t.Fatal("pingDatabase(nil) = true, want false")
	}
}

func TestGetDBStatus_UsesHealthCheckerWhenAvailable(t *testing.T) {
	originalChecker := healthChecker
	defer func() { healthChecker = originalChecker }()

	now := time.Date(2026, 3, 18, 13, 0, 0, 0, time.UTC)
	healthChecker = &DBHealthChecker{
		paramHealthy: true,
		dataHealthy:  true,
		lastCheck:    now,
	}

	status := GetDBStatus()

	if status["healthy"] != true {
		t.Fatalf("healthy = %#v, want true", status["healthy"])
	}
	if status["last_check"] != now {
		t.Fatalf("last_check = %#v, want %#v", status["last_check"], now)
	}
}

func TestIsDBHealthy_UsesHealthCheckerWhenAvailable(t *testing.T) {
	originalChecker := healthChecker
	defer func() { healthChecker = originalChecker }()

	healthChecker = &DBHealthChecker{
		paramHealthy: true,
		dataHealthy:  false,
	}

	if IsDBHealthy() {
		t.Fatal("IsDBHealthy() = true, want false")
	}
}
