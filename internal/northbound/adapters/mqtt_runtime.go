//go:build !no_paho_mqtt

package adapters

import (
	"time"
)

func (a *MQTTAdapter) SetReconnectInterval(interval time.Duration) {
	a.mu.Lock()
	if interval <= 0 {
		interval = defaultReconnectInterval
	}
	a.reconnectInterval = interval
	a.mu.Unlock()
}

func (a *MQTTAdapter) lifecycleState() adapterLifecycleState {
	return adapterLifecycleState{
		adapterType:    "mqtt",
		logLabel:       "MQTT",
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

func (a *MQTTAdapter) reconnectState() adapterReconnectState {
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

// Start 启动适配器的后台线程
func (a *MQTTAdapter) Start() {
	a.lifecycleState().start(a.runLoop, a.signalReconnect)
}

// Stop 停止适配器的后台线程
func (a *MQTTAdapter) Stop() {
	a.lifecycleState().stop()
}

// SetInterval 设置发送周期
func (a *MQTTAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	if interval < minUploadInterval {
		interval = minUploadInterval
	}
	a.interval = interval
	a.lastSend = time.Time{}
	a.mu.Unlock()
}

// IsEnabled 检查是否启用
func (a *MQTTAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// IsConnected 检查连接状态
func (a *MQTTAdapter) IsConnected() bool {
	return a.reconnectState().isConnected()
}

// Close 关闭
func (a *MQTTAdapter) Close() error {
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

// runLoop 单协程事件循环（数据/报警/重连）
func (a *MQTTAdapter) runLoop() {
	defer func() {
		a.mu.Lock()
		transition := updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
		logLoopStateTransition("mqtt", a.name, transition)
		a.wg.Done()
	}()

	a.mu.RLock()
	interval := a.interval
	stopChan := a.stopChan
	dataChan := a.dataChan
	reconnectNow := a.reconnectNow
	a.mu.RUnlock()

	runMQTTLikeLoop(mqttLikeLoopConfig{
		logLabel:     "MQTT",
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

// GetStats 获取适配器统计信息
func (a *MQTTAdapter) RuntimeStatsSnapshot() RuntimeStatsSnapshot {
	a.mu.RLock()
	enabled := a.enabled
	initialized := a.initialized
	connected := a.connected
	loopState := a.loopState
	interval := a.interval
	a.mu.RUnlock()

	a.pendingMu.RLock()
	pendingCount := len(a.pendingData)
	a.pendingMu.RUnlock()

	a.alarmMu.RLock()
	alarmCount := len(a.pendingAlarms)
	a.alarmMu.RUnlock()

	return RuntimeStatsSnapshot{
		Name:         a.name,
		Type:         "mqtt",
		Enabled:      enabled,
		Initialized:  initialized,
		Connected:    connected,
		LoopState:    loopState.String(),
		IntervalMS:   interval.Milliseconds(),
		PendingData:  pendingCount,
		PendingAlarm: alarmCount,
		Broker:       a.broker,
		Topic:        a.topic,
		AlarmTopic:   a.alarmTopic,
		ClientID:     a.clientID,
		QOS:          a.qos,
		Retain:       a.retain,
	}
}

func (a *MQTTAdapter) GetStats() map[string]interface{} {
	return a.RuntimeStatsSnapshot().ToMap()
}

// GetLastSendTime 获取最后发送时间
func (a *MQTTAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

// PendingCommandCount 获取待处理命令数量
func (a *MQTTAdapter) PendingCommandCount() int {
	return 0
}
