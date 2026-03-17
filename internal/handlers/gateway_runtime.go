package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/database"
)

type gatewayRuntimeConfig struct {
	CollectorDeviceSyncInterval     string `json:"collector_device_sync_interval"`
	CollectorCommandPollInterval    string `json:"collector_command_poll_interval"`
	NorthboundMQTTReconnectInterval string `json:"northbound_mqtt_reconnect_interval"`
	DriverSerialReadTimeout         string `json:"driver_serial_read_timeout"`
	DriverTCPDialTimeout            string `json:"driver_tcp_dial_timeout"`
	DriverTCPReadTimeout            string `json:"driver_tcp_read_timeout"`
	DriverSerialOpenBackoff         string `json:"driver_serial_open_backoff"`
	DriverTCPDialBackoff            string `json:"driver_tcp_dial_backoff"`
	DriverSerialOpenRetries         *int   `json:"driver_serial_open_retries"`
	DriverTCPDialRetries            *int   `json:"driver_tcp_dial_retries"`
}

type runtimeConfigChange struct {
	From interface{} `json:"from"`
	To   interface{} `json:"to"`
}

type runtimeConfigAuditView struct {
	ID               int64                          `json:"id"`
	OperatorUserID   int64                          `json:"operator_user_id"`
	OperatorUsername string                         `json:"operator_username"`
	SourceIP         string                         `json:"source_ip"`
	CreatedAt        string                         `json:"created_at"`
	Changes          map[string]runtimeConfigChange `json:"changes,omitempty"`
	ChangesRaw       string                         `json:"changes_raw,omitempty"`
}

func (h *Handler) GetGatewayRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	if h.appConfig == nil {
		WriteSuccess(w, map[string]interface{}{})
		return
	}

	WriteSuccess(w, h.gatewayRuntimeConfigView())
}

func (h *Handler) UpdateGatewayRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	if h.appConfig == nil {
		writeServerErrorWithLog(w, apiErrUpdateRuntimeConfigFailed, nil)
		return
	}

	var payload gatewayRuntimeConfig
	if !parseRequestOrWriteBadRequestDefault(w, r, &payload) {
		return
	}

	changes, err := h.applyGatewayRuntimeConfig(&payload)
	if err != nil {
		WriteBadRequestCode(w, apiErrUpdateRuntimeConfigFailed.Code, apiErrUpdateRuntimeConfigFailed.Message+": "+err.Error())
		return
	}

	if len(changes) > 0 {
		h.recordRuntimeConfigAudit(r, changes)
	}

	h.GetGatewayRuntimeConfig(w, r)
}

func (h *Handler) applyGatewayRuntimeConfig(payload *gatewayRuntimeConfig) (map[string]runtimeConfigChange, error) {
	if payload == nil || h.appConfig == nil {
		return nil, nil
	}

	changes := make(map[string]runtimeConfigChange)

	if err := applyGatewayDurationConfigChange(changes, "collector_device_sync_interval", payload.CollectorDeviceSyncInterval, &h.appConfig.CollectorDeviceSyncInterval); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "collector_command_poll_interval", payload.CollectorCommandPollInterval, &h.appConfig.CollectorCommandPollInterval); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "northbound_mqtt_reconnect_interval", payload.NorthboundMQTTReconnectInterval, &h.appConfig.NorthboundMQTTReconnectInterval); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "driver_serial_read_timeout", payload.DriverSerialReadTimeout, &h.appConfig.DriverSerialReadTimeout); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "driver_tcp_dial_timeout", payload.DriverTCPDialTimeout, &h.appConfig.DriverTCPDialTimeout); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "driver_tcp_read_timeout", payload.DriverTCPReadTimeout, &h.appConfig.DriverTCPReadTimeout); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "driver_serial_open_backoff", payload.DriverSerialOpenBackoff, &h.appConfig.DriverSerialOpenBackoff); err != nil {
		return nil, err
	}
	if err := applyGatewayDurationConfigChange(changes, "driver_tcp_dial_backoff", payload.DriverTCPDialBackoff, &h.appConfig.DriverTCPDialBackoff); err != nil {
		return nil, err
	}

	if err := applyGatewayRetryConfigChange(changes, "driver_serial_open_retries", payload.DriverSerialOpenRetries, &h.appConfig.DriverSerialOpenRetries); err != nil {
		return nil, err
	}
	if err := applyGatewayRetryConfigChange(changes, "driver_tcp_dial_retries", payload.DriverTCPDialRetries, &h.appConfig.DriverTCPDialRetries); err != nil {
		return nil, err
	}

	if h.collector != nil {
		h.collector.SetRuntimeIntervals(h.appConfig.CollectorDeviceSyncInterval, h.appConfig.CollectorCommandPollInterval)
	}

	if h.driverExecutor != nil {
		h.driverExecutor.SetTimeouts(h.appConfig.DriverSerialReadTimeout, h.appConfig.DriverTCPDialTimeout, h.appConfig.DriverTCPReadTimeout)
		h.driverExecutor.SetRetries(h.appConfig.DriverSerialOpenRetries, h.appConfig.DriverTCPDialRetries, h.appConfig.DriverSerialOpenBackoff, h.appConfig.DriverTCPDialBackoff)
	}

	h.applyNorthboundRuntimeConfigFromHandler()

	return changes, nil
}

func (h *Handler) GetGatewayRuntimeAudits(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			WriteBadRequestCode(w, apiErrUpdateRuntimeConfigFailed.Code, apiErrUpdateRuntimeConfigFailed.Message+": invalid limit")
			return
		}
		limit = parsed
	}

	items, err := database.ListRuntimeConfigAudits(limit)
	if err != nil {
		writeServerErrorWithLog(w, apiErrListRuntimeConfigAuditsFailed, err)
		return
	}

	WriteSuccess(w, buildRuntimeAuditViews(items))
}

func (h *Handler) recordRuntimeConfigAudit(r *http.Request, changes map[string]runtimeConfigChange) {
	if len(changes) == 0 {
		return
	}

	changeJSON, err := json.Marshal(changes)
	if err != nil {
		return
	}

	session := auth.SessionFromContext(r.Context())
	operatorUserID := int64(0)
	operatorUsername := "unknown"
	if session != nil {
		operatorUserID = session.UserID
		if strings.TrimSpace(session.Username) != "" {
			operatorUsername = strings.TrimSpace(session.Username)
		}
	}

	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		remoteAddr = forwarded
	}

	_, _ = database.CreateRuntimeConfigAudit(&database.RuntimeConfigAudit{
		OperatorUserID:   operatorUserID,
		OperatorUsername: operatorUsername,
		SourceIP:         remoteAddr,
		Changes:          string(changeJSON),
	})
}

func (h *Handler) applyNorthboundRuntimeConfigFromHandler() {
	if h.northboundMgr == nil || h.appConfig == nil {
		return
	}

	for _, name := range h.northboundMgr.ListRuntimeNames() {
		adapter, err := h.northboundMgr.GetAdapter(name)
		if err != nil || adapter == nil {
			continue
		}
		mqttAdapter, ok := adapter.(interface{ SetReconnectInterval(time.Duration) })
		if !ok {
			continue
		}
		mqttAdapter.SetReconnectInterval(h.appConfig.NorthboundMQTTReconnectInterval)
	}
}

func parseOptionalDuration(raw string) (time.Duration, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, false, fmt.Errorf("invalid duration: %s", raw)
	}
	if value <= 0 {
		return 0, false, fmt.Errorf("duration must be > 0: %s", raw)
	}
	return value, true, nil
}

func (h *Handler) gatewayRuntimeConfigView() map[string]interface{} {
	collectorDeviceSyncInterval, collectorCommandPollInterval := h.gatewayCollectorRuntimeIntervals()

	return map[string]interface{}{
		"collector_device_sync_interval":     collectorDeviceSyncInterval.String(),
		"collector_command_poll_interval":    collectorCommandPollInterval.String(),
		"northbound_mqtt_reconnect_interval": h.appConfig.NorthboundMQTTReconnectInterval.String(),
		"driver_serial_read_timeout":         h.appConfig.DriverSerialReadTimeout.String(),
		"driver_tcp_dial_timeout":            h.appConfig.DriverTCPDialTimeout.String(),
		"driver_tcp_read_timeout":            h.appConfig.DriverTCPReadTimeout.String(),
		"driver_serial_open_backoff":         h.appConfig.DriverSerialOpenBackoff.String(),
		"driver_tcp_dial_backoff":            h.appConfig.DriverTCPDialBackoff.String(),
		"driver_serial_open_retries":         h.appConfig.DriverSerialOpenRetries,
		"driver_tcp_dial_retries":            h.appConfig.DriverTCPDialRetries,
	}
}

func (h *Handler) gatewayCollectorRuntimeIntervals() (time.Duration, time.Duration) {
	if h.collector == nil {
		return h.appConfig.CollectorDeviceSyncInterval, h.appConfig.CollectorCommandPollInterval
	}
	return h.collector.GetRuntimeIntervals()
}

func applyGatewayDurationConfigChange(changes map[string]runtimeConfigChange, key, raw string, target *time.Duration) error {
	parsed, ok, err := parseOptionalDuration(raw)
	if err != nil || !ok {
		return err
	}

	recordRuntimeConfigChange(changes, key, target.String(), parsed.String())
	*target = parsed
	return nil
}

func applyGatewayRetryConfigChange(changes map[string]runtimeConfigChange, key string, value *int, target *int) error {
	if value == nil {
		return nil
	}
	if *value < 0 {
		return fmt.Errorf("%s must be >= 0", key)
	}

	recordRuntimeConfigChange(changes, key, *target, *value)
	*target = *value
	return nil
}

func recordRuntimeConfigChange(changes map[string]runtimeConfigChange, field string, from, to interface{}) {
	if changes == nil || strings.TrimSpace(field) == "" {
		return
	}
	if reflect.DeepEqual(from, to) {
		return
	}
	changes[field] = runtimeConfigChange{From: from, To: to}
}

func buildRuntimeAuditViews(items []*database.RuntimeConfigAudit) []*runtimeConfigAuditView {
	if len(items) == 0 {
		return []*runtimeConfigAuditView{}
	}

	views := make([]*runtimeConfigAuditView, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}

		view := &runtimeConfigAuditView{
			ID:               item.ID,
			OperatorUserID:   item.OperatorUserID,
			OperatorUsername: item.OperatorUsername,
			SourceIP:         item.SourceIP,
			CreatedAt:        item.CreatedAt,
		}

		raw := strings.TrimSpace(item.Changes)
		if raw != "" {
			parsed := make(map[string]runtimeConfigChange)
			if err := json.Unmarshal([]byte(raw), &parsed); err == nil && len(parsed) > 0 {
				view.Changes = parsed
			} else {
				view.ChangesRaw = item.Changes
			}
		}

		views = append(views, view)
	}

	return views
}
