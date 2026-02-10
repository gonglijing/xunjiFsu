package handlers

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestFilterSystemDataCache(t *testing.T) {
	input := []*models.DataCache{
		{DeviceID: 1, FieldName: "temp", Value: "20"},
		nil,
		{DeviceID: models.SystemStatsDeviceID, FieldName: "cpu_usage", Value: "12.3"},
		{DeviceID: 2, FieldName: "hum", Value: "56"},
	}

	got := filterSystemDataCache(input)
	if len(got) != 2 {
		t.Fatalf("len = %d, want %d", len(got), 2)
	}
	if got[0].DeviceID != 1 {
		t.Fatalf("first DeviceID = %d, want %d", got[0].DeviceID, 1)
	}
	if got[1].DeviceID != 2 {
		t.Fatalf("second DeviceID = %d, want %d", got[1].DeviceID, 2)
	}
}

func TestFilterSystemDataPoints(t *testing.T) {
	input := []*database.DataPoint{
		{DeviceID: 9, FieldName: "voltage", Value: "220"},
		nil,
		{DeviceID: models.SystemStatsDeviceID, FieldName: "mem_usage", Value: "41.2"},
		{DeviceID: 10, FieldName: "current", Value: "3.5"},
	}

	got := filterSystemDataPoints(input)
	if len(got) != 2 {
		t.Fatalf("len = %d, want %d", len(got), 2)
	}
	if got[0].DeviceID != 9 {
		t.Fatalf("first DeviceID = %d, want %d", got[0].DeviceID, 9)
	}
	if got[1].DeviceID != 10 {
		t.Fatalf("second DeviceID = %d, want %d", got[1].DeviceID, 10)
	}
}
