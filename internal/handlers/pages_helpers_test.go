package handlers

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestSummarizeDeviceStats(t *testing.T) {
	devices := []*models.Device{{Enabled: 1}, {Enabled: 0}, nil, {Enabled: 1}}
	stats := summarizeDeviceStats(devices)

	if stats.Total != 4 {
		t.Fatalf("Total = %d, want %d", stats.Total, 4)
	}
	if stats.Enabled != 2 {
		t.Fatalf("Enabled = %d, want %d", stats.Enabled, 2)
	}
}

func TestSummarizeNorthboundStats(t *testing.T) {
	configs := []*models.NorthboundConfig{{Enabled: 1}, {Enabled: 0}, nil}
	stats := summarizeNorthboundStats(configs)

	if stats.Total != 3 {
		t.Fatalf("Total = %d, want %d", stats.Total, 3)
	}
	if stats.Enabled != 1 {
		t.Fatalf("Enabled = %d, want %d", stats.Enabled, 1)
	}
}

func TestSummarizeAlarmStats(t *testing.T) {
	now := time.Date(2026, 2, 8, 12, 0, 0, 0, time.UTC)
	today := now.Truncate(24 * time.Hour)
	alarms := []*models.AlarmLog{
		{Acknowledged: 0, TriggeredAt: today.Add(1 * time.Hour)},
		{Acknowledged: 1, TriggeredAt: today.Add(-1 * time.Hour)},
		nil,
	}

	stats := summarizeAlarmStats(alarms, now)
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
