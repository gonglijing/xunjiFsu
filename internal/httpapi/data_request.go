package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (api *DataAPI) loadDeviceDataCacheByRequest(w http.ResponseWriter, r *http.Request) ([]*models.DataCache, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}

	cache, err := api.service.LoadDeviceDataCache(id)
	if err != nil {
		writeServerErrorWithLog(w, errGetDeviceCacheFailed, err)
		return nil, false
	}
	return cache, true
}
