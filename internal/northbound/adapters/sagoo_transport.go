package adapters

import (
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// publish 发布MQTT消息
func (a *SagooAdapter) publish(topic string, payload []byte) error {
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
			return err
		}
		a.subscribeCommandTopics(client)
	}

	token := client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(timeout) {
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		return err
	}

	return nil
}

// subscribeCommandTopics 订阅命令主题
func (a *SagooAdapter) subscribeCommandTopics(client mqtt.Client) {
	a.mu.RLock()
	cfg := a.config
	a.mu.RUnlock()

	if cfg == nil {
		return
	}

	pk := strings.TrimSpace(cfg.ProductKey)
	dk := strings.TrimSpace(cfg.DeviceKey)
	if pk == "" || dk == "" {
		return
	}

	a.subscribe(client, sagooSysTopic(pk, dk, "thing/service/property/set"), a.handlePropertySet)
	a.subscribe(client, sagooSysTopic(pk, dk, "thing/service/+"), a.handleServiceCall)
	a.subscribe(client, sagooSysTopic(pk, dk, "thing/config/push"), a.handleConfigPush)
}

func (a *SagooAdapter) subscribe(client mqtt.Client, topic string, handler mqtt.MessageHandler) {
	a.mu.RLock()
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	token := client.Subscribe(topic, qos, handler)
	if !token.WaitTimeout(timeout) {
		return
	}
}
