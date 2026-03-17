package database

import "testing"

func TestNorthboundLastConnectedValue(t *testing.T) {
	if got := northboundLastConnectedValue(false); got != nil {
		t.Fatalf("northboundLastConnectedValue(false) = %#v, want nil", got)
	}

	got := northboundLastConnectedValue(true)
	if got == nil {
		t.Fatal("northboundLastConnectedValue(true) = nil, want timestamp pointer")
	}
}
