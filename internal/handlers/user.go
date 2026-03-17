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
		writeServerErrorWithLog(w, apiErrListUsersFailed, err)
		return
	}

	WriteSuccess(w, sanitizeUsers(users))
}

// CreateUser 创建用户
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	user, ok := parseUserPayload(w, r)
	if !ok {
		return
	}

	id, err := database.CreateUser(user)
	if err != nil {
		writeServerErrorWithLog(w, apiErrCreateUserFailed, err)
		return
	}

	user.ID = id
	WriteCreated(w, sanitizeUser(user))
}

// UpdateUser 更新用户
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	user, ok := parseUserPayload(w, r)
	if !ok {
		return
	}

	user.ID = id
	if err := database.UpdateUser(user); err != nil {
		writeServerErrorWithLog(w, apiErrUpdateUserFailed, err)
		return
	}

	WriteSuccess(w, sanitizeUser(user))
}

// DeleteUser 删除用户
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDOrWriteBadRequestDefault(w, r)
	if !ok {
		return
	}

	if err := database.DeleteUser(id); err != nil {
		writeServerErrorWithLog(w, apiErrDeleteUserFailed, err)
		return
	}

	WriteDeleted(w)
}

func parseUserPayload(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	var user models.User
	if !parseRequestOrWriteBadRequestDefault(w, r, &user) {
		return nil, false
	}
	return &user, true
}

func sanitizeUsers(users []*models.User) []*models.User {
	for _, user := range users {
		sanitizeUser(user)
	}
	return users
}

func sanitizeUser(user *models.User) *models.User {
	if user != nil {
		user.Password = ""
	}
	return user
}
