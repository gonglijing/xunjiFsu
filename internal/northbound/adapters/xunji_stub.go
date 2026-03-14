//go:build no_paho_mqtt

package adapters

import (
	"fmt"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

type XunjiAdapter struct {
	name string
}

func NewXunjiAdapter(name string) *XunjiAdapter {
	return &XunjiAdapter{name: name}
}

func (a *XunjiAdapter) SetReconnectInterval(interval time.Duration) {}
func (a *XunjiAdapter) Name() string                                { return a.name }
func (a *XunjiAdapter) Type() string                                { return "xunji" }
func (a *XunjiAdapter) Initialize(configStr string) error {
	return fmt.Errorf("xunji adapter is disabled (build tag no_paho_mqtt)")
}
func (a *XunjiAdapter) Start()       {}
func (a *XunjiAdapter) Stop()        {}
func (a *XunjiAdapter) Close() error { return nil }
func (a *XunjiAdapter) Send(data *models.CollectData) error {
	return fmt.Errorf("xunji adapter is disabled (build tag no_paho_mqtt)")
}
func (a *XunjiAdapter) SendAlarm(alarm *models.AlarmPayload) error {
	return fmt.Errorf("xunji adapter is disabled (build tag no_paho_mqtt)")
}
func (a *XunjiAdapter) SetInterval(interval time.Duration) {}
func (a *XunjiAdapter) IsEnabled() bool                    { return false }
func (a *XunjiAdapter) IsConnected() bool                  { return false }
func (a *XunjiAdapter) RuntimeStatsSnapshot() RuntimeStatsSnapshot {
	return RuntimeStatsSnapshot{
		Name:      a.name,
		Type:      "xunji",
		Enabled:   false,
		Connected: false,
		Error:     "xunji adapter is disabled (build tag no_paho_mqtt)",
	}
}
func (a *XunjiAdapter) GetStats() map[string]interface{} { return a.RuntimeStatsSnapshot().ToMap() }
func (a *XunjiAdapter) GetLastSendTime() time.Time       { return time.Time{} }
func (a *XunjiAdapter) PendingCommandCount() int         { return 0 }
