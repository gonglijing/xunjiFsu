package app

import (
	"fmt"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/auth"
	"github.com/gonglijing/xunjiFsu/internal/collector"
	"github.com/gonglijing/xunjiFsu/internal/config"
	"github.com/gonglijing/xunjiFsu/internal/driver"
	"github.com/gonglijing/xunjiFsu/internal/httpapi"
	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

type apiRouteDeps struct {
	status        *httpapi.StatusAPI
	data          *httpapi.DataAPI
	driver        *httpapi.DriverAPI
	northbound    *httpapi.NorthboundAPI
	device        *httpapi.DeviceAPI
	deviceExec    *httpapi.DeviceExecAPI
	deviceRuntime *httpapi.DeviceRuntimeAPI
	debugModbus   *httpapi.DebugModbusAPI
	gateway       *httpapi.GatewayAPI
	resource      *httpapi.ResourceAPI
	user          *httpapi.UserAPI
	threshold     *httpapi.ThresholdAPI
	alarm         *httpapi.AlarmAPI
}

func newAPIRouteDeps(
	cfg *config.Config,
	collect *collector.Collector,
	executor *driver.DriverExecutor,
	driverManager *driver.DriverManager,
	northboundMgr *northbound.NorthboundManager,
	authManager *auth.JWTManager,
) *apiRouteDeps {
	driverCount := func() int {
		if driverManager == nil {
			return 0
		}
		return len(driverManager.ListDrivers())
	}

	return &apiRouteDeps{
		status:        httpapi.NewStatusAPI(service.NewStatusService(collect, driverCount)),
		data:          httpapi.NewDataAPI(service.NewDataService()),
		driver:        httpapi.NewDriverAPI(service.NewDriverService(driverManager, driverManager, cfg.DriversDir)),
		northbound:    httpapi.NewNorthboundAPI(service.NewNorthboundService(northboundMgr, service.NorthboundRuntimeHooks{Rebuild: newNorthboundRuntimeRebuilder(northboundMgr)}), northboundMgr),
		device:        httpapi.NewDeviceAPI(service.NewDeviceService(collect)),
		deviceExec:    httpapi.NewDeviceExecAPI(service.NewDeviceExecService(driverManager)),
		deviceRuntime: httpapi.NewDeviceRuntimeAPI(service.NewDeviceRuntimeService(collect)),
		debugModbus:   httpapi.NewDebugModbusAPI(),
		gateway: httpapi.NewGatewayAPI(
			service.NewGatewayConfigService(),
			service.NewGatewayRuntimeService(cfg, collect, executor, northboundMgr),
		),
		resource:  httpapi.NewResourceAPI(service.NewResourceService(executor)),
		user:      httpapi.NewUserAPI(service.NewUserService(), authManager),
		threshold: httpapi.NewThresholdAPI(service.NewThresholdService()),
		alarm:     httpapi.NewAlarmAPI(service.NewAlarmService()),
	}
}

func newNorthboundRuntimeRebuilder(northboundMgr *northbound.NorthboundManager) func(*models.NorthboundConfig) error {
	return func(cfg *models.NorthboundConfig) error {
		if cfg == nil {
			return nil
		}

		northboundMgr.RemoveAdapter(cfg.Name)
		northboundMgr.SetInterval(cfg.Name, time.Duration(cfg.UploadInterval)*time.Millisecond)

		if cfg.Enabled == 0 {
			northboundMgr.SetEnabled(cfg.Name, false)
			return nil
		}

		adapter := adapters.NewAdapter(cfg.Type, cfg.Name)
		if adapter == nil {
			return fmt.Errorf("unsupported northbound type: %s", cfg.Type)
		}
		if err := adapter.Initialize(resolveNorthboundAdapterConfig(cfg)); err != nil {
			northboundMgr.SetEnabled(cfg.Name, false)
			return err
		}
		adapter.SetInterval(time.Duration(cfg.UploadInterval) * time.Millisecond)
		northboundMgr.RegisterAdapter(cfg.Name, adapter)
		northboundMgr.SetEnabled(cfg.Name, true)
		return nil
	}
}

func resolveNorthboundAdapterConfig(config *models.NorthboundConfig) string {
	if config == nil {
		return ""
	}
	if trimmed := config.Config; trimmed != "" && trimmed != "{}" {
		return config.Config
	}
	return adapters.BuildConfigFromModel(config)
}
