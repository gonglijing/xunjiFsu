package adapters

type adapterLoopState uint8

const (
	adapterLoopStopped adapterLoopState = iota
	adapterLoopRunning
	adapterLoopStopping
)

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
