package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
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

func (h *Handler) GetGatewayRuntimeConfig(w http.ResponseWriter, r *http.Request) {
	if h.appConfig == nil {
		WriteSuccess(w, map[string]interface{}{})
		return
	}

	collectorDeviceSyncInterval := h.appConfig.CollectorDeviceSyncInterval
	collectorCommandPollInterval := h.appConfig.CollectorCommandPollInterval
	if h.collector != nil {
		collectorDeviceSyncInterval, collectorCommandPollInterval = h.collector.GetRuntimeIntervals()
	}

	WriteSuccess(w, map[string]interface{}{
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
	})
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

	if interval, ok, err := parseOptionalDuration(payload.CollectorDeviceSyncInterval); err != nil {
		return nil, err
	} else if ok {
		changes["collector_device_sync_interval"] = runtimeConfigChange{From: h.appConfig.CollectorDeviceSyncInterval.String(), To: interval.String()}
		h.appConfig.CollectorDeviceSyncInterval = interval
	}

	if interval, ok, err := parseOptionalDuration(payload.CollectorCommandPollInterval); err != nil {
		return nil, err
	} else if ok {
		changes["collector_command_poll_interval"] = runtimeConfigChange{From: h.appConfig.CollectorCommandPollInterval.String(), To: interval.String()}
		h.appConfig.CollectorCommandPollInterval = interval
	}

	if interval, ok, err := parseOptionalDuration(payload.NorthboundMQTTReconnectInterval); err != nil {
		return nil, err
	} else if ok {
		changes["northbound_mqtt_reconnect_interval"] = runtimeConfigChange{From: h.appConfig.NorthboundMQTTReconnectInterval.String(), To: interval.String()}
		h.appConfig.NorthboundMQTTReconnectInterval = interval
	}

	if timeout, ok, err := parseOptionalDuration(payload.DriverSerialReadTimeout); err != nil {
		return nil, err
	} else if ok {
		changes["driver_serial_read_timeout"] = runtimeConfigChange{From: h.appConfig.DriverSerialReadTimeout.String(), To: timeout.String()}
		h.appConfig.DriverSerialReadTimeout = timeout
	}

	if timeout, ok, err := parseOptionalDuration(payload.DriverTCPDialTimeout); err != nil {
		return nil, err
	} else if ok {
		changes["driver_tcp_dial_timeout"] = runtimeConfigChange{From: h.appConfig.DriverTCPDialTimeout.String(), To: timeout.String()}
		h.appConfig.DriverTCPDialTimeout = timeout
	}

	if timeout, ok, err := parseOptionalDuration(payload.DriverTCPReadTimeout); err != nil {
		return nil, err
	} else if ok {
		changes["driver_tcp_read_timeout"] = runtimeConfigChange{From: h.appConfig.DriverTCPReadTimeout.String(), To: timeout.String()}
		h.appConfig.DriverTCPReadTimeout = timeout
	}

	if backoff, ok, err := parseOptionalDuration(payload.DriverSerialOpenBackoff); err != nil {
		return nil, err
	} else if ok {
		changes["driver_serial_open_backoff"] = runtimeConfigChange{From: h.appConfig.DriverSerialOpenBackoff.String(), To: backoff.String()}
		h.appConfig.DriverSerialOpenBackoff = backoff
	}

	if backoff, ok, err := parseOptionalDuration(payload.DriverTCPDialBackoff); err != nil {
		return nil, err
	} else if ok {
		changes["driver_tcp_dial_backoff"] = runtimeConfigChange{From: h.appConfig.DriverTCPDialBackoff.String(), To: backoff.String()}
		h.appConfig.DriverTCPDialBackoff = backoff
	}

	if payload.DriverSerialOpenRetries != nil {
		if *payload.DriverSerialOpenRetries < 0 {
			return nil, fmt.Errorf("driver_serial_open_retries must be >= 0")
		}
		changes["driver_serial_open_retries"] = runtimeConfigChange{From: h.appConfig.DriverSerialOpenRetries, To: *payload.DriverSerialOpenRetries}
		h.appConfig.DriverSerialOpenRetries = *payload.DriverSerialOpenRetries
	}

	if payload.DriverTCPDialRetries != nil {
		if *payload.DriverTCPDialRetries < 0 {
			return nil, fmt.Errorf("driver_tcp_dial_retries must be >= 0")
		}
		changes["driver_tcp_dial_retries"] = runtimeConfigChange{From: h.appConfig.DriverTCPDialRetries, To: *payload.DriverTCPDialRetries}
		h.appConfig.DriverTCPDialRetries = *payload.DriverTCPDialRetries
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

	WriteSuccess(w, items)
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
