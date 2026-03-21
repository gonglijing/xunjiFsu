package adapters

import "log/slog"

type adapterLoopState uint8

const (
	adapterLoopStopped adapterLoopState = iota
	adapterLoopRunning
	adapterLoopStopping
)

type loopStateTransition struct {
	from    adapterLoopState
	to      adapterLoopState
	changed bool
}

func (s adapterLoopState) String() string {
	switch s {
	case adapterLoopRunning:
		return "running"
	case adapterLoopStopping:
		return "stopping"
	default:
		return "stopped"
	}
}

func updateLoopState(state *adapterLoopState, next adapterLoopState) loopStateTransition {
	if state == nil {
		return loopStateTransition{}
	}

	from := *state
	if from == next {
		return loopStateTransition{from: from, to: next}
	}

	*state = next
	return loopStateTransition{from: from, to: next, changed: true}
}

func logLoopStateTransition(adapterType, adapterName string, transition loopStateTransition) {
	if !transition.changed {
		return
	}
	slog.Info("Northbound loop state transition", "type", adapterType, "name", adapterName, "from", transition.from.String(), "to", transition.to.String())
}
