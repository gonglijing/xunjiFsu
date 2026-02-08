package handlers

import (
	"testing"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestShouldSyncGatewayIdentity(t *testing.T) {
	tests := []struct {
		name string
		cfg  *models.NorthboundConfig
		want bool
	}{
		{name: "nil", cfg: nil, want: false},
		{name: "xunji", cfg: &models.NorthboundConfig{Type: "xunji"}, want: true},
		{name: "mqtt", cfg: &models.NorthboundConfig{Type: "mqtt"}, want: false},
		{name: "ithings mixed", cfg: &models.NorthboundConfig{Type: " iThings "}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSyncGatewayIdentity(tt.cfg); got != tt.want {
				t.Fatalf("shouldSyncGatewayIdentity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractGatewayIdentity(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *models.GatewayConfig
		wantProduct   string
		wantDevice    string
		wantAvailable bool
	}{
		{name: "nil", cfg: nil, wantAvailable: false},
		{name: "missing", cfg: &models.GatewayConfig{ProductKey: "pk", DeviceKey: ""}, wantAvailable: false},
		{name: "ok", cfg: &models.GatewayConfig{ProductKey: " pk ", DeviceKey: " dk "}, wantProduct: "pk", wantDevice: "dk", wantAvailable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			productKey, deviceKey, ok := extractGatewayIdentity(toDatabaseGatewayConfig(tt.cfg))
			if ok != tt.wantAvailable {
				t.Fatalf("ok = %v, want %v", ok, tt.wantAvailable)
			}
			if productKey != tt.wantProduct {
				t.Fatalf("productKey = %q, want %q", productKey, tt.wantProduct)
			}
			if deviceKey != tt.wantDevice {
				t.Fatalf("deviceKey = %q, want %q", deviceKey, tt.wantDevice)
			}
		})
	}
}
