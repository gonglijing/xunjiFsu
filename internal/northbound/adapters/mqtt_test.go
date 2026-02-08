package adapters

import (
	"testing"
)

func TestMQTTAdapterStartStopCanRestart(t *testing.T) {
	a := NewMQTTAdapter("mqtt-test")

	a.mu.Lock()
	a.initialized = true
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

func TestMQTTAdapterReconnectGuard(t *testing.T) {
	a := NewMQTTAdapter("mqtt-test")

	a.mu.Lock()
	a.reconnecting = true
	a.mu.Unlock()

	a.reconnect()

	a.mu.RLock()
	stillReconnecting := a.reconnecting
	a.mu.RUnlock()

	if !stillReconnecting {
		t.Fatalf("expected reconnect guard to keep reconnecting=true unchanged")
	}
}
