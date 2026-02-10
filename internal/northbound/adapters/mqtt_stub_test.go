//go:build no_paho_mqtt

package adapters

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestMQTTStub_DisabledBehavior(t *testing.T) {
	a := NewMQTTAdapter("mqtt-stub")
	if a == nil {
		t.Fatalf("expected stub adapter not nil")
	}
	if err := a.Initialize(`{"broker":"tcp://127.0.0.1:1883","topic":"t"}`); err == nil {
		t.Fatalf("expected initialize error when mqtt disabled")
	}
	if err := a.Send(&models.CollectData{DeviceID: 1}); err == nil {
		t.Fatalf("expected send error when mqtt disabled")
	}
	if err := a.SendAlarm(&models.AlarmPayload{DeviceID: 1}); err == nil {
		t.Fatalf("expected alarm send error when mqtt disabled")
	}
	stats := a.GetStats()
	if stats["type"] != "mqtt" {
		t.Fatalf("unexpected stats type: %v", stats["type"])
	}
}
