package handlers

import (
	"testing"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/config"
)

func TestParseOptionalDuration(t *testing.T) {
	if _, ok, err := parseOptionalDuration(""); err != nil || ok {
		t.Fatalf("empty should return (ok=false, err=nil), got ok=%v err=%v", ok, err)
	}

	if got, ok, err := parseOptionalDuration("150ms"); err != nil || !ok || got != 150*time.Millisecond {
		t.Fatalf("parse 150ms failed, got (%v, %v, %v)", got, ok, err)
	}

	if _, _, err := parseOptionalDuration("bad"); err == nil {
		t.Fatalf("expected parse error for invalid duration")
	}

	if _, _, err := parseOptionalDuration("0s"); err == nil {
		t.Fatalf("expected parse error for non-positive duration")
	}
}

func TestApplyGatewayRuntimeConfig_NegativeRetries(t *testing.T) {
	negative := -1
	h := &Handler{}
	h.appConfig = &config.Config{}

	_, err := h.applyGatewayRuntimeConfig(&gatewayRuntimeConfig{DriverSerialOpenRetries: &negative})
	if err == nil {
		t.Fatalf("expected error for negative driver_serial_open_retries")
	}

	_, err = h.applyGatewayRuntimeConfig(&gatewayRuntimeConfig{DriverTCPDialRetries: &negative})
	if err == nil {
		t.Fatalf("expected error for negative driver_tcp_dial_retries")
	}
}
