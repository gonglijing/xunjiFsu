package httpapi

import "net/http"

func (api *ResourceAPI) GetResources(w http.ResponseWriter, r *http.Request) {
	resources, err := api.service.ListResources()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListResourcesFailed, err)
		return
	}
	WriteSuccess(w, resources)
}
