package handlers

import (
	"net/http"
	"sync"
	"time"
)

type rateState struct {
	mu    sync.Mutex
	times []time.Time
}

// RateLimiter 请求限流器
type RateLimiter struct {
	requests sync.Map // key:string(ip) -> *rateState
	limit    int
	window   time.Duration
	enabled  bool
}

// NewRateLimiter 创建限流器
func NewRateLimiter(requestsPerMinute int, enabled bool) *RateLimiter {
	return &RateLimiter{
		limit:   requestsPerMinute,
		window:  time.Minute,
		enabled: enabled,
	}
}

func (rl *RateLimiter) getOrCreateState(ip string) *rateState {
	if ip == "" {
		ip = "unknown"
	}
	if existing, ok := rl.requests.Load(ip); ok {
		return existing.(*rateState)
	}
	state := &rateState{}
	actual, _ := rl.requests.LoadOrStore(ip, state)
	return actual.(*rateState)
}

func trimRecentTimes(times []time.Time, windowStart time.Time) []time.Time {
	if len(times) == 0 {
		return times[:0]
	}
	writeIdx := 0
	for _, t := range times {
		if t.After(windowStart) {
			times[writeIdx] = t
			writeIdx++
		}
	}
	return times[:writeIdx]
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(ip string) bool {
	if !rl.enabled {
		return true
	}

	state := rl.getOrCreateState(ip)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)
	state.times = trimRecentTimes(state.times, windowStart)

	if len(state.times) >= rl.limit {
		return false
	}

	state.times = append(state.times, now)
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

type bruteForceState struct {
	mu           sync.Mutex
	failures     []time.Time
	blockedUntil time.Time
}

// BruteForceLimiter 暴力破解防护
type BruteForceLimiter struct {
	states    sync.Map // key:string(ip) -> *bruteForceState
	limit     int
	window    time.Duration
	blockTime time.Duration
}

// NewBruteForceLimiter 创建暴力破解防护器
func NewBruteForceLimiter(maxFailures int, blockDuration time.Duration) *BruteForceLimiter {
	return &BruteForceLimiter{
		limit:     maxFailures,
		window:    15 * time.Minute,
		blockTime: blockDuration,
	}
}

func (b *BruteForceLimiter) getOrCreateState(ip string) *bruteForceState {
	if ip == "" {
		ip = "unknown"
	}
	if existing, ok := b.states.Load(ip); ok {
		return existing.(*bruteForceState)
	}
	state := &bruteForceState{}
	actual, _ := b.states.LoadOrStore(ip, state)
	return actual.(*bruteForceState)
}

// RecordFailure 记录登录失败
func (b *BruteForceLimiter) RecordFailure(ip string) {
	state := b.getOrCreateState(ip)
	state.mu.Lock()
	defer state.mu.Unlock()

	now := time.Now()
	if !state.blockedUntil.IsZero() && now.Before(state.blockedUntil) {
		return
	}
	if !state.blockedUntil.IsZero() && !now.Before(state.blockedUntil) {
		state.blockedUntil = time.Time{}
	}

	windowStart := now.Add(-b.window)
	state.failures = trimRecentTimes(state.failures, windowStart)
	state.failures = append(state.failures, now)

	if len(state.failures) >= b.limit {
		state.blockedUntil = now.Add(b.blockTime)
		state.failures = state.failures[:0]
	}
}

// RecordSuccess 记录登录成功
func (b *BruteForceLimiter) RecordSuccess(ip string) {
	state := b.getOrCreateState(ip)
	state.mu.Lock()
	state.failures = state.failures[:0]
	state.blockedUntil = time.Time{}
	state.mu.Unlock()
}

// IsBlocked 检查IP是否被封禁
func (b *BruteForceLimiter) IsBlocked(ip string) bool {
	state := b.getOrCreateState(ip)
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.blockedUntil.IsZero() {
		return false
	}
	if time.Now().Before(state.blockedUntil) {
		return true
	}
	state.blockedUntil = time.Time{}
	return false
}

// BlockStatus 返回封禁状态
func (b *BruteForceLimiter) BlockStatus(ip string) (bool, time.Duration) {
	state := b.getOrCreateState(ip)
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.blockedUntil.IsZero() {
		return false, 0
	}
	if remaining := time.Until(state.blockedUntil); remaining > 0 {
		return true, remaining
	}
	state.blockedUntil = time.Time{}
	return false, 0
}
