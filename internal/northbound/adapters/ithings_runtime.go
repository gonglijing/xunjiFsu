package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *IThingsAdapter) lifecycleState() adapterLifecycleState {
	return adapterLifecycleState{
		adapterType:    "ithings",
		logLabel:       "iThings",
		adapterName:    a.name,
		mu:             &a.mu,
		wg:             &a.wg,
		initialized:    &a.initialized,
		enabled:        &a.enabled,
		connected:      &a.connected,
		loopState:      &a.loopState,
		stopChan:       &a.stopChan,
		workSignalChan: &a.flushNow,
	}
}

func (a *IThingsAdapter) Start() {
	a.lifecycleState().start(a.runLoop, nil)
}

func (a *IThingsAdapter) Stop() {
	a.lifecycleState().stop()
}

func (a *IThingsAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = resolveInterval(int(interval.Milliseconds()), defaultReportInterval)
	a.mu.Unlock()
}

func (a *IThingsAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *IThingsAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

func (a *IThingsAdapter) Close() error {
	return a.lifecycleState().close(
		func() { _ = a.flushRealtime() },
		func() { _ = a.flushAlarmBatch() },
		func() disconnectableClient {
			client := a.client
			a.client = nil
			a.config = nil
			a.realtimeQueue = nil
			a.alarmQueue = nil
			a.commandQueue = nil
			a.requestStates = nil
			return client
		},
	)
}

func (a *IThingsAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	if limit <= 0 {
		limit = 20
	}

	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	if !a.isInitialized() {
		return nil, fmt.Errorf("adapter not initialized")
	}

	if len(a.commandQueue) == 0 {
		return nil, nil
	}

	if limit > len(a.commandQueue) {
		limit = len(a.commandQueue)
	}

	items := make([]*models.NorthboundCommand, limit)
	copy(items, a.commandQueue[:limit])
	clear(a.commandQueue[:limit])
	a.commandQueue = a.commandQueue[limit:]

	return items, nil
}

func (a *IThingsAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	if result == nil || strings.TrimSpace(result.RequestID) == "" {
		return nil
	}

	a.mu.RLock()
	initialized := a.initialized
	a.mu.RUnlock()
	if !initialized {
		return nil
	}

	state, ready := a.applyCommandResult(result)
	if !ready {
		return nil
	}

	code := state.Code
	if code == 0 {
		if state.Success {
			code = 0
		} else {
			code = 500
		}
	}
	message := strings.TrimSpace(state.Message)
	if message == "" {
		if state.Success {
			message = "success"
		} else {
			message = "failed"
		}
	}

	payload := map[string]interface{}{
		"msgToken":  state.RequestID,
		"code":      code,
		"msg":       message,
		"timestamp": time.Now().UnixMilli(),
	}

	if state.TopicType == "action" {
		payload["method"] = "actionReply"
		if strings.TrimSpace(state.ActionID) != "" {
			payload["actionID"] = state.ActionID
		} else if strings.TrimSpace(state.FieldName) != "" {
			payload["actionID"] = state.FieldName
		}
		if strings.TrimSpace(state.FieldName) != "" {
			payload["data"] = map[string]interface{}{
				state.FieldName: convertFieldValue(state.Value),
			}
		}
		topic := renderIThingsTopic(a.upActionTopicTemplate, state.ProductID, state.DeviceName)
		body, _ := json.Marshal(payload)
		return a.publish(topic, body)
	}

	payload["method"] = "controlReply"
	topic := renderIThingsTopic(a.upPropertyTopicTemplate, state.ProductID, state.DeviceName)
	body, _ := json.Marshal(payload)
	return a.publish(topic, body)
}

func (a *IThingsAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.dataMu.RLock()
	pendingData := len(a.realtimeQueue)
	a.dataMu.RUnlock()

	a.alarmMu.RLock()
	pendingAlarm := len(a.alarmQueue)
	a.alarmMu.RUnlock()

	a.commandMu.RLock()
	pendingCmd := len(a.commandQueue)
	a.commandMu.RUnlock()

	return map[string]interface{}{
		"name":                       a.name,
		"type":                       "ithings",
		"enabled":                    a.enabled,
		"initialized":                a.initialized,
		"connected":                  a.connected && a.client != nil && a.client.IsConnected(),
		"loop_state":                 a.loopState.String(),
		"interval_ms":                a.reportEvery.Milliseconds(),
		"pending_data":               pendingData,
		"pending_alarm":              pendingAlarm,
		"pending_cmd":                pendingCmd,
		"up_property_topic_template": a.upPropertyTopicTemplate,
		"down_property_topic":        a.downPropertyTopic,
		"down_action_topic":          a.downActionTopic,
	}
}

func (a *IThingsAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

func (a *IThingsAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}

func (a *IThingsAdapter) runLoop() {
	defer func() {
		a.mu.Lock()
		transition := updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
		logLoopStateTransition("ithings", a.name, transition)
		a.wg.Done()
	}()

	a.mu.RLock()
	reportInterval := a.reportEvery
	alarmInterval := a.alarmEvery
	flushNow := a.flushNow
	stopChan := a.stopChan
	a.mu.RUnlock()

	runBufferedFlushLoop(bufferedFlushLoopConfig{
		logLabel:       "iThings",
		reportLabel:    "realtime",
		reportInterval: reportInterval,
		alarmInterval:  alarmInterval,
		stopChan:       stopChan,
		flushNow:       flushNow,
		flushData: func() error {
			return a.flushRealtime()
		},
		flushAlarm: func() error {
			return a.flushAlarmBatch()
		},
		alarmQueueEmpty: func() bool {
			a.alarmMu.RLock()
			defer a.alarmMu.RUnlock()
			return len(a.alarmQueue) == 0
		},
	})
}
