package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	defaultPandaXGatewayTelemetryTopic  = "v1/gateway/telemetry"
	defaultPandaXGatewayAttributesTopic = "v1/gateway/attributes"
	defaultPandaXTelemetryTopic         = "v1/devices/me/telemetry"
	defaultPandaXAttributesTopic        = "v1/devices/me/attributes"
	defaultPandaXRowTopic               = "v1/devices/me/row"
	defaultPandaXRPCRequestTopic        = "v1/devices/me/rpc/request"
	defaultPandaXRPCResponseTopic       = "v1/devices/me/rpc/response"
	defaultPandaXEventPrefix            = "v1/devices/event"
	defaultPandaXAlarmIdentifier        = "alarm"
	defaultPandaXReconnectInterval      = 5 * time.Second
	maxPandaXReconnectInterval          = 5 * time.Minute
)

// PandaXConfig PandaX 北向配置
type PandaXConfig struct {
	ServerURL string `json:"serverUrl"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	ClientID  string `json:"clientId"`
	QOS       int    `json:"qos"`
	Retain    bool   `json:"retain"`
	KeepAlive int    `json:"keepAlive"`
	Timeout   int    `json:"connectTimeout"`

	UploadIntervalMs     int `json:"uploadIntervalMs"`
	AlarmFlushIntervalMs int `json:"alarmFlushIntervalMs"`
	AlarmBatchSize       int `json:"alarmBatchSize"`
	AlarmQueueSize       int `json:"alarmQueueSize"`
	RealtimeQueueSize    int `json:"realtimeQueueSize"`
	CommandQueueSize     int `json:"commandQueueSize"`

	GatewayMode            bool   `json:"gatewayMode"`
	SubDeviceTokenMode     string `json:"subDeviceTokenMode"`
	TelemetryTopic         string `json:"telemetryTopic"`
	AttributesTopic        string `json:"attributesTopic"`
	RowTopic               string `json:"rowTopic"`
	GatewayTelemetryTopic  string `json:"gatewayTelemetryTopic"`
	GatewayAttributesTopic string `json:"gatewayAttributesTopic"`
	EventTopicPrefix       string `json:"eventTopicPrefix"`
	AlarmTopic             string `json:"alarmTopic"`
	AlarmIdentifier        string `json:"alarmIdentifier"`
	RPCRequestTopic        string `json:"rpcRequestTopic"`
	RPCResponseTopic       string `json:"rpcResponseTopic"`

	ProductKey string `json:"productKey"`
	DeviceKey  string `json:"deviceKey"`
}

// SystemStatsProvider 系统属性提供者接口
type SystemStatsProvider interface {
	CollectSystemStatsOnce() *models.SystemStats
}

// PandaXAdapter PandaX 北向适配器
type PandaXAdapter struct {
	name   string
	config *PandaXConfig

	client  mqtt.Client
	qos     byte
	retain  bool
	timeout time.Duration

	reportEvery time.Duration
	alarmEvery  time.Duration
	alarmBatch  int
	alarmCap    int
	realtimeCap int
	commandCap  int

	telemetryTopic         string
	attributesTopic        string
	rowTopic               string
	gatewayTelemetryTopic  string
	gatewayAttributesTopic string
	eventTopicPrefix       string
	alarmTopic             string
	alarmIdentifier        string
	rpcRequestTopic        string
	rpcResponseTopic       string

	realtimeQueue []*models.CollectData
	dataMu        sync.RWMutex

	alarmQueue []*models.AlarmPayload
	alarmMu    sync.RWMutex

	commandQueue []*models.NorthboundCommand
	commandMu    sync.RWMutex

	flushNow     chan struct{}
	stopChan     chan struct{}
	reconnectNow chan struct{}
	wg           sync.WaitGroup

	mu                sync.RWMutex
	initialized       bool
	enabled           bool
	connected         bool
	reconnectInterval time.Duration
	lastSend          time.Time
	seq               uint64

	// 系统属性提供者
	systemStatsProvider SystemStatsProvider
}

// NewPandaXAdapter 创建 PandaX 适配器
func NewPandaXAdapter(name string) *PandaXAdapter {
	return &PandaXAdapter{
		name:              name,
		flushNow:          make(chan struct{}, 1),
		stopChan:          make(chan struct{}),
		reconnectNow:      make(chan struct{}, 1),
		realtimeQueue:     make([]*models.CollectData, 0),
		alarmQueue:        make([]*models.AlarmPayload, 0),
		commandQueue:      make([]*models.NorthboundCommand, 0),
		reconnectInterval: defaultPandaXReconnectInterval,
	}
}

func (a *PandaXAdapter) Name() string {
	return a.name
}

func (a *PandaXAdapter) Type() string {
	return "pandax"
}

// SetSystemStatsProvider 设置系统属性提供者
func (a *PandaXAdapter) SetSystemStatsProvider(provider SystemStatsProvider) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.systemStatsProvider = provider
	log.Printf("[PandaX-%s] SetSystemStatsProvider: 系统属性提供者已设置", a.name)
}

func (a *PandaXAdapter) Initialize(configStr string) error {
	log.Printf("[PandaX-%s] Initialize: 开始初始化", a.name)

	cfg, err := parsePandaXConfig(configStr)
	if err != nil {
		log.Printf("[PandaX-%s] Initialize: 配置解析失败: %v", a.name, err)
		return err
	}
	log.Printf("[PandaX-%s] Initialize: 配置解析成功, serverUrl=%s, username=%s, gatewayMode=%v",
		a.name, cfg.ServerURL, cfg.Username, cfg.GatewayMode)

	_ = a.Close()

	broker := normalizeBroker(cfg.ServerURL)
	log.Printf("[PandaX-%s] Initialize: broker=%s", a.name, broker)

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = fmt.Sprintf("pandax-%s-%d", a.name, time.Now().UnixNano())
	}
	log.Printf("[PandaX-%s] Initialize: clientId=%s", a.name, clientID)

	client, err := a.connectPandaXMQTT(broker, clientID, cfg.Username, cfg.Password, cfg.KeepAlive, cfg.Timeout)
	if err != nil {
		log.Printf("[PandaX-%s] Initialize: MQTT 连接失败: %v", a.name, err)
		return fmt.Errorf("failed to connect MQTT: %w", err)
	}
	log.Printf("[PandaX-%s] Initialize: MQTT 连接成功", a.name)

	topicTelemetry := pickFirstNonEmpty(cfg.TelemetryTopic, defaultPandaXTelemetryTopic)
	topicAttributes := pickFirstNonEmpty(cfg.AttributesTopic, defaultPandaXAttributesTopic)
	topicRow := pickFirstNonEmpty(cfg.RowTopic, defaultPandaXRowTopic)
	topicGatewayTelemetry := pickFirstNonEmpty(cfg.GatewayTelemetryTopic, defaultPandaXGatewayTelemetryTopic)
	topicGatewayAttributes := pickFirstNonEmpty(cfg.GatewayAttributesTopic, defaultPandaXGatewayAttributesTopic)
	eventTopicPrefix := pickFirstNonEmpty(cfg.EventTopicPrefix, defaultPandaXEventPrefix)
	alarmIdentifier := pickFirstNonEmpty(cfg.AlarmIdentifier, defaultPandaXAlarmIdentifier)
	alarmTopic := strings.TrimSpace(cfg.AlarmTopic)
	if alarmTopic == "" {
		alarmTopic = strings.TrimRight(eventTopicPrefix, "/") + "/" + alarmIdentifier
	}

	a.mu.Lock()
	a.config = cfg
	a.client = client
	a.qos = clampQOS(cfg.QOS)
	a.retain = cfg.Retain
	a.timeout = time.Duration(maxInt2(cfg.Timeout, 10)) * time.Second
	a.reportEvery = resolveInterval(cfg.UploadIntervalMs, defaultReportInterval)
	a.alarmEvery = resolveInterval(cfg.AlarmFlushIntervalMs, defaultAlarmInterval)
	a.alarmBatch = resolvePositive(cfg.AlarmBatchSize, defaultAlarmBatch)
	a.alarmCap = resolvePositive(cfg.AlarmQueueSize, defaultAlarmQueue)
	a.realtimeCap = resolvePositive(cfg.RealtimeQueueSize, defaultRealtimeQueue)
	a.commandCap = resolvePositive(cfg.CommandQueueSize, defaultRealtimeQueue)
	a.telemetryTopic = topicTelemetry
	a.attributesTopic = topicAttributes
	a.rowTopic = topicRow
	a.gatewayTelemetryTopic = topicGatewayTelemetry
	a.gatewayAttributesTopic = topicGatewayAttributes
	a.eventTopicPrefix = eventTopicPrefix
	a.alarmTopic = alarmTopic
	a.alarmIdentifier = alarmIdentifier
	a.rpcRequestTopic = pickFirstNonEmpty(cfg.RPCRequestTopic, defaultPandaXRPCRequestTopic)
	a.rpcResponseTopic = pickFirstNonEmpty(cfg.RPCResponseTopic, defaultPandaXRPCResponseTopic)
	a.flushNow = make(chan struct{}, 1)
	a.stopChan = make(chan struct{})
	a.reconnectNow = make(chan struct{}, 1)
	a.initialized = true
	a.connected = true
	a.enabled = false
	a.mu.Unlock()

	a.subscribeRPCTopics(client)

	log.Printf("PandaX adapter initialized: %s (broker=%s)", a.name, broker)
	return nil
}

func (a *PandaXAdapter) Start() {
	needReconnect := false
	a.mu.Lock()
	if a.initialized && !a.enabled {
		a.enabled = true
		if a.stopChan == nil {
			a.stopChan = make(chan struct{})
		}
		if a.reconnectNow == nil {
			a.reconnectNow = make(chan struct{}, 1)
		}
		needReconnect = !a.connected
		a.wg.Add(3)
		go a.reportLoop()
		go a.alarmLoop()
		go a.reconnectLoop()
		log.Printf("[PandaX-%s] Start: 适配器已启动, reportInterval=%v, alarmInterval=%v",
			a.name, a.reportEvery, a.alarmEvery)
	}
	a.mu.Unlock()

	if needReconnect {
		a.signalReconnect()
	}
}

func (a *PandaXAdapter) Stop() {
	a.mu.Lock()
	if a.enabled {
		a.enabled = false
		if a.stopChan != nil {
			close(a.stopChan)
			a.stopChan = nil
		}
		log.Printf("[PandaX-%s] Stop: 适配器已停止", a.name)
	}
	a.mu.Unlock()
	a.wg.Wait()
}

func (a *PandaXAdapter) SetInterval(interval time.Duration) {
	a.mu.Lock()
	a.reportEvery = resolveInterval(int(interval.Milliseconds()), defaultReportInterval)
	a.mu.Unlock()
}

func (a *PandaXAdapter) SetReconnectInterval(interval time.Duration) {
	a.mu.Lock()
	if interval <= 0 {
		interval = defaultPandaXReconnectInterval
	}
	if interval > maxPandaXReconnectInterval {
		interval = maxPandaXReconnectInterval
	}
	a.reconnectInterval = interval
	a.mu.Unlock()
}

func (a *PandaXAdapter) IsEnabled() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.enabled
}

func (a *PandaXAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

func (a *PandaXAdapter) Send(data *models.CollectData) error {
	if data == nil {
		log.Printf("[PandaX-%s] Send: data is nil", a.name)
		return nil
	}

	log.Printf("[PandaX-%s] Send: deviceId=%d, deviceKey=%s, fields=%d",
		a.name, data.DeviceID, data.DeviceKey, len(data.Fields))

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(cloneCollectData(data))
	queueLen := len(a.realtimeQueue)
	a.dataMu.Unlock()

	log.Printf("[PandaX-%s] Send: enqueued, queueLen=%d", a.name, queueLen)
	return nil
}

func (a *PandaXAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.enqueueAlarmLocked(cloneAlarmPayload(alarm))
	queueLen := len(a.alarmQueue)
	a.alarmMu.Unlock()

	log.Printf("[PandaX-%s] SendAlarm: deviceKey=%s, fieldName=%s, severity=%s, message=%s, queueLen=%d",
		a.name, alarm.DeviceKey, alarm.FieldName, alarm.Severity, alarm.Message, queueLen)

	if queueLen >= a.alarmBatch {
		log.Printf("[PandaX-%s] SendAlarm: 触发批量上报, batchSize=%d", a.name, a.alarmBatch)
	}

	return nil
}

func (a *PandaXAdapter) Close() error {
	log.Printf("[PandaX-%s] Close: 开始关闭", a.name)

	a.Stop()

	a.mu.Lock()
	client := a.client
	a.initialized = false
	a.connected = false
	a.enabled = false
	a.mu.Unlock()

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

func (a *PandaXAdapter) PullCommands(limit int) ([]*models.NorthboundCommand, error) {
	if limit <= 0 {
		limit = 20
	}

	a.commandMu.Lock()
	defer a.commandMu.Unlock()

	if !a.isInitialized() {
		log.Printf("[PandaX-%s] PullCommands: 适配器未初始化", a.name)
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

	log.Printf("[PandaX-%s] PullCommands: 取出 %d 条命令", a.name, len(items))
	return items, nil
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

	resp := map[string]interface{}{
		"requestId": result.RequestID,
		"method":    "write",
		"params": map[string]interface{}{
			"success":    result.Success,
			"code":       code,
			"message":    message,
			"productKey": result.ProductKey,
			"deviceKey":  result.DeviceKey,
			"fieldName":  result.FieldName,
			"value":      convertFieldValue(result.Value),
		},
	}
	body, _ := json.Marshal(resp)

	return a.publish(topic, body)
}

func (a *PandaXAdapter) GetStats() map[string]interface{} {
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
		"name":                    a.name,
		"type":                    "pandax",
		"enabled":                 a.enabled,
		"initialized":             a.initialized,
		"connected":               a.connected && a.client != nil && a.client.IsConnected(),
		"interval_ms":             a.reportEvery.Milliseconds(),
		"pending_data":            pendingData,
		"pending_alarm":           pendingAlarm,
		"pending_cmd":             pendingCmd,
		"telemetry_topic":         a.telemetryTopic,
		"gateway_telemetry_topic": a.gatewayTelemetryTopic,
		"rpc_request_topic":       a.rpcRequestTopic,
		"rpc_response_topic":      a.rpcResponseTopic,
	}
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

func (a *PandaXAdapter) reportLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.reportEvery
	stopChan := a.stopChan
	a.mu.RUnlock()

	log.Printf("[PandaX-%s] reportLoop: 启动, interval=%v (主动获取数据库最新数据)", a.name, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			log.Printf("[PandaX-%s] reportLoop: 退出", a.name)
			return
		case <-ticker.C:
			// 主动从数据库获取最新数据上报
			if err := a.fetchAndPublishLatestData(); err != nil {
				log.Printf("[PandaX-%s] reportLoop: fetchAndPublishLatestData 失败: %v", a.name, err)
			}
		}
	}
}

// fetchAndPublishLatestData 从数据库获取所有设备的最新数据并上报
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

		// 判断是否是网关系统数据
		isSystemStats := dev.DeviceID == models.SystemStatsDeviceID

		// 构建 CollectData
		data := &models.CollectData{
			DeviceID:   dev.DeviceID,
			DeviceName: dev.DeviceName,
			Timestamp:  dev.CollectedAt,
			Fields:     dev.Fields,
		}

		// 发送到队列
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

	// 如果没有系统属性数据，尝试获取当前系统属性
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

	// 触发一次发送
	if err := a.flushRealtime(); err != nil {
		log.Printf("[PandaX-%s] fetchAndPublishLatestData: flushRealtime 失败: %v", a.name, err)
	}

	log.Printf("[PandaX-%s] fetchAndPublishLatestData: 完成, 设备数=%d, 系统属性=%d",
		a.name, successCount, systemStatsCount)
	return nil
}

// fetchCurrentSystemStats 获取当前系统属性
func (a *PandaXAdapter) fetchCurrentSystemStats() *models.CollectData {
	a.mu.RLock()
	provider := a.systemStatsProvider
	a.mu.RUnlock()

	if provider == nil {
		log.Printf("[PandaX-%s] fetchCurrentSystemStats: 系统属性提供者未设置", a.name)
		return nil
	}

	// 采集当前系统属性
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

func (a *PandaXAdapter) alarmLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	interval := a.alarmEvery
	flushNow := a.flushNow
	stopChan := a.stopChan
	a.mu.RUnlock()

	log.Printf("[PandaX-%s] alarmLoop: 启动, interval=%v, batch=%d", a.name, interval, a.alarmBatch)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			log.Printf("[PandaX-%s] alarmLoop: 正在清空剩余报警队列...", a.name)
			for {
				if err := a.flushAlarmBatch(); err != nil {
					log.Printf("[PandaX-%s] alarmLoop: flushAlarmBatch 失败: %v", a.name, err)
					return
				}
				a.alarmMu.RLock()
				empty := len(a.alarmQueue) == 0
				a.alarmMu.RUnlock()
				if empty {
					log.Printf("[PandaX-%s] alarmLoop: 退出", a.name)
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		case <-ticker.C:
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("[PandaX-%s] alarmLoop: flushAlarmBatch 失败: %v", a.name, err)
			}
		case <-flushNow:
			log.Printf("[PandaX-%s] alarmLoop: 收到 flush 信号", a.name)
			if err := a.flushAlarmBatch(); err != nil {
				log.Printf("[PandaX-%s] alarmLoop: flushAlarmBatch 失败: %v", a.name, err)
			}
		}
	}
}

func (a *PandaXAdapter) flushRealtime() error {
	a.dataMu.Lock()
	if len(a.realtimeQueue) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := make([]*models.CollectData, len(a.realtimeQueue))
	copy(batch, a.realtimeQueue)
	clear(a.realtimeQueue)
	a.realtimeQueue = a.realtimeQueue[:0]
	a.dataMu.Unlock()

	log.Printf("[PandaX-%s] flushRealtime: 开始发送 %d 条数据", a.name, len(batch))

	// 批量构建 payload
	topic, body := a.buildBatchRealtimePublish(batch)

	// 发送批量数据
	if err := a.publish(topic, body); err != nil {
		log.Printf("[PandaX-%s] flushRealtime: 发送失败: %v", a.name, err)
		// 将数据放回队列
		a.dataMu.Lock()
		a.prependRealtime(batch)
		a.dataMu.Unlock()
		return err
	}

	log.Printf("[PandaX-%s] flushRealtime: 发送成功 %d 条数据", a.name, len(batch))
	return nil
}

// buildBatchRealtimePublish 批量构建网关遥测 payload
// 格式: {"设备名1": {"ts": ts, "values": {...}}, "设备名2": {"ts": ts, "values": {...}}}
func (a *PandaXAdapter) buildBatchRealtimePublish(batch []*models.CollectData) (string, []byte) {
	a.mu.RLock()
	topic := a.gatewayTelemetryTopic
	a.mu.RUnlock()

	if len(batch) == 0 {
		return topic, []byte("{}")
	}

	// 构建批量 payload
	payload := make(map[string]interface{}, len(batch))
	for _, data := range batch {
		if data == nil {
			continue
		}

		// 获取子设备标识符
		subToken := a.resolveSubDeviceToken(data)

		// 构建 values
		values := make(map[string]interface{}, len(data.Fields))
		for key, value := range data.Fields {
			values[key] = convertFieldValue(value)
		}

		// 获取时间戳（毫秒）
		ts := data.Timestamp.UnixMilli()
		if ts <= 0 {
			ts = time.Now().UnixMilli()
		}

		payload[subToken] = map[string]interface{}{
			"ts":     ts,
			"values": values,
		}
	}

	body, _ := json.Marshal(payload)
	log.Printf("[PandaX-%s] buildBatchRealtimePublish: deviceCount=%d, payloadSize=%d", a.name, len(payload), len(body))

	return topic, body
}

func (a *PandaXAdapter) flushAlarmBatch() error {
	a.alarmMu.Lock()
	if len(a.alarmQueue) == 0 {
		a.alarmMu.Unlock()
		return nil
	}
	count := a.alarmBatch
	if count > len(a.alarmQueue) {
		count = len(a.alarmQueue)
	}
	batch := make([]*models.AlarmPayload, count)
	copy(batch, a.alarmQueue[:count])
	clear(a.alarmQueue[:count])
	a.alarmQueue = a.alarmQueue[count:]
	a.alarmMu.Unlock()

	log.Printf("[PandaX-%s] flushAlarmBatch: 开始发送 %d 条报警", a.name, len(batch))

	successCount := 0
	for _, item := range batch {
		topic, body := a.buildAlarmPublish(item)
		if err := a.publish(topic, body); err != nil {
			log.Printf("[PandaX-%s] flushAlarmBatch: 发送失败 deviceKey=%s field=%s: %v",
				a.name, item.DeviceKey, item.FieldName, err)
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
		successCount++
	}

	log.Printf("[PandaX-%s] flushAlarmBatch: 发送成功 %d/%d", a.name, successCount, len(batch))
	return nil
}

func (a *PandaXAdapter) buildAlarmPublish(alarm *models.AlarmPayload) (string, []byte) {
	a.mu.RLock()
	topic := a.alarmTopic
	a.mu.RUnlock()

	if alarm == nil {
		return topic, []byte("{}")
	}

	payload := map[string]interface{}{
		"device_name":  alarm.DeviceName,
		"product_key":  alarm.ProductKey,
		"device_key":   alarm.DeviceKey,
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"severity":     alarm.Severity,
		"message":      alarm.Message,
		"ts":           time.Now().UnixMilli(),
	}
	body, _ := json.Marshal(payload)
	return topic, body
}

func (a *PandaXAdapter) publish(topic string, payload []byte) error {
	a.mu.RLock()
	client := a.client
	timeout := a.timeout
	qos := a.qos
	retain := a.retain
	a.mu.RUnlock()

	if client == nil {
		log.Printf("[PandaX-%s] publish: MQTT client 为 nil", a.name)
		a.markDisconnected()
		return fmt.Errorf("mqtt client is nil")
	}

	if !client.IsConnected() {
		a.markDisconnected()
		return fmt.Errorf("mqtt client not connected")
	}

	log.Printf("[PandaX-%s] publish: topic=%s, size=%d", a.name, topic, len(payload))

	token := client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(timeout) {
		log.Printf("[PandaX-%s] publish: 发布超时", a.name)
		a.markDisconnected()
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		log.Printf("[PandaX-%s] publish: 发布失败: %v", a.name, err)
		a.markDisconnected()
		return err
	}

	a.mu.Lock()
	a.lastSend = time.Now()
	a.connected = true
	a.mu.Unlock()

	log.Printf("[PandaX-%s] publish: 发送成功", a.name)
	return nil
}

func (a *PandaXAdapter) connectPandaXMQTT(broker, clientID, username, password string, keepAliveSec, timeoutSec int) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetAutoReconnect(false)
	opts.SetConnectRetry(false)
	if keepAliveSec > 0 {
		opts.SetKeepAlive(time.Duration(keepAliveSec) * time.Second)
	}

	timeout := 10 * time.Second
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}
	opts.SetConnectTimeout(timeout)

	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		if err != nil {
			log.Printf("[PandaX-%s] MQTT connection lost: %v", a.name, err)
		} else {
			log.Printf("[PandaX-%s] MQTT connection lost", a.name)
		}
		a.markDisconnected()
	}
	opts.OnConnect = func(_ mqtt.Client) {
		log.Printf("[PandaX-%s] MQTT connected: %s", a.name, broker)
		a.mu.Lock()
		a.connected = true
		a.mu.Unlock()
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(timeout) {
		return nil, fmt.Errorf("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}

	return client, nil
}

func (a *PandaXAdapter) reconnectLoop() {
	defer a.wg.Done()

	a.mu.RLock()
	stopChan := a.stopChan
	reconnectNow := a.reconnectNow
	a.mu.RUnlock()

	if stopChan == nil || reconnectNow == nil {
		return
	}

	for {
		select {
		case <-stopChan:
			return
		case <-reconnectNow:
		}

		failures := 0
		for {
			if !a.shouldReconnect() {
				break
			}

			if err := a.reconnectOnce(); err != nil {
				failures++
				delay := pandaXReconnectDelay(a.currentReconnectInterval(), failures)
				log.Printf("[PandaX-%s] reconnect failed (attempt=%d): %v, retry in %v", a.name, failures, err, delay)
				select {
				case <-stopChan:
					return
				case <-time.After(delay):
				}
				continue
			}

			log.Printf("[PandaX-%s] reconnect success", a.name)
			break
		}
	}
}

func (a *PandaXAdapter) reconnectOnce() error {
	a.mu.RLock()
	client := a.client
	timeout := a.timeout
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("mqtt client is nil")
	}

	token := client.Connect()
	if !token.WaitTimeout(timeout) {
		a.markDisconnected()
		return fmt.Errorf("mqtt reconnect timeout")
	}
	if err := token.Error(); err != nil {
		a.markDisconnected()
		return err
	}

	a.mu.Lock()
	a.connected = true
	a.mu.Unlock()
	a.subscribeRPCTopics(client)

	return nil
}

func (a *PandaXAdapter) shouldReconnect() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.initialized || !a.enabled || a.client == nil {
		return false
	}

	return !a.client.IsConnected()
}

func (a *PandaXAdapter) currentReconnectInterval() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.reconnectInterval <= 0 {
		return defaultPandaXReconnectInterval
	}
	if a.reconnectInterval > maxPandaXReconnectInterval {
		return maxPandaXReconnectInterval
	}
	return a.reconnectInterval
}

func (a *PandaXAdapter) signalReconnect() {
	a.mu.RLock()
	reconnectNow := a.reconnectNow
	a.mu.RUnlock()

	if reconnectNow == nil {
		return
	}

	select {
	case reconnectNow <- struct{}{}:
	default:
	}
}

func (a *PandaXAdapter) markDisconnected() {
	a.mu.Lock()
	a.connected = false
	enabled := a.enabled
	a.mu.Unlock()

	if enabled {
		a.signalReconnect()
	}
}

func pandaXReconnectDelay(base time.Duration, failures int) time.Duration {
	if base <= 0 {
		base = defaultPandaXReconnectInterval
	}
	if base > maxPandaXReconnectInterval {
		base = maxPandaXReconnectInterval
	}
	if failures <= 0 {
		return base
	}

	delay := base
	for attempt := 1; attempt < failures; attempt++ {
		if delay >= maxPandaXReconnectInterval/2 {
			return maxPandaXReconnectInterval
		}
		delay *= 2
	}

	if delay > maxPandaXReconnectInterval {
		return maxPandaXReconnectInterval
	}
	return delay
}

func (a *PandaXAdapter) subscribeRPCTopics(client mqtt.Client) {
	a.mu.RLock()
	rpcReqTopic := a.rpcRequestTopic
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	topics := make(map[string]struct{})
	if strings.TrimSpace(rpcReqTopic) != "" {
		topics[strings.TrimSpace(rpcReqTopic)] = struct{}{}
		if !strings.HasSuffix(strings.TrimSpace(rpcReqTopic), "/+") {
			topics[strings.TrimRight(strings.TrimSpace(rpcReqTopic), "/")+"/+"] = struct{}{}
		}
	}

	log.Printf("[PandaX-%s] subscribeRPCTopics: 开始订阅 topics=%v", a.name, topics)

	for topic := range topics {
		token := client.Subscribe(topic, qos, a.handleRPCRequest)
		if !token.WaitTimeout(timeout) {
			log.Printf("[PandaX-%s] subscribeRPCTopics: 订阅超时 topic=%s", a.name, topic)
			continue
		}
		if err := token.Error(); err != nil {
			log.Printf("[PandaX-%s] subscribeRPCTopics: 订阅失败 topic=%s: %v", a.name, topic, err)
		} else {
			log.Printf("[PandaX-%s] subscribeRPCTopics: 订阅成功 topic=%s", a.name, topic)
		}
	}
}

func (a *PandaXAdapter) handleRPCRequest(_ mqtt.Client, message mqtt.Message) {
	log.Printf("[PandaX-%s] handleRPCRequest: 收到 RPC topic=%s", a.name, message.Topic())

	var req struct {
		RequestID string      `json:"requestId"`
		Method    string      `json:"method"`
		Params    interface{} `json:"params"`
	}
	if err := json.Unmarshal(message.Payload(), &req); err != nil {
		log.Printf("[PandaX-%s] handleRPCRequest: JSON 解析失败: %v", a.name, err)
		return
	}

	if strings.TrimSpace(req.RequestID) == "" {
		req.RequestID = requestIDFromPandaXRPCTopic(message.Topic())
	}

	log.Printf("[PandaX-%s] handleRPCRequest: requestId=%s, method=%s", a.name, req.RequestID, req.Method)

	commands := a.buildCommandsFromRPC(req.RequestID, req.Method, req.Params)
	if len(commands) == 0 {
		log.Printf("[PandaX-%s] handleRPCRequest: 无有效命令", a.name)
		return
	}

	a.commandMu.Lock()
	a.commandQueue = appendCommandQueueWithCap(a.commandQueue, commands, a.commandCap)
	queueLen := len(a.commandQueue)
	a.commandMu.Unlock()

	log.Printf("[PandaX-%s] handleRPCRequest: 入队 %d 条命令, queueLen=%d", a.name, len(commands), queueLen)
}

func (a *PandaXAdapter) buildCommandsFromRPC(requestID, method string, params interface{}) []*models.NorthboundCommand {
	defaultPK, defaultDK := a.defaultIdentity()
	commands := buildPandaXCommands(requestID, method, params, defaultPK, defaultDK)
	if len(commands) == 0 {
		return nil
	}

	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		cmd.Source = "pandax.rpc.request"
	}

	return commands
}

func buildPandaXCommands(requestID, method string, params interface{}, defaultPK, defaultDK string) []*models.NorthboundCommand {
	out := make([]*models.NorthboundCommand, 0)
	appendProperties := func(pk, dk string, props map[string]interface{}) {
		if len(props) == 0 {
			return
		}
		keys := make([]string, 0, len(props))
		for key := range props {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" {
				continue
			}
			keys = append(keys, trimmed)
		}
		if len(keys) == 0 {
			return
		}
		sort.Strings(keys)

		for _, key := range keys {
			item := &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: pk,
				DeviceKey:  dk,
				FieldName:  key,
				Value:      stringifyAny(props[key]),
			}
			if strings.TrimSpace(item.ProductKey) == "" || strings.TrimSpace(item.DeviceKey) == "" {
				continue
			}
			out = append(out, item)
		}
	}

	obj, ok := params.(map[string]interface{})
	if ok {
		pk := pickFirstNonEmpty(pickConfigString(obj, "productKey", "product_key"), defaultPK)
		dk := pickFirstNonEmpty(pickConfigString(obj, "deviceKey", "device_key"), defaultDK)

		if props, ok := mapFromAny(obj["properties"]); ok {
			appendProperties(pk, dk, props)
		}

		for _, key := range []string{"sub_device", "subDevice"} {
			sub, ok := mapFromAny(obj[key])
			if !ok {
				continue
			}
			subPK := pk
			subDK := dk
			if identity, ok := mapFromAny(sub["identity"]); ok {
				subPK = pickFirstNonEmpty(pickConfigString(identity, "productKey", "product_key"), subPK)
				subDK = pickFirstNonEmpty(pickConfigString(identity, "deviceKey", "device_key"), subDK)
			}
			if props, ok := mapFromAny(sub["properties"]); ok {
				appendProperties(subPK, subDK, props)
			}
		}

		for _, key := range []string{"sub_devices", "subDevices"} {
			list, ok := obj[key].([]interface{})
			if !ok || len(list) == 0 {
				continue
			}
			for _, item := range list {
				row, ok := mapFromAny(item)
				if !ok {
					continue
				}
				subPK := pk
				subDK := dk
				if identity, ok := mapFromAny(row["identity"]); ok {
					subPK = pickFirstNonEmpty(pickConfigString(identity, "productKey", "product_key"), subPK)
					subDK = pickFirstNonEmpty(pickConfigString(identity, "deviceKey", "device_key"), subDK)
				}
				if props, ok := mapFromAny(row["properties"]); ok {
					appendProperties(subPK, subDK, props)
				}
			}
		}

		if fieldName := strings.TrimSpace(pickConfigString(obj, "fieldName", "field_name")); fieldName != "" {
			if rawValue, exists := obj["value"]; exists {
				appendProperties(pk, dk, map[string]interface{}{fieldName: rawValue})
			}
		}

		if len(out) == 0 {
			generic := make(map[string]interface{})
			for key, value := range obj {
				if isPandaXReservedRPCKey(key) {
					continue
				}
				generic[key] = value
			}
			appendProperties(pk, dk, generic)
		}
	}

	if len(out) == 0 && strings.TrimSpace(method) != "" {
		if strings.TrimSpace(defaultPK) != "" && strings.TrimSpace(defaultDK) != "" {
			out = append(out, &models.NorthboundCommand{
				RequestID:  requestID,
				ProductKey: defaultPK,
				DeviceKey:  defaultDK,
				FieldName:  strings.TrimSpace(method),
				Value:      stringifyAny(params),
			})
		}
	}

	return out
}

func isPandaXReservedRPCKey(key string) bool {
	switch strings.TrimSpace(key) {
	case "productKey", "product_key", "deviceKey", "device_key",
		"properties", "sub_device", "subDevice", "sub_devices", "subDevices",
		"fieldName", "field_name", "value":
		return true
	default:
		return false
	}
}

func (a *PandaXAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	if len(a.realtimeQueue) >= a.realtimeCap {
		a.realtimeQueue[0] = nil
		a.realtimeQueue = a.realtimeQueue[1:]
	}
	a.realtimeQueue = append(a.realtimeQueue, item)
}

func (a *PandaXAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	queue := make([]*models.CollectData, 0, len(items)+len(a.realtimeQueue))
	queue = append(queue, items...)
	queue = append(queue, a.realtimeQueue...)
	if len(queue) > a.realtimeCap {
		clear(queue[a.realtimeCap:])
		queue = queue[:a.realtimeCap]
	}
	a.realtimeQueue = queue[:len(queue):len(queue)]
}

func (a *PandaXAdapter) enqueueAlarmLocked(item *models.AlarmPayload) {
	if item == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	if len(a.alarmQueue) >= a.alarmCap {
		a.alarmQueue[0] = nil
		a.alarmQueue = a.alarmQueue[1:]
	}
	a.alarmQueue = append(a.alarmQueue, item)
}

func (a *PandaXAdapter) prependAlarms(items []*models.AlarmPayload) {
	if len(items) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	queue := make([]*models.AlarmPayload, 0, len(items)+len(a.alarmQueue))
	queue = append(queue, items...)
	queue = append(queue, a.alarmQueue...)
	if len(queue) > a.alarmCap {
		clear(queue[a.alarmCap:])
		queue = queue[:a.alarmCap]
	}
	a.alarmQueue = queue[:len(queue):len(queue)]
}

func (a *PandaXAdapter) defaultIdentity() (string, string) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.config == nil {
		return "", ""
	}
	return strings.TrimSpace(a.config.ProductKey), strings.TrimSpace(a.config.DeviceKey)
}

func (a *PandaXAdapter) resolveSubDeviceToken(data *models.CollectData) string {
	a.mu.RLock()
	cfg := a.config
	a.mu.RUnlock()

	if cfg == nil {
		if strings.TrimSpace(data.DeviceName) != "" {
			return strings.TrimSpace(data.DeviceName)
		}
		if strings.TrimSpace(data.DeviceKey) != "" {
			return strings.TrimSpace(data.DeviceKey)
		}
		return defaultDeviceToken(data.DeviceID)
	}

	pk := pickFirstNonEmpty(data.ProductKey, cfg.ProductKey)
	name := pickFirstNonEmpty(data.DeviceName, data.DeviceKey)
	dk := pickFirstNonEmpty(data.DeviceKey, data.DeviceName)

	switch strings.ToLower(strings.TrimSpace(cfg.SubDeviceTokenMode)) {
	case "devicekey", "device_key":
		if dk != "" {
			return dk
		}
	case "product_devicekey", "product_device_key":
		if pk != "" && dk != "" {
			return pk + "_" + dk
		}
	case "product_devicename", "product_device_name":
		if pk != "" && name != "" {
			return pk + "_" + name
		}
	default:
		if name != "" {
			return name
		}
	}

	if dk != "" {
		return dk
	}
	if name != "" {
		return name
	}
	return defaultDeviceToken(data.DeviceID)
}

func (a *PandaXAdapter) isInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

func parsePandaXConfig(configStr string) (*PandaXConfig, error) {
	raw := make(map[string]interface{})
	if err := json.Unmarshal([]byte(configStr), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := &PandaXConfig{
		ServerURL: normalizePandaXServerURL(
			pickConfigString(raw, "serverUrl", "broker", "server_url"),
			pickConfigString(raw, "protocol"),
			pickConfigInt(raw, 0, "port"),
		),
		Username:               strings.TrimSpace(pickConfigString(raw, "username", "token", "deviceToken")),
		Password:               strings.TrimSpace(pickConfigString(raw, "password")),
		ClientID:               strings.TrimSpace(pickConfigString(raw, "clientId", "client_id")),
		QOS:                    pickConfigInt(raw, 0, "qos"),
		Retain:                 pickConfigBool(raw, false, "retain"),
		KeepAlive:              pickConfigInt(raw, 60, "keepAlive", "keep_alive"),
		Timeout:                pickConfigInt(raw, 10, "connectTimeout", "connect_timeout", "timeout"),
		UploadIntervalMs:       pickConfigInt(raw, int(defaultReportInterval.Milliseconds()), "uploadIntervalMs", "upload_interval_ms", "reportIntervalMs"),
		AlarmFlushIntervalMs:   pickConfigInt(raw, int(defaultAlarmInterval.Milliseconds()), "alarmFlushIntervalMs"),
		AlarmBatchSize:         pickConfigInt(raw, defaultAlarmBatch, "alarmBatchSize"),
		AlarmQueueSize:         pickConfigInt(raw, defaultAlarmQueue, "alarmQueueSize"),
		RealtimeQueueSize:      pickConfigInt(raw, defaultRealtimeQueue, "realtimeQueueSize"),
		GatewayMode:            pickConfigBool(raw, true, "gatewayMode"),
		SubDeviceTokenMode:     strings.TrimSpace(pickConfigString(raw, "subDeviceTokenMode")),
		TelemetryTopic:         strings.TrimSpace(pickConfigString(raw, "telemetryTopic", "topic")),
		AttributesTopic:        strings.TrimSpace(pickConfigString(raw, "attributesTopic")),
		RowTopic:               strings.TrimSpace(pickConfigString(raw, "rowTopic")),
		GatewayTelemetryTopic:  strings.TrimSpace(pickConfigString(raw, "gatewayTelemetryTopic")),
		GatewayAttributesTopic: strings.TrimSpace(pickConfigString(raw, "gatewayAttributesTopic")),
		EventTopicPrefix:       strings.TrimSpace(pickConfigString(raw, "eventTopicPrefix")),
		AlarmTopic:             strings.TrimSpace(pickConfigString(raw, "alarmTopic")),
		AlarmIdentifier:        strings.TrimSpace(pickConfigString(raw, "alarmIdentifier")),
		RPCRequestTopic:        strings.TrimSpace(pickConfigString(raw, "rpcRequestTopic")),
		RPCResponseTopic:       strings.TrimSpace(pickConfigString(raw, "rpcResponseTopic")),
		ProductKey:             strings.TrimSpace(pickConfigString(raw, "productKey", "product_key")),
		DeviceKey:              strings.TrimSpace(pickConfigString(raw, "deviceKey", "device_key")),
	}

	cfg.CommandQueueSize = pickConfigInt(raw, cfg.RealtimeQueueSize, "commandQueueSize")

	if err := normalizePandaXConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func normalizePandaXConfig(cfg *PandaXConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.ServerURL == "" {
		return fmt.Errorf("serverUrl is required")
	}
	if cfg.Username == "" {
		return fmt.Errorf("username is required")
	}
	if !cfg.GatewayMode {
		return fmt.Errorf("PandaX adapter only supports gatewayMode=true")
	}
	if cfg.QOS < 0 || cfg.QOS > 2 {
		return fmt.Errorf("qos must be between 0 and 2")
	}
	if cfg.UploadIntervalMs <= 0 {
		cfg.UploadIntervalMs = int(defaultReportInterval.Milliseconds())
	}
	if cfg.AlarmFlushIntervalMs <= 0 {
		cfg.AlarmFlushIntervalMs = int(defaultAlarmInterval.Milliseconds())
	}
	if cfg.AlarmBatchSize <= 0 {
		cfg.AlarmBatchSize = defaultAlarmBatch
	}
	if cfg.AlarmQueueSize <= 0 {
		cfg.AlarmQueueSize = defaultAlarmQueue
	}
	if cfg.RealtimeQueueSize <= 0 {
		cfg.RealtimeQueueSize = defaultRealtimeQueue
	}
	if cfg.CommandQueueSize <= 0 {
		cfg.CommandQueueSize = cfg.RealtimeQueueSize
	}

	return nil
}

func normalizePandaXServerURL(serverURL, protocol string, port int) string {
	return normalizeServerURLWithPort(serverURL, protocol, port)
}

func requestIDFromPandaXRPCTopic(topic string) string {
	parts := splitTopic(topic)
	if len(parts) < 6 {
		return ""
	}
	if parts[0] != "v1" || parts[1] != "devices" || parts[2] != "me" || parts[3] != "rpc" || parts[4] != "request" {
		return ""
	}
	return strings.TrimSpace(parts[5])
}

func maxInt2(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func (a *PandaXAdapter) nextID(prefix string) string {
	n := atomic.AddUint64(&a.seq, 1)
	millis := strconv.FormatInt(time.Now().UnixMilli(), 10)
	return prefix + "_" + millis + "_" + strconv.FormatUint(n, 10)
}

func formatMetricFloat2(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func defaultDeviceToken(deviceID int64) string {
	return "device_" + strconv.FormatInt(deviceID, 10)
}
