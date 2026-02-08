package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// GetResources 列出资源
func (h *Handler) GetResources(w http.ResponseWriter, r *http.Request) {
	resources, err := database.ListResources()
	if err != nil {
		writeServerErrorWithLog(w, apiErrListResourcesFailed, err)
		return
	}
	WriteSuccess(w, resources)
}

// CreateResource 新建资源
func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var resource models.Resource
	if !parseRequestOrWriteBadRequestDefault(w, r, &resource) {
		return
	}
	id, err := database.AddResource(&resource)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateResourceFailed, err)
		return
	}
	resource.ID = id
	WriteCreated(w, resource)
}

// UpdateResource 更新资源
func (h *Handler) UpdateResource(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	var resource models.Resource
	if !parseRequestOrWriteBadRequestDefault(w, r, &resource) {
		return
	}
	resource.ID = id
	if err := database.UpdateResource(&resource); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateResourceFailed, err)
		return
	}
	WriteSuccess(w, resource)
}

// DeleteResource 删除资源
func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	if err := database.DeleteResource(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteResourceFailed, err)
		return
	}
	WriteDeleted(w)
}

// ToggleResource 启停资源
func (h *Handler) ToggleResource(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}
	res, err := database.GetResourceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrResourceNotFound)
		return
	}
	newState := 0
	if res.Enabled == 0 {
		newState = 1
	}
	if err := database.ToggleResource(id, newState); err != nil {
		writeServerErrorWithLog(w, apiErrToggleResourceFailed, err)
		return
	}
	res.Enabled = newState
	WriteSuccess(w, res)
}
