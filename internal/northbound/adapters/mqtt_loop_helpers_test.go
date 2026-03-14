package adapters

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRunMQTTLikeLoop_StopFlushesAndExits(t *testing.T) {
	stopChan := make(chan struct{})
	dataChan := make(chan struct{}, 1)
	reconnectNow := make(chan struct{}, 1)
	done := make(chan struct{})

	var flushDataCalls atomic.Int32
	var flushAlarmCalls atomic.Int32

	go func() {
		runMQTTLikeLoop(mqttLikeLoopConfig{
			logLabel:     "test",
			adapterName:  "loop",
			interval:     time.Hour,
			stopChan:     stopChan,
			dataChan:     dataChan,
			reconnectNow: reconnectNow,
			flushData: func() {
				flushDataCalls.Add(1)
			},
			flushAlarm: func() {
				flushAlarmCalls.Add(1)
			},
			shouldReconnect: func() bool { return false },
			reconnectOnce:   func() error { return nil },
			reconnectDelay:  func() time.Duration { return time.Millisecond },
		})
		close(done)
	}()

	close(stopChan)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("runMQTTLikeLoop did not exit after stop")
	}

	if got := flushDataCalls.Load(); got != 1 {
		t.Fatalf("flushDataCalls=%d, want=1", got)
	}
	if got := flushAlarmCalls.Load(); got != 1 {
		t.Fatalf("flushAlarmCalls=%d, want=1", got)
	}
}
