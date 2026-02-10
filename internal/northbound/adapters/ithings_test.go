package adapters

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestParseIThingsConfig_Defaults(t *testing.T) {
	config := `{"serverUrl":"tcp://localhost:1883","username":"u","productKey":"pk","deviceKey":"dk","gatewayMode":true}`

	cfg, err := parseIThingsConfig(config)
	if err != nil {
		t.Fatalf("parseIThingsConfig() error = %v", err)
	}

	if cfg.UploadIntervalMs != int((5 * time.Second).Milliseconds()) {
		t.Fatalf("UploadIntervalMs=%d, want=%d", cfg.UploadIntervalMs, int((5 * time.Second).Milliseconds()))
	}
	if cfg.AlarmFlushIntervalMs != int((2 * time.Second).Milliseconds()) {
		t.Fatalf("AlarmFlushIntervalMs=%d, want=%d", cfg.AlarmFlushIntervalMs, int((2 * time.Second).Milliseconds()))
	}
	if cfg.UpPropertyTopicTemplate == "" || cfg.DownPropertyTopic == "" {
		t.Fatalf("expected default topics applied, got up=%q down=%q", cfg.UpPropertyTopicTemplate, cfg.DownPropertyTopic)
	}
	if cfg.SubDeviceNameMode != cfg.DeviceNameMode {
		t.Fatalf("SubDeviceNameMode=%q, want DeviceNameMode=%q", cfg.SubDeviceNameMode, cfg.DeviceNameMode)
	}
}

func TestIThingsPullCommands_PopsInBatch(t *testing.T) {
	adapter := NewIThingsAdapter("ithings-test")
	adapter.initialized = true
	adapter.commandQueue = []*models.NorthboundCommand{
		{RequestID: "1"},
		{RequestID: "2"},
		{RequestID: "3"},
	}

	items, err := adapter.PullCommands(2)
	if err != nil {
		t.Fatalf("PullCommands() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items)=%d, want=2", len(items))
	}
	if len(adapter.commandQueue) != 1 {
		t.Fatalf("remaining queue=%d, want=1", len(adapter.commandQueue))
	}
}

func TestResolveDeviceNameByMode(t *testing.T) {
	if got := resolveDeviceNameByMode("dev-name", "dev-key", "device_name"); got != "dev-name" {
		t.Fatalf("device_name mode got=%q, want=dev-name", got)
	}
	if got := resolveDeviceNameByMode("dev-name", "dev-key", "device_key"); got != "dev-key" {
		t.Fatalf("device_key mode got=%q, want=dev-key", got)
	}
	if got := resolveDeviceNameByMode("", " dev-key ", ""); got != "dev-key" {
		t.Fatalf("default mode got=%q, want=dev-key", got)
	}
}
