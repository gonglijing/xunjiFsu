package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func parseNorthboundConfigRequest(w http.ResponseWriter, r *http.Request) (*models.NorthboundConfig, bool) {
	var config models.NorthboundConfig
	if err := ParseRequest(r, &config); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	normalizeNorthboundConfig(&config)
	if err := validateNorthboundConfig(&config); err != nil {
		WriteBadRequestCode(w, errNorthboundConfigInvalid.Code, errNorthboundConfigInvalid.Message+": "+err.Error())
		return nil, false
	}
	return &config, true
}

func (api *NorthboundAPI) loadNorthboundConfig(w http.ResponseWriter, r *http.Request) (*models.NorthboundConfig, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}
	config, err := api.service.LoadConfig(id)
	if err != nil {
		WriteNotFoundDef(w, errNorthboundConfigNotFound)
		return nil, false
	}
	return config, true
}
