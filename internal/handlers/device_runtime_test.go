package handlers

import (
	"testing"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestBuildDeviceRuntimeStatusList(t *testing.T) {
	devices := []*models.Device{
		{ID: 2},
		nil,
		{ID: 1},
		{ID: 3},
	}
	runtimeStatusMap := map[int64]collectorpkg.DeviceRuntimeStatus{
		1: {
			Registered:          true,
			ConsecutiveFailures: 2,
			LastErrorKind:       "timeout",
		},
		3: {
			DeviceID:   3,
			Registered: true,
		},
	}

	statuses := buildDeviceRuntimeStatusList(devices, runtimeStatusMap)
	if len(statuses) != 3 {
		t.Fatalf("expected 3 statuses, got %d", len(statuses))
	}

	if statuses[0].DeviceID != 1 || statuses[1].DeviceID != 2 || statuses[2].DeviceID != 3 {
		t.Fatalf("expected sorted device ids [1,2,3], got [%d,%d,%d]", statuses[0].DeviceID, statuses[1].DeviceID, statuses[2].DeviceID)
	}

	if !statuses[0].Registered || statuses[0].ConsecutiveFailures != 2 || statuses[0].LastErrorKind != "timeout" {
		t.Fatalf("device 1 runtime not preserved: %+v", statuses[0])
	}

	if statuses[1].Registered {
		t.Fatalf("device 2 should default to registered=false")
	}
}
