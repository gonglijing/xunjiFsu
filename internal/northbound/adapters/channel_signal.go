package adapters

func signalStructChan(ch chan struct{}) {
	if ch == nil {
		return
	}

	select {
	case ch <- struct{}{}:
	default:
	}
}
