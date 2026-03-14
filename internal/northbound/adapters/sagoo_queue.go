package adapters

import (
	"encoding/json"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// Send 发送数据（加入缓冲队列）
func (a *SagooAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(cloneCollectData(data))
	a.dataMu.Unlock()

	return nil
}

// SendAlarm 发送报警
func (a *SagooAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.enqueueAlarmLocked(cloneAlarmPayload(alarm))
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
	batch := make([]*models.CollectData, len(a.latestData))
	copy(batch, a.latestData)
	clear(a.latestData)
	a.latestData = a.latestData[:0]
	topic := a.topic
	a.dataMu.Unlock()

	for _, data := range batch {
		message := a.buildMessage(data)
		if err := a.publish(topic, message); err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			return err
		}
	}

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
	batch := make([]*models.AlarmPayload, count)
	copy(batch, a.alarmQueue[:count])
	clear(a.alarmQueue[:count])
	a.alarmQueue = a.alarmQueue[count:]
	topic := a.alarmTopic
	a.alarmMu.Unlock()

	for _, alarm := range batch {
		message := a.buildAlarmMessage(alarm)
		if err := a.publish(topic, message); err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			return err
		}
	}

	return nil
}

// buildMessage 构建循迹消息
func (a *SagooAdapter) buildMessage(data *models.CollectData) []byte {
	if data == nil {
		return []byte("{}")
	}

	properties := make(map[string]interface{}, len(data.Fields))
	for key, value := range data.Fields {
		properties[key] = convertFieldValue(value)
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

	msg := map[string]interface{}{
		"id":      a.nextID("msg"),
		"version": "1.0",
		"sys": map[string]interface{}{
			"ack": 0,
		},
		"method": "thing.event.property.pack.post",
		"params": map[string]interface{}{
			"properties": map[string]interface{}{},
			"events":     map[string]interface{}{},
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]string{
						"productKey": subPK,
						"deviceKey":  subDK,
					},
					"properties": properties,
					"events":     map[string]interface{}{},
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

	eventValue := map[string]interface{}{
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"message":      alarm.Message,
	}

	event := map[string]interface{}{
		"value": eventValue,
		"time":  time.Now().UnixMilli(),
	}

	events := map[string]interface{}{
		"alarm": event,
	}

	msg := map[string]interface{}{
		"id":      a.nextID("alarm"),
		"version": "1.0",
		"sys": map[string]interface{}{
			"ack": 0,
		},
		"method": "thing.event.property.pack.post",
		"params": map[string]interface{}{
			"properties": map[string]interface{}{},
			"events":     map[string]interface{}{},
			"subDevices": []interface{}{
				map[string]interface{}{
					"identity": map[string]string{
						"productKey": pickFirstNonEmpty2(alarm.ProductKey, defaultPK),
						"deviceKey":  pickFirstNonEmpty2(alarm.DeviceKey, defaultDK),
					},
					"properties": map[string]interface{}{},
					"events":     events,
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
