package collector

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

func TestListDeviceRuntimeStatus(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	device := &models.Device{ID: 701, Name: "d-701", CollectInterval: 1500, StorageInterval: 30}
	task := newCollectTask(device, nil)
	now := time.Now().Add(-time.Minute).Round(time.Millisecond)
	task.nextRun = now.Add(10 * time.Second)
	task.lastRun = now
	task.lastStored = now.Add(3 * time.Second)
	task.consecutiveFailures = 2
	task.lastError = "i/o timeout"
	task.lastErrorKind = collectErrorKindTimeout
	task.lastErrorAt = now.Add(5 * time.Second)

	c.mu.Lock()
	c.tasks[device.ID] = task
	c.mu.Unlock()

	statuses := c.ListDeviceRuntimeStatus()
	status, ok := statuses[device.ID]
	if !ok {
		t.Fatalf("expected runtime status for device %d", device.ID)
	}
	if !status.Registered {
		t.Fatalf("expected registered=true")
	}
	if status.DeviceID != device.ID {
		t.Fatalf("device id mismatch: got %d want %d", status.DeviceID, device.ID)
	}
	if status.CollectIntervalMs != 1500 {
		t.Fatalf("collect interval mismatch: got %d want 1500", status.CollectIntervalMs)
	}
	if status.StorageIntervalSec != 30 {
		t.Fatalf("storage interval mismatch: got %d want 30", status.StorageIntervalSec)
	}
	if status.NextRunAt.IsZero() || !status.NextRunAt.Equal(task.nextRun) {
		t.Fatalf("next run mismatch")
	}
	if status.LastRunAt.IsZero() || !status.LastRunAt.Equal(task.lastRun) {
		t.Fatalf("last run mismatch")
	}
	if status.LastStoredAt.IsZero() || !status.LastStoredAt.Equal(task.lastStored) {
		t.Fatalf("last stored mismatch")
	}
	if status.ConsecutiveFailures != 2 {
		t.Fatalf("consecutive failures mismatch: got %d", status.ConsecutiveFailures)
	}
	if status.LastError != "i/o timeout" {
		t.Fatalf("last error mismatch: got %q", status.LastError)
	}
	if status.LastErrorKind != "timeout" {
		t.Fatalf("last error kind mismatch: got %q", status.LastErrorKind)
	}
	if status.LastErrorAt.IsZero() || !status.LastErrorAt.Equal(task.lastErrorAt) {
		t.Fatalf("last error at mismatch")
	}
}

func TestGetDeviceRuntimeStatus_NotFound(t *testing.T) {
	mgr := northbound.NewNorthboundManager()
	c := NewCollector(nil, mgr)

	if _, ok := c.GetDeviceRuntimeStatus(999); ok {
		t.Fatalf("expected not found status")
	}
}
