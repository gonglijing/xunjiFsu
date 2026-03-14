//go:build !no_paho_mqtt

package adapters

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// connectMQTT 创建并连接通用 MQTT 客户端。
func connectMQTT(broker, clientID, username, password string, keepAliveSec, timeoutSec int) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().AddBroker(broker)
	opts.SetClientID(clientID)
	if username != "" {
		opts.SetUsername(username)
	}
	if password != "" {
		opts.SetPassword(password)
	}
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(2 * time.Second)
	if keepAliveSec > 0 {
		opts.SetKeepAlive(time.Duration(keepAliveSec) * time.Second)
	}
	if timeoutSec > 0 {
		opts.SetConnectTimeout(time.Duration(timeoutSec) * time.Second)
	} else {
		opts.SetConnectTimeout(10 * time.Second)
	}
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		if err != nil {
			log.Printf("MQTT connection lost: %v", err)
		}
	}
	opts.OnConnect = func(_ mqtt.Client) {
		log.Printf("MQTT connected: %s", broker)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	timeout := 10 * time.Second
	if timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}
	if !token.WaitTimeout(timeout) {
		return nil, fmt.Errorf("mqtt connect timeout")
	}
	if err := token.Error(); err != nil {
		return nil, err
	}
	return client, nil
}
