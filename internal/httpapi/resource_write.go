package httpapi

import (
	"net/http"
)

func (api *ResourceAPI) CreateResource(w http.ResponseWriter, r *http.Request) {
	resource, ok := parseResourceRequest(w, r)
	if !ok {
		return
	}

	resource, err := api.service.CreateResource(resource)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateResourceFailed, err)
		return
	}
	WriteCreated(w, resource)
}

func (api *ResourceAPI) UpdateResource(w http.ResponseWriter, r *http.Request) {
	resourceModel, ok := api.loadResourceByRequest(w, r)
	if !ok {
		return
	}

	resource, ok := parseResourceRequest(w, r)
	if !ok {
		return
	}
	resource.ID = resourceModel.ID

	resource, err := api.service.UpdateResource(resource)
	if err != nil {
		writeServerErrorWithLog(w, apiErrUpdateResourceFailed, err)
		return
	}
	WriteSuccess(w, resource)
}

func (api *ResourceAPI) DeleteResource(w http.ResponseWriter, r *http.Request) {
	resource, ok := api.loadResourceByRequest(w, r)
	if !ok {
		return
	}

	if err := api.service.DeleteResource(resource.ID); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteResourceFailed, err)
		return
	}
	WriteDeleted(w)
}

func (api *ResourceAPI) ToggleResource(w http.ResponseWriter, r *http.Request) {
	resource, ok := api.loadResourceByRequest(w, r)
	if !ok {
		return
	}

	resource, err := api.service.ToggleResource(resource.ID)
	if err != nil {
		WriteNotFoundDef(w, apiErrResourceNotFound)
		return
	}
	WriteSuccess(w, resource)
}
