package adapters

import (
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (a *IThingsAdapter) publish(topic string, payload []byte) error {
	a.mu.RLock()
	client := a.client
	timeout := a.timeout
	qos := a.qos
	retain := a.retain
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("mqtt client is nil")
	}

	if !client.IsConnected() {
		token := client.Connect()
		if !token.WaitTimeout(timeout) {
			return fmt.Errorf("mqtt connect timeout")
		}
		if err := token.Error(); err != nil {
			a.mu.Lock()
			a.connected = false
			a.mu.Unlock()
			return err
		}
		a.subscribeDownTopics(client)
	}

	token := client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		a.mu.Lock()
		a.connected = false
		a.mu.Unlock()
		return err
	}

	a.mu.Lock()
	a.lastSend = time.Now()
	a.connected = true
	a.mu.Unlock()

	return nil
}

func (a *IThingsAdapter) subscribeDownTopics(client mqtt.Client) {
	a.mu.RLock()
	downPropertyTopic := strings.TrimSpace(a.downPropertyTopic)
	downActionTopic := strings.TrimSpace(a.downActionTopic)
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	topics := make(map[string]struct{})
	if downPropertyTopic != "" {
		topics[downPropertyTopic] = struct{}{}
	}
	if downActionTopic != "" {
		topics[downActionTopic] = struct{}{}
	}

	for topic := range topics {
		token := client.Subscribe(topic, qos, a.handleDownlink)
		if !token.WaitTimeout(timeout) {
			continue
		}
		if err := token.Error(); err != nil {
			log.Printf("iThings subscribe failed topic=%s: %v", topic, err)
		}
	}
}
