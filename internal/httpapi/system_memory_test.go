package httpapi

import (
	"runtime"
	"testing"
	"time"
)

func TestReadProcStatusKB_ParsesVmRSS(t *testing.T) {
	kb, ok := readProcStatusKB("Name:\ttest\nVmRSS:\t   12345 kB\n", "VmRSS")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if kb != 12345 {
		t.Fatalf("kb = %d, want 12345", kb)
	}
}

func TestReadLastGCPauseMS_UsesLatestPauseSlot(t *testing.T) {
	memStats := &runtime.MemStats{
		NumGC: 3,
	}
	memStats.PauseNs[2] = 5 * uint64(time.Microsecond)

	got := readLastGCPauseMS(memStats)
	if got != 0.005 {
		t.Fatalf("readLastGCPauseMS() = %v, want 0.005", got)
	}
}
