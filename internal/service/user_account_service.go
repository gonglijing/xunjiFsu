package service

import (
	"github.com/gonglijing/xunjiFsu/internal/database"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) ListUsers() ([]*models.User, error) {
	return database.ListUsers()
}

func (s *UserService) LoadUser(id int64) (*models.User, error) {
	return database.LoadUser(id)
}

func (s *UserService) CreateUser(user *models.User) (*models.User, error) {
	if user == nil {
		return nil, nil
	}
	id, err := database.CreateUser(user)
	if err != nil {
		return nil, err
	}
	user.ID = id
	return user, nil
}

func (s *UserService) UpdateUser(user *models.User) (*models.User, error) {
	if user == nil {
		return nil, nil
	}
	if err := database.UpdateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) DeleteUser(id int64) error {
	return database.DeleteUser(id)
}
