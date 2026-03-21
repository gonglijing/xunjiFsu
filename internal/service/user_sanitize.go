package service

import "github.com/gonglijing/xunjiFsu/internal/models"

func SanitizeUsers(users []*models.User) []*models.User {
	for _, user := range users {
		SanitizeUser(user)
	}
	return users
}

func SanitizeUser(user *models.User) *models.User {
	if user != nil {
		user.Password = ""
	}
	return user
}
