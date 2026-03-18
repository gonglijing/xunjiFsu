package handlers

import (
	"net/http"
	"time"
)

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	ReadTimeout     time.Duration // 读取超时
	WriteTimeout    time.Duration // 写入超时
	IdleTimeout     time.Duration // 空闲超时
	ShutdownTimeout time.Duration // 关闭超时
}

// DefaultTimeoutConfig 默认超时配置
func DefaultTimeoutConfig() *TimeoutConfig {
	return &TimeoutConfig{
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// TimeoutMiddleware 创建超时中间件
func TimeoutMiddleware(cfg *TimeoutConfig) func(http.Handler) http.Handler {
	timeout := 30 * time.Second
	if cfg != nil && cfg.ReadTimeout > 0 {
		timeout = cfg.ReadTimeout
	}

	return func(next http.Handler) http.Handler {
		if timeout <= 0 {
			return next
		}
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}

// WithTimeout 为处理器添加超时
func WithTimeout(handler http.HandlerFunc, timeout time.Duration) http.HandlerFunc {
	if timeout <= 0 {
		return handler
	}

	timeoutHandler := http.TimeoutHandler(http.HandlerFunc(handler), timeout, "Request timeout")
	return func(w http.ResponseWriter, r *http.Request) {
		timeoutHandler.ServeHTTP(w, r)
	}
}
