package handlers

import (
	"net/http"

	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 用户管理 ====================

// GetUsers 获取所有用户
func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := database.GetAllUsers()
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	// 清除敏感信息
	for i := range users {
		users[i].Password = ""
	}

	WriteSuccess(w, users)
}

// CreateUser 创建用户
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := ParseRequest(r, &user); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	id, err := database.CreateUser(&user)
	if err != nil {
		WriteServerError(w, err.Error())
		return
	}

	user.ID = id
	user.Password = ""
	WriteCreated(w, user)
}

// UpdateUser 更新用户
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	var user models.User
	if err := ParseRequest(r, &user); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	user.ID = id
	if err := database.UpdateUser(&user); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	user.Password = ""
	WriteSuccess(w, user)
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := ParseID(r)
	if err != nil {
		WriteBadRequest(w, "Invalid ID")
		return
	}

	if err := database.DeleteUser(id); err != nil {
		WriteServerError(w, err.Error())
		return
	}

	WriteSuccess(w, nil)
}
