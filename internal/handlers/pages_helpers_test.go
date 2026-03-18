package handlers

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestBuildDeviceStats(t *testing.T) {
	devices := []*models.Device{{Enabled: 1}, {Enabled: 0}, nil, {Enabled: 1}}
	stats := buildDeviceStats(devices)

	if stats.Total != 4 {
		t.Fatalf("Total = %d, want %d", stats.Total, 4)
	}
	if stats.Enabled != 2 {
		t.Fatalf("Enabled = %d, want %d", stats.Enabled, 2)
	}
}

func TestBuildNorthboundStats(t *testing.T) {
	configs := []*models.NorthboundConfig{{Enabled: 1}, {Enabled: 0}, nil}
	stats := buildNorthboundStats(configs)

	if stats.Total != 3 {
		t.Fatalf("Total = %d, want %d", stats.Total, 3)
	}
	if stats.Enabled != 1 {
		t.Fatalf("Enabled = %d, want %d", stats.Enabled, 1)
	}
}

func TestBuildAlarmStats(t *testing.T) {
	now := time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)
	today := now.Truncate(24 * time.Hour)
	alarms := []*models.AlarmLog{
		{Acknowledged: 0, TriggeredAt: today.Add(1 * time.Hour)},
		{Acknowledged: 1, TriggeredAt: today.Add(-1 * time.Hour)},
		nil,
	}

	stats := buildAlarmStats(alarms, now)
	if stats.Total != 3 {
		t.Fatalf("Total = %d, want %d", stats.Total, 3)
	}
	if stats.Unacked != 1 {
		t.Fatalf("Unacked = %d, want %d", stats.Unacked, 1)
	}
	if stats.Today != 1 {
		t.Fatalf("Today = %d, want %d", stats.Today, 1)
	}
}

func TestBuildStatusData(t *testing.T) {
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	status := buildStatusData(
		[]*models.Device{{Enabled: 1}, {Enabled: 0}},
		[]*models.NorthboundConfig{{Enabled: 1}},
		[]*models.AlarmLog{{Acknowledged: 0, TriggeredAt: now}},
		3,
		true,
		now,
	)

	if !status.CollectorRunning {
		t.Fatal("CollectorRunning = false, want true")
	}
	if status.Devices.Total != 2 || status.Devices.Enabled != 1 {
		t.Fatalf("unexpected device stats: %#v", status.Devices)
	}
	if status.Northbound.Total != 1 || status.Northbound.Enabled != 1 {
		t.Fatalf("unexpected northbound stats: %#v", status.Northbound)
	}
	if status.Alarms.Total != 1 || status.Alarms.Unacked != 1 {
		t.Fatalf("unexpected alarm stats: %#v", status.Alarms)
	}
	if status.Drivers.Total != 3 {
		t.Fatalf("status.Drivers.Total = %d, want 3", status.Drivers.Total)
	}
	if !status.Timestamp.Equal(now) {
		t.Fatalf("status.Timestamp = %#v, want %#v", status.Timestamp, now)
	}
}

func TestBuildCollectorStatusResponse(t *testing.T) {
	response := buildCollectorStatusResponse("started")
	if response["status"] != "started" {
		t.Fatalf("response[status] = %q, want %q", response["status"], "started")
	}
}
