package app

import "testing"

func TestInitDatabases_NilConfig(t *testing.T) {
	err := initDatabases(nil)
	if err == nil {
		t.Fatal("initDatabases(nil) = nil, want error")
	}
	if err.Error() != "config is nil" {
		t.Fatalf("initDatabases(nil) = %q, want %q", err.Error(), "config is nil")
	}
}
