package handlers

import (
	"net/http"
	"time"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetNorthboundConfigs 获取所有北向配置
func (h *Handler) GetNorthboundConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := database.GetAllNorthboundConfigs()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListNorthboundConfigsFailed, err)
		return
	}

	views := make([]*northboundConfigView, 0, len(configs))
	for _, cfg := range configs {
		views = append(views, h.buildNorthboundConfigView(cfg))
	}

	WriteSuccess(w, views)
}

// CreateNorthboundConfig 创建北向配置
func (h *Handler) CreateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	config, ok := parseAndPrepareNorthboundConfig(w, r)
	if !ok {
		return
	}

	if config.Enabled == 1 {
		if err := h.registerNorthboundAdapter(config); err != nil {
			WriteBadRequestCode(w, apiErrNorthboundInitializeFailed.Code, apiErrNorthboundInitializeFailed.Message+": "+err.Error())
			return
		}
	}
	h.northboundMgr.SetInterval(config.Name, time.Duration(config.UploadInterval)*time.Millisecond)
	h.northboundMgr.SetEnabled(config.Name, config.Enabled == 1)

	id, err := database.CreateNorthboundConfig(config)
	if err != nil {
		h.northboundMgr.RemoveAdapter(config.Name)
		writeServerErrorWithLog(w, apiErrCreateNorthboundConfigFailed, err)
		return
	}

	config.ID = id
	WriteCreated(w, h.buildNorthboundConfigView(config))
}

// UpdateNorthboundConfig 更新北向配置
func (h *Handler) UpdateNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	oldConfig, err := database.GetNorthboundConfigByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrNorthboundConfigNotFound)
		return
	}

	config, ok := parseAndPrepareNorthboundConfig(w, r)
	if !ok {
		return
	}

	if oldConfig.Name != config.Name {
		h.northboundMgr.RemoveAdapter(oldConfig.Name)
	}

	if err := h.rebuildNorthboundRuntime(config); err != nil {
		h.rollbackNorthboundRuntime(oldConfig)
		WriteBadRequestCode(w, apiErrNorthboundInitializeFailed.Code, apiErrNorthboundInitializeFailed.Message+": "+err.Error())
		return
	}

	config.ID = id
	if err := database.UpdateNorthboundConfig(config); err != nil {
		h.northboundMgr.RemoveAdapter(config.Name)
		h.rollbackNorthboundRuntime(oldConfig)
		writeServerErrorWithLog(w, apiErrUpdateNorthboundConfigFailed, err)
		return
	}

	WriteSuccess(w, h.buildNorthboundConfigView(config))
}

// DeleteNorthboundConfig 删除北向配置
func (h *Handler) DeleteNorthboundConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	config, err := database.GetNorthboundConfigByID(id)
	if err == nil {
		h.northboundMgr.RemoveAdapter(config.Name)
	}

	if err := database.DeleteNorthboundConfig(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteNorthboundConfigFailed, err)
		return
	}

	WriteDeleted(w)
}

func parseAndPrepareNorthboundConfig(w http.ResponseWriter, r *http.Request) (*models.NorthboundConfig, bool) {
	var config models.NorthboundConfig
	if !parseRequestOrWriteBadRequestDefault(w, r, &config) {
		return nil, false
	}

	normalizeNorthboundConfig(&config)
	if err := validateNorthboundConfig(&config); err != nil {
		writeNorthboundConfigInvalid(w, err)
		return nil, false
	}
	if err := enrichNorthboundConfigWithGatewayIdentity(&config); err != nil {
		writeNorthboundConfigInvalid(w, err)
		return nil, false
	}

	return &config, true
}

func writeNorthboundConfigInvalid(w http.ResponseWriter, err error) {
	if err == nil {
		WriteBadRequestDef(w, apiErrNorthboundConfigInvalid)
		return
	}
	WriteBadRequestCode(w, apiErrNorthboundConfigInvalid.Code, apiErrNorthboundConfigInvalid.Message+": "+err.Error())
}

func (h *Handler) rollbackNorthboundRuntime(oldConfig *models.NorthboundConfig) {
	if oldConfig != nil {
		_ = h.rebuildNorthboundRuntime(oldConfig)
	}
}
