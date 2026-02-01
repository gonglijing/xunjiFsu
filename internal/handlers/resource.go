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
        WriteServerError(w, err.Error())
        return
    }
    WriteSuccess(w, resources)
}

// CreateResource 新建资源
func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
    var resource models.Resource
    if err := ParseRequest(r, &resource); err != nil {
        WriteBadRequest(w, "invalid body")
        return
    }
    id, err := database.AddResource(&resource)
    if err != nil {
        WriteServerError(w, err.Error())
        return
    }
    resource.ID = id
    WriteCreated(w, resource)
}

// UpdateResource 更新资源
func (h *Handler) UpdateResource(w http.ResponseWriter, r *http.Request) {
    id, err := ParseID(r)
    if err != nil {
        WriteBadRequest(w, "invalid id")
        return
    }
    var resource models.Resource
    if err := ParseRequest(r, &resource); err != nil {
        WriteBadRequest(w, "invalid body")
        return
    }
    resource.ID = id
    if err := database.UpdateResource(&resource); err != nil {
        WriteServerError(w, err.Error())
        return
    }
    WriteSuccess(w, resource)
}

// DeleteResource 删除资源
func (h *Handler) DeleteResource(w http.ResponseWriter, r *http.Request) {
    id, err := ParseID(r)
    if err != nil {
        WriteBadRequest(w, "invalid id")
        return
    }
    if err := database.DeleteResource(id); err != nil {
        WriteServerError(w, err.Error())
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// ToggleResource 启停资源
func (h *Handler) ToggleResource(w http.ResponseWriter, r *http.Request) {
    id, err := ParseID(r)
    if err != nil {
        WriteBadRequest(w, "invalid id")
        return
    }
    res, err := database.GetResourceByID(id)
    if err != nil {
        WriteNotFound(w, "resource not found")
        return
    }
    newState := 0
    if res.Enabled == 0 {
        newState = 1
    }
    if err := database.ToggleResource(id, newState); err != nil {
        WriteServerError(w, err.Error())
        return
    }
    res.Enabled = newState
    WriteSuccess(w, res)
}
