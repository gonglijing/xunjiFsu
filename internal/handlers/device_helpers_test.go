package handlers

import (
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestNormalizeWriteParams_ExplicitFieldAndValue(t *testing.T) {
	config := map[string]string{
		"field_name": "temperature",
		"value":      "26.5",
	}
	params := map[string]interface{}{
		"temperature": 26.5,
	}

	if err := normalizeWriteParams(config, params); err != nil {
		t.Fatalf("normalizeWriteParams returned error: %v", err)
	}

	if got := config["field_name"]; got != "temperature" {
		t.Fatalf("field_name = %q, want %q", got, "temperature")
	}
	if got := config["value"]; got != "26.5" {
		t.Fatalf("value = %q, want %q", got, "26.5")
	}
}

func TestNormalizeWriteParams_FromSingleParam(t *testing.T) {
	config := map[string]string{}
	params := map[string]interface{}{
		"setpoint": 30,
	}

	if err := normalizeWriteParams(config, params); err != nil {
		t.Fatalf("normalizeWriteParams returned error: %v", err)
	}

	if got := config["field_name"]; got != "setpoint" {
		t.Fatalf("field_name = %q, want %q", got, "setpoint")
	}
	if got := config["value"]; got != "30" {
		t.Fatalf("value = %q, want %q", got, "30")
	}
}

func TestNormalizeWriteParams_FromSubDeviceProperties(t *testing.T) {
	config := map[string]string{}
	params := map[string]interface{}{
		"subDevice": map[string]interface{}{
			"identity": map[string]interface{}{
				"productKey": "pk-1",
				"deviceKey":  "dk-1",
			},
			"properties": map[string]interface{}{
				"humidity": 55.2,
			},
		},
	}

	if err := normalizeWriteParams(config, params); err != nil {
		t.Fatalf("normalizeWriteParams returned error: %v", err)
	}

	if got := config["field_name"]; got != "humidity" {
		t.Fatalf("field_name = %q, want %q", got, "humidity")
	}
	if got := config["value"]; got != "55.2" {
		t.Fatalf("value = %q, want %q", got, "55.2")
	}
}

func TestNormalizeWriteParams_Ambiguous(t *testing.T) {
	config := map[string]string{}
	params := map[string]interface{}{
		"temperature": 25,
		"humidity":    60,
	}

	err := normalizeWriteParams(config, params)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %q, want contains ambiguous", err.Error())
	}
}

func TestEnrichExecuteIdentity_UsesDeviceIdentity(t *testing.T) {
	config := map[string]string{}
	device := &models.Device{ProductKey: "dev-pk", DeviceKey: "dev-dk"}

	enrichExecuteIdentity(config, device)

	if got := config["product_key"]; got != "dev-pk" {
		t.Fatalf("product_key = %q, want %q", got, "dev-pk")
	}
	if got := config["productKey"]; got != "dev-pk" {
		t.Fatalf("productKey = %q, want %q", got, "dev-pk")
	}
	if got := config["device_key"]; got != "dev-dk" {
		t.Fatalf("device_key = %q, want %q", got, "dev-dk")
	}
	if got := config["deviceKey"]; got != "dev-dk" {
		t.Fatalf("deviceKey = %q, want %q", got, "dev-dk")
	}
}
