package database

import (
	"database/sql"

	"github.com/gonglijing/xunjiFsu/internal/models"
)

const selectUserFields = `SELECT id, username, password, role, created_at, updated_at FROM users`

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
	return loadUser(selectUserFields+" WHERE username = ?", username)
}

// LoadUser 根据ID获取用户
func LoadUser(id int64) (*models.User, error) {
	return loadUser(selectUserFields+" WHERE id = ?", id)
}

// ListUsers 获取所有用户
func ListUsers() ([]*models.User, error) {
	return listUsers(selectUserFields+" ORDER BY id", nil)
}

type userScanner interface {
	Scan(dest ...any) error
}

func loadUser(query string, args ...any) (*models.User, error) {
	user := &models.User{}
	err := scanUser(ParamDB.QueryRow(query, args...), user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func listUsers(query string, args []any) ([]*models.User, error) {
	return queryList[*models.User](ParamDB,
		query,
		args,
		func(rows *sql.Rows) (*models.User, error) {
			user := &models.User{}
			if err := scanUser(rows, user); err != nil {
				return nil, err
			}
			return user, nil
		},
	)
}

func scanUser(scanner userScanner, user *models.User) error {
	return scanner.Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
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
