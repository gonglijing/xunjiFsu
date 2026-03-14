package adapters

import (
	"testing"
	"time"
)

func TestReconnectSchedulerScheduleAndStop(t *testing.T) {
	var scheduler reconnectScheduler

	scheduler.Schedule(10 * time.Millisecond)
	if scheduler.Channel() == nil {
		t.Fatal("Channel() = nil after Schedule")
	}

	select {
	case <-scheduler.Channel():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("scheduled timer did not fire")
	}

	scheduler.Stop()
	if scheduler.Channel() != nil {
		t.Fatal("Channel() not cleared after Stop")
	}
}

func TestReconnectSchedulerRescheduleClearsPendingSignal(t *testing.T) {
	var scheduler reconnectScheduler

	scheduler.Schedule(10 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	scheduler.Stop()

	start := time.Now()
	scheduler.Schedule(80 * time.Millisecond)

	select {
	case <-scheduler.Channel():
		if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
			t.Fatalf("timer fired too early after reschedule: %v", elapsed)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("rescheduled timer did not fire")
	}

	scheduler.Close()
	if scheduler.Channel() != nil {
		t.Fatal("Channel() not cleared after Close")
	}
}
