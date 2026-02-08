package handlers

import (
	"strings"
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestIsGatewayIdentityNorthboundType(t *testing.T) {
	tests := []struct {
		name   string
		nbType string
		want   bool
	}{
		{name: "xunji", nbType: "xunji", want: true},
		{name: "pandax", nbType: "pandax", want: true},
		{name: "ithings mixed case", nbType: "iThings", want: true},
		{name: "mqtt", nbType: "mqtt", want: false},
		{name: "empty", nbType: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGatewayIdentityNorthboundType(tt.nbType); got != tt.want {
				t.Fatalf("isGatewayIdentityNorthboundType(%q) = %v, want %v", tt.nbType, got, tt.want)
			}
		})
	}
}

func TestHasSchemaConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *models.NorthboundConfig
		want bool
	}{
		{name: "nil", cfg: nil, want: false},
		{name: "empty", cfg: &models.NorthboundConfig{Config: ""}, want: false},
		{name: "empty object", cfg: &models.NorthboundConfig{Config: "{}"}, want: false},
		{name: "empty object with spaces", cfg: &models.NorthboundConfig{Config: "  {}  "}, want: false},
		{name: "non empty", cfg: &models.NorthboundConfig{Config: `{"server":"mqtt://a"}`}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSchemaConfig(tt.cfg); got != tt.want {
				t.Fatalf("hasSchemaConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateNorthboundConfig_RequiredFieldsFallback(t *testing.T) {
	config := &models.NorthboundConfig{
		Name: "demo",
		Type: "MQTT",
	}

	err := validateNorthboundConfig(config)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "server_url or config is required for MQTT type") {
		t.Fatalf("error = %q, want required server_url error", err.Error())
	}
}

func TestValidateNorthboundConfig_SchemaConfigBypassesLegacyRequiredFields(t *testing.T) {
	config := &models.NorthboundConfig{
		Name:   "demo",
		Type:   "mqtt",
		Config: `{"broker":"tcp://127.0.0.1:1883","topic":"demo"}`,
	}

	if err := validateNorthboundConfig(config); err != nil {
		t.Fatalf("validateNorthboundConfig returned error: %v", err)
	}
}
