package adapters

import (
	"testing"
	"time"
)

func TestMQTTAdapterStartStopCanRestart(t *testing.T) {
	a := NewMQTTAdapter("mqtt-test")

	a.mu.Lock()
	a.initialized = true
	a.connected = true
	a.interval = time.Hour
	a.mu.Unlock()

	a.Start()
	if !a.IsEnabled() {
		t.Fatalf("expected adapter enabled after Start")
	}

	a.Stop()
	if a.IsEnabled() {
		t.Fatalf("expected adapter disabled after Stop")
	}

	a.Start()
	if !a.IsEnabled() {
		t.Fatalf("expected adapter enabled after restart")
	}

	a.Stop()
}

func TestMQTTSingleLoop_StopThenCloseSafe(t *testing.T) {
	a := NewMQTTAdapter("mqtt-test")
	a.mu.Lock()
	a.initialized = true
	a.connected = true
	a.interval = time.Hour
	a.mu.Unlock()

	a.Start()
	a.Stop()
	if err := a.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if a.IsEnabled() {
		t.Fatalf("expected adapter disabled after Close")
	}
	if a.loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", a.loopState.String())
	}
}
