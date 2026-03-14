package adapters

import (
	"encoding/json"
	"fmt"
	"log"
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
		go a.runLoop()
		log.Printf("[PandaX-%s] Start: 适配器已启动, reportInterval=%v, alarmInterval=%v",
			a.name, a.reportEvery, a.alarmEvery)
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
		log.Printf("[PandaX-%s] Stop: 适配器已停止", a.name)
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
	log.Printf("[PandaX-%s] Close: 开始关闭", a.name)

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
		log.Printf("[PandaX-%s] Close: 断开 MQTT 连接", a.name)
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

	log.Printf("[PandaX-%s] Close: 已关闭", a.name)
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

func (a *PandaXAdapter) GetStats() map[string]interface{} {
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

func (a *PandaXAdapter) runLoop() {
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

	log.Printf("[PandaX-%s] runLoop: 启动, report=%v, alarm=%v", a.name, reportInterval, alarmInterval)

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
			log.Printf("[PandaX-%s] runLoop: 停止并清空报警", a.name)
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
					log.Printf("[PandaX-%s] runLoop: flushAlarmBatch 失败: %v", a.name, err)
				},
				onDrained: func() {
					log.Printf("[PandaX-%s] runLoop: 已退出", a.name)
				},
			})
			return
		case <-reportTicker.C:
			if err := a.fetchAndPublishLatestData(); err != nil {
				log.Printf("[PandaX-%s] runLoop: fetchAndPublishLatestData 失败: %v", a.name, err)
			}
		case <-alarmTicker.C:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("[PandaX-%s] runLoop: flushAlarmBatch 失败: %v", a.name, err)
			}
		case <-flushNow:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("[PandaX-%s] runLoop: flushAlarmBatch 失败: %v", a.name, err)
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
				log.Printf("[PandaX-%s] reconnect failed (attempt=%d): %v, retry in %v", a.name, reconnectFailures, err, delay)
				reconnect.Schedule(delay)
				continue
			}
			log.Printf("[PandaX-%s] reconnect success", a.name)
			stopReconnect()
		}
	}
}

func (a *PandaXAdapter) fetchAndPublishLatestData() error {
	devices, err := database.GetAllDevicesLatestData()
	if err != nil {
		return fmt.Errorf("获取设备最新数据失败: %w", err)
	}

	log.Printf("[PandaX-%s] fetchAndPublishLatestData: 获取到 %d 条数据", a.name, len(devices))

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
			log.Printf("[PandaX-%s] fetchAndPublishLatestData: 网关系统属性 deviceId=%d, deviceName=%s, fields=%d",
				a.name, dev.DeviceID, dev.DeviceName, len(dev.Fields))
			systemStatsCount++
		} else {
			log.Printf("[PandaX-%s] fetchAndPublishLatestData: 设备 deviceId=%d, deviceName=%s, fields=%d",
				a.name, dev.DeviceID, dev.DeviceName, len(dev.Fields))
			successCount++
		}
	}

	if systemStatsCount == 0 {
		if sysData := a.fetchCurrentSystemStats(); sysData != nil {
			a.dataMu.Lock()
			a.enqueueRealtimeLocked(sysData)
			queueLen := len(a.realtimeQueue)
			a.dataMu.Unlock()
			log.Printf("[PandaX-%s] fetchAndPublishLatestData: 当前系统属性, fields=%d, queueLen=%d",
				a.name, len(sysData.Fields), queueLen)
		}
	}

	if err := a.flushRealtime(); err != nil {
		log.Printf("[PandaX-%s] fetchAndPublishLatestData: flushRealtime 失败: %v", a.name, err)
	}

	log.Printf("[PandaX-%s] fetchAndPublishLatestData: 完成, 设备数=%d, 系统属性=%d",
		a.name, successCount, systemStatsCount)
	return nil
}

func (a *PandaXAdapter) fetchCurrentSystemStats() *models.CollectData {
	a.mu.RLock()
	provider := a.systemStatsProvider
	a.mu.RUnlock()

	if provider == nil {
		log.Printf("[PandaX-%s] fetchCurrentSystemStats: 系统属性提供者未设置", a.name)
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
			"cpu_usage":     formatMetricFloat2(stats.CpuUsage),
			"mem_total":     formatMetricFloat2(stats.MemTotal),
			"mem_used":      formatMetricFloat2(stats.MemUsed),
			"mem_usage":     formatMetricFloat2(stats.MemUsage),
			"mem_available": formatMetricFloat2(stats.MemAvailable),
			"disk_total":    formatMetricFloat2(stats.DiskTotal),
			"disk_used":     formatMetricFloat2(stats.DiskUsed),
			"disk_usage":    formatMetricFloat2(stats.DiskUsage),
			"disk_free":     formatMetricFloat2(stats.DiskFree),
			"uptime":        strconv.FormatInt(stats.Uptime, 10),
			"load_1":        formatMetricFloat2(stats.Load1),
			"load_5":        formatMetricFloat2(stats.Load5),
			"load_15":       formatMetricFloat2(stats.Load15),
		},
	}
}
