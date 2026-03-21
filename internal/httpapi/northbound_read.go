package httpapi

import (
	"net/http"
	"strings"
)

func (api *NorthboundAPI) GetNorthboundConfigs(w http.ResponseWriter, r *http.Request) {
	configs, err := api.service.ListConfigs()
	if err != nil {
		writeServerErrorWithLog(w, errListNorthboundConfigsFailed, err)
		return
	}
	views := make([]*northboundConfigView, 0, len(configs))
	for _, cfg := range configs {
		views = append(views, api.buildNorthboundConfigView(cfg))
	}
	WriteSuccess(w, views)
}

func (api *NorthboundAPI) GetNorthboundStatus(w http.ResponseWriter, r *http.Request) {
	items, err := api.service.ListStatusItems()
	if err != nil {
		writeServerErrorWithLog(w, errListNorthboundStatusFailed, err)
		return
	}
	WriteSuccess(w, items)
}

func (api *NorthboundAPI) GetNorthboundSchema(w http.ResponseWriter, r *http.Request) {
	nbType := strings.TrimSpace(r.URL.Query().Get("type"))
	if nbType == "" {
		nbType = "pandax"
	}
	response, err := loadNorthboundSchema(nbType)
	if err != nil {
		WriteBadRequest(w, err.Error())
		return
	}
	WriteSuccess(w, response)
}
