package adapters

import (
	"sync"
	"sync/atomic"
	"testing"
)

type testDisconnectableClient struct {
	connected       bool
	disconnectCalls int
}

func (c *testDisconnectableClient) IsConnected() bool {
	return c.connected
}

func (c *testDisconnectableClient) Disconnect(quiesce uint) {
	c.disconnectCalls++
	c.connected = false
}

func TestAdapterLifecycleStateStartStop(t *testing.T) {
	var (
		mu           sync.RWMutex
		wg           sync.WaitGroup
		initialized  = true
		enabled      bool
		connected    bool
		loopState    = adapterLoopStopped
		stopChan     chan struct{}
		workSignal   chan struct{}
		reconnectNow chan struct{}
	)

	state := adapterLifecycleState{
		adapterType:    "test",
		logLabel:       "Test",
		adapterName:    "lifecycle",
		mu:             &mu,
		wg:             &wg,
		initialized:    &initialized,
		enabled:        &enabled,
		connected:      &connected,
		loopState:      &loopState,
		stopChan:       &stopChan,
		workSignalChan: &workSignal,
		reconnectChan:  &reconnectNow,
	}

	var reconnectSignals atomic.Int32
	started := make(chan struct{}, 1)
	runLoop := func() {
		started <- struct{}{}
		<-stopChan
		wg.Done()
	}

	state.start(runLoop, func() {
		reconnectSignals.Add(1)
	})

	<-started
	if !enabled {
		t.Fatal("expected enabled after start")
	}
	if loopState != adapterLoopRunning {
		t.Fatalf("loopState=%s, want=running", loopState.String())
	}
	if reconnectSignals.Load() != 1 {
		t.Fatalf("reconnectSignals=%d, want=1", reconnectSignals.Load())
	}
	if stopChan == nil || workSignal == nil || reconnectNow == nil {
		t.Fatal("expected lifecycle channels initialized")
	}

	state.stop()

	if enabled {
		t.Fatal("expected disabled after stop")
	}
	if loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", loopState.String())
	}
	if stopChan != nil {
		t.Fatal("expected stopChan cleared after stop")
	}
}

func TestAdapterLifecycleStateCloseResetsState(t *testing.T) {
	var (
		mu           sync.RWMutex
		wg           sync.WaitGroup
		initialized  = true
		enabled      bool
		connected                         = true
		loopState                         = adapterLoopStopped
		stopChan                          = make(chan struct{})
		workSignal                        = make(chan struct{}, 1)
		reconnectNow                      = make(chan struct{}, 1)
		rawClient                         = &testDisconnectableClient{connected: true}
		client       disconnectableClient = rawClient
	)

	state := adapterLifecycleState{
		adapterType:    "test",
		logLabel:       "Test",
		adapterName:    "lifecycle",
		mu:             &mu,
		wg:             &wg,
		initialized:    &initialized,
		enabled:        &enabled,
		connected:      &connected,
		loopState:      &loopState,
		stopChan:       &stopChan,
		workSignalChan: &workSignal,
		reconnectChan:  &reconnectNow,
	}

	var flushDataCalls atomic.Int32
	var flushAlarmCalls atomic.Int32

	if err := state.close(
		func() { flushDataCalls.Add(1) },
		func() { flushAlarmCalls.Add(1) },
		func() disconnectableClient {
			current := client
			client = nil
			return current
		},
	); err != nil {
		t.Fatalf("close() error = %v", err)
	}

	if initialized || enabled || connected {
		t.Fatalf("state not reset: initialized=%v enabled=%v connected=%v", initialized, enabled, connected)
	}
	if loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", loopState.String())
	}
	if stopChan != nil || workSignal != nil || reconnectNow != nil {
		t.Fatal("expected lifecycle channels cleared after close")
	}
	if flushDataCalls.Load() != 1 || flushAlarmCalls.Load() != 1 {
		t.Fatalf("flush calls mismatch: data=%d alarm=%d", flushDataCalls.Load(), flushAlarmCalls.Load())
	}
	if client != nil {
		t.Fatal("expected client cleared")
	}
	if rawClient.disconnectCalls != 1 {
		t.Fatalf("disconnectCalls=%d, want=1", rawClient.disconnectCalls)
	}
}

func TestAdapterLifecycleStateOptionalReconnectSignal(t *testing.T) {
	var (
		mu          sync.RWMutex
		wg          sync.WaitGroup
		initialized = true
		enabled     bool
		connected   = true
		loopState   = adapterLoopStopped
		stopChan    chan struct{}
		workSignal  chan struct{}
	)

	state := adapterLifecycleState{
		adapterType:    "test",
		logLabel:       "Test",
		adapterName:    "lifecycle",
		mu:             &mu,
		wg:             &wg,
		initialized:    &initialized,
		enabled:        &enabled,
		connected:      &connected,
		loopState:      &loopState,
		stopChan:       &stopChan,
		workSignalChan: &workSignal,
	}

	started := make(chan struct{}, 1)
	state.start(func() {
		started <- struct{}{}
		<-stopChan
		wg.Done()
	}, nil)

	<-started
	if workSignal == nil {
		t.Fatal("expected workSignal initialized")
	}
	state.stop()
}
