package adapters

import (
	"sync/atomic"
	"testing"
)

func TestDrainAlarmQueueOnStop_DrainsUntilEmpty(t *testing.T) {
	var flushCalls atomic.Int32
	var drained atomic.Bool

	drainAlarmQueueOnStop(alarmDrainConfig{
		flushAlarm: func() error {
			flushCalls.Add(1)
			return nil
		},
		alarmQueueEmpty: func() bool {
			return flushCalls.Load() >= 3
		},
		onDrained: func() {
			drained.Store(true)
		},
	})

	if got := flushCalls.Load(); got != 3 {
		t.Fatalf("flushCalls=%d, want=3", got)
	}
	if !drained.Load() {
		t.Fatal("expected onDrained to be called")
	}
}
