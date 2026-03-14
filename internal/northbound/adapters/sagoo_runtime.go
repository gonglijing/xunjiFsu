package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *SagooAdapter) lifecycleState() adapterLifecycleState {
	return adapterLifecycleState{
		adapterType:    "sagoo",
		logLabel:       "Sagoo",
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

// Start 启动适配器的后台线程
func (a *SagooAdapter) Start() {
	a.lifecycleState().start(a.runLoop, nil)
}

// Stop 停止适配器的后台线程
func (a *SagooAdapter) Stop() {
	a.lifecycleState().stop()
}

// SetInterval 设置发送周期
func (a *SagooAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = interval
	a.mu.Unlock()
}

// IsEnabled 检查是否启用
func (a *SagooAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

// IsConnected 检查连接状态
func (a *SagooAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

// Close 关闭
func (a *SagooAdapter) Close() error {
	return a.lifecycleState().close(
		func() { _ = a.flushLatestData() },
		func() { _ = a.flushAlarmBatch() },
		func() disconnectableClient {
			client := a.client
			a.client = nil
			a.config = nil
			a.latestData = nil
			a.alarmQueue = nil
			a.commandQueue = nil
			return client
		},
	)
}

// PullCommands 拉取待执行命令
func (a *SagooAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
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

	out := make([]*models.NorthboundCommand, limit)
	copy(out, a.commandQueue[:limit])
	clear(a.commandQueue[:limit])
	a.commandQueue = a.commandQueue[limit:]

	return out, nil
}

// ReportCommandResult 上报命令执行结果
func (a *SagooAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	if result == nil {
		return nil
	}

	a.mu.RLock()
	initialized := a.initialized
	cfg := a.config
	a.mu.RUnlock()

	if !initialized || cfg == nil {
		return nil
	}

	pk := pickFirstNonEmpty2(strings.TrimSpace(cfg.ProductKey), result.ProductKey)
	dk := pickFirstNonEmpty2(strings.TrimSpace(cfg.DeviceKey), result.DeviceKey)
	if pk == "" || dk == "" {
		return nil
	}

	code := result.Code
	if code == 0 && result.Success {
		code = 200
	}
	msg := result.Message
	if msg == "" && result.Success {
		msg = "success"
	}

	resp := map[string]interface{}{
		"code":    code,
		"id":      result.RequestID,
		"message": msg,
		"version": "1.0.0",
		"data": map[string]interface{}{
			result.FieldName: result.Value,
		},
	}
	body, _ := json.Marshal(resp)

	topic := sagooSysTopic(pk, dk, "thing/service/property/set_reply")
	return a.publish(topic, body)
}

// runLoop 单协程事件循环（实时与报警）
func (a *SagooAdapter) runLoop() {
	defer func() {
		a.mu.Lock()
		transition := updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
		logLoopStateTransition("sagoo", a.name, transition)
		a.wg.Done()
	}()

	a.mu.RLock()
	reportInterval := a.reportEvery
	alarmInterval := a.alarmEvery
	flushNow := a.flushNow
	stopChan := a.stopChan
	a.mu.RUnlock()

	runBufferedFlushLoop(bufferedFlushLoopConfig{
		logLabel:       "Sagoo",
		reportLabel:    "report",
		reportInterval: reportInterval,
		alarmInterval:  alarmInterval,
		stopChan:       stopChan,
		flushNow:       flushNow,
		flushData: func() error {
			return a.flushLatestData()
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

// GetStats 获取适配器统计信息
func (a *SagooAdapter) GetStats() map[string]interface{} {
	a.mu.RLock()
	productKey := ""
	deviceKey := ""
	if a.config != nil {
		productKey = strings.TrimSpace(a.config.ProductKey)
		deviceKey = strings.TrimSpace(a.config.DeviceKey)
	}
	defer a.mu.RUnlock()

	a.dataMu.RLock()
	dataCount := len(a.latestData)
	a.dataMu.RUnlock()

	a.alarmMu.RLock()
	alarmCount := len(a.alarmQueue)
	a.alarmMu.RUnlock()

	a.commandMu.RLock()
	commandCount := len(a.commandQueue)
	a.commandMu.RUnlock()

	return map[string]interface{}{
		"name":          a.name,
		"type":          "sagoo",
		"enabled":       a.enabled,
		"initialized":   a.initialized,
		"connected":     a.connected && a.client != nil && a.client.IsConnected(),
		"loop_state":    a.loopState.String(),
		"interval_ms":   a.reportEvery.Milliseconds(),
		"pending_data":  dataCount,
		"pending_alarm": alarmCount,
		"pending_cmd":   commandCount,
		"product_key":   productKey,
		"device_key":    deviceKey,
	}
}

// GetLastSendTime 获取最后发送时间（返回零值，因为是内部管理）
func (a *SagooAdapter) GetLastSendTime() time.Time {
	return time.Time{}
}

// PendingCommandCount 获取待处理命令数量
func (a *SagooAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}
