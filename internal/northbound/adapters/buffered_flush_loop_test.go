package adapters

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestRunBufferedFlushLoop_StopDrainsAlarmQueue(t *testing.T) {
	stopChan := make(chan struct{})
	flushNow := make(chan struct{}, 1)
	done := make(chan struct{})

	var alarmFlushCalls atomic.Int32

	go func() {
		runBufferedFlushLoop(bufferedFlushLoopConfig{
			logLabel:       "test",
			reportLabel:    "report",
			reportInterval: time.Hour,
			alarmInterval:  time.Hour,
			stopChan:       stopChan,
			flushNow:       flushNow,
			flushData:      func() error { return nil },
			flushAlarm: func() error {
				alarmFlushCalls.Add(1)
				return nil
			},
			alarmQueueEmpty: func() bool {
				return alarmFlushCalls.Load() >= 2
			},
		})
		close(done)
	}()

	close(stopChan)

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("runBufferedFlushLoop did not exit after draining alarms")
	}

	if got := alarmFlushCalls.Load(); got != 2 {
		t.Fatalf("alarmFlushCalls=%d, want=2", got)
	}
}
