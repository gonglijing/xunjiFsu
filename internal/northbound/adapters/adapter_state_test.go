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

func TestUpdateLoopState(t *testing.T) {
	state := adapterLoopStopped

	transition := updateLoopState(&state, adapterLoopRunning)
	if !transition.changed {
		t.Fatal("expected changed transition")
	}
	if transition.from != adapterLoopStopped || transition.to != adapterLoopRunning {
		t.Fatalf("unexpected transition: %+v", transition)
	}
	if state != adapterLoopRunning {
		t.Fatalf("state=%s, want=running", state.String())
	}

	transition = updateLoopState(&state, adapterLoopRunning)
	if transition.changed {
		t.Fatal("expected unchanged transition")
	}
	if transition.from != adapterLoopRunning || transition.to != adapterLoopRunning {
		t.Fatalf("unexpected unchanged transition: %+v", transition)
	}
}
