//go:build !no_paho_mqtt

package adapters

import (
	"log"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

// Send 发送数据（加入缓冲队列）
func (a *MQTTAdapter) Send(data *models.CollectData) error {
	if data == nil {
		return nil
	}

	a.pendingMu.Lock()
	a.pendingData = appendQueueItemWithCap(a.pendingData, data, mqttPendingDataCap)
	a.pendingMu.Unlock()

	signalStructChan(a.dataChan)

	return nil
}

// SendAlarm 发送报警
func (a *MQTTAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	if alarm == nil {
		return nil
	}

	a.alarmMu.Lock()
	a.pendingAlarms = appendQueueItemWithCap(a.pendingAlarms, alarm, mqttPendingAlarmCap)
	a.alarmMu.Unlock()

	signalStructChan(a.dataChan)

	return nil
}

// flushPendingData 发送待处理数据
func (a *MQTTAdapter) flushPendingData() {
	a.pendingMu.Lock()
	if len(a.pendingData) == 0 {
		a.pendingMu.Unlock()
		return
	}
	batch := a.pendingData
	a.pendingData = nil
	a.pendingMu.Unlock()

	for idx, data := range batch {
		if err := a.publish(a.topic, data); err != nil {
			log.Printf("MQTT [%s] send data failed: %v", a.name, err)
			remaining := batch[idx:]
			a.pendingMu.Lock()
			a.pendingData = prependQueueWithCap(a.pendingData, remaining, mqttPendingDataCap)
			a.pendingMu.Unlock()
			if !a.IsConnected() {
				a.signalReconnect()
			}
			clear(batch)
			return
		}
	}
	clear(batch)
}

// flushAlarms 发送报警
func (a *MQTTAdapter) flushAlarms() error {
	a.alarmMu.Lock()
	if len(a.pendingAlarms) == 0 {
		a.alarmMu.Unlock()
		return nil
	}
	batch := a.pendingAlarms
	a.pendingAlarms = nil
	a.alarmMu.Unlock()

	for idx, alarm := range batch {
		if err := a.publish(a.alarmTopic, alarm); err != nil {
			log.Printf("MQTT [%s] send alarm failed: %v", a.name, err)
			remaining := batch[idx:]
			a.alarmMu.Lock()
			a.pendingAlarms = prependQueueWithCap(a.pendingAlarms, remaining, mqttPendingAlarmCap)
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
