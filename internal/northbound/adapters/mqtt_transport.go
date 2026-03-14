//go:build !no_paho_mqtt

package adapters

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// connectMQTT 创建并连接MQTT客户端
func (a *MQTTAdapter) connectMQTT(settings mqttInitSettings, username, password string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(settings.broker).
		SetClientID(settings.clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(defaultReconnectInterval)

	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	if settings.cleanSession {
		opts.SetCleanSession(true)
	} else {
		opts.SetCleanSession(false)
	}
	if settings.keepAlive > 0 {
		opts.SetKeepAlive(settings.keepAlive)
	}
	opts.SetConnectTimeout(settings.timeout)

	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		if err != nil {
			log.Printf("MQTT [%s] connection lost: %v", a.name, err)
		}
		a.markDisconnected()
	}
	opts.OnConnect = func(_ mqtt.Client) {
		log.Printf("MQTT [%s] connected: %s", a.name, settings.broker)
		a.mu.Lock()
		a.connected = true
		a.mu.Unlock()
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()

	if !token.WaitTimeout(settings.timeout) {
		return nil, fmt.Errorf("MQTT [%s] connect timeout", a.name)
	}
	if err := token.Error(); err != nil {
		return nil, err
	}

	return client, nil
}

func (a *MQTTAdapter) signalReconnect() {
	a.reconnectState().signalReconnect()
}

func (a *MQTTAdapter) markDisconnected() {
	a.reconnectState().markDisconnected()
}

func (a *MQTTAdapter) reconnectOnce() error {
	log.Printf("MQTT [%s] attempting to reconnect...", a.name)
	a.mu.RLock()
	settings := mqttInitSettings{
		broker:       a.broker,
		clientID:     a.clientID,
		cleanSession: a.cleanSession,
		timeout:      a.timeout,
		keepAlive:    a.keepAlive,
	}
	username := a.username
	password := a.password
	a.mu.RUnlock()

	client, err := a.connectMQTT(settings, username, password)
	if err != nil {
		return err
	}

	a.mu.Lock()
	oldClient := a.client
	a.client = client
	a.connected = true
	a.mu.Unlock()

	if oldClient != nil && oldClient != client && oldClient.IsConnected() {
		oldClient.Disconnect(250)
	}

	log.Printf("MQTT [%s] reconnected successfully", a.name)
	return nil
}

// publish 发布消息
func (a *MQTTAdapter) publish(topic string, payload interface{}) error {
	a.mu.RLock()
	if !a.initialized || !a.enabled {
		a.mu.RUnlock()
		return fmt.Errorf("adapter not initialized or disabled")
	}
	client := a.client
	qos := a.qos
	retain := a.retain
	timeout := a.timeout
	a.mu.RUnlock()

	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	var body []byte
	if data, ok := payload.(*models.CollectData); ok {
		msg := map[string]interface{}{
			"device_name": data.DeviceName,
			"device_id":   data.DeviceID,
			"timestamp":   data.Timestamp.Unix(),
			"fields":      data.Fields,
		}
		body, _ = json.Marshal(msg)
	} else if alarm, ok := payload.(*models.AlarmPayload); ok {
		msg := map[string]interface{}{
			"device_id":    alarm.DeviceID,
			"device_name":  alarm.DeviceName,
			"field_name":   alarm.FieldName,
			"actual_value": alarm.ActualValue,
			"threshold":    alarm.Threshold,
			"operator":     alarm.Operator,
			"severity":     alarm.Severity,
			"message":      alarm.Message,
			"timestamp":    time.Now().Unix(),
		}
		body, _ = json.Marshal(msg)
	} else {
		return fmt.Errorf("unknown payload type")
	}

	token := client.Publish(topic, qos, retain, body)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		a.markDisconnected()
		return err
	}

	a.mu.Lock()
	a.lastSend = time.Now()
	a.connected = true
	a.mu.Unlock()

	return nil
}

func (a *MQTTAdapter) currentReconnectInterval() time.Duration {
	return a.reconnectState().currentReconnectInterval()
}

func (a *MQTTAdapter) shouldReconnect() bool {
	return a.reconnectState().shouldReconnect()
}
