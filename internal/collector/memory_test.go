package collector

import "testing"

func TestParseSystemMemoryMB_UsesMemAvailable(t *testing.T) {
	total, used, available, ok := parseSystemMemoryMB(`MemTotal:       4096000 kB
MemFree:         512000 kB
MemAvailable:   1536000 kB
Buffers:         128000 kB
Cached:          256000 kB
`)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if total != 4000 {
		t.Fatalf("total = %v, want 4000", total)
	}
	if available != 1500 {
		t.Fatalf("available = %v, want 1500", available)
	}
	if used != 2500 {
		t.Fatalf("used = %v, want 2500", used)
	}
}

func TestParseSystemMemoryMB_FallsBackWithoutMemAvailable(t *testing.T) {
	total, used, available, ok := parseSystemMemoryMB(`MemTotal:       2048000 kB
MemFree:         256000 kB
Buffers:         128000 kB
Cached:          640000 kB
`)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if total != 2000 {
		t.Fatalf("total = %v, want 2000", total)
	}
	if available != 1000 {
		t.Fatalf("available = %v, want 1000", available)
	}
	if used != 1000 {
		t.Fatalf("used = %v, want 1000", used)
	}
}
