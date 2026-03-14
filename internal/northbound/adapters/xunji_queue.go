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
	a.pendingData = appendQueueItemWithCap(a.pendingData, data, xunjiPendingDataCap)
	a.pendingMu.Unlock()

	signalStructChan(a.dataChan)

	return nil
}

func (a *XunjiAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.pendingAlarms = appendQueueItemWithCap(a.pendingAlarms, alarm, xunjiPendingAlarmCap)
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
	a.pendingData = nil
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
		clear(batch)
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
	a.pendingAlarms = nil
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
			clear(batch)
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

type xunjiRealtimePayloadItem struct {
	TS     int64             `json:"ts"`
	Values jsonFieldValueMap `json:"values"`
}

type xunjiAlarmMessage struct {
	DeviceID    int64   `json:"device_id"`
	DeviceName  string  `json:"device_name"`
	FieldName   string  `json:"field_name"`
	ActualValue float64 `json:"actual_value"`
	Threshold   float64 `json:"threshold"`
	Operator    string  `json:"operator"`
	Severity    string  `json:"severity"`
	Message     string  `json:"message"`
	Timestamp   int64   `json:"timestamp"`
}

func (a *XunjiAdapter) buildBatchRealtimePayload(batch []*models.CollectData) []byte {
	if len(batch) == 0 {
		return []byte("{}")
	}

	a.mu.RLock()
	mode := a.subDeviceTokenMode
	a.mu.RUnlock()

	payload := make(map[string]xunjiRealtimePayloadItem, len(batch))
	for _, data := range batch {
		if data == nil {
			continue
		}
		token := resolveXunjiSubToken(data, mode)
		ts := data.Timestamp.UnixMilli()
		if ts <= 0 {
			ts = time.Now().UnixMilli()
		}

		payload[token] = xunjiRealtimePayloadItem{
			TS:     ts,
			Values: jsonFieldValueMap(data.Fields),
		}
	}

	body, _ := json.Marshal(payload)
	return body
}

func (a *XunjiAdapter) buildAlarmMessage(alarm *models.AlarmPayload) []byte {
	if alarm == nil {
		return []byte("{}")
	}

	msg := xunjiAlarmMessage{
		DeviceID:    alarm.DeviceID,
		DeviceName:  alarm.DeviceName,
		FieldName:   alarm.FieldName,
		ActualValue: alarm.ActualValue,
		Threshold:   alarm.Threshold,
		Operator:    alarm.Operator,
		Severity:    alarm.Severity,
		Message:     alarm.Message,
		Timestamp:   time.Now().UnixMilli(),
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
