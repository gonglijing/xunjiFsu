package adapters

import "testing"

func TestSignalStructChanNilSafe(t *testing.T) {
	signalStructChan(nil)
}

func TestSignalStructChanNonBlocking(t *testing.T) {
	ch := make(chan struct{}, 1)

	signalStructChan(ch)
	signalStructChan(ch)

	if got := len(ch); got != 1 {
		t.Fatalf("len(ch)=%d, want=1", got)
	}
}
