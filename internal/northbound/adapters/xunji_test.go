package adapters

import "testing"

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
	adapter := NewXunJiAdapter("xunji-test")
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
	adapter := NewXunJiAdapter("xunji-test")
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
