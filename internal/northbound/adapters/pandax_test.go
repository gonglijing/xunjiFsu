package adapters

import (
	"strings"
	"testing"
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
