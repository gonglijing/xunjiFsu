//go:build !no_paho_mqtt

package adapters

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
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

func TestMQTTSend_DropsOldestWhenQueueFull(t *testing.T) {
	a := NewMQTTAdapter("mqtt-test")

	for i := 0; i < mqttPendingDataCap+5; i++ {
		if err := a.Send(&models.CollectData{DeviceID: int64(i)}); err != nil {
			t.Fatalf("Send() error = %v", err)
		}
	}

	a.pendingMu.RLock()
	defer a.pendingMu.RUnlock()
	if len(a.pendingData) != mqttPendingDataCap {
		t.Fatalf("len(pendingData)=%d, want=%d", len(a.pendingData), mqttPendingDataCap)
	}
	if got := a.pendingData[0].DeviceID; got != 5 {
		t.Fatalf("oldest deviceID=%d, want=5", got)
	}
}

func TestMQTTSendAlarm_DropsOldestWhenQueueFull(t *testing.T) {
	a := NewMQTTAdapter("mqtt-test")

	for i := 0; i < mqttPendingAlarmCap+5; i++ {
		if err := a.SendAlarm(&models.AlarmPayload{DeviceID: int64(i)}); err != nil {
			t.Fatalf("SendAlarm() error = %v", err)
		}
	}

	a.alarmMu.RLock()
	defer a.alarmMu.RUnlock()
	if len(a.pendingAlarms) != mqttPendingAlarmCap {
		t.Fatalf("len(pendingAlarms)=%d, want=%d", len(a.pendingAlarms), mqttPendingAlarmCap)
	}
	if got := a.pendingAlarms[0].DeviceID; got != 5 {
		t.Fatalf("oldest alarm deviceID=%d, want=5", got)
	}
}
