package adapters

import (
	"encoding/json"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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

func TestIThingsBuildRealtimePublish(t *testing.T) {
	adapter := NewIThingsAdapter("ithings-test")
	adapter.config = &IThingsConfig{ProductKey: "gwpk", DeviceKey: "gwdk"}
	adapter.upPropertyTopicTemplate = "$thing/up/property/{productID}/{deviceName}"
	adapter.deviceNameMode = "device_key"
	adapter.subDeviceNameMode = "device_key"

	topic, body, err := adapter.buildRealtimePublish(&models.CollectData{
		DeviceID:   1,
		DeviceName: "pump-1",
		DeviceKey:  "dk-1",
		Timestamp:  time.Unix(1700000000, 0),
		Fields: map[string]string{
			"temperature": "23.5",
			"running":     "true",
		},
	})
	if err != nil {
		t.Fatalf("buildRealtimePublish() error = %v", err)
	}
	if topic != "$thing/up/property/gwpk/gwdk" {
		t.Fatalf("topic=%q", topic)
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	subDevices, ok := decoded["subDevices"].([]interface{})
	if !ok || len(subDevices) != 1 {
		t.Fatalf("subDevices=%v", decoded["subDevices"])
	}
	subDevice, _ := subDevices[0].(map[string]interface{})
	if subDevice["deviceName"] != "dk-1" {
		t.Fatalf("deviceName=%v, want=dk-1", subDevice["deviceName"])
	}
	properties, ok := subDevice["properties"].([]interface{})
	if !ok || len(properties) != 1 {
		t.Fatalf("properties=%v", subDevice["properties"])
	}
	item, _ := properties[0].(map[string]interface{})
	params, ok := item["params"].(map[string]interface{})
	if !ok {
		t.Fatalf("params=%v", item["params"])
	}
	if params["temperature"] != 23.5 {
		t.Fatalf("temperature=%v, want=23.5", params["temperature"])
	}
	if params["running"] != true {
		t.Fatalf("running=%v, want=true", params["running"])
	}
}

func TestIThingsBuildAlarmPublish(t *testing.T) {
	adapter := NewIThingsAdapter("ithings-test")
	adapter.config = &IThingsConfig{ProductKey: "gwpk", DeviceKey: "gwdk"}
	adapter.upEventTopicTemplate = "$thing/up/event/{productID}/{deviceName}"
	adapter.deviceNameMode = "device_key"
	adapter.alarmEventID = "alarm"
	adapter.alarmEventType = "alert"

	topic, body, err := adapter.buildAlarmPublish(&models.AlarmPayload{
		DeviceID:    1,
		DeviceName:  "pump-1",
		DeviceKey:   "dk-1",
		FieldName:   "temperature",
		ActualValue: 23.5,
		Threshold:   30,
		Operator:    ">",
		Severity:    "high",
		Message:     "too hot",
	})
	if err != nil {
		t.Fatalf("buildAlarmPublish() error = %v", err)
	}
	if topic != "$thing/up/event/gwpk/gwdk" {
		t.Fatalf("topic=%q", topic)
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	params, ok := decoded["params"].(map[string]interface{})
	if !ok {
		t.Fatalf("params=%v", decoded["params"])
	}
	if params["device_key"] != "dk-1" {
		t.Fatalf("device_key=%v, want=dk-1", params["device_key"])
	}
	if params["actual_value"] != 23.5 {
		t.Fatalf("actual_value=%v, want=23.5", params["actual_value"])
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

func TestBuildIThingsInitSettings_Defaults(t *testing.T) {
	cfg, err := parseIThingsConfig(`{"serverUrl":"127.0.0.1","port":1883,"username":"u","productKey":"pk","deviceKey":"dk","gatewayMode":true}`)
	if err != nil {
		t.Fatalf("parseIThingsConfig() error = %v", err)
	}

	settings := buildIThingsInitSettings("ithings-test", cfg)
	if settings.broker != "tcp://127.0.0.1" {
		t.Fatalf("broker=%q, want=tcp://127.0.0.1", settings.broker)
	}
	if settings.reportEvery != 5*time.Second {
		t.Fatalf("reportEvery=%v, want=5s", settings.reportEvery)
	}
	if settings.alarmEvery != 2*time.Second {
		t.Fatalf("alarmEvery=%v, want=2s", settings.alarmEvery)
	}
	if settings.downPropertyTopic != defaultIThingsDownPropertyTopic {
		t.Fatalf("downPropertyTopic=%q", settings.downPropertyTopic)
	}
	if settings.downActionTopic != defaultIThingsDownActionTopic {
		t.Fatalf("downActionTopic=%q", settings.downActionTopic)
	}
	if settings.deviceNameMode != "deviceKey" || settings.subDeviceNameMode != "deviceKey" {
		t.Fatalf("device modes mismatch: %q/%q", settings.deviceNameMode, settings.subDeviceNameMode)
	}
}

func TestIThingsApplyConfig_ResetsRuntimeState(t *testing.T) {
	cfg, err := parseIThingsConfig(`{"serverUrl":"tcp://localhost:1883","username":"u","productKey":"pk","deviceKey":"dk","gatewayMode":true}`)
	if err != nil {
		t.Fatalf("parseIThingsConfig() error = %v", err)
	}

	adapter := NewIThingsAdapter("ithings-test")
	adapter.requestStates = map[string]*iThingsRequestState{"old": {RequestID: "old"}}
	adapter.enabled = true
	adapter.loopState = adapterLoopRunning

	settings := buildIThingsInitSettings("ithings-test", cfg)
	var client mqtt.Client
	adapter.applyConfig(cfg, client, settings)

	if adapter.enabled {
		t.Fatal("expected adapter disabled after applyConfig")
	}
	if adapter.loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", adapter.loopState.String())
	}
	if adapter.flushNow == nil || adapter.stopChan == nil {
		t.Fatal("expected runtime channels initialized")
	}
	if adapter.requestStates == nil || len(adapter.requestStates) != 0 {
		t.Fatalf("requestStates=%v, want empty map", adapter.requestStates)
	}
}

func TestIThingsSingleLoop_StopThenCloseSafe(t *testing.T) {
	adapter := NewIThingsAdapter("ithings-test")
	adapter.initialized = true
	adapter.reportEvery = time.Hour
	adapter.alarmEvery = time.Hour

	adapter.Start()
	adapter.Stop()
	if err := adapter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if adapter.IsEnabled() {
		t.Fatal("adapter should be disabled after Close")
	}
	if adapter.loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", adapter.loopState.String())
	}
}
