package database

import (
	"fmt"
	"log/slog"

	"github.com/gonglijing/xunjiFsu/internal/pwdutil"
)

// InitDefaultData 初始化默认数据
func InitDefaultData() error {
	var count int
	err := ParamDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		slog.Warn("Failed to query users count, trying to create default user", "error", err)
		_, err := ParamDB.Exec(
			"INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
			"admin", pwdutil.Hash("123456"), "admin",
		)
		if err != nil {
			return fmt.Errorf("failed to create default user: %w", err)
		}
		slog.Info("Created default admin user")
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
		slog.Info("Created default admin user")
	}

	return nil
}
