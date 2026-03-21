package service

import "time"

func (s *GatewayRuntimeService) RuntimeConfigView() GatewayRuntimeView {
	if s == nil || s.appConfig == nil {
		return GatewayRuntimeView{}
	}

	collectorDeviceSyncInterval, collectorCommandPollInterval := s.collectorRuntimeIntervals()

	return GatewayRuntimeView{
		CollectorDeviceSyncInterval:     collectorDeviceSyncInterval.String(),
		CollectorCommandPollInterval:    collectorCommandPollInterval.String(),
		CollectorWorkers:                s.collectorWorkers(),
		NorthboundMQTTReconnectInterval: s.appConfig.NorthboundMQTTReconnectInterval.String(),
		DriverSerialReadTimeout:         s.appConfig.DriverSerialReadTimeout.String(),
		DriverTCPDialTimeout:            s.appConfig.DriverTCPDialTimeout.String(),
		DriverTCPReadTimeout:            s.appConfig.DriverTCPReadTimeout.String(),
		DriverSerialOpenBackoff:         s.appConfig.DriverSerialOpenBackoff.String(),
		DriverTCPDialBackoff:            s.appConfig.DriverTCPDialBackoff.String(),
		DriverSerialOpenRetries:         s.appConfig.DriverSerialOpenRetries,
		DriverTCPDialRetries:            s.appConfig.DriverTCPDialRetries,
	}
}

func (s *GatewayRuntimeService) collectorRuntimeIntervals() (time.Duration, time.Duration) {
	if s.collector == nil {
		return s.appConfig.CollectorDeviceSyncInterval, s.appConfig.CollectorCommandPollInterval
	}
	return s.collector.GetRuntimeIntervals()
}

func (s *GatewayRuntimeService) collectorWorkers() int {
	if s.collector == nil {
		return s.appConfig.CollectorWorkers
	}
	return s.collector.GetMaxConcurrentCollects()
}
