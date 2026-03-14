//go:build !no_paho_mqtt

package adapters

import (
	"time"
)

func (a *XunjiAdapter) lifecycleState() adapterLifecycleState {
	return adapterLifecycleState{
		adapterType:    "xunji",
		logLabel:       "Xunji",
		adapterName:    a.name,
		mu:             &a.mu,
		wg:             &a.wg,
		initialized:    &a.initialized,
		enabled:        &a.enabled,
		connected:      &a.connected,
		loopState:      &a.loopState,
		stopChan:       &a.stopChan,
		workSignalChan: &a.dataChan,
		reconnectChan:  &a.reconnectNow,
	}
}

func (a *XunjiAdapter) reconnectState() adapterReconnectState {
	return adapterReconnectState{
		mu:                &a.mu,
		initialized:       &a.initialized,
		enabled:           &a.enabled,
		connected:         &a.connected,
		reconnectInterval: &a.reconnectInterval,
		reconnectNow:      &a.reconnectNow,
		client: func() connectionStatus {
			return a.client
		},
		normalizeInterval: normalizeReconnectInterval,
	}
}

func (a *XunjiAdapter) Start() {
	a.lifecycleState().start(a.runLoop, a.signalReconnect)
}

func (a *XunjiAdapter) Stop() {
	a.lifecycleState().stop()
}

func (a *XunjiAdapter) Close() error {
	return a.lifecycleState().close(
		a.flushPendingData,
		func() { _ = a.flushAlarms() },
		func() disconnectableClient {
			client := a.client
			a.client = nil
			return client
		},
	)
}

func (a *XunjiAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	if interval < minUploadInterval {
		interval = minUploadInterval
	}
	a.interval = interval
	a.lastSend = time.Time{}
	a.mu.Unlock()
}

func (a *XunjiAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *XunjiAdapter) IsConnected() bool {
	return a.reconnectState().isConnected()
}

func (a *XunjiAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	enabled := a.enabled
	initialized := a.initialized
	connected := a.connected
	loopState := a.loopState
	interval := a.interval
	topic := a.topic
	alarmTopic := a.alarmTopic
	gatewayName := a.gatewayName
	a.mu.RUnlock()

	a.pendingMu.RLock()
	pendingCount := len(a.pendingData)
	a.pendingMu.RUnlock()

	a.alarmMu.RLock()
	alarmCount := len(a.pendingAlarms)
	a.alarmMu.RUnlock()

	return map[string]interface{}{
		"name":          a.name,
		"type":          "xunji",
		"enabled":       enabled,
		"initialized":   initialized,
		"connected":     connected,
		"loop_state":    loopState.String(),
		"interval_ms":   interval.Milliseconds(),
		"pending_data":  pendingCount,
		"pending_alarm": alarmCount,
		"topic":         topic,
		"alarm_topic":   alarmTopic,
		"gateway_name":  gatewayName,
	}
}

func (a *XunjiAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

func (a *XunjiAdapter) PendingCommandCount() int { return 0 }

func (a *XunjiAdapter) runLoop() {
	defer func() {
		a.mu.Lock()
		transition := updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
		logLoopStateTransition("xunji", a.name, transition)
		a.wg.Done()
	}()

	a.mu.RLock()
	interval := a.interval
	stopChan := a.stopChan
	dataChan := a.dataChan
	reconnectNow := a.reconnectNow
	a.mu.RUnlock()

	runMQTTLikeLoop(mqttLikeLoopConfig{
		logLabel:     "Xunji",
		adapterName:  a.name,
		interval:     interval,
		stopChan:     stopChan,
		dataChan:     dataChan,
		reconnectNow: reconnectNow,
		flushData: func() {
			a.flushPendingData()
		},
		flushAlarm: func() {
			_ = a.flushAlarms()
		},
		shouldReconnect: func() bool {
			return a.shouldReconnect()
		},
		reconnectOnce: func() error {
			return a.reconnectOnce()
		},
		reconnectDelay: func() time.Duration {
			return a.currentReconnectInterval()
		},
	})
}

func (a *XunjiAdapter) SetReconnectInterval(interval time.Duration) {
	a.mu.Lock()
	if interval <= 0 {
		interval = defaultReconnectInterval
	}
	a.reconnectInterval = interval
	a.mu.Unlock()
}
