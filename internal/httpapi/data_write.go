package httpapi

import "net/http"

func (api *DataAPI) ClearHistoryData(w http.ResponseWriter, r *http.Request) {
	query, err := parseHistoryPointQuery(r)
	if err != nil {
		WriteBadRequestCode(w, errHistoryPointQueryDef.Code, errHistoryPointQueryDef.Message+": "+err.Error())
		return
	}

	deleted, err := api.service.ClearHistoryPoint(query.DeviceID, query.FieldName)
	if err != nil {
		writeServerErrorWithLog(w, errClearHistoryData, err)
		return
	}
	WriteSuccess(w, deletedCountView{Deleted: deleted})
}
