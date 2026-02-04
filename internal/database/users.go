package database

import (
	"database/sql"
	"github.com/gonglijing/xunjiFsu/internal/models"
)

// ==================== 用户操作 (param.db - 直接写) ====================

// CreateUser 创建用户
func CreateUser(user *models.User) (int64, error) {
	result, err := ParamDB.Exec(
		"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
		user.Username, user.Password, user.Role,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetUserByUsername 根据用户名获取用户
func GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}
	err := ParamDB.QueryRow(
		"SELECT id, username, password, role, created_at, updated_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID 根据ID获取用户
func GetUserByID(id int64) (*models.User, error) {
	user := &models.User{}
	err := ParamDB.QueryRow(
		"SELECT id, username, password, role, created_at, updated_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetAllUsers 获取所有用户
func GetAllUsers() ([]*models.User, error) {
	return queryList[*models.User](ParamDB,
		"SELECT id, username, password, role, created_at, updated_at FROM users ORDER BY id",
		nil,
		func(rows *sql.Rows) (*models.User, error) {
			user := &models.User{}
			if err := rows.Scan(&user.ID, &user.Username, &user.Password, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
				return nil, err
			}
			return user, nil
		},
	)
}

// UpdateUser 更新用户
func UpdateUser(user *models.User) error {
	_, err := ParamDB.Exec(
		"UPDATE users SET username = ?, password = ?, role = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
		user.Username, user.Password, user.Role, user.ID,
	)
	return err
}

// DeleteUser 删除用户
func DeleteUser(id int64) error {
	_, err := ParamDB.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}
