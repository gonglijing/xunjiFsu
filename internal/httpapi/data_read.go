package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/service"
)

func (api *DataAPI) GetDataCache(w http.ResponseWriter, r *http.Request) {
	cache, err := api.service.LoadAllDataCache()
	if err != nil {
		writeServerErrorWithLog(w, errListDataCacheFailed, err)
		return
	}
	WriteSuccess(w, cache)
}

func (api *DataAPI) GetDataCacheByDeviceID(w http.ResponseWriter, r *http.Request) {
	cache, ok := api.loadDeviceDataCacheByRequest(w, r)
	if !ok {
		return
	}
	WriteSuccess(w, cache)
}

func (api *DataAPI) GetHistoryData(w http.ResponseWriter, r *http.Request) {
	query, err := parseHistoryDataQuery(r)
	if err != nil {
		WriteBadRequestCode(w, errHistoryDataQueryDef.Code, errHistoryDataQueryDef.Message+": "+err.Error())
		return
	}

	points, err := api.service.QueryHistoryData(service.HistoryDataQuery{
		DeviceID:  query.DeviceID,
		FieldName: query.FieldName,
		StartTime: query.StartTime,
		EndTime:   query.EndTime,
	})
	if err != nil {
		writeServerErrorWithLog(w, errQueryHistoryData, err)
		return
	}
	WriteSuccess(w, points)
}
