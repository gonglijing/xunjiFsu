package adapters

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func (a *PandaXAdapter) publish(topic string, payload []byte) error {
	a.mu.RLock()
	client := a.client
	timeout := a.timeout
	qos := a.qos
	retain := a.retain
	a.mu.RUnlock()

	if client == nil {
		slog.Info(fmt.Sprintf("[PandaX-%s] publish: MQTT client 为 nil", a.name))
		a.markDisconnected()
		return fmt.Errorf("mqtt client is nil")
	}

	if !client.IsConnected() {
		a.markDisconnected()
		return fmt.Errorf("mqtt client not connected")
	}

	slog.Info(fmt.Sprintf("[PandaX-%s] publish: topic=%s, size=%d", a.name, topic, len(payload)))

	token := client.Publish(topic, qos, retain, payload)
	if !token.WaitTimeout(timeout) {
		slog.Info(fmt.Sprintf("[PandaX-%s] publish: 发布超时", a.name))
		a.markDisconnected()
		return fmt.Errorf("mqtt publish timeout")
	}
	if err := token.Error(); err != nil {
		slog.Info(fmt.Sprintf("[PandaX-%s] publish: 发布失败: %v", a.name, err))
		a.markDisconnected()
		return err
	}

	a.mu.Lock()
	a.lastSend = time.Now()
	a.connected = true
	a.mu.Unlock()

	slog.Info(fmt.Sprintf("[PandaX-%s] publish: 发送成功", a.name))
	return nil
}

func (a *PandaXAdapter) connectPandaXMQTT(broker, clientID, username, password string, keepAliveSec, timeoutSec int) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetAutoReconnect(false)
	opts.SetConnectRetry(false)
	if keepAliveSec > 0 {
		opts.SetKeepAlive(time.Duration(keepAliveSec) * time.Second)
	}

	timeout := 10 * time.Second
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}
	opts.SetConnectTimeout(timeout)

	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		if err != nil {
			slog.Info(fmt.Sprintf("[PandaX-%s] MQTT connection lost: %v", a.name, err))
		} else {
			slog.Info(fmt.Sprintf("[PandaX-%s] MQTT connection lost", a.name))
		}
		a.markDisconnected()
	}
	opts.OnConnect = func(_ mqtt.Client) {
		slog.Info(fmt.Sprintf("[PandaX-%s] MQTT connected: %s", a.name, broker))
		a.mu.Lock()
		a.connected = true
		a.mu.Unlock()
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(timeout) {
		return nil, fmt.Errorf("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}

	return client, nil
}

func (a *PandaXAdapter) reconnectOnce() error {
	a.mu.RLock()
	client := a.client
	timeout := a.timeout
	a.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("mqtt client is nil")
	}

	token := client.Connect()
	if !token.WaitTimeout(timeout) {
		a.markDisconnected()
		return fmt.Errorf("mqtt reconnect timeout")
	}
	if err := token.Error(); err != nil {
		a.markDisconnected()
		return err
	}

	a.mu.Lock()
	a.connected = true
	a.mu.Unlock()
	a.subscribeRPCTopics(client)

	return nil
}

func (a *PandaXAdapter) shouldReconnect() bool {
	return a.reconnectState().shouldReconnect()
}

func (a *PandaXAdapter) currentReconnectInterval() time.Duration {
	return a.reconnectState().currentReconnectInterval()
}

func (a *PandaXAdapter) signalReconnect() {
	a.reconnectState().signalReconnect()
}

func (a *PandaXAdapter) markDisconnected() {
	a.reconnectState().markDisconnected()
}

func pandaXReconnectDelay(base time.Duration, failures int) time.Duration {
	if base <= 0 {
		base = defaultPandaXReconnectInterval
	}
	if base > maxPandaXReconnectInterval {
		base = maxPandaXReconnectInterval
	}
	if failures <= 0 {
		return base
	}

	delay := base
	for attempt := 1; attempt < failures; attempt++ {
		if delay >= maxPandaXReconnectInterval/2 {
			return maxPandaXReconnectInterval
		}
		delay *= 2
	}

	if delay > maxPandaXReconnectInterval {
		return maxPandaXReconnectInterval
	}
	return delay
}

func (a *PandaXAdapter) subscribeRPCTopics(client mqtt.Client) {
	a.mu.RLock()
	rpcReqTopic := a.rpcRequestTopic
	qos := a.qos
	timeout := a.timeout
	a.mu.RUnlock()

	topics := make(map[string]struct{})
	if strings.TrimSpace(rpcReqTopic) != "" {
		topics[strings.TrimSpace(rpcReqTopic)] = struct{}{}
		if !strings.HasSuffix(strings.TrimSpace(rpcReqTopic), "/+") {
			topics[strings.TrimRight(strings.TrimSpace(rpcReqTopic), "/")+"/+"] = struct{}{}
		}
	}

	slog.Info(fmt.Sprintf("[PandaX-%s] subscribeRPCTopics: 开始订阅 topics=%v", a.name, topics))

	for topic := range topics {
		token := client.Subscribe(topic, qos, a.handleRPCRequest)
		if !token.WaitTimeout(timeout) {
			slog.Info(fmt.Sprintf("[PandaX-%s] subscribeRPCTopics: 订阅超时 topic=%s", a.name, topic))
			continue
		}
		if err := token.Error(); err != nil {
			slog.Info(fmt.Sprintf("[PandaX-%s] subscribeRPCTopics: 订阅失败 topic=%s: %v", a.name, topic, err))
		} else {
			slog.Info(fmt.Sprintf("[PandaX-%s] subscribeRPCTopics: 订阅成功 topic=%s", a.name, topic))
		}
	}
}
