package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/northbound/adapters"
)

func (api *NorthboundAPI) CreateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	config, ok := parseNorthboundConfigRequest(w, r)
	if !ok {
		return
	}
	config, err := api.service.CreateConfig(config)
	if err != nil {
		writeServerErrorWithLog(w, errCreateNorthboundConfig, err)
		return
	}
	WriteCreated(w, api.buildNorthboundConfigView(config))
}

func (api *NorthboundAPI) UpdateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	oldConfig, ok := api.loadNorthboundConfig(w, r)
	if !ok {
		return
	}
	config, ok := parseNorthboundConfigRequest(w, r)
	if !ok {
		return
	}
	config, err := api.service.UpdateConfig(oldConfig, config)
	if err != nil {
		writeServerErrorWithLog(w, errUpdateNorthboundConfig, err)
		return
	}
	WriteSuccess(w, api.buildNorthboundConfigView(config))
}

func (api *NorthboundAPI) DeleteNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	config, ok := api.loadNorthboundConfig(w, r)
	if !ok {
		return
	}
	if err := api.service.DeleteConfig(config); err != nil {
		writeServerErrorWithLog(w, errDeleteNorthboundConfig, err)
		return
	}
	WriteDeleted(w)
}

func (api *NorthboundAPI) ToggleNorthboundEnable(w http.ResponseWriter, r *http.Request) {
	config, ok := api.loadNorthboundConfig(w, r)
	if !ok {
		return
	}
	nextState, err := api.service.ToggleEnabled(config)
	if err != nil {
		WriteBadRequestCode(w, errNorthboundInitializeFailed.Code, errNorthboundInitializeFailed.Message+": "+err.Error())
		return
	}
	WriteSuccess(w, northboundEnabledView{Enabled: nextState})
}

func (api *NorthboundAPI) ReloadNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	config, ok := api.loadNorthboundConfig(w, r)
	if !ok {
		return
	}
	if err := api.service.ReloadConfig(config); err != nil {
		WriteBadRequestCode(w, errNorthboundReloadFailed.Code, errNorthboundReloadFailed.Message+": "+err.Error())
		return
	}
	WriteSuccess(w, api.buildNorthboundConfigView(config))
}

func (api *NorthboundAPI) SyncNorthboundDevices(w http.ResponseWriter, r *http.Request) {
	config, ok := api.loadNorthboundConfig(w, r)
	if !ok {
		return
	}
	if err := api.syncNorthboundDevices(config); err != nil {
		WriteBadRequestCode(w, errNorthboundSyncDevicesFailed.Code, errNorthboundSyncDevicesFailed.Message+": "+err.Error())
		return
	}
	WriteSuccess(w, northboundSyncView{
		ID:      config.ID,
		Name:    config.Name,
		Type:    config.Type,
		Message: "同步设备已触发",
	})
}

func (api *NorthboundAPI) syncNorthboundDevices(config *models.NorthboundConfig) error {
	if config == nil {
		return fmt.Errorf("northbound config is nil")
	}
	if config.Enabled == 0 {
		return fmt.Errorf("northbound is disabled")
	}
	adapter, err := api.service.RuntimeAdapterForConfig(config)
	if err != nil {
		return err
	}
	deviceSyncAdapter, ok := adapter.(adapters.NorthboundAdapterWithDeviceSync)
	if !ok {
		return fmt.Errorf("adapter type %s does not support device sync", normalizeNorthboundType(config.Type))
	}
	return deviceSyncAdapter.SyncDevices()
}
