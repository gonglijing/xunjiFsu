package database

import (
	"fmt"
	"log"

	"github.com/gonglijing/xunjiFsu/internal/pwdutil"
)

// InitDefaultData 初始化默认数据
func InitDefaultData() error {
	var count int
	err := ParamDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		// 表可能不存在或为空，返回错误
		log.Printf("Warning: Failed to query users count: %v, trying to create default user", err)
		// 尝试直接创建用户
		_, err := ParamDB.Exec(
			"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
			"admin", pwdutil.Hash("123456"), "admin",
		)
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}
		log.Println("Created default admin user")
		return nil
	}

	if count == 0 {
		_, err := ParamDB.Exec(
			"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
			"admin", pwdutil.Hash("123456"), "admin",
		)
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}
		log.Println("Created default admin user")
	}

	return nil
}
