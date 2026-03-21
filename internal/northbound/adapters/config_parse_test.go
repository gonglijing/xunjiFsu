package adapters

import "testing"

func TestParseAdapterRawConfig_PickValues(t *testing.T) {
	raw, err := parseAdapterRawConfig(`{
		" serverUrl ": "ignored",
		"serverUrl": " broker.example.com ",
		"protocol": " ssl ",
		"port": 8883,
		"username": " demo ",
		"retain": true
	}`)
	if err != nil {
		t.Fatalf("parseAdapterRawConfig() error = %v", err)
	}

	if got := raw.pickString("username"); got != "demo" {
		t.Fatalf("raw.pickString() = %q, want demo", got)
	}
	if got := raw.pickInt(0, "port"); got != 8883 {
		t.Fatalf("raw.pickInt() = %d, want 8883", got)
	}
	if got := raw.pickBool(false, "retain"); !got {
		t.Fatalf("raw.pickBool() = %v, want true", got)
	}
	if got := raw.pickNormalizedServerURL("serverUrl"); got != "ssl://broker.example.com:8883" {
		t.Fatalf("raw.pickNormalizedServerURL() = %q, want ssl://broker.example.com:8883", got)
	}
}

func TestParseAdapterRawConfig_InvalidJSON(t *testing.T) {
	if _, err := parseAdapterRawConfig(`{`); err == nil {
		t.Fatal("parseAdapterRawConfig() error = nil, want non-nil")
	}
}
