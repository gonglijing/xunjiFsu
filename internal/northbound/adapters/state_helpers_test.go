package adapters

import "testing"

func TestAdapterLoopStateString(t *testing.T) {
	if got := adapterLoopStopped.String(); got != "stopped" {
		t.Fatalf("stopped.String()=%q", got)
	}
	if got := adapterLoopRunning.String(); got != "running" {
		t.Fatalf("running.String()=%q", got)
	}
	if got := adapterLoopStopping.String(); got != "stopping" {
		t.Fatalf("stopping.String()=%q", got)
	}
}
