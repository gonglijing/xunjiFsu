package handlers

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter 请求限流器
type RateLimiter struct {
	mu          sync.RWMutex
	requests    map[string][]time.Time // IP -> 请求时间列表
	limit       int                    // 限流阈值
	window      time.Duration          // 时间窗口
	enabled     bool                   // 是否启用
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requestsPerMinute int, enabled bool) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
		enabled:  enabled,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(ip string) bool {
	if !rl.enabled {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// 清理过期的请求记录
	times := rl.requests[ip]
	var validTimes []time.Time
	for _, t := range times {
		if t.After(windowStart) {
			validTimes = append(validTimes, t)
		}
	}

	// 检查是否超过限制
	if len(validTimes) >= rl.limit {
		rl.requests[ip] = validTimes
		return false
	}

	// 记录新请求
	validTimes = append(validTimes, now)
	rl.requests[ip] = validTimes
	return true
}

// RateLimitMiddleware 创建限流中间件
func RateLimitMiddleware(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow(r.RemoteAddr) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BruteForceLimiter 暴力破解防护
type BruteForceLimiter struct {
	mu          sync.RWMutex
	failures    map[string][]time.Time // IP -> 失败时间列表
	limit       int                    // 失败次数阈值
	window      time.Duration          // 时间窗口
	blockTime   time.Duration          // 封禁时间
	blocked     map[string]time.Time   // 被封禁的IP
}

// NewBruteForceLimiter 创建暴力破解防护器
func NewBruteForceLimiter(maxFailures int, blockDuration time.Duration) *BruteForceLimiter {
	return &BruteForceLimiter{
		failures:  make(map[string][]time.Time),
		limit:     maxFailures,
		window:    15 * time.Minute,
		blockTime: blockDuration,
		blocked:   make(map[string]time.Time),
	}
}

// RecordFailure 记录登录失败
func (b *BruteForceLimiter) RecordFailure(ip string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-b.window)

	// 检查是否已被封禁
	if blockedUntil, exists := b.blocked[ip]; exists {
		if now.Before(blockedUntil) {
			return // 仍在封禁期
		}
		delete(b.blocked, ip) // 封禁已解除
	}

	// 清理过期的失败记录
	times := b.failures[ip]
	var validTimes []time.Time
	for _, t := range times {
		if t.After(windowStart) {
			validTimes = append(validTimes, t)
		}
	}

	// 添加新失败记录
	validTimes = append(validTimes, now)
	b.failures[ip] = validTimes

	// 检查是否需要封禁
	if len(validTimes) >= b.limit {
		b.blocked[ip] = now.Add(b.blockTime)
		delete(b.failures, ip)
	}
}

// RecordSuccess 记录登录成功
func (b *BruteForceLimiter) RecordSuccess(ip string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.failures, ip)
}

// IsBlocked 检查IP是否被封禁
func (b *BruteForceLimiter) IsBlocked(ip string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if blockedUntil, exists := b.blocked[ip]; exists {
		if time.Now().Before(blockedUntil) {
			return true
		}
		delete(b.blocked, ip)
	}
	return false
}

// BlockStatus 返回封禁状态
func (b *BruteForceLimiter) BlockStatus(ip string) (bool, time.Duration) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if blockedUntil, exists := b.blocked[ip]; exists {
		if remaining := time.Until(blockedUntil); remaining > 0 {
			return true, remaining
		}
		delete(b.blocked, ip)
	}
	return false, 0
}
