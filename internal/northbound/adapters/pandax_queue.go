package adapters

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *PandaXAdapter) Send(data *models.CollectData) error {
	if data == nil {
		log.Printf("[PandaX-%s] Send: data is nil", a.name)
		return nil
	}

	log.Printf("[PandaX-%s] Send: deviceId=%d, deviceKey=%s, fields=%d",
		a.name, data.DeviceID, data.DeviceKey, len(data.Fields))

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(data)
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
	a.enqueueAlarmLocked(alarm)
	queueLen := len(a.alarmQueue)
	a.alarmMu.Unlock()

	log.Printf("[PandaX-%s] SendAlarm: deviceKey=%s, fieldName=%s, severity=%s, message=%s, queueLen=%d",
		a.name, alarm.DeviceKey, alarm.FieldName, alarm.Severity, alarm.Message, queueLen)

	if queueLen >= a.alarmBatch {
		log.Printf("[PandaX-%s] SendAlarm: 触发批量上报, batchSize=%d", a.name, a.alarmBatch)
	}

	return nil
}

func (a *PandaXAdapter) flushRealtime() error {
	a.dataMu.Lock()
	if len(a.realtimeQueue) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := a.realtimeQueue
	a.realtimeQueue = nil
	a.dataMu.Unlock()

	log.Printf("[PandaX-%s] flushRealtime: 开始发送 %d 条数据", a.name, len(batch))

	topic, body := a.buildBatchRealtimePublish(batch)
	if err := a.publish(topic, body); err != nil {
		log.Printf("[PandaX-%s] flushRealtime: 发送失败: %v", a.name, err)
		a.dataMu.Lock()
		a.prependRealtime(batch)
		a.dataMu.Unlock()
		clear(batch)
		return err
	}

	log.Printf("[PandaX-%s] flushRealtime: 发送成功 %d 条数据", a.name, len(batch))
	clear(batch)
	return nil
}

func (a *PandaXAdapter) buildBatchRealtimePublish(batch []*models.CollectData) (string, []byte) {
	a.mu.RLock()
	topic := a.gatewayTelemetryTopic
	a.mu.RUnlock()

	if len(batch) == 0 {
		return topic, []byte("{}")
	}

	payload := make(map[string]interface{}, len(batch))
	for _, data := range batch {
		if data == nil {
			continue
		}

		subToken := a.resolveSubDeviceToken(data)
		values := make(map[string]interface{}, len(data.Fields))
		for key, value := range data.Fields {
			values[key] = convertFieldValue(value)
		}

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
	batch := a.alarmQueue[:count]
	a.alarmQueue = a.alarmQueue[count:]
	a.alarmQueue = a.alarmQueue[:len(a.alarmQueue):len(a.alarmQueue)]
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
			clear(batch)
			return err
		}
		successCount++
	}

	log.Printf("[PandaX-%s] flushAlarmBatch: 发送成功 %d/%d", a.name, successCount, len(batch))
	clear(batch)
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

func (a *PandaXAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.realtimeQueue = appendQueueItemWithCap(a.realtimeQueue, item, a.realtimeCap)
}

func (a *PandaXAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.realtimeQueue = prependQueueWithCap(a.realtimeQueue, items, a.realtimeCap)
}

func (a *PandaXAdapter) enqueueAlarmLocked(item *models.AlarmPayload) {
	if item == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = appendQueueItemWithCap(a.alarmQueue, item, a.alarmCap)
}

func (a *PandaXAdapter) prependAlarms(items []*models.AlarmPayload) {
	if len(items) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = prependQueueWithCap(a.alarmQueue, items, a.alarmCap)
}
