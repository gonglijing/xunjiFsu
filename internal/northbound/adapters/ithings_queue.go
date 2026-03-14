package adapters

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (a *IThingsAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.dataMu.Lock()
	a.enqueueRealtimeLocked(data)
	a.dataMu.Unlock()

	return nil
}

func (a *IThingsAdapter) SendAlarm(alarm *models.AlarmPayload) error {
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

func (a *IThingsAdapter) flushRealtime() error {
	a.dataMu.Lock()
	if len(a.realtimeQueue) == 0 {
		a.dataMu.Unlock()
		return nil
	}
	batch := a.realtimeQueue
	a.realtimeQueue = nil
	a.dataMu.Unlock()

	for _, item := range batch {
		topic, body, err := a.buildRealtimePublish(item)
		if err != nil {
			a.dataMu.Lock()
			a.prependRealtime(batch)
			a.dataMu.Unlock()
			clear(batch)
			return err
		}
		if err := a.publish(topic, body); err != nil {
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

func (a *IThingsAdapter) flushAlarmBatch() error {
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

	for _, item := range batch {
		topic, body, err := a.buildAlarmPublish(item)
		if err != nil {
			a.alarmMu.Lock()
			a.prependAlarms(batch)
			a.alarmMu.Unlock()
			clear(batch)
			return err
		}
		if err := a.publish(topic, body); err != nil {
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

func (a *IThingsAdapter) buildRealtimePublish(data *models.CollectData) (string, []byte, error) {
	a.mu.RLock()
	cfg := a.config
	upPropertyTpl := a.upPropertyTopicTemplate
	deviceNameMode := a.deviceNameMode
	subDeviceNameMode := a.subDeviceNameMode
	a.mu.RUnlock()

	if data == nil {
		return upPropertyTpl, []byte("{}"), nil
	}

	values := make(map[string]interface{}, len(data.Fields))
	for key, value := range data.Fields {
		values[key] = convertFieldValue(value)
	}

	ts := data.Timestamp.UnixMilli()
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}

	if cfg == nil {
		return "", nil, fmt.Errorf("ithings config is nil")
	}

	gatewayProductID := strings.TrimSpace(cfg.ProductKey)
	gatewayDeviceName := strings.TrimSpace(cfg.DeviceKey)
	if gatewayProductID == "" || gatewayDeviceName == "" {
		return "", nil, fmt.Errorf("productKey and deviceKey are required for iThings gateway mode")
	}
	topic := renderIThingsTopic(upPropertyTpl, gatewayProductID, gatewayDeviceName)

	subProductID := pickFirstNonEmpty2(strings.TrimSpace(data.ProductKey), gatewayProductID)
	subDeviceName := pickFirstNonEmpty2(a.resolveCollectDeviceName(data, subDeviceNameMode), a.resolveCollectDeviceName(data, deviceNameMode))
	if subDeviceName == "" {
		subDeviceName = defaultDeviceToken(data.DeviceID)
	}

	payload := map[string]interface{}{
		"method":     "packReport",
		"msgToken":   a.nextID("pack"),
		"timestamp":  ts,
		"properties": []interface{}{},
		"events":     []interface{}{},
		"subDevices": []interface{}{
			map[string]interface{}{
				"productID":  subProductID,
				"deviceName": subDeviceName,
				"properties": []interface{}{
					map[string]interface{}{
						"timestamp": ts,
						"params":    values,
					},
				},
				"events": []interface{}{},
			},
		},
	}
	body, _ := json.Marshal(payload)
	return topic, body, nil
}

func (a *IThingsAdapter) buildAlarmPublish(alarm *models.AlarmPayload) (string, []byte, error) {
	a.mu.RLock()
	cfg := a.config
	upEventTpl := a.upEventTopicTemplate
	alarmEventID := a.alarmEventID
	alarmEventType := a.alarmEventType
	deviceNameMode := a.deviceNameMode
	a.mu.RUnlock()

	if alarm == nil {
		return upEventTpl, []byte("{}"), nil
	}

	if cfg == nil {
		return "", nil, fmt.Errorf("ithings config is nil")
	}
	gatewayProductID := strings.TrimSpace(cfg.ProductKey)
	gatewayDeviceName := strings.TrimSpace(cfg.DeviceKey)
	if gatewayProductID == "" || gatewayDeviceName == "" {
		return "", nil, fmt.Errorf("productKey and deviceKey are required for iThings gateway mode")
	}
	topic := renderIThingsTopic(upEventTpl, gatewayProductID, gatewayDeviceName)

	subProductID := pickFirstNonEmpty2(strings.TrimSpace(alarm.ProductKey), gatewayProductID)
	subDeviceName := strings.TrimSpace(a.resolveAlarmDeviceName(alarm, deviceNameMode))
	if subDeviceName == "" {
		subDeviceName = defaultDeviceToken(alarm.DeviceID)
	}

	params := map[string]interface{}{
		"device_name":  alarm.DeviceName,
		"product_key":  subProductID,
		"device_key":   subDeviceName,
		"field_name":   alarm.FieldName,
		"actual_value": alarm.ActualValue,
		"threshold":    alarm.Threshold,
		"operator":     alarm.Operator,
		"severity":     alarm.Severity,
		"message":      alarm.Message,
	}
	payload := map[string]interface{}{
		"method":    "eventPost",
		"msgToken":  a.nextID("alarm"),
		"timestamp": time.Now().UnixMilli(),
		"eventID":   alarmEventID,
		"type":      alarmEventType,
		"params":    params,
	}
	body, _ := json.Marshal(payload)
	return topic, body, nil
}

func (a *IThingsAdapter) enqueueRealtimeLocked(item *models.CollectData) {
	if item == nil {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.realtimeQueue = appendQueueItemWithCap(a.realtimeQueue, item, a.realtimeCap)
}

func (a *IThingsAdapter) prependRealtime(items []*models.CollectData) {
	if len(items) == 0 {
		return
	}
	if a.realtimeCap <= 0 {
		a.realtimeCap = defaultRealtimeQueue
	}
	a.realtimeQueue = prependQueueWithCap(a.realtimeQueue, items, a.realtimeCap)
}

func (a *IThingsAdapter) enqueueAlarmLocked(item *models.AlarmPayload) {
	if item == nil {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = appendQueueItemWithCap(a.alarmQueue, item, a.alarmCap)
}

func (a *IThingsAdapter) prependAlarms(items []*models.AlarmPayload) {
	if len(items) == 0 {
		return
	}
	if a.alarmCap <= 0 {
		a.alarmCap = defaultAlarmQueue
	}
	a.alarmQueue = prependQueueWithCap(a.alarmQueue, items, a.alarmCap)
}
