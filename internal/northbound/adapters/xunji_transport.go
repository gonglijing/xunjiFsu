//go:build !no_paho_mqtt

package adapters

import (
	"fmt"
	"log"
	"time"
)

func (a *XunjiAdapter) publishBytes(topic string, payload []byte) error {
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

	token := client.Publish(topic, qos, retain, payload)
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

func (a *XunjiAdapter) signalReconnect() {
	a.reconnectState().signalReconnect()
}

func (a *XunjiAdapter) markDisconnected() {
	a.reconnectState().markDisconnected()
}

func (a *XunjiAdapter) reconnectOnce() error {
	log.Printf("Xunji [%s] attempting to reconnect...", a.name)
	client, err := connectMQTT(a.broker, a.clientID, a.username, a.password, int(a.keepAlive.Seconds()), int(a.timeout.Seconds()))
	if err != nil {
		return err
	}

	a.mu.Lock()
	oldClient := a.client
	a.client = client
	a.connected = true
	a.mu.Unlock()

	if oldClient != nil && oldClient != a.client && oldClient.IsConnected() {
		oldClient.Disconnect(250)
	}

	log.Printf("Xunji [%s] reconnected successfully", a.name)
	return nil
}

func (a *XunjiAdapter) currentReconnectInterval() time.Duration {
	return a.reconnectState().currentReconnectInterval()
}

func (a *XunjiAdapter) shouldReconnect() bool {
	return a.reconnectState().shouldReconnect()
}
