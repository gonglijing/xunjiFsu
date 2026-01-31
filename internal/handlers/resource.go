package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 资源管理 ====================

// GetResources 获取所有资源
func (h *Handler) GetResources(w http.ResponseWriter, r *http.Request) {
	resources, err := database.GetAllResources()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}
	WriteSuccess(w, resources)
}

// CreateResource 创建资源
func (h *Handler) CreateResource(w http.ResponseWriter, r *http.Request) {
	var resource models.Resource
	if err := ParseRequest(r, &resource); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	id, err := database.CreateResource(&resource)
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
		WriteBadRequest(w, "Invalid ID")
		return
	}

	var resource models.Resource
	if err := ParseRequest(r, &resource); err != nil {
		WriteBadRequest(w, "Invalid request body")
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
		WriteBadRequest(w, "Invalid ID")
		return
	}

	if err := database.DeleteResource(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
}

// OpenResource 打开资源
func (h *Handler) OpenResource(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	resource, err := database.GetResourceByID(id)
	if err != nil {
		WriteNotFound(w, "Resource not found")
		return
	}

	if err := h.resourceMgr.OpenResource(resource); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, map[string]string{"status": "opened"})
}

// CloseResource 关闭资源
func (h *Handler) CloseResource(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	resource, err := database.GetResourceByID(id)
	if err != nil {
		WriteNotFound(w, "Resource not found")
		return
	}

	if err := h.resourceMgr.CloseResource(id, resource.Type); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, map[string]string{"status": "closed"})
}
