package adapters

import (
	"strings"
	"testing"
	"time"
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
