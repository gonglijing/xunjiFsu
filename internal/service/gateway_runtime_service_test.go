package service

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/platform/config"
)

func TestParseOptionalDuration(t *testing.T) {
	if _, ok, err := ParseOptionalDuration(""); err != nil || ok {
		t.Fatalf("empty should return (ok=false, err=nil), got ok=%v err=%v", ok, err)
	}

	if got, ok, err := ParseOptionalDuration("150ms"); err != nil || !ok || got != 150*time.Millisecond {
		t.Fatalf("parse 150ms failed, got (%v, %v, %v)", got, ok, err)
	}

	if _, _, err := ParseOptionalDuration("bad"); err == nil {
		t.Fatalf("expected parse error for invalid duration")
	}

	if _, _, err := ParseOptionalDuration("0s"); err == nil {
		t.Fatalf("expected parse error for non-positive duration")
	}
}

func TestRuntimeConfigView_UsesTypedView(t *testing.T) {
	svc := NewGatewayRuntimeService(&config.Config{
		CollectorDeviceSyncInterval:     3 * time.Second,
		CollectorCommandPollInterval:    5 * time.Second,
		CollectorWorkers:                7,
		NorthboundMQTTReconnectInterval: 9 * time.Second,
		DriverSerialReadTimeout:         11 * time.Second,
		DriverTCPDialTimeout:            13 * time.Second,
		DriverTCPReadTimeout:            15 * time.Second,
		DriverSerialOpenBackoff:         17 * time.Second,
		DriverTCPDialBackoff:            19 * time.Second,
		DriverSerialOpenRetries:         2,
		DriverTCPDialRetries:            4,
	}, nil, nil, nil)

	view := svc.RuntimeConfigView()
	if view.CollectorDeviceSyncInterval != "3s" {
		t.Fatalf("CollectorDeviceSyncInterval = %q", view.CollectorDeviceSyncInterval)
	}
	if view.CollectorWorkers != 7 {
		t.Fatalf("CollectorWorkers = %d", view.CollectorWorkers)
	}
	if view.DriverTCPDialRetries != 4 {
		t.Fatalf("DriverTCPDialRetries = %d", view.DriverTCPDialRetries)
	}
}

func TestApplyRuntimeConfig_NegativeRetries(t *testing.T) {
	negative := -1
	svc := NewGatewayRuntimeService(&config.Config{}, nil, nil, nil)

	_, err := svc.ApplyRuntimeConfig(&GatewayRuntimeConfig{DriverSerialOpenRetries: &negative})
	if err == nil {
		t.Fatalf("expected error for negative driver_serial_open_retries")
	}

	_, err = svc.ApplyRuntimeConfig(&GatewayRuntimeConfig{DriverTCPDialRetries: &negative})
	if err == nil {
		t.Fatalf("expected error for negative driver_tcp_dial_retries")
	}

	zero := 0
	_, err = svc.ApplyRuntimeConfig(&GatewayRuntimeConfig{CollectorWorkers: &zero})
	if err == nil {
		t.Fatalf("expected error for non-positive collector_workers")
	}
}

func TestApplyDurationConfigChange(t *testing.T) {
	changes := make(map[string]RuntimeConfigChange)
	target := 5 * time.Second

	if err := applyDurationConfigChange(changes, "driver_tcp_read_timeout", "8s", &target); err != nil {
		t.Fatalf("applyDurationConfigChange returned error: %v", err)
	}
	if target != 8*time.Second {
		t.Fatalf("target = %v, want 8s", target)
	}
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}
}

func TestApplyRetryConfigChange(t *testing.T) {
	changes := make(map[string]RuntimeConfigChange)
	target := 1
	value := 3

	if err := applyRetryConfigChange(changes, "driver_tcp_dial_retries", &value, &target); err != nil {
		t.Fatalf("applyRetryConfigChange returned error: %v", err)
	}
	if target != 3 {
		t.Fatalf("target = %d, want 3", target)
	}

	negative := -1
	if err := applyRetryConfigChange(changes, "driver_tcp_dial_retries", &negative, &target); err == nil {
		t.Fatal("expected negative retry error")
	}
}

func TestApplyPositiveIntConfigChange(t *testing.T) {
	changes := make(map[string]RuntimeConfigChange)
	target := 4
	value := 6

	if err := applyPositiveIntConfigChange(changes, "collector_workers", &value, &target); err != nil {
		t.Fatalf("applyPositiveIntConfigChange returned error: %v", err)
	}
	if target != 6 {
		t.Fatalf("target = %d, want 6", target)
	}

	zero := 0
	if err := applyPositiveIntConfigChange(changes, "collector_workers", &zero, &target); err == nil {
		t.Fatal("expected non-positive value error")
	}
}

func TestRecordRuntimeConfigChange(t *testing.T) {
	changes := make(map[string]RuntimeConfigChange)

	recordRuntimeConfigChange(changes, "x", "1s", "1s")
	if len(changes) != 0 {
		t.Fatalf("expected same from/to not recorded")
	}

	recordRuntimeConfigChange(changes, "x", "1s", "2s")
	if len(changes) != 1 {
		t.Fatalf("expected one change recorded")
	}
}
