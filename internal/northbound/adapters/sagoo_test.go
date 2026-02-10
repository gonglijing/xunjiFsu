package adapters

import (
	"testing"
	"time"
)

func TestExtractCommandProperties_DirectParams(t *testing.T) {
	params := map[string]interface{}{
		"temperature": 25.3,
		"enabled":     true,
	}

	properties, pk, dk := extractCommandProperties(params)
	if pk != "" || dk != "" {
		t.Fatalf("unexpected identity: pk=%q dk=%q", pk, dk)
	}
	if len(properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(properties))
	}
	if got, ok := properties["temperature"]; !ok || got != 25.3 {
		t.Fatalf("temperature property mismatch, got=%v", got)
	}
	if got, ok := properties["enabled"]; !ok || got != true {
		t.Fatalf("enabled property mismatch, got=%v", got)
	}
}

func TestExtractCommandProperties_DirectParamsWithRootIdentity(t *testing.T) {
	params := map[string]interface{}{
		"identity": map[string]interface{}{
			"productKey": "sub-pk",
			"deviceKey":  "sub-dk",
		},
		"voltage": 220,
	}

	properties, pk, dk := extractCommandProperties(params)
	if pk != "sub-pk" || dk != "sub-dk" {
		t.Fatalf("expected identity sub-pk/sub-dk, got %q/%q", pk, dk)
	}
	if len(properties) != 1 || properties["voltage"] != 220 {
		t.Fatalf("unexpected properties: %+v", properties)
	}
}

func TestEnqueueCommandFromPropertySet_UseRootIdentity(t *testing.T) {
	adapter := NewSagooAdapter("sagoo-test")
	adapter.commandCap = 10

	adapter.enqueueCommandFromPropertySet(
		"gw-pk",
		"gw-dk",
		"req-1",
		map[string]interface{}{"setPoint": 42},
		"sub-pk",
		"sub-dk",
	)

	if len(adapter.commandQueue) != 1 {
		t.Fatalf("expected 1 command, got %d", len(adapter.commandQueue))
	}
	command := adapter.commandQueue[0]
	if command.ProductKey != "sub-pk" || command.DeviceKey != "sub-dk" {
		t.Fatalf("expected sub identity, got %q/%q", command.ProductKey, command.DeviceKey)
	}
	if command.FieldName != "setPoint" || command.Value != "42" {
		t.Fatalf("unexpected command payload: %+v", command)
	}
}

func TestEnqueueCommandFromPropertySet_SubDevicesCompatible(t *testing.T) {
	adapter := NewSagooAdapter("sagoo-test")
	adapter.commandCap = 10

	adapter.enqueueCommandFromPropertySet(
		"gw-pk",
		"gw-dk",
		"req-2",
		map[string]interface{}{
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]interface{}{
						"productKey": "sub-pk-2",
						"deviceKey":  "sub-dk-2",
					},
					"properties": map[string]interface{}{
						"mode": "auto",
					},
				},
			},
		},
		"",
		"",
	)

	if len(adapter.commandQueue) != 1 {
		t.Fatalf("expected 1 command, got %d", len(adapter.commandQueue))
	}
	command := adapter.commandQueue[0]
	if command.ProductKey != "sub-pk-2" || command.DeviceKey != "sub-dk-2" {
		t.Fatalf("expected subDevices identity, got %q/%q", command.ProductKey, command.DeviceKey)
	}
	if command.FieldName != "mode" || command.Value != "auto" {
		t.Fatalf("unexpected command payload: %+v", command)
	}
}

func TestParseIdentityMap(t *testing.T) {
	pk, dk := parseIdentityMap(map[string]interface{}{
		"productKey": " pk ",
		"deviceKey":  " dk ",
	})
	if pk != "pk" || dk != "dk" {
		t.Fatalf("expected trimmed identity, got %q/%q", pk, dk)
	}
}

func TestParseSagooConfig_SnakeCaseCompatibility(t *testing.T) {
	config := `{
		"product_key": "pk",
		"device_key": "dk",
		"server_url": "127.0.0.1",
		"port": 1883,
		"client_id": "cid",
		"keep_alive": 30,
		"connect_timeout": 9,
		"upload_interval_ms": 3000,
		"alarm_flush_interval_ms": 1000,
		"alarm_batch_size": 10,
		"alarm_queue_size": 100,
		"realtime_queue_size": 200
	}`

	cfg, err := parseSagooConfig(config)
	if err != nil {
		t.Fatalf("parseSagooConfig() error = %v", err)
	}
	if cfg.ProductKey != "pk" || cfg.DeviceKey != "dk" {
		t.Fatalf("identity mismatch: %+v", cfg)
	}
	if cfg.ServerURL != "tcp://127.0.0.1:1883" {
		t.Fatalf("server url mismatch: %q", cfg.ServerURL)
	}
	if cfg.ClientID != "cid" {
		t.Fatalf("client id mismatch: %q", cfg.ClientID)
	}
	if cfg.KeepAlive != 30 || cfg.Timeout != 9 {
		t.Fatalf("mqtt options mismatch: keepAlive=%d timeout=%d", cfg.KeepAlive, cfg.Timeout)
	}
	if cfg.UploadIntervalMs != 3000 || cfg.AlarmFlushIntervalMs != 1000 {
		t.Fatalf("interval mismatch: upload=%d alarm=%d", cfg.UploadIntervalMs, cfg.AlarmFlushIntervalMs)
	}
	if cfg.AlarmBatchSize != 10 || cfg.AlarmQueueSize != 100 || cfg.RealtimeQueueSize != 200 {
		t.Fatalf("queue mismatch: batch=%d alarmQueue=%d realtimeQueue=%d", cfg.AlarmBatchSize, cfg.AlarmQueueSize, cfg.RealtimeQueueSize)
	}
}

func TestParseSagooConfig_Defaults(t *testing.T) {
	config := `{"productKey":"pk","deviceKey":"dk","serverUrl":"tcp://127.0.0.1:1883"}`
	cfg, err := parseSagooConfig(config)
	if err != nil {
		t.Fatalf("parseSagooConfig() error = %v", err)
	}

	if cfg.KeepAlive != 60 || cfg.Timeout != 10 {
		t.Fatalf("default mqtt options mismatch: keepAlive=%d timeout=%d", cfg.KeepAlive, cfg.Timeout)
	}
	if cfg.UploadIntervalMs != int((5 * time.Second).Milliseconds()) {
		t.Fatalf("default upload interval mismatch: %d", cfg.UploadIntervalMs)
	}
	if cfg.AlarmFlushIntervalMs != int((2 * time.Second).Milliseconds()) {
		t.Fatalf("default alarm flush mismatch: %d", cfg.AlarmFlushIntervalMs)
	}
}
