package adapters

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type pandaXCommandResultParams struct {
	Success    bool               `json:"success"`
	Code       int                `json:"code"`
	Message    string             `json:"message"`
	ProductKey string             `json:"productKey"`
	DeviceKey  string             `json:"deviceKey"`
	FieldName  string             `json:"fieldName"`
	Value      jsonConvertedValue `json:"value"`
}

type pandaXCommandResultPayload struct {
	RequestID string                    `json:"requestId"`
	Method    string                    `json:"method"`
	Params    pandaXCommandResultParams `json:"params"`
}

func (a *PandaXAdapter) Start() {
	needReconnect := false
	transition := loopStateTransition{}
	a.mu.Lock()
	if a.initialized && !a.enabled && a.loopState == adapterLoopStopped {
		a.enabled = true
		transition = updateLoopState(&a.loopState, adapterLoopRunning)
		if a.stopChan == nil {
			a.stopChan = make(chan struct{})
		}
		if a.flushNow == nil {
			a.flushNow = make(chan struct{}, 1)
		}
		if a.reconnectNow == nil {
			a.reconnectNow = make(chan struct{}, 1)
		}
		needReconnect = !a.connected
		a.wg.Add(1)
		go a.executeLoop()
		slog.Info("PandaX adapter started", "adapter", a.name, "report_interval", a.reportEvery, "alarm_interval", a.alarmEvery)
	}
	a.mu.Unlock()
	logLoopStateTransition("pandax", a.name, transition)

	if needReconnect {
		a.signalReconnect()
	}
}

func (a *PandaXAdapter) Stop() {
	transitionStopping := loopStateTransition{}
	transitionStopped := loopStateTransition{}

	a.mu.Lock()
	stopChan := a.stopChan
	if a.enabled {
		a.enabled = false
		transitionStopping = updateLoopState(&a.loopState, adapterLoopStopping)
		if stopChan != nil {
			close(stopChan)
		}
		slog.Info("PandaX adapter stopped", "adapter", a.name)
	}
	a.mu.Unlock()
	logLoopStateTransition("pandax", a.name, transitionStopping)

	a.wg.Wait()
	if stopChan != nil {
		a.mu.Lock()
		if a.stopChan == stopChan {
			a.stopChan = nil
		}
		transitionStopped = updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
	}
	logLoopStateTransition("pandax", a.name, transitionStopped)
}

func (a *PandaXAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = resolveInterval(int(interval.Milliseconds()), defaultReportInterval)
	a.mu.Unlock()
}

func (a *PandaXAdapter) reconnectState() adapterReconnectState {
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
		normalizeInterval: normalizePandaXReconnectInterval,
	}
}

func (a *PandaXAdapter) SetReconnectInterval(interval time.Duration) {
	a.mu.Lock()
	a.reconnectInterval = normalizePandaXReconnectInterval(interval)
	a.mu.Unlock()
}

func (a *PandaXAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *PandaXAdapter) IsConnected() bool {
	return a.reconnectState().isConnected()
}

func (a *PandaXAdapter) Close() error {
	slog.Info("PandaX close start", "adapter", a.name)

	a.Stop()

	transitionStopped := loopStateTransition{}
	a.mu.Lock()
	client := a.client
	a.initialized = false
	a.connected = false
	a.enabled = false
	transitionStopped = updateLoopState(&a.loopState, adapterLoopStopped)
	a.mu.Unlock()
	logLoopStateTransition("pandax", a.name, transitionStopped)

	_ = a.flushRealtime()
	_ = a.flushAlarmBatch()

	if client != nil && client.IsConnected() {
		slog.Info("PandaX close disconnect MQTT", "adapter", a.name)
		client.Disconnect(250)
	}

	a.mu.Lock()
	a.client = nil
	a.config = nil
	a.flushNow = nil
	a.stopChan = nil
	a.reconnectNow = nil
	a.realtimeQueue = nil
	a.alarmQueue = nil
	a.commandQueue = nil
	a.mu.Unlock()

	slog.Info("PandaX close completed", "adapter", a.name)
	return nil
}

func (a *PandaXAdapter) ReportCommandResult(result *models.NorthboundCommandResult) error {
	if result == nil || strings.TrimSpace(result.RequestID) == "" {
		return nil
	}

	a.mu.RLock()
	initialized := a.initialized
	topic := a.rpcResponseTopic
	a.mu.RUnlock()
	if !initialized {
		return nil
	}

	code := result.Code
	if code == 0 {
		if result.Success {
			code = 200
		} else {
			code = 500
		}
	}
	message := strings.TrimSpace(result.Message)
	if message == "" {
		if result.Success {
			message = "success"
		} else {
			message = "failed"
		}
	}

	resp := pandaXCommandResultPayload{
		RequestID: result.RequestID,
		Method:    "write",
		Params: pandaXCommandResultParams{
			Success:    result.Success,
			Code:       code,
			Message:    message,
			ProductKey: result.ProductKey,
			DeviceKey:  result.DeviceKey,
			FieldName:  result.FieldName,
			Value:      jsonConvertedValue(result.Value),
		},
	}
	body, _ := json.Marshal(resp)

	return a.publish(topic, body)
}

func (a *PandaXAdapter) RuntimeStatsSnapshot() RuntimeStatsSnapshot {
	a.mu.RLock()
	enabled := a.enabled
	initialized := a.initialized
	connected := a.connected && a.client != nil && a.client.IsConnected()
	loopState := a.loopState.String()
	intervalMS := a.reportEvery.Milliseconds()
	telemetryTopic := a.telemetryTopic
	gatewayTelemetryTopic := a.gatewayTelemetryTopic
	gatewayRegisterTopic := a.gatewayRegisterTopic
	rpcRequestTopic := a.rpcRequestTopic
	rpcResponseTopic := a.rpcResponseTopic
	a.mu.RUnlock()

	a.dataMu.RLock()
	pendingData := len(a.realtimeQueue)
	a.dataMu.RUnlock()

	a.alarmMu.RLock()
	pendingAlarm := len(a.alarmQueue)
	a.alarmMu.RUnlock()

	a.commandMu.RLock()
	pendingCmd := len(a.commandQueue)
	a.commandMu.RUnlock()

	return RuntimeStatsSnapshot{
		Name:                  a.name,
		Type:                  "pandax",
		Enabled:               enabled,
		Initialized:           initialized,
		Connected:             connected,
		LoopState:             loopState,
		IntervalMS:            intervalMS,
		PendingData:           pendingData,
		PendingAlarm:          pendingAlarm,
		PendingCmd:            pendingCmd,
		TelemetryTopic:        telemetryTopic,
		GatewayTelemetryTopic: gatewayTelemetryTopic,
		GatewayRegisterTopic:  gatewayRegisterTopic,
		RPCRequestTopic:       rpcRequestTopic,
		RPCResponseTopic:      rpcResponseTopic,
	}
}

func (a *PandaXAdapter) GetStats() map[string]any {
	return a.RuntimeStatsSnapshot().ToMap()
}

func (a *PandaXAdapter) GetLastSendTime() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastSend
}

func (a *PandaXAdapter) PendingCommandCount() int {
	a.commandMu.RLock()
	defer a.commandMu.RUnlock()
	return len(a.commandQueue)
}

func (a *PandaXAdapter) executeLoop() {
	defer func() {
		a.mu.Lock()
		transition := updateLoopState(&a.loopState, adapterLoopStopped)
		a.mu.Unlock()
		logLoopStateTransition("pandax", a.name, transition)
		a.wg.Done()
	}()

	a.mu.RLock()
	reportInterval := a.reportEvery
	alarmInterval := a.alarmEvery
	flushNow := a.flushNow
	reconnectNow := a.reconnectNow
	stopChan := a.stopChan
	a.mu.RUnlock()

	if reportInterval <= 0 {
		reportInterval = defaultReportInterval
	}
	if alarmInterval <= 0 {
		alarmInterval = defaultAlarmInterval
	}

	slog.Info("PandaX run loop started", "adapter", a.name, "report_interval", reportInterval, "alarm_interval", alarmInterval)

	reportTicker := time.NewTicker(reportInterval)
	alarmTicker := time.NewTicker(alarmInterval)
	defer reportTicker.Stop()
	defer alarmTicker.Stop()

	var reconnect reconnectScheduler
	reconnectFailures := 0

	stopReconnect := func() {
		reconnect.Stop()
		reconnectFailures = 0
	}
	defer reconnect.Close()

	for {
		select {
		case <-stopChan:
			slog.Info("PandaX run loop stopping and draining alarms", "adapter", a.name)
			stopReconnect()
			drainAlarmQueueOnStop(alarmDrainConfig{
				flushAlarm: func() error {
					return a.flushAlarmBatch()
				},
				alarmQueueEmpty: func() bool {
					a.alarmMu.RLock()
					defer a.alarmMu.RUnlock()
					return len(a.alarmQueue) == 0
				},
				onFlushError: func(err error) {
					slog.Info("PandaX run loop alarm flush failed", "adapter", a.name, "error", err)
				},
				onDrained: func() {
					slog.Info("PandaX run loop exited", "adapter", a.name)
				},
			})
			return
		case <-reportTicker.C:
			if err := a.fetchAndPublishLatestData(); err != nil {
				slog.Info("PandaX run loop fetch and publish failed", "adapter", a.name, "error", err)
			}
		case <-alarmTicker.C:
			if err := a.flushAlarmBatch(); err != nil {
				slog.Info("PandaX run loop alarm flush failed", "adapter", a.name, "error", err)
			}
		case <-flushNow:
			if err := a.flushAlarmBatch(); err != nil {
				slog.Info("PandaX run loop alarm flush failed", "adapter", a.name, "error", err)
			}
		case <-reconnectNow:
			if a.shouldReconnect() {
				reconnectFailures = 0
				reconnect.Schedule(0)
			}
		case <-reconnect.Channel():
			if !a.shouldReconnect() {
				stopReconnect()
				continue
			}
			if err := a.reconnectOnce(); err != nil {
				reconnectFailures++
				delay := pandaXReconnectDelay(a.currentReconnectInterval(), reconnectFailures)
				slog.Info("PandaX reconnect failed",
					"adapter", a.name,
					"attempt", reconnectFailures,
					"error", err,
					"retry_in", delay)
				reconnect.Schedule(delay)
				continue
			}
			slog.Info("PandaX reconnect succeeded", "adapter", a.name)
			stopReconnect()
		}
	}
}

func (a *PandaXAdapter) fetchAndPublishLatestData() error {
	devices, err := database.GetAllDevicesLatestData()
	if err != nil {
		return fmt.Errorf("获取设备最新数据失败: %w", err)
	}

	slog.Info("PandaX latest data loaded", "adapter", a.name, "count", len(devices))

	successCount := 0
	systemStatsCount := 0

	for _, dev := range devices {
		if dev == nil || len(dev.Fields) == 0 {
			continue
		}

		isSystemStats := dev.DeviceID == models.SystemStatsDeviceID
		data := &models.CollectData{
			DeviceID:   dev.DeviceID,
			DeviceName: dev.DeviceName,
			Timestamp:  dev.CollectedAt,
			Fields:     dev.Fields,
		}

		a.dataMu.Lock()
		a.enqueueRealtimeLocked(data)
		a.dataMu.Unlock()

		if isSystemStats {
			slog.Info("PandaX system stats enqueued",
				"adapter", a.name,
				"device_id", dev.DeviceID,
				"device_name", dev.DeviceName,
				"fields", len(dev.Fields))
			systemStatsCount++
		} else {
			slog.Info("PandaX device data enqueued",
				"adapter", a.name,
				"device_id", dev.DeviceID,
				"device_name", dev.DeviceName,
				"fields", len(dev.Fields))
			successCount++
		}
	}

	if systemStatsCount == 0 {
		if sysData := a.fetchCurrentSystemStats(); sysData != nil {
			a.dataMu.Lock()
			a.enqueueRealtimeLocked(sysData)
			queueLen := len(a.realtimeQueue)
			a.dataMu.Unlock()
			slog.Info("PandaX current system stats enqueued",
				"adapter", a.name,
				"fields", len(sysData.Fields),
				"queue_len", queueLen)
		}
	}

	if err := a.flushRealtime(); err != nil {
		slog.Info("PandaX latest data flush failed", "adapter", a.name, "error", err)
	}

	slog.Info("PandaX latest data publish completed",
		"adapter", a.name,
		"devices", successCount,
		"system_stats", systemStatsCount)
	return nil
}

func (a *PandaXAdapter) fetchCurrentSystemStats() *models.CollectData {
	a.mu.RLock()
	provider := a.systemStatsProvider
	a.mu.RUnlock()

	if provider == nil {
		slog.Info("PandaX current system stats provider missing", "adapter", a.name)
		return nil
	}

	stats := provider.CollectSystemStatsOnce()
	if stats == nil {
		return nil
	}

	return &models.CollectData{
		DeviceID:   models.SystemStatsDeviceID,
		DeviceName: models.SystemStatsDeviceName,
		Timestamp:  time.Unix(0, stats.Timestamp*int64(time.Millisecond)),
		Fields: map[string]string{
			"cpu_usage":     formatMetricFloat(stats.CpuUsage),
			"mem_total":     formatMetricFloat(stats.MemTotal),
			"mem_used":      formatMetricFloat(stats.MemUsed),
			"mem_usage":     formatMetricFloat(stats.MemUsage),
			"mem_available": formatMetricFloat(stats.MemAvailable),
			"disk_total":    formatMetricFloat(stats.DiskTotal),
			"disk_used":     formatMetricFloat(stats.DiskUsed),
			"disk_usage":    formatMetricFloat(stats.DiskUsage),
			"disk_free":     formatMetricFloat(stats.DiskFree),
			"uptime":        strconv.FormatInt(stats.Uptime, 10),
			"load_1":        formatMetricFloat(stats.Load1),
			"load_5":        formatMetricFloat(stats.Load5),
			"load_15":       formatMetricFloat(stats.Load15),
		},
	}
}
