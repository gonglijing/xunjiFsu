package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *ThresholdAPI) loadThresholdByRequest(w http.ResponseWriter, r *http.Request) (*models.Threshold, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}

	threshold, err := api.service.LoadThreshold(id)
	if err != nil {
		WriteNotFoundDef(w, errThresholdNotFound)
		return nil, false
	}
	return threshold, true
}

func parseThresholdRequest(w http.ResponseWriter, r *http.Request) (*models.Threshold, bool) {
	var threshold models.Threshold
	if err := ParseRequest(r, &threshold); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	service.NormalizeThresholdInput(&threshold)
	return &threshold, true
}
