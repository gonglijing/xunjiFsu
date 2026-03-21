package httpapi

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

func (api *ResourceAPI) loadResourceByRequest(w http.ResponseWriter, r *http.Request) (*models.Resource, bool) {
	id, ok := parseIDOrWriteBadRequest(w, r, apiErrInvalidID)
	if !ok {
		return nil, false
	}

	resource, err := api.service.LoadResource(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrResourceNotFound)
		return nil, false
	}
	return resource, true
}

func parseResourceRequest(w http.ResponseWriter, r *http.Request) (*models.Resource, bool) {
	var resource models.Resource
	if err := ParseRequest(r, &resource); err != nil {
		WriteBadRequestDef(w, apiErrInvalidRequestBody)
		return nil, false
	}
	return &resource, true
}
