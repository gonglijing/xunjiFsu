//go:build no_paho_mqtt

package adapters

import (
	"fmt"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

const (
	mqttPendingDataCap  = 1000
	mqttPendingAlarmCap = 100
)

type MQTTAdapter struct {
	name string
}

func NewMQTTAdapter(name string) *MQTTAdapter {
	return &MQTTAdapter{name: name}
}

func (a *MQTTAdapter) SetReconnectInterval(interval time.Duration) {}
func (a *MQTTAdapter) Name() string                                { return a.name }
func (a *MQTTAdapter) Type() string                                { return "mqtt" }
func (a *MQTTAdapter) Initialize(configStr string) error {
	return fmt.Errorf("mqtt adapter is disabled (build tag no_paho_mqtt)")
}
func (a *MQTTAdapter) Start()       {}
func (a *MQTTAdapter) Stop()        {}
func (a *MQTTAdapter) Close() error { return nil }
func (a *MQTTAdapter) Send(data *models.CollectData) error {
	return fmt.Errorf("mqtt adapter is disabled (build tag no_paho_mqtt)")
}
func (a *MQTTAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	return fmt.Errorf("mqtt adapter is disabled (build tag no_paho_mqtt)")
}
func (a *MQTTAdapter) SetInterval(interval time.Duration) {}
func (a *MQTTAdapter) IsEnabled() bool                    { return false }
func (a *MQTTAdapter) IsConnected() bool                  { return false }
func (a *MQTTAdapter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"type":      "mqtt",
		"enabled":   false,
		"connected": false,
		"error":     "mqtt adapter is disabled (build tag no_paho_mqtt)",
	}
}
func (a *MQTTAdapter) GetLastSendTime() time.Time { return time.Time{} }
func (a *MQTTAdapter) PendingCommandCount() int   { return 0 }
