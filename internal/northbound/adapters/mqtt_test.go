//go:build !no_paho_mqtt

package adapters

import (
	"strings"
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

func TestParseMQTTConfig_Defaults(t *testing.T) {
	cfg, err := parseMQTTConfig(`{"broker":"127.0.0.1:1883","topic":"test/topic"}`)
	if err != nil {
		t.Fatalf("parseMQTTConfig() error = %v", err)
	}

	if cfg.ConnectTimeout != 10 {
		t.Fatalf("ConnectTimeout=%d, want=10", cfg.ConnectTimeout)
	}
	if cfg.KeepAlive != 60 {
		t.Fatalf("KeepAlive=%d, want=60", cfg.KeepAlive)
	}

	settings := buildMQTTInitSettings(cfg)
	if settings.broker != "tcp://127.0.0.1:1883" {
		t.Fatalf("broker=%q, want=tcp://127.0.0.1:1883", settings.broker)
	}
	if settings.alarmTopic != "test/topic/alarm" {
		t.Fatalf("alarmTopic=%q, want=test/topic/alarm", settings.alarmTopic)
	}
	if settings.interval != 5*time.Second {
		t.Fatalf("interval=%v, want=5s", settings.interval)
	}
	if !strings.HasPrefix(settings.clientID, "fsu-mqtt-") {
		t.Fatalf("clientID=%q, want fsu-mqtt-*", settings.clientID)
	}
}

func TestParseMQTTConfig_InvalidQOS(t *testing.T) {
	_, err := parseMQTTConfig(`{"broker":"tcp://127.0.0.1:1883","topic":"test/topic","qos":3}`)
	if err == nil {
		t.Fatalf("expected parseMQTTConfig() error")
	}
}

func TestMQTTApplyConfig_SetsRuntimeFields(t *testing.T) {
	cfg, err := parseMQTTConfig(`{"broker":"127.0.0.1:1883","topic":"test/topic","clean_session":true}`)
	if err != nil {
		t.Fatalf("parseMQTTConfig() error = %v", err)
	}

	adapter := NewMQTTAdapter("mqtt-test")
	settings := buildMQTTInitSettings(cfg)
	adapter.applyConfig(cfg, nil, settings)

	if adapter.broker != "tcp://127.0.0.1:1883" {
		t.Fatalf("broker=%q, want=tcp://127.0.0.1:1883", adapter.broker)
	}
	if adapter.topic != "test/topic" || adapter.alarmTopic != "test/topic/alarm" {
		t.Fatalf("topics mismatch: %q / %q", adapter.topic, adapter.alarmTopic)
	}
	if adapter.clientID != settings.clientID {
		t.Fatalf("clientID=%q, want=%q", adapter.clientID, settings.clientID)
	}
	if !adapter.cleanSession {
		t.Fatal("expected cleanSession=true")
	}
	if !adapter.initialized || !adapter.connected {
		t.Fatalf("state mismatch: initialized=%v connected=%v", adapter.initialized, adapter.connected)
	}
	if adapter.loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", adapter.loopState.String())
	}
}
