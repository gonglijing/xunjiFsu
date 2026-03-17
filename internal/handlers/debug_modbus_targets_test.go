package handlers

import (
	"strings"
	"testing"
)

func TestResolveModbusTCPTarget_DirectEndpoint(t *testing.T) {
	endpoint, fromResource, err := resolveModbusTCPTarget(" 127.0.0.1:502 ", nil)
	if err != nil {
		t.Fatalf("resolveModbusTCPTarget() error = %v", err)
	}
	if fromResource {
		t.Fatal("fromResource = true, want false")
	}
	if endpoint != "127.0.0.1:502" {
		t.Fatalf("endpoint = %q, want %q", endpoint, "127.0.0.1:502")
	}
}

func TestValidateResolvedModbusTCPEndpoint(t *testing.T) {
	if err := validateResolvedModbusTCPEndpoint("127.0.0.1:502", false); err != nil {
		t.Fatalf("validateResolvedModbusTCPEndpoint() error = %v", err)
	}
}

func TestValidateResolvedModbusTCPEndpoint_FromResource(t *testing.T) {
	err := validateResolvedModbusTCPEndpoint("bad-endpoint", true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "resource path is invalid") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "resource path is invalid")
	}
}
