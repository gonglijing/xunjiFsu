//go:build !no_extism

package driver

import "testing"

func TestParseDriverVersionOutput_UsesDataFallback(t *testing.T) {
	output := []byte(`{
		"success": true,
		"data": {
			"version": "1.2.3",
			"product_key": "prod-a"
		}
	}`)

	version, productKey, err := parseDriverVersionOutput(output)
	if err != nil {
		t.Fatalf("parseDriverVersionOutput() error = %v", err)
	}
	if version != "1.2.3" {
		t.Fatalf("version = %q, want 1.2.3", version)
	}
	if productKey != "prod-a" {
		t.Fatalf("productKey = %q, want prod-a", productKey)
	}
}

func TestParseDriverVersionOutput_FailsOnDriverError(t *testing.T) {
	output := []byte(`{
		"success": false,
		"error": "version not available"
	}`)

	_, _, err := parseDriverVersionOutput(output)
	if err == nil {
		t.Fatal("parseDriverVersionOutput() error = nil, want non-nil")
	}
	if err.Error() != "version not available" {
		t.Fatalf("parseDriverVersionOutput() error = %q, want %q", err.Error(), "version not available")
	}
}
