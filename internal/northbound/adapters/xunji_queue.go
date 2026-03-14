//go:build !no_paho_mqtt

package adapters

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *XunjiAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.pendingMu.Lock()
	a.pendingData = appendQueueItemWithCap(a.pendingData, cloneCollectData(data), xunjiPendingDataCap)
	a.pendingMu.Unlock()

	signalStructChan(a.dataChan)

	return nil
}

func (a *XunjiAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.pendingAlarms = appendQueueItemWithCap(a.pendingAlarms, cloneAlarmPayload(alarm), xunjiPendingAlarmCap)
	a.alarmMu.Unlock()

	signalStructChan(a.dataChan)

	return nil
}

func (a *XunjiAdapter) flushPendingData() {
	a.pendingMu.Lock()
	if len(a.pendingData) == 0 {
		a.pendingMu.Unlock()
		return
	}
	batch := a.pendingData
	a.pendingData = a.pendingData[:0]
	a.pendingMu.Unlock()

	body := a.buildBatchRealtimePayload(batch)
	topic := a.currentTopic()
	if err := a.publishBytes(topic, body); err != nil {
		logXunjiSendFailure(a.name, "data", err)
		a.pendingMu.Lock()
		a.pendingData = prependQueueWithCap(a.pendingData, batch, xunjiPendingDataCap)
		a.pendingMu.Unlock()
		if !a.IsConnected() {
			a.signalReconnect()
		}
		return
	}

	clear(batch)
}

func (a *XunjiAdapter) flushAlarms() error {
	a.alarmMu.Lock()
	if len(a.pendingAlarms) == 0 {
		a.alarmMu.Unlock()
		return nil
	}
	batch := a.pendingAlarms
	a.pendingAlarms = a.pendingAlarms[:0]
	a.alarmMu.Unlock()

	topic := a.currentAlarmTopic()
	for idx, alarm := range batch {
		body := a.buildAlarmMessage(alarm)
		if err := a.publishBytes(topic, body); err != nil {
			logXunjiSendFailure(a.name, "alarm", err)
			remaining := batch[idx:]
			a.alarmMu.Lock()
			a.pendingAlarms = prependQueueWithCap(a.pendingAlarms, remaining, xunjiPendingAlarmCap)
			a.alarmMu.Unlock()
			if !a.IsConnected() {
				a.signalReconnect()
			}
			return err
		}
	}

	clear(batch)
	return nil
}

func (a *XunjiAdapter) currentTopic() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.topic
}

func (a *XunjiAdapter) currentAlarmTopic() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.alarmTopic
}

func (a *XunjiAdapter) buildBatchRealtimePayload(batch []*models.CollectData) []byte {
	if len(batch) == 0 {
		return []byte("{}")
	}

	a.mu.RLock()
	mode := a.subDeviceTokenMode
	a.mu.RUnlock()

	payload := make(map[string]interface{}, len(batch))
	for _, data := range batch {
		if data == nil {
			continue
		}
		token := resolveXunjiSubToken(data, mode)
		values := make(map[string]interface{}, len(data.Fields))
		for key, value := range data.Fields {
			values[key] = convertFieldValue(value)
		}

		ts := data.Timestamp.UnixMilli()
		if ts <= 0 {
			ts = time.Now().UnixMilli()
		}

		payload[token] = map[string]interface{}{
			"ts":     ts,
			"values": values,
		}
	}

	body, _ := json.Marshal(payload)
	return body
}

func (a *XunjiAdapter) buildAlarmMessage(alarm *models.AlarmPayload) []byte {
	if alarm == nil {
		return []byte("{}")
	}

	msg := map[string]interface{}{
		"device_id":    alarm.DeviceID,
		"device_name":  alarm.DeviceName,
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"severity":     alarm.Severity,
		"message":      alarm.Message,
		"timestamp":    time.Now().UnixMilli(),
	}
	body, _ := json.Marshal(msg)
	return body
}

func logXunjiSendFailure(adapterName, kind string, err error) {
	if err == nil {
		return
	}
	// Shared helper keeps queue methods focused on queue flow.
	log.Printf("Xunji [%s] send %s failed: %v", adapterName, kind, err)
}
