//go:build !no_paho_mqtt

package adapters

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestParseXunjiConfig_Defaults(t *testing.T) {
	cfg, err := parseXunjiConfig(`{"serverUrl":"localhost","port":1883,"qos":1}`)
	if err != nil {
		t.Fatalf("parseXunjiConfig() error = %v", err)
	}
	if cfg.ServerURL != "tcp://localhost:1883" {
		t.Fatalf("ServerURL=%q, want=%q", cfg.ServerURL, "tcp://localhost:1883")
	}
	if cfg.Topic != defaultXunjiTopicTemplate {
		t.Fatalf("Topic=%q, want=%q", cfg.Topic, defaultXunjiTopicTemplate)
	}
	if cfg.UploadIntervalMs != int(defaultReportInterval.Milliseconds()) {
		t.Fatalf("UploadIntervalMs=%d, want=%d", cfg.UploadIntervalMs, int(defaultReportInterval.Milliseconds()))
	}
}

func TestRenderXunjiTopic(t *testing.T) {
	got := renderXunjiTopic("v1/gateway/{gatewayname}", "Main Gateway")
	if got != "v1/gateway/Main_Gateway" {
		t.Fatalf("renderXunjiTopic()=%q, want=%q", got, "v1/gateway/Main_Gateway")
	}
}

func TestResolveXunjiSubTokenModes(t *testing.T) {
	data := &models.CollectData{
		DeviceID:   1,
		DeviceName: "dev-name",
		ProductKey: "prod-a",
		DeviceKey:  "dev-key",
	}

	if got := resolveXunjiSubToken(data, ""); got != "dev-name" {
		t.Fatalf("default token=%q, want=dev-name", got)
	}
	if got := resolveXunjiSubToken(data, "device_key"); got != "dev-key" {
		t.Fatalf("device_key token=%q, want=dev-key", got)
	}
	if got := resolveXunjiSubToken(data, "product_devicekey"); got != "prod-a_dev-key" {
		t.Fatalf("product_devicekey token=%q, want=prod-a_dev-key", got)
	}
}

func TestBuildXunjiInitSettings_Defaults(t *testing.T) {
	cfg, err := parseXunjiConfig(`{"serverUrl":"localhost","port":1883,"gatewayName":"Main Gateway"}`)
	if err != nil {
		t.Fatalf("parseXunjiConfig() error = %v", err)
	}

	settings := buildXunjiInitSettings("xunji-test", cfg)
	if settings.broker != "tcp://localhost:1883" {
		t.Fatalf("broker=%q, want=tcp://localhost:1883", settings.broker)
	}
	if settings.gatewayName != "Main_Gateway" {
		t.Fatalf("gatewayName=%q, want=Main_Gateway", settings.gatewayName)
	}
	if settings.topic != "v1/gateway/Main_Gateway" {
		t.Fatalf("topic=%q, want=v1/gateway/Main_Gateway", settings.topic)
	}
	if settings.alarmTopic != "v1/gateway/Main_Gateway/alarm" {
		t.Fatalf("alarmTopic=%q, want=v1/gateway/Main_Gateway/alarm", settings.alarmTopic)
	}
	if settings.timeout != 10*time.Second {
		t.Fatalf("timeout=%v, want=10s", settings.timeout)
	}
	if settings.keepAlive != 60*time.Second {
		t.Fatalf("keepAlive=%v, want=60s", settings.keepAlive)
	}
	if settings.interval != 5*time.Second {
		t.Fatalf("interval=%v, want=5s", settings.interval)
	}
	if !strings.HasPrefix(settings.clientID, "xunji-xunji-test-") {
		t.Fatalf("clientID=%q, want xunji-xunji-test-*", settings.clientID)
	}
}

func TestXunjiApplyConfig_SetsRuntimeFields(t *testing.T) {
	cfg, err := parseXunjiConfig(`{"serverUrl":"localhost","port":1883,"gatewayName":"GW 1","subDeviceTokenMode":"device_key"}`)
	if err != nil {
		t.Fatalf("parseXunjiConfig() error = %v", err)
	}

	adapter := NewXunjiAdapter("xunji-test")
	settings := buildXunjiInitSettings("xunji-test", cfg)
	adapter.applyConfig(cfg, nil, settings)

	if adapter.broker != "tcp://localhost:1883" {
		t.Fatalf("broker=%q", adapter.broker)
	}
	if adapter.gatewayName != "GW_1" {
		t.Fatalf("gatewayName=%q, want=GW_1", adapter.gatewayName)
	}
	if adapter.topic != "v1/gateway/GW_1" || adapter.alarmTopic != "v1/gateway/GW_1/alarm" {
		t.Fatalf("topics mismatch: %q / %q", adapter.topic, adapter.alarmTopic)
	}
	if adapter.subDeviceTokenMode != "device_key" {
		t.Fatalf("subDeviceTokenMode=%q", adapter.subDeviceTokenMode)
	}
	if !adapter.initialized || !adapter.connected {
		t.Fatalf("state mismatch: initialized=%v connected=%v", adapter.initialized, adapter.connected)
	}
	if adapter.loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", adapter.loopState.String())
	}
}

func TestXunjiBuildBatchRealtimePayload(t *testing.T) {
	a := NewXunjiAdapter("xunji-test")
	a.subDeviceTokenMode = "device_key"
	ts1 := time.Unix(1700000000, 0)
	ts2 := time.Unix(1700000100, 0)
	body := a.buildBatchRealtimePayload([]*models.CollectData{
		{
			DeviceID:   1,
			DeviceName: "pump-1",
			DeviceKey:  "dk-1",
			Timestamp:  ts1,
			Fields: map[string]string{
				"temperature": "23.5",
				"running":     "true",
			},
		},
		{
			DeviceID:   2,
			DeviceName: "meter-2",
			DeviceKey:  "dk-2",
			Timestamp:  ts2,
			Fields: map[string]string{
				"pressure": "1.2",
			},
		},
	})

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	item1, ok := decoded["dk-1"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing sub token dk-1")
	}
	values1, ok := item1["values"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing values for dk-1")
	}
	if values1["temperature"] != 23.5 {
		t.Fatalf("temperature=%v, want=23.5", values1["temperature"])
	}
	if values1["running"] != true {
		t.Fatalf("running=%v, want=true", values1["running"])
	}
}

func TestXunjiSend_DropsOldestWhenQueueFull(t *testing.T) {
	a := NewXunjiAdapter("xunji-test")

	for i := 0; i < xunjiPendingDataCap+5; i++ {
		if err := a.Send(&models.CollectData{DeviceID: int64(i)}); err != nil {
			t.Fatalf("Send() error = %v", err)
		}
	}

	a.pendingMu.RLock()
	defer a.pendingMu.RUnlock()
	if len(a.pendingData) != xunjiPendingDataCap {
		t.Fatalf("len(pendingData)=%d, want=%d", len(a.pendingData), xunjiPendingDataCap)
	}
	if got := a.pendingData[0].DeviceID; got != 5 {
		t.Fatalf("oldest deviceID=%d, want=5", got)
	}
}

func TestXunjiAdapterStartStopCanRestart(t *testing.T) {
	a := NewXunjiAdapter("xunji-test")
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
