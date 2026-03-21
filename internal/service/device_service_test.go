package service

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

func TestNextDeviceEnabledState(t *testing.T) {
	if nextDeviceEnabledState(0) != 1 {
		t.Fatal("nextDeviceEnabledState(0) should return 1")
	}
	if nextDeviceEnabledState(1) != 0 {
		t.Fatal("nextDeviceEnabledState(1) should return 0")
	}
}

func TestBuildResourceMap(t *testing.T) {
	resources := []*models.Resource{
		nil,
		{ID: 1, Name: "serial-1"},
		{ID: 2, Name: "net-1"},
	}

	got := buildResourceMap(resources)

	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[1] == nil || got[1].Name != "serial-1" {
		t.Fatalf("got[1] = %#v", got[1])
	}
}

func TestEnrichDeviceDisplay(t *testing.T) {
	driverID := int64(7)
	resourceID := int64(3)
	device := &models.Device{
		DriverID:   &driverID,
		ResourceID: &resourceID,
	}

	enrichDeviceDisplay(device, map[int64]string{7: "demo-driver"}, map[int64]*models.Resource{
		3: {ID: 3, Name: "rs485", Type: "serial", Path: "/dev/ttyUSB0"},
	})

	if device.DriverName != "demo-driver" {
		t.Fatalf("device.DriverName = %q, want demo-driver", device.DriverName)
	}
	if device.ResourceName != "rs485" || device.ResourceType != "serial" || device.ResourcePath != "/dev/ttyUSB0" {
		t.Fatalf("unexpected resource display fields: %#v", device)
	}
}
