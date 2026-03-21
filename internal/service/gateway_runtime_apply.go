package service

import "time"

func (s *GatewayRuntimeService) ApplyRuntimeConfig(payload *GatewayRuntimeConfig) (map[string]RuntimeConfigChange, error) {
	if s == nil || payload == nil || s.appConfig == nil {
		return nil, nil
	}

	changes := make(map[string]RuntimeConfigChange)

	if err := applyDurationConfigChange(changes, "collector_device_sync_interval", payload.CollectorDeviceSyncInterval, &s.appConfig.CollectorDeviceSyncInterval); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "collector_command_poll_interval", payload.CollectorCommandPollInterval, &s.appConfig.CollectorCommandPollInterval); err != nil {
		return nil, err
	}
	if err := applyPositiveIntConfigChange(changes, "collector_workers", payload.CollectorWorkers, &s.appConfig.CollectorWorkers); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "northbound_mqtt_reconnect_interval", payload.NorthboundMQTTReconnectInterval, &s.appConfig.NorthboundMQTTReconnectInterval); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "driver_serial_read_timeout", payload.DriverSerialReadTimeout, &s.appConfig.DriverSerialReadTimeout); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "driver_tcp_dial_timeout", payload.DriverTCPDialTimeout, &s.appConfig.DriverTCPDialTimeout); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "driver_tcp_read_timeout", payload.DriverTCPReadTimeout, &s.appConfig.DriverTCPReadTimeout); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "driver_serial_open_backoff", payload.DriverSerialOpenBackoff, &s.appConfig.DriverSerialOpenBackoff); err != nil {
		return nil, err
	}
	if err := applyDurationConfigChange(changes, "driver_tcp_dial_backoff", payload.DriverTCPDialBackoff, &s.appConfig.DriverTCPDialBackoff); err != nil {
		return nil, err
	}
	if err := applyRetryConfigChange(changes, "driver_serial_open_retries", payload.DriverSerialOpenRetries, &s.appConfig.DriverSerialOpenRetries); err != nil {
		return nil, err
	}
	if err := applyRetryConfigChange(changes, "driver_tcp_dial_retries", payload.DriverTCPDialRetries, &s.appConfig.DriverTCPDialRetries); err != nil {
		return nil, err
	}

	s.applyCollectorRuntime()
	s.applyDriverRuntime()
	s.applyNorthboundRuntime()

	return changes, nil
}

func ParseOptionalDuration(raw string) (time.Duration, bool, error) {
	return parseOptionalDuration(raw)
}

func (s *GatewayRuntimeService) applyCollectorRuntime() {
	if s.collector == nil || s.appConfig == nil {
		return
	}
	s.collector.SetRuntimeIntervals(s.appConfig.CollectorDeviceSyncInterval, s.appConfig.CollectorCommandPollInterval)
	s.collector.SetMaxConcurrentCollects(s.appConfig.CollectorWorkers)
}

func (s *GatewayRuntimeService) applyDriverRuntime() {
	if s.driverExecutor == nil || s.appConfig == nil {
		return
	}
	s.driverExecutor.SetTimeouts(s.appConfig.DriverSerialReadTimeout, s.appConfig.DriverTCPDialTimeout, s.appConfig.DriverTCPReadTimeout)
	s.driverExecutor.SetRetries(s.appConfig.DriverSerialOpenRetries, s.appConfig.DriverTCPDialRetries, s.appConfig.DriverSerialOpenBackoff, s.appConfig.DriverTCPDialBackoff)
}

func (s *GatewayRuntimeService) applyNorthboundRuntime() {
	if s.northboundMgr == nil || s.appConfig == nil {
		return
	}

	for _, name := range s.northboundMgr.ListRuntimeNames() {
		adapter, err := s.northboundMgr.GetAdapter(name)
		if err != nil || adapter == nil {
			continue
		}
		mqttAdapter, ok := adapter.(interface{ SetReconnectInterval(time.Duration) })
		if !ok {
			continue
		}
		mqttAdapter.SetReconnectInterval(s.appConfig.NorthboundMQTTReconnectInterval)
	}
}
