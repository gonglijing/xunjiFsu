package database

import (
	"os"
	"strconv"
)

// getEnvInt 从环境变量获取整数配置
func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if val, err := strconv.Atoi(v); err == nil {
			return val
		}
	}
	return defaultVal
}
