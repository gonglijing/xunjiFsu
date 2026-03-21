package adapters

import (
	"sync"
	"testing"
	"time"
)

type testConnectionStatus struct {
	connected bool
}

func (c *testConnectionStatus) IsConnected() bool {
	return c.connected
}

func TestAdapterReconnectStateMarkDisconnectedSignalsReconnect(t *testing.T) {
	var (
		mu                sync.RWMutex
		initialized       = true
		enabled           = true
		connected         = true
		reconnectInterval = time.Second
		reconnectNow      = make(chan struct{}, 1)
		client            = &testConnectionStatus{connected: true}
	)

	state := adapterReconnectState{
		mu:                &mu,
		initialized:       &initialized,
		enabled:           &enabled,
		connected:         &connected,
		reconnectInterval: &reconnectInterval,
		reconnectNow:      &reconnectNow,
		client: func() connectionStatus {
			return client
		},
	}

	state.markDisconnected()

	if connected {
		t.Fatal("expected connected=false after markDisconnected")
	}
	if len(reconnectNow) != 1 {
		t.Fatalf("len(reconnectNow)=%d, want=1", len(reconnectNow))
	}
}

func TestAdapterReconnectStateConnectionChecks(t *testing.T) {
	var (
		mu                sync.RWMutex
		initialized       = true
		enabled           = true
		connected         = true
		reconnectInterval time.Duration
		reconnectNow      = make(chan struct{}, 1)
		client            = &testConnectionStatus{connected: true}
	)

	state := adapterReconnectState{
		mu:                &mu,
		initialized:       &initialized,
		enabled:           &enabled,
		connected:         &connected,
		reconnectInterval: &reconnectInterval,
		reconnectNow:      &reconnectNow,
		client: func() connectionStatus {
			return client
		},
	}

	if got := state.currentReconnectInterval(); got != defaultReconnectInterval {
		t.Fatalf("currentReconnectInterval()=%v, want=%v", got, defaultReconnectInterval)
	}
	if !state.isConnected() {
		t.Fatal("expected isConnected()=true")
	}

	client.connected = false
	if !state.shouldReconnect() {
		t.Fatal("expected shouldReconnect()=true when client disconnected")
	}
	if state.isConnected() {
		t.Fatal("expected isConnected()=false when client disconnected")
	}
}

func TestAdapterReconnectStateCustomNormalizeInterval(t *testing.T) {
	var (
		mu                sync.RWMutex
		initialized       = true
		enabled           = true
		connected         = true
		reconnectInterval = 10 * time.Minute
		client            = &testConnectionStatus{connected: true}
	)

	state := adapterReconnectState{
		mu:                &mu,
		initialized:       &initialized,
		enabled:           &enabled,
		connected:         &connected,
		reconnectInterval: &reconnectInterval,
		client: func() connectionStatus {
			return client
		},
		normalizeInterval: normalizePandaXReconnectInterval,
	}

	if got := state.currentReconnectInterval(); got != maxPandaXReconnectInterval {
		t.Fatalf("currentReconnectInterval()=%v, want=%v", got, maxPandaXReconnectInterval)
	}
}
