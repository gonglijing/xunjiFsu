//go:build no_paho_mqtt

package adapters

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestXunjiStub_DisabledBehavior(t *testing.T) {
	a := NewXunjiAdapter("xunji-stub")
	if a == nil {
		t.Fatalf("expected stub adapter not nil")
	}
	if err := a.Initialize(`{"serverUrl":"tcp://127.0.0.1:1883"}`); err == nil {
		t.Fatalf("expected initialize error when xunji disabled")
	}
	if err := a.Send(&models.CollectData{DeviceID: 1}); err == nil {
		t.Fatalf("expected send error when xunji disabled")
	}
	if err := a.SendAlarm(&models.AlarmPayload{DeviceID: 1}); err == nil {
		t.Fatalf("expected alarm send error when xunji disabled")
	}
	stats := a.GetStats()
	if stats["type"] != "xunji" {
		t.Fatalf("unexpected stats type: %v", stats["type"])
	}
}
