package service

import (
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
)

type GatewayRuntimeConfig struct {
	CollectorDeviceSyncInterval     string `json:"collector_device_sync_interval"`
	CollectorCommandPollInterval    string `json:"collector_command_poll_interval"`
	CollectorWorkers                *int   `json:"collector_workers"`
	NorthboundMQTTReconnectInterval string `json:"northbound_mqtt_reconnect_interval"`
	DriverSerialReadTimeout         string `json:"driver_serial_read_timeout"`
	DriverTCPDialTimeout            string `json:"driver_tcp_dial_timeout"`
	DriverTCPReadTimeout            string `json:"driver_tcp_read_timeout"`
	DriverSerialOpenBackoff         string `json:"driver_serial_open_backoff"`
	DriverTCPDialBackoff            string `json:"driver_tcp_dial_backoff"`
	DriverSerialOpenRetries         *int   `json:"driver_serial_open_retries"`
	DriverTCPDialRetries            *int   `json:"driver_tcp_dial_retries"`
}

type GatewayRuntimeView struct {
	CollectorDeviceSyncInterval     string `json:"collector_device_sync_interval"`
	CollectorCommandPollInterval    string `json:"collector_command_poll_interval"`
	CollectorWorkers                int    `json:"collector_workers"`
	NorthboundMQTTReconnectInterval string `json:"northbound_mqtt_reconnect_interval"`
	DriverSerialReadTimeout         string `json:"driver_serial_read_timeout"`
	DriverTCPDialTimeout            string `json:"driver_tcp_dial_timeout"`
	DriverTCPReadTimeout            string `json:"driver_tcp_read_timeout"`
	DriverSerialOpenBackoff         string `json:"driver_serial_open_backoff"`
	DriverTCPDialBackoff            string `json:"driver_tcp_dial_backoff"`
	DriverSerialOpenRetries         int    `json:"driver_serial_open_retries"`
	DriverTCPDialRetries            int    `json:"driver_tcp_dial_retries"`
}

type RuntimeConfigChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

type RuntimeConfigActor struct {
	UserID   int64
	Username string
	SourceIP string
}

type GatewayRuntimeService struct {
	appConfig      *config.Config
	collector      *collector.Collector
	driverExecutor *driver.DriverExecutor
	northboundMgr  *northbound.NorthboundManager
}

func NewGatewayRuntimeService(
	appConfig *config.Config,
	collector *collector.Collector,
	driverExecutor *driver.DriverExecutor,
	northboundMgr *northbound.NorthboundManager,
) *GatewayRuntimeService {
	return &GatewayRuntimeService{
		appConfig:      appConfig,
		collector:      collector,
		driverExecutor: driverExecutor,
		northboundMgr:  northboundMgr,
	}
}
