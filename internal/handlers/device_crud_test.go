package handlers

import (
	"testing"

	collectorpkg "github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestBuildDeviceListItem_WithRuntimeStatus(t *testing.T) {
	device := &models.Device{ID: 11, Name: "d-11"}
	statusMap := map[int64]collectorpkg.DeviceRuntimeStatus{
		11: {
			DeviceID:            11,
			Registered:          true,
			ConsecutiveFailures: 3,
			LastError:           "timeout",
			LastErrorKind:       "timeout",
		},
	}

	item := buildDeviceListItem(device, statusMap)
	if item == nil {
		t.Fatalf("expected non-nil item")
	}
	if item.Device != device {
		t.Fatalf("expected same device pointer")
	}
	if !item.CollectRuntime.Registered {
		t.Fatalf("expected registered runtime")
	}
	if item.CollectRuntime.ConsecutiveFailures != 3 {
		t.Fatalf("expected consecutive failures=3, got %d", item.CollectRuntime.ConsecutiveFailures)
	}
	if item.CollectRuntime.DeviceID != 11 {
		t.Fatalf("expected runtime device id=11, got %d", item.CollectRuntime.DeviceID)
	}
}

func TestBuildDeviceListItem_DefaultRuntimeStatus(t *testing.T) {
	device := &models.Device{ID: 22, Name: "d-22"}

	item := buildDeviceListItem(device, map[int64]collectorpkg.DeviceRuntimeStatus{})
	if item == nil {
		t.Fatalf("expected non-nil item")
	}
	if item.CollectRuntime.Registered {
		t.Fatalf("expected default registered=false")
	}
	if item.CollectRuntime.DeviceID != 22 {
		t.Fatalf("expected default runtime device id=22, got %d", item.CollectRuntime.DeviceID)
	}
}
