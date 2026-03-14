package adapters

import (
	"log"
	"sync"
)

type disconnectableClient interface {
	IsConnected() bool
	Disconnect(quiesce uint)
}

type adapterLifecycleState struct {
	adapterType    string
	logLabel       string
	adapterName    string
	mu             *sync.RWMutex
	wg             *sync.WaitGroup
	initialized    *bool
	enabled        *bool
	connected      *bool
	loopState      *adapterLoopState
	stopChan       *chan struct{}
	workSignalChan *chan struct{}
	reconnectChan  *chan struct{}
}

func (s adapterLifecycleState) start(runLoop func(), signalReconnect func()) {
	needReconnect := false
	transition := loopStateTransition{}

	s.mu.Lock()
	if *s.initialized && !*s.enabled && *s.loopState == adapterLoopStopped {
		if *s.stopChan == nil {
			*s.stopChan = make(chan struct{})
		}
		if s.workSignalChan != nil && *s.workSignalChan == nil {
			*s.workSignalChan = make(chan struct{}, 1)
		}
		if s.reconnectChan != nil && *s.reconnectChan == nil {
			*s.reconnectChan = make(chan struct{}, 1)
		}
		*s.enabled = true
		transition = updateLoopState(s.loopState, adapterLoopRunning)
		needReconnect = !*s.connected
		s.wg.Add(1)
		go runLoop()
		log.Printf("%s adapter started: %s", s.logLabel, s.adapterName)
	}
	s.mu.Unlock()
	logLoopStateTransition(s.adapterType, s.adapterName, transition)

	if needReconnect && signalReconnect != nil {
		signalReconnect()
	}
}

func (s adapterLifecycleState) stop() {
	transitionStopping := loopStateTransition{}
	transitionStopped := loopStateTransition{}

	s.mu.Lock()
	stopChan := *s.stopChan
	if *s.enabled {
		*s.enabled = false
		transitionStopping = updateLoopState(s.loopState, adapterLoopStopping)
		if stopChan != nil {
			close(stopChan)
		}
	}
	s.mu.Unlock()
	logLoopStateTransition(s.adapterType, s.adapterName, transitionStopping)

	s.wg.Wait()
	if stopChan != nil {
		s.mu.Lock()
		if *s.stopChan == stopChan {
			*s.stopChan = nil
		}
		transitionStopped = updateLoopState(s.loopState, adapterLoopStopped)
		s.mu.Unlock()
	}
	logLoopStateTransition(s.adapterType, s.adapterName, transitionStopped)
	log.Printf("%s adapter stopped: %s", s.logLabel, s.adapterName)
}

func (s adapterLifecycleState) close(flushData, flushAlarm func(), clearClient func() disconnectableClient) error {
	s.stop()

	if flushData != nil {
		flushData()
	}
	if flushAlarm != nil {
		flushAlarm()
	}

	transitionStopped := loopStateTransition{}
	var client disconnectableClient

	s.mu.Lock()
	*s.initialized = false
	*s.connected = false
	*s.enabled = false
	transitionStopped = updateLoopState(s.loopState, adapterLoopStopped)
	if clearClient != nil {
		client = clearClient()
	}
	*s.stopChan = nil
	if s.workSignalChan != nil {
		*s.workSignalChan = nil
	}
	if s.reconnectChan != nil {
		*s.reconnectChan = nil
	}
	s.mu.Unlock()
	logLoopStateTransition(s.adapterType, s.adapterName, transitionStopped)

	if client != nil && client.IsConnected() {
		client.Disconnect(250)
	}

	return nil
}
