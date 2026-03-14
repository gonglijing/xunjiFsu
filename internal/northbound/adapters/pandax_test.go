package adapters

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNormalizePandaXServerURL(t *testing.T) {
	tests := []struct {
		name     string
		server   string
		protocol string
		port     int
		want     string
	}{
		{
			name:   "host only with port",
			server: "127.0.0.1",
			port:   1883,
			want:   "tcp://127.0.0.1:1883",
		},
		{
			name:   "host with protocol and port",
			server: "tcp://127.0.0.1",
			port:   1883,
			want:   "tcp://127.0.0.1:1883",
		},
		{
			name:   "host with protocol and existing port",
			server: "tcp://127.0.0.1:2883",
			port:   1883,
			want:   "tcp://127.0.0.1:2883",
		},
		{
			name:   "host without protocol but custom protocol",
			server: "broker.example.com",
			port:   8883,
			want:   "tcp://broker.example.com:8883",
		},
		{
			name:     "host with ssl protocol",
			server:   "broker.example.com",
			protocol: "ssl",
			port:     8883,
			want:     "ssl://broker.example.com:8883",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := normalizePandaXServerURL(testCase.server, testCase.protocol, testCase.port)
			if got != testCase.want {
				t.Fatalf("normalizePandaXServerURL() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestParsePandaXConfig_UsesPortAndGatewayValidation(t *testing.T) {
	config := `{
		"serverUrl": "localhost",
		"port": 1883,
		"username": "token",
		"gatewayMode": true,
		"qos": 1
	}`

	cfg, err := parsePandaXConfig(config)
	if err != nil {
		t.Fatalf("parsePandaXConfig() error = %v", err)
	}
	if cfg.ServerURL != "tcp://localhost:1883" {
		t.Fatalf("unexpected server url: %q", cfg.ServerURL)
	}
	if !cfg.GatewayMode {
		t.Fatalf("expected gateway mode true")
	}
}

func TestParsePandaXConfig_GatewayModeFalseRejected(t *testing.T) {
	config := `{
		"serverUrl": "tcp://localhost:1883",
		"username": "token",
		"gatewayMode": false
	}`

	_, err := parsePandaXConfig(config)
	if err == nil {
		t.Fatalf("expected parsePandaXConfig() error")
	}
	if !strings.Contains(err.Error(), "gatewayMode=true") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPandaXReconnectDelay_BackoffAndCap(t *testing.T) {
	base := 5 * time.Second

	cases := []struct {
		name     string
		failures int
		want     time.Duration
	}{
		{name: "first retry uses base", failures: 1, want: 5 * time.Second},
		{name: "second retry doubles", failures: 2, want: 10 * time.Second},
		{name: "third retry doubles again", failures: 3, want: 20 * time.Second},
		{name: "capped at five minutes", failures: 8, want: 5 * time.Minute},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := pandaXReconnectDelay(base, testCase.failures)
			if got != testCase.want {
				t.Fatalf("pandaXReconnectDelay() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestPandaXAdapter_SetReconnectIntervalClamp(t *testing.T) {
	adapter := NewPandaXAdapter("test")

	adapter.SetReconnectInterval(0)
	if adapter.reconnectInterval != defaultPandaXReconnectInterval {
		t.Fatalf("reconnectInterval = %v, want %v", adapter.reconnectInterval, defaultPandaXReconnectInterval)
	}

	adapter.SetReconnectInterval(10 * time.Minute)
	if adapter.reconnectInterval != maxPandaXReconnectInterval {
		t.Fatalf("reconnectInterval = %v, want %v", adapter.reconnectInterval, maxPandaXReconnectInterval)
	}

	adapter.SetReconnectInterval(30 * time.Second)
	if adapter.reconnectInterval != 30*time.Second {
		t.Fatalf("reconnectInterval = %v, want %v", adapter.reconnectInterval, 30*time.Second)
	}
}

func TestParsePandaXConfig_Defaults(t *testing.T) {
	config := `{"serverUrl":"tcp://localhost:1883","username":"token","gatewayMode":true}`

	cfg, err := parsePandaXConfig(config)
	if err != nil {
		t.Fatalf("parsePandaXConfig() error = %v", err)
	}

	if cfg.UploadIntervalMs != int((5 * time.Second).Milliseconds()) {
		t.Fatalf("UploadIntervalMs=%d, want=%d", cfg.UploadIntervalMs, int((5 * time.Second).Milliseconds()))
	}
	if cfg.AlarmFlushIntervalMs != int((2 * time.Second).Milliseconds()) {
		t.Fatalf("AlarmFlushIntervalMs=%d, want=%d", cfg.AlarmFlushIntervalMs, int((2 * time.Second).Milliseconds()))
	}
	if cfg.CommandQueueSize != cfg.RealtimeQueueSize {
		t.Fatalf("CommandQueueSize=%d, want same as RealtimeQueueSize=%d", cfg.CommandQueueSize, cfg.RealtimeQueueSize)
	}
	if cfg.GatewayRegisterTopic != defaultPandaXGatewayRegisterTopic {
		t.Fatalf("GatewayRegisterTopic=%q, want=%q", cfg.GatewayRegisterTopic, defaultPandaXGatewayRegisterTopic)
	}
}

func TestBuildPandaXInitSettings_Defaults(t *testing.T) {
	cfg, err := parsePandaXConfig(`{"serverUrl":"127.0.0.1","port":1883,"username":"token","gatewayMode":true}`)
	if err != nil {
		t.Fatalf("parsePandaXConfig() error = %v", err)
	}

	settings := buildPandaXInitSettings("pandax-test", cfg)
	if settings.broker != "tcp://127.0.0.1:1883" {
		t.Fatalf("broker=%q, want=tcp://127.0.0.1:1883", settings.broker)
	}
	if settings.telemetryTopic != defaultPandaXTelemetryTopic {
		t.Fatalf("telemetryTopic=%q", settings.telemetryTopic)
	}
	if settings.gatewayRegisterTopic != defaultPandaXGatewayRegisterTopic {
		t.Fatalf("gatewayRegisterTopic=%q", settings.gatewayRegisterTopic)
	}
	if settings.alarmTopic != defaultPandaXEventPrefix+"/"+defaultPandaXAlarmIdentifier {
		t.Fatalf("alarmTopic=%q", settings.alarmTopic)
	}
	if settings.rpcRequestTopic != defaultPandaXRPCRequestTopic || settings.rpcResponseTopic != defaultPandaXRPCResponseTopic {
		t.Fatalf("rpc topics mismatch: %q / %q", settings.rpcRequestTopic, settings.rpcResponseTopic)
	}
	if settings.reportEvery != 5*time.Second || settings.alarmEvery != 2*time.Second {
		t.Fatalf("interval mismatch: report=%v alarm=%v", settings.reportEvery, settings.alarmEvery)
	}
	if !strings.HasPrefix(settings.clientID, "pandax-pandax-test-") {
		t.Fatalf("clientID=%q, want pandax-pandax-test-*", settings.clientID)
	}
}

func TestPandaXApplyConfig_SetsRuntimeFields(t *testing.T) {
	cfg, err := parsePandaXConfig(`{"serverUrl":"tcp://localhost:1883","username":"token","gatewayMode":true}`)
	if err != nil {
		t.Fatalf("parsePandaXConfig() error = %v", err)
	}

	adapter := NewPandaXAdapter("pandax-test")
	settings := buildPandaXInitSettings("pandax-test", cfg)
	adapter.applyConfig(cfg, nil, settings)

	if adapter.telemetryTopic != defaultPandaXTelemetryTopic {
		t.Fatalf("telemetryTopic=%q", adapter.telemetryTopic)
	}
	if adapter.gatewayRegisterTopic != defaultPandaXGatewayRegisterTopic {
		t.Fatalf("gatewayRegisterTopic=%q", adapter.gatewayRegisterTopic)
	}
	if adapter.rpcRequestTopic != defaultPandaXRPCRequestTopic || adapter.rpcResponseTopic != defaultPandaXRPCResponseTopic {
		t.Fatalf("rpc topics mismatch: %q / %q", adapter.rpcRequestTopic, adapter.rpcResponseTopic)
	}
	if adapter.flushNow == nil || adapter.stopChan == nil || adapter.reconnectNow == nil {
		t.Fatal("expected runtime channels initialized")
	}
	if adapter.enabled {
		t.Fatal("expected adapter disabled after applyConfig")
	}
	if !adapter.initialized || !adapter.connected {
		t.Fatalf("state mismatch: initialized=%v connected=%v", adapter.initialized, adapter.connected)
	}
	if adapter.loopState != adapterLoopStopped {
		t.Fatalf("loopState=%s, want=stopped", adapter.loopState.String())
	}
}

func TestPandaXBuildSyncDevicesPayload(t *testing.T) {
	adapter := NewPandaXAdapter("pandax-test")
	adapter.config = &PandaXConfig{
		SubDeviceTokenMode: "product_device_name",
	}
	adapter.gatewayRegisterTopic = "v1/gateway/register/telemetry"

	devices := []*models.Device{
		{ID: 1, Name: "pump-1", ProductKey: "prodA", DeviceKey: "devA"},
		{ID: 2, Name: "meter-2", ProductKey: "prodB", DeviceKey: "devB"},
	}
	latest := []*database.LatestDeviceData{
		{
			DeviceID:    1,
			DeviceName:  "pump-1",
			Fields:      map[string]string{"temp": "23.5", "running": "true"},
			CollectedAt: time.Unix(1700000000, 0),
		},
		{
			DeviceID:    2,
			DeviceName:  "meter-2",
			Fields:      map[string]string{"pressure": "1.8"},
			CollectedAt: time.Unix(1700000100, 0),
		},
	}

	topic, body, count, err := adapter.buildSyncDevicesPayload(devices, latest)
	if err != nil {
		t.Fatalf("buildSyncDevicesPayload() error = %v", err)
	}
	if topic != "v1/gateway/register/telemetry" {
		t.Fatalf("topic=%q", topic)
	}
	if count != 2 {
		t.Fatalf("count=%d, want=2", count)
	}

	decoded := make(map[string]interface{})
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	subDevices, ok := decoded["subDevices"].([]interface{})
	if !ok {
		t.Fatalf("subDevices missing")
	}
	if len(subDevices) != 2 {
		t.Fatalf("len(subDevices)=%d, want=2", len(subDevices))
	}

	first, _ := subDevices[0].(map[string]interface{})
	if _, exists := first["token"]; exists {
		t.Fatalf("token should be omitted in sync payload")
	}
	if first["productKey"] != "prodA" {
		t.Fatalf("first productKey=%v, want=prodA", first["productKey"])
	}
	values, ok := first["values"].(map[string]interface{})
	if !ok {
		t.Fatalf("first values missing")
	}
	if values["running"] != true {
		t.Fatalf("running=%v, want=true", values["running"])
	}
	if values["temp"] != 23.5 {
		t.Fatalf("temp=%v, want=23.5", values["temp"])
	}
	if _, exists := first["deviceKey"]; exists {
		t.Fatalf("deviceKey should be omitted in sync payload")
	}
	if _, exists := first["fields"]; exists {
		t.Fatalf("fields should be omitted in sync payload")
	}
}

func TestResolveSyncSubDeviceProductKey_PrioritizesResolved(t *testing.T) {
	device := &models.Device{ID: 7, ProductKey: "device-pk"}
	resolved := map[int64]string{7: "driver-pk"}

	if got := resolveSyncSubDeviceProductKey(device, resolved); got != "driver-pk" {
		t.Fatalf("resolveSyncSubDeviceProductKey()=%q, want=%q", got, "driver-pk")
	}

	if got := resolveSyncSubDeviceProductKey(device, nil); got != "device-pk" {
		t.Fatalf("resolveSyncSubDeviceProductKey()=%q, want=%q", got, "device-pk")
	}

	if got := resolveSyncSubDeviceProductKey(nil, resolved); got != "" {
		t.Fatalf("resolveSyncSubDeviceProductKey()=%q, want empty", got)
	}
}

func TestResolveDriverProductKey_UsesDriverField(t *testing.T) {
	driver := &models.Driver{ProductKey: "  fixed-driver-pk  "}
	if got := resolveDriverProductKey(driver); got != "fixed-driver-pk" {
		t.Fatalf("resolveDriverProductKey()=%q, want=%q", got, "fixed-driver-pk")
	}
}

func TestPandaXBuildSyncDevicesPayload_PrecheckFailsWhenProductHasNoFields(t *testing.T) {
	adapter := NewPandaXAdapter("pandax-test")
	adapter.config = &PandaXConfig{SubDeviceTokenMode: "product_device_name"}
	adapter.gatewayRegisterTopic = "v1/gateway/register/telemetry"

	devices := []*models.Device{
		{ID: 1, Name: "pump-1", ProductKey: "prodA", DeviceKey: "devA"},
	}

	_, _, _, err := adapter.buildSyncDevicesPayload(devices, nil)
	if err == nil {
		t.Fatal("expected precheck error")
	}
	if !strings.Contains(err.Error(), "prodA") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSortSyncSubDevices_ByProductThenDeviceName(t *testing.T) {
	subDevices := []pandaXSyncSubDevice{
		{ProductKey: "prodB", DeviceName: "zeta"},
		{ProductKey: "prodA", DeviceName: "zeta"},
		{ProductKey: "prodA", DeviceName: "alpha"},
	}

	sortSyncSubDevices(subDevices)

	if subDevices[0].ProductKey != "prodA" || subDevices[0].DeviceName != "alpha" {
		t.Fatalf("first subDevice = %+v", subDevices[0])
	}
	if subDevices[1].ProductKey != "prodA" || subDevices[1].DeviceName != "zeta" {
		t.Fatalf("second subDevice = %+v", subDevices[1])
	}
	if subDevices[2].ProductKey != "prodB" || subDevices[2].DeviceName != "zeta" {
		t.Fatalf("third subDevice = %+v", subDevices[2])
	}
}

func TestSyncTimestampOrNow_UsesFallbackForZeroTime(t *testing.T) {
	nowMS := int64(1700000000123)

	if got := syncTimestampOrNow(time.Time{}, nowMS); got != nowMS {
		t.Fatalf("syncTimestampOrNow()=%d, want=%d", got, nowMS)
	}

	ts := time.Unix(1700000000, 0)
	if got := syncTimestampOrNow(ts, nowMS); got != ts.UnixMilli() {
		t.Fatalf("syncTimestampOrNow()=%d, want=%d", got, ts.UnixMilli())
	}
}

func TestPandaXPullCommands_PopsInBatch(t *testing.T) {
	adapter := NewPandaXAdapter("pandax-test")
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

func TestIsPandaXReservedRPCKey(t *testing.T) {
	reserved := []string{"productKey", "device_key", "subDevices", "field_name", "value"}
	for _, key := range reserved {
		if !isPandaXReservedRPCKey(key) {
			t.Fatalf("expected reserved key: %q", key)
		}
	}

	notReserved := []string{"temperature", "method", "id", "custom_field"}
	for _, key := range notReserved {
		if isPandaXReservedRPCKey(key) {
			t.Fatalf("unexpected reserved key: %q", key)
		}
	}
}

type stubSystemStatsProvider struct {
	stats *models.SystemStats
}

func (s stubSystemStatsProvider) CollectSystemStatsOnce() *models.SystemStats {
	return s.stats
}

func TestFormatMetricFloat2(t *testing.T) {
	if got := formatMetricFloat2(1.234); got != "1.23" {
		t.Fatalf("formatMetricFloat2()=%q, want=1.23", got)
	}
	if got := formatMetricFloat2(1); got != "1.00" {
		t.Fatalf("formatMetricFloat2()=%q, want=1.00", got)
	}
}

func TestDefaultDeviceToken(t *testing.T) {
	if got := defaultDeviceToken(42); got != "device_42" {
		t.Fatalf("defaultDeviceToken()=%q, want=device_42", got)
	}
}

func TestFetchCurrentSystemStats_FormatValues(t *testing.T) {
	adapter := NewPandaXAdapter("pandax-test")
	adapter.systemStatsProvider = stubSystemStatsProvider{stats: &models.SystemStats{
		CpuUsage:     1.236,
		MemTotal:     1024.5,
		MemUsed:      600.1,
		MemUsage:     58.63,
		MemAvailable: 424.4,
		DiskTotal:    256.0,
		DiskUsed:     100.6,
		DiskUsage:    39.3,
		DiskFree:     155.4,
		Uptime:       123,
		Load1:        0.14,
		Load5:        0.28,
		Load15:       0.52,
		Timestamp:    1700000000000,
	}}

	data := adapter.fetchCurrentSystemStats()
	if data == nil {
		t.Fatal("fetchCurrentSystemStats() returned nil")
	}
	if data.Fields["cpu_usage"] != "1.24" {
		t.Fatalf("cpu_usage=%q, want=1.24", data.Fields["cpu_usage"])
	}
	if data.Fields["uptime"] != "123" {
		t.Fatalf("uptime=%q, want=123", data.Fields["uptime"])
	}
}

func TestRequestIDFromPandaXRPCTopic(t *testing.T) {
	cases := []struct {
		name  string
		topic string
		want  string
	}{
		{name: "normal", topic: "v1/devices/me/rpc/request/123", want: "123"},
		{name: "leading trailing slash", topic: "/v1/devices/me/rpc/request/abc/", want: "abc"},
		{name: "extra segments", topic: "v1/devices/me/rpc/request/xyz/extra", want: "xyz"},
		{name: "spaces", topic: "  /v1/devices/me/rpc/request/ 777 / ", want: "777"},
		{name: "invalid prefix", topic: "v1/devices/me/rpc/response/123", want: ""},
		{name: "missing request id", topic: "v1/devices/me/rpc/request", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := requestIDFromPandaXRPCTopic(tc.topic)
			if got != tc.want {
				t.Fatalf("requestIDFromPandaXRPCTopic()=%q, want=%q", got, tc.want)
			}
		})
	}
}

func TestPandaXSingleLoop_StopThenCloseSafe(t *testing.T) {
	adapter := NewPandaXAdapter("pandax-test")
	adapter.initialized = true
	adapter.connected = true
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
