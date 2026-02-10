package handlers

import (
	"net/http"
	"sort"
	"sync"
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

// RequestTimer 请求计时器
type RequestTimer struct {
	mu            sync.RWMutex
	durations     []time.Duration
	maxRecords    int
	slowThreshold time.Duration
}

// NewRequestTimer 创建请求计时器
func NewRequestTimer(maxRecords int, slowThreshold time.Duration) *RequestTimer {
	return &RequestTimer{
		durations:     make([]time.Duration, 0, maxRecords),
		maxRecords:    maxRecords,
		slowThreshold: slowThreshold,
	}
}

// Record 记录请求耗时
func (t *RequestTimer) Record(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.durations = append(t.durations, d)
	if len(t.durations) > t.maxRecords {
		t.durations = t.durations[len(t.durations)-t.maxRecords:]
	}
}

// GetStats 获取统计信息
func (t *RequestTimer) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.durations) == 0 {
		return map[string]interface{}{
			"count":      0,
			"avg_ms":     0,
			"p50_ms":     0,
			"p95_ms":     0,
			"p99_ms":     0,
			"slow_count": 0,
		}
	}

	// 计算统计
	sorted := make([]time.Duration, len(t.durations))
	copy(sorted, t.durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	avg := time.Duration(0)
	for _, d := range sorted {
		avg += d
	}
	avg /= time.Duration(len(sorted))

	p50 := sorted[len(sorted)*50/100]
	p95 := sorted[len(sorted)*95/100]
	p99 := sorted[len(sorted)*99/100]

	slowCount := 0
	for _, d := range sorted {
		if d >= t.slowThreshold {
			slowCount++
		}
	}

	return map[string]interface{}{
		"count":      len(sorted),
		"avg_ms":     avg.Milliseconds(),
		"p50_ms":     p50.Milliseconds(),
		"p95_ms":     p95.Milliseconds(),
		"p99_ms":     p99.Milliseconds(),
		"slow_count": slowCount,
	}
}

// TimerMiddleware 请求计时中间件
func TimerMiddleware(timer *RequestTimer, name string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			duration := time.Since(start)
			timer.Record(duration)
		})
	}
}
