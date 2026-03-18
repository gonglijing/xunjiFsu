package app

import (
	"errors"
	"net/http"
	"testing"
	"time"

	appconfig "github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

func TestShouldLoadNorthboundConfig(t *testing.T) {
	if shouldLoadNorthboundConfig(nil) {
		t.Fatal("nil config should not be loaded")
	}
	if shouldLoadNorthboundConfig(&models.NorthboundConfig{Enabled: 0}) {
		t.Fatal("disabled config should not be loaded")
	}
	if !shouldLoadNorthboundConfig(&models.NorthboundConfig{Enabled: 1}) {
		t.Fatal("enabled config should be loaded")
	}
}

func TestBuildNorthboundConfigPayload(t *testing.T) {
	config := &models.NorthboundConfig{
		Type:      "mqtt",
		ServerURL: "tcp://127.0.0.1:1883",
		Topic:     "demo/topic",
	}

	got := buildNorthboundConfigPayload(config)
	if got == "" {
		t.Fatal("expected payload from model, got empty string")
	}

	config.Config = `{"server":"tcp://127.0.0.1:1883","topic":"schema-topic"}`
	got = buildNorthboundConfigPayload(config)
	if got != config.Config {
		t.Fatalf("buildNorthboundConfigPayload() = %q, want schema config", got)
	}
}

func TestShouldLoadDriver(t *testing.T) {
	if shouldLoadDriver(nil) {
		t.Fatal("nil driver should not be loaded")
	}
	if shouldLoadDriver(&models.Driver{Enabled: 0}) {
		t.Fatal("disabled driver should not be loaded")
	}
	if !shouldLoadDriver(&models.Driver{Enabled: 1}) {
		t.Fatal("enabled driver should be loaded")
	}
}

func TestResolveDriverFilePath(t *testing.T) {
	cfg := &appconfig.Config{DriversDir: "/tmp/drivers"}

	if got := resolveDriverFilePath(nil, nil); got != "" {
		t.Fatalf("resolveDriverFilePath(nil, nil) = %q, want empty", got)
	}

	driverModel := &models.Driver{Name: "demo"}
	if got := resolveDriverFilePath(cfg, driverModel); got != "/tmp/drivers/demo.wasm" {
		t.Fatalf("resolveDriverFilePath() = %q, want derived path", got)
	}

	driverModel.FilePath = "/custom/demo.wasm"
	if got := resolveDriverFilePath(cfg, driverModel); got != "/custom/demo.wasm" {
		t.Fatalf("resolveDriverFilePath() = %q, want existing file path", got)
	}
}

func TestBuildHTTPServer(t *testing.T) {
	cfg := &appconfig.Config{
		ListenAddr:       ":9090",
		HTTPReadTimeout:  10 * time.Second,
		HTTPWriteTimeout: 20 * time.Second,
		HTTPIdleTimeout:  30 * time.Second,
	}

	server := buildHTTPServer(cfg, http.NotFoundHandler())

	if server.Addr != ":9090" {
		t.Fatalf("server.Addr = %q, want :9090", server.Addr)
	}
	if server.ReadTimeout != 10*time.Second {
		t.Fatalf("server.ReadTimeout = %v, want 10s", server.ReadTimeout)
	}
	if server.WriteTimeout != 20*time.Second {
		t.Fatalf("server.WriteTimeout = %v, want 20s", server.WriteTimeout)
	}
	if server.IdleTimeout != 30*time.Second {
		t.Fatalf("server.IdleTimeout = %v, want 30s", server.IdleTimeout)
	}
	if server.Handler == nil {
		t.Fatal("server.Handler = nil, want non-nil")
	}
}

func TestResolveHTTPServerMode(t *testing.T) {
	tests := []struct {
		name string
		cfg  *appconfig.Config
		want httpServerMode
	}{
		{name: "nil config", cfg: nil, want: httpServerModePlain},
		{name: "plain http", cfg: &appconfig.Config{}, want: httpServerModePlain},
		{
			name: "auto tls",
			cfg:  &appconfig.Config{TLSAuto: true, TLSDomain: "gw.example.com"},
			want: httpServerModeAutoTLS,
		},
		{
			name: "manual tls",
			cfg:  &appconfig.Config{TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"},
			want: httpServerModeManualTLS,
		},
		{
			name: "auto tls wins over manual tls",
			cfg:  &appconfig.Config{TLSAuto: true, TLSDomain: "gw.example.com", TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"},
			want: httpServerModeAutoTLS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveHTTPServerMode(tt.cfg); got != tt.want {
				t.Fatalf("resolveHTTPServerMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapHTTPServerError(t *testing.T) {
	if err := wrapHTTPServerError(nil); err != nil {
		t.Fatalf("wrapHTTPServerError(nil) = %v, want nil", err)
	}
	if err := wrapHTTPServerError(http.ErrServerClosed); err != nil {
		t.Fatalf("wrapHTTPServerError(ErrServerClosed) = %v, want nil", err)
	}

	serveErr := errors.New("listen failed")
	wrapped := wrapHTTPServerError(serveErr)
	if wrapped == nil {
		t.Fatal("wrapHTTPServerError() = nil, want wrapped error")
	}
	if wrapped.Error() != "server error: listen failed" {
		t.Fatalf("wrapped.Error() = %q, want %q", wrapped.Error(), "server error: listen failed")
	}
}
