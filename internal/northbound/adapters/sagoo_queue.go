package adapters

import (
	"encoding/json"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type sagooSysPayload struct {
	Ack int `json:"ack"`
}

type sagooIdentityPayload struct {
	ProductKey string `json:"productKey"`
	DeviceKey  string `json:"deviceKey"`
}

type sagooRealtimeSubDevice struct {
	Identity   sagooIdentityPayload `json:"identity"`
	Properties jsonFieldValueMap    `json:"properties"`
	Events     struct{}             `json:"events"`
}

type sagooRealtimeParams struct {
	Properties struct{}                 `json:"properties"`
	Events     struct{}                 `json:"events"`
	SubDevices []sagooRealtimeSubDevice `json:"subDevices"`
}

type sagooRealtimeMessage struct {
	ID      string              `json:"id"`
	Version string              `json:"version"`
	Sys     sagooSysPayload     `json:"sys"`
	Method  string              `json:"method"`
	Params  sagooRealtimeParams `json:"params"`
}

type sagooAlarmValue struct {
	FieldName   string  `json:"field_name"`
	ActualValue float64 `json:"actual_value"`
	Threshold   float64 `json:"threshold"`
	Operator    string  `json:"operator"`
	Message     string  `json:"message"`
}

type sagooAlarmEvent struct {
	Value sagooAlarmValue `json:"value"`
	Time  int64           `json:"time"`
}

type sagooAlarmEvents struct {
	Alarm sagooAlarmEvent `json:"alarm"`
}

type sagooAlarmSubDevice struct {
	Identity   sagooIdentityPayload `json:"identity"`
	Properties struct{}             `json:"properties"`
	Events     sagooAlarmEvents     `json:"events"`
}

type sagooAlarmParams struct {
	Properties struct{}              `json:"properties"`
	Events     struct{}              `json:"events"`
	SubDevices []sagooAlarmSubDevice `json:"subDevices"`
}

type sagooAlarmMessage struct {
	ID      string           `json:"id"`
	Version string           `json:"version"`
	Sys     sagooSysPayload  `json:"sys"`
	Method  string           `json:"method"`
	Params  sagooAlarmParams `json:"params"`
}

// Send 发送数据（加入缓冲队列）
func (a *SagooAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(data)
	a.dataMu.Unlock()

	return nil
}

// SendAlarm 发送报警
func (a *SagooAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.enqueueAlarmLocked(alarm)
	needFlush := len(a.alarmQueue) >= a.alarmBatch
	flushNow := a.flushNow
	a.alarmMu.Unlock()

	if needFlush {
		signalStructChan(flushNow)
	}

	return nil
}

// flushLatestData 发送实时数据
func (a *SagooAdapter) flushLatestData() error {
	a.dataMu.Lock()
	if len(a.latestData) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := a.latestData
	a.latestData = nil
	topic := a.topic
	a.dataMu.Unlock()

	for _, data := range batch {
		message := a.buildMessage(data)
		if err := a.publish(topic, message); err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			clear(batch)
			return err
		}
	}

	clear(batch)
	return nil
}

// flushAlarmBatch 发送报警批次
func (a *SagooAdapter) flushAlarmBatch() error {
	a.alarmMu.Lock()
	if len(a.alarmQueue) == 0 {
		a.alarmMu.Unlock()
		return nil
	}
	count := a.alarmBatch
	if count > len(a.alarmQueue) {
		count = len(a.alarmQueue)
	}
	batch := a.alarmQueue[:count]
	a.alarmQueue = a.alarmQueue[count:]
	a.alarmQueue = a.alarmQueue[:len(a.alarmQueue):len(a.alarmQueue)]
	topic := a.alarmTopic
	a.alarmMu.Unlock()

	for _, alarm := range batch {
		message := a.buildAlarmMessage(alarm)
		if err := a.publish(topic, message); err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			clear(batch)
			return err
		}
	}

	clear(batch)
	return nil
}

// buildMessage 构建循迹消息
func (a *SagooAdapter) buildMessage(data *models.CollectData) []byte {
	if data == nil {
		return []byte("{}")
	}

	defaultPK, defaultDK := a.defaultIdentity()
	subPK := data.ProductKey
	subDK := data.DeviceKey
	if subPK == "" {
		subPK = defaultPK
	}
	if subDK == "" {
		subDK = defaultDK
	}

	msg := sagooRealtimeMessage{
		ID:      a.nextID("msg"),
		Version: "1.0",
		Sys:     sagooSysPayload{Ack: 0},
		Method:  "thing.event.property.pack.post",
		Params: sagooRealtimeParams{
			Properties: struct{}{},
			Events:     struct{}{},
			SubDevices: []sagooRealtimeSubDevice{
				{
					Identity: sagooIdentityPayload{
						ProductKey: subPK,
						DeviceKey:  subDK,
					},
					Properties: jsonFieldValueMap(data.Fields),
					Events:     struct{}{},
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(msg)
	return jsonBytes
}

// buildAlarmMessage 构建报警消息
func (a *SagooAdapter) buildAlarmMessage(alarm *models.AlarmPayload) []byte {
	if alarm == nil {
		return []byte("{}")
	}

	defaultPK, defaultDK := a.defaultIdentity()

	msg := sagooAlarmMessage{
		ID:      a.nextID("alarm"),
		Version: "1.0",
		Sys:     sagooSysPayload{Ack: 0},
		Method:  "thing.event.property.pack.post",
		Params: sagooAlarmParams{
			Properties: struct{}{},
			Events:     struct{}{},
			SubDevices: []sagooAlarmSubDevice{
				{
					Identity: sagooIdentityPayload{
						ProductKey: pickFirstNonEmpty2(alarm.ProductKey, defaultPK),
						DeviceKey:  pickFirstNonEmpty2(alarm.DeviceKey, defaultDK),
					},
					Properties: struct{}{},
					Events: sagooAlarmEvents{
						Alarm: sagooAlarmEvent{
							Value: sagooAlarmValue{
								FieldName:   alarm.FieldName,
								ActualValue: alarm.ActualValue,
								Threshold:   alarm.Threshold,
								Operator:    alarm.Operator,
								Message:     alarm.Message,
							},
							Time: time.Now().UnixMilli(),
						},
					},
				},
			},
		},
	}

	jsonBytes, _ := json.Marshal(msg)
	return jsonBytes
}

func (a *SagooAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.latestData = appendQueueItemWithCap(a.latestData, item, a.realtimeCap)
}

func (a *SagooAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.latestData = prependQueueWithCap(a.latestData, items, a.realtimeCap)
}

func (a *SagooAdapter) enqueueAlarmLocked(alarm *models.AlarmPayload) {
	if alarm == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = appendQueueItemWithCap(a.alarmQueue, alarm, a.alarmCap)
}

func (a *SagooAdapter) prependAlarms(alarms []*models.AlarmPayload) {
	if len(alarms) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = prependQueueWithCap(a.alarmQueue, alarms, a.alarmCap)
}
