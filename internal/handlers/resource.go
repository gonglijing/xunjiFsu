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
	resource, ok := parseResourcePayload(w, r)
	if !ok {
		return
	}
	id, err := database.AddResource(resource)
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

	resource, ok := parseResourcePayload(w, r)
	if !ok {
		return
	}
	resource.ID = id
	if err := database.UpdateResource(resource); err != nil {
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
	resource, ok := loadResourceOrWriteNotFound(w, r)
	if !ok {
		return
	}

	newState := nextResourceEnabledState(resource.Enabled)
	if err := database.ToggleResource(resource.ID, newState); err != nil {
		writeServerErrorWithLog(w, apiErrToggleResourceFailed, err)
		return
	}
	resource.Enabled = newState
	WriteSuccess(w, resource)
}

func parseResourcePayload(w http.ResponseWriter, r *http.Request) (*models.Resource, bool) {
	var resource models.Resource
	if !parseRequestOrWriteBadRequestDefault(w, r, &resource) {
		return nil, false
	}
	return &resource, true
}

func loadResourceOrWriteNotFound(w http.ResponseWriter, r *http.Request) (*models.Resource, bool) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return nil, false
	}

	resource, err := database.GetResourceByID(id)
	if err != nil {
		WriteNotFoundDef(w, apiErrResourceNotFound)
		return nil, false
	}

	return resource, true
}

func nextResourceEnabledState(enabled int) int {
	if enabled == 0 {
		return 1
	}
	return 0
}
