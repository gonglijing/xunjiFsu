package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
)

// 数据缓存
func (h *Handler) GetDataCache(w http.ResponseWriter, r *http.Request) {
	cache, err := database.GetAllDataCache()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListDataCacheFailed, err)
		return
	}
	WriteSuccess(w, filterSystemDataCache(cache))
}

func (h *Handler) GetDataCacheByDeviceID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	cache, err := database.GetDataCacheByDeviceID(id)
	if err != nil {
		writeServerErrorWithLog(w, apiErrGetDeviceDataCacheFailed, err)
		return
	}
	WriteSuccess(w, cache)
}

func (h *Handler) GetHistoryData(w http.ResponseWriter, r *http.Request) {
	query, err := parseHistoryDataQuery(r)
	if err != nil {
		WriteBadRequestCode(w, apiErrHistoryDataQueryInvalid.Code, apiErrHistoryDataQueryInvalid.Message+": "+err.Error())
		return
	}

	points, err := queryDataPoints(query)
	if err != nil {
		writeServerErrorWithLog(w, apiErrQueryHistoryDataFailed, err)
		return
	}

	WriteSuccess(w, points)
}

func queryDataPoints(query historyDataQuery) ([]*database.DataPoint, error) {
	if query.DeviceID != nil {
		if query.FieldName != "" {
			return database.GetDataPointsByDeviceFieldAndTime(*query.DeviceID, query.FieldName, query.StartTime, query.EndTime, 2000)
		}
		if !query.StartTime.IsZero() || !query.EndTime.IsZero() {
			return database.GetDataPointsByDeviceAndTimeLimit(*query.DeviceID, query.StartTime, query.EndTime, 2000)
		}
		return database.GetDataPointsByDevice(*query.DeviceID, 1000)
	}

	points, err := database.GetLatestDataPoints(1000)
	if err != nil {
		return nil, err
	}

	return filterSystemDataPoints(points), nil
}
