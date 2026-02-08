package adapters

import "testing"

func TestNormalizeServerURLWithPort(t *testing.T) {
	tests := []struct {
		name     string
		server   string
		protocol string
		port     int
		want     string
	}{
		{name: "empty", server: "", protocol: "tcp", port: 1883, want: ""},
		{name: "host only", server: "127.0.0.1", protocol: "tcp", port: 1883, want: "tcp://127.0.0.1:1883"},
		{name: "host with protocol", server: "tcp://127.0.0.1", protocol: "tcp", port: 1883, want: "tcp://127.0.0.1:1883"},
		{name: "host with existing port", server: "tcp://127.0.0.1:2883", protocol: "tcp", port: 1883, want: "tcp://127.0.0.1:2883"},
		{name: "ssl host", server: "broker.example.com", protocol: "ssl", port: 8883, want: "ssl://broker.example.com:8883"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := normalizeServerURLWithPort(testCase.server, testCase.protocol, testCase.port)
			if got != testCase.want {
				t.Fatalf("normalizeServerURLWithPort() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBuildBrokerURL_NoDoublePort(t *testing.T) {
	got := buildBrokerURL("tcp://example.com:1883", 2883)
	if got != "tcp://example.com:1883" {
		t.Fatalf("buildBrokerURL() = %q, want %q", got, "tcp://example.com:1883")
	}
}
