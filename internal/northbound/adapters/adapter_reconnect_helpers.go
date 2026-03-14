package adapters

import (
	"sync"
	"time"
)

type connectionStatus interface {
	IsConnected() bool
}

type adapterReconnectState struct {
	mu                *sync.RWMutex
	initialized       *bool
	enabled           *bool
	connected         *bool
	reconnectInterval *time.Duration
	reconnectNow      *chan struct{}
	client            func() connectionStatus
	normalizeInterval func(time.Duration) time.Duration
}

func normalizeReconnectInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return defaultReconnectInterval
	}
	return interval
}

func normalizePandaXReconnectInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return defaultPandaXReconnectInterval
	}
	if interval > maxPandaXReconnectInterval {
		return maxPandaXReconnectInterval
	}
	return interval
}

func (s adapterReconnectState) signalReconnect() {
	s.mu.RLock()
	var reconnectNow chan struct{}
	if s.reconnectNow != nil {
		reconnectNow = *s.reconnectNow
	}
	s.mu.RUnlock()
	signalStructChan(reconnectNow)
}

func (s adapterReconnectState) markDisconnected() {
	s.mu.Lock()
	*s.connected = false
	enabled := *s.enabled
	s.mu.Unlock()
	if enabled {
		s.signalReconnect()
	}
}

func (s adapterReconnectState) currentReconnectInterval() time.Duration {
	s.mu.RLock()
	interval := *s.reconnectInterval
	s.mu.RUnlock()

	if s.normalizeInterval != nil {
		return s.normalizeInterval(interval)
	}
	return normalizeReconnectInterval(interval)
}

func (s adapterReconnectState) shouldReconnect() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client := s.client()
	if !*s.initialized || !*s.enabled || client == nil {
		return false
	}
	return !client.IsConnected()
}

func (s adapterReconnectState) isConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client := s.client()
	return *s.connected && client != nil && client.IsConnected()
}
