package adapters

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type pandaXRealtimePayloadItem struct {
	TS     int64             `json:"ts"`
	Values jsonFieldValueMap `json:"values"`
}

type pandaXAlarmPublishPayload struct {
	DeviceName  string  `json:"device_name"`
	ProductKey  string  `json:"product_key"`
	DeviceKey   string  `json:"device_key"`
	FieldName   string  `json:"field_name"`
	ActualValue float64 `json:"actual_value"`
	Threshold   float64 `json:"threshold"`
	Operator    string  `json:"operator"`
	Severity    string  `json:"severity"`
	Message     string  `json:"message"`
	TS          int64   `json:"ts"`
}

func (a *PandaXAdapter) Send(data *models.CollectData) error {
	if data == nil {
		slog.Info("PandaX send skipped nil data", "adapter", a.name)
		return nil
	}

	slog.Info("PandaX enqueue realtime data",
		"adapter", a.name,
		"device_id", data.DeviceID,
		"device_key", data.DeviceKey,
		"fields", len(data.Fields))

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(data)
	queueLen := len(a.realtimeQueue)
	a.dataMu.Unlock()

	slog.Info("PandaX realtime data enqueued", "adapter", a.name, "queue_len", queueLen)
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

	slog.Info("PandaX alarm enqueued",
		"adapter", a.name,
		"device_key", alarm.DeviceKey,
		"field_name", alarm.FieldName,
		"severity", alarm.Severity,
		"message", alarm.Message,
		"queue_len", queueLen)

	if queueLen >= a.alarmBatch {
		slog.Info("PandaX alarm batch threshold reached", "adapter", a.name, "batch_size", a.alarmBatch)
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

	slog.Info("PandaX realtime flush start", "adapter", a.name, "count", len(batch))

	topic, body := a.buildBatchRealtimePublish(batch)
	if err := a.publish(topic, body); err != nil {
		slog.Info("PandaX realtime flush failed", "adapter", a.name, "error", err)
		a.dataMu.Lock()
		a.prependRealtime(batch)
		a.dataMu.Unlock()
		clear(batch)
		return err
	}

	slog.Info("PandaX realtime flush succeeded", "adapter", a.name, "count", len(batch))
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

	payload := make(map[string]pandaXRealtimePayloadItem, len(batch))
	for _, data := range batch {
		if data == nil {
			continue
		}

		subToken := a.resolveSubDeviceToken(data)
		ts := data.Timestamp.UnixMilli()
		if ts <= 0 {
			ts = time.Now().UnixMilli()
		}

		payload[subToken] = pandaXRealtimePayloadItem{
			TS:     ts,
			Values: jsonFieldValueMap(data.Fields),
		}
	}

	body, _ := json.Marshal(payload)
	slog.Info("PandaX build realtime payload", "adapter", a.name, "device_count", len(payload), "payload_size", len(body))

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

	slog.Info("PandaX alarm flush start", "adapter", a.name, "count", len(batch))

	successCount := 0
	for _, item := range batch {
		topic, body := a.buildAlarmPublish(item)
		if err := a.publish(topic, body); err != nil {
			slog.Info("PandaX alarm flush failed",
				"adapter", a.name,
				"device_key", item.DeviceKey,
				"field_name", item.FieldName,
				"error", err)
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			clear(batch)
			return err
		}
		successCount++
	}

	slog.Info("PandaX alarm flush succeeded", "adapter", a.name, "success_count", successCount, "count", len(batch))
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

	payload := pandaXAlarmPublishPayload{
		DeviceName:  alarm.DeviceName,
		ProductKey:  alarm.ProductKey,
		DeviceKey:   alarm.DeviceKey,
		FieldName:   alarm.FieldName,
		ActualValue: alarm.ActualValue,
		Threshold:   alarm.Threshold,
		Operator:    alarm.Operator,
		Severity:    alarm.Severity,
		Message:     alarm.Message,
		TS:          time.Now().UnixMilli(),
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
