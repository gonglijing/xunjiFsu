package httpapi

import (
	"github.com/gonglijing/xunjiFsu/internal/northbound"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

type NorthboundAPI struct {
	service *service.NorthboundService
	manager *northbound.NorthboundManager
}

type northboundEnabledView struct {
	Enabled int `json:"enabled"`
}

type northboundSyncView struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func NewNorthboundAPI(northboundService *service.NorthboundService, manager *northbound.NorthboundManager) *NorthboundAPI {
	return &NorthboundAPI{service: northboundService, manager: manager}
}

var (
	errNorthboundConfigNotFound    = APIErrorDef{Code: "E_NORTHBOUND_NOT_FOUND", Message: "Northbound config not found"}
	errNorthboundConfigInvalid     = APIErrorDef{Code: "E_NORTHBOUND_CONFIG_INVALID", Message: "config 参数无效"}
	errNorthboundInitializeFailed  = APIErrorDef{Code: "E_NORTHBOUND_INITIALIZE_FAILED", Message: "北向初始化失败"}
	errNorthboundReloadFailed      = APIErrorDef{Code: "E_NORTHBOUND_RELOAD_FAILED", Message: "北向重载失败"}
	errNorthboundSyncDevicesFailed = APIErrorDef{Code: "E_NORTHBOUND_SYNC_DEVICES_FAILED", Message: "同步设备失败"}
	errListNorthboundConfigsFailed = APIErrorDef{Code: "E_LIST_NORTHBOUND_CONFIGS_FAILED", Message: "获取北向配置失败"}
	errCreateNorthboundConfig      = APIErrorDef{Code: "E_CREATE_NORTHBOUND_CONFIG_FAILED", Message: "创建北向配置失败"}
	errUpdateNorthboundConfig      = APIErrorDef{Code: "E_UPDATE_NORTHBOUND_CONFIG_FAILED", Message: "更新北向配置失败"}
	errDeleteNorthboundConfig      = APIErrorDef{Code: "E_DELETE_NORTHBOUND_CONFIG_FAILED", Message: "删除北向配置失败"}
	errListNorthboundStatusFailed  = APIErrorDef{Code: "E_LIST_NORTHBOUND_STATUS_FAILED", Message: "获取北向运行态失败"}
)
