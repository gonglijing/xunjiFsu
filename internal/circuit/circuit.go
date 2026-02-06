package circuit

import (
	"sync"
	"time"
)

var log = &nullLogger{}

type nullLogger struct{}

func (l *nullLogger) Printf(format string, v ...interface{}) {}
func (l *nullLogger) Println(v ...interface{})              {}

// CircuitState 熔断器状态
type CircuitState int

const (
	Closed CircuitState = iota // 关闭状态，正常运行
	Open                      // 打开状态，拒绝请求
	HalfOpen                  // 半开状态，尝试恢复
)

// String 返回状态字符串
func (s CircuitState) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Config 熔断器配置
type Config struct {
	FailureThreshold  int           // 失败次数阈值
	FailureWindow     time.Duration // 失败计数时间窗口
	SuccessThreshold  int           // 半开状态下成功次数阈值
	RecoveryTimeout   time.Duration // 恢复尝试间隔
	RequestTimeout    time.Duration // 单个请求超时
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		FailureThreshold:  5,           // 5次失败
		FailureWindow:     time.Minute, // 1分钟窗口
		SuccessThreshold:  3,           // 需要3次成功
		RecoveryTimeout:   time.Second * 30, // 30秒后尝试恢复
		RequestTimeout:    time.Second * 10, // 请求超时10秒
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	state           CircuitState
	config          *Config
	failures        []time.Time // 失败时间戳
	successes       []time.Time // 成功时间戳
	lastFailure     time.Time
	mu              sync.RWMutex
	requestCount    int64
	failureCount    int64
	successCount    int64
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig()
	}
	return &CircuitBreaker{
		state:  Closed,
		config: config,
		failures: make([]time.Time, 0),
		successes: make([]time.Time, 0),
	}
}

// Execute 执行受保护的函数
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	cb.requestCount++
	cb.mu.Unlock()

	if !cb.allowRequest() {
		return &CircuitOpenError{
			RetryAfter: cb.retryAfter(),
		}
	}

	err := fn()
	cb.recordResult(err == nil)
	return err
}

// allowRequest 检查是否允许请求
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true
	case Open:
		if time.Since(cb.lastFailure) >= cb.config.RecoveryTimeout {
			cb.state = HalfOpen
			cb.successes = cb.successes[:0]
			cb.failures = cb.failures[:0]
			return true
		}
		return false
	case HalfOpen:
		return true
	default:
		return true
	}
}

// recordResult 记录结果
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	if success {
		cb.successCount++
		cb.successes = append(cb.successes, now)
		cb.cleanup(now)

		if cb.state == HalfOpen {
			if len(cb.successes) >= cb.config.SuccessThreshold {
				cb.state = Closed
				cb.failures = cb.failures[:0]
				log.Printf("Circuit breaker state: %s -> %s", HalfOpen, Closed)
			}
		}
	} else {
		cb.failureCount++
		cb.lastFailure = now
		cb.failures = append(cb.failures, now)
		cb.cleanup(now)

		if cb.state == HalfOpen {
			cb.state = Open
			log.Printf("Circuit breaker state: %s -> %s (half-open failure)", HalfOpen, Open)
		} else if cb.state == Closed {
			if len(cb.failures) >= cb.config.FailureThreshold {
				cb.state = Open
				log.Printf("Circuit breaker state: %s -> %s (failure threshold reached)", Closed, Open)
			}
		}
	}
}

// cleanup 清理过期记录
func (cb *CircuitBreaker) cleanup(now time.Time) {
	windowStart := now.Add(-cb.config.FailureWindow)

	var validFailures []time.Time
	for _, t := range cb.failures {
		if t.After(windowStart) {
			validFailures = append(validFailures, t)
		}
	}
	cb.failures = validFailures

	var validSuccesses []time.Time
	for _, t := range cb.successes {
		if t.After(windowStart) {
			validSuccesses = append(validSuccesses, t)
		}
	}
	cb.successes = validSuccesses
}

// retryAfter 返回重试等待时间
func (cb *CircuitBreaker) retryAfter() time.Duration {
	return cb.config.RecoveryTimeout - time.Since(cb.lastFailure)
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 当处于打开状态且超过恢复超时时间时，自动进入半开状态
	if cb.state == Open && time.Since(cb.lastFailure) >= cb.config.RecoveryTimeout {
		cb.state = HalfOpen
		cb.successes = cb.successes[:0]
		cb.failures = cb.failures[:0]
	}

	return cb.state
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = Closed
	cb.failures = cb.failures[:0]
	cb.successes = cb.successes[:0]
	cb.requestCount = 0
	cb.failureCount = 0
	cb.successCount = 0
}

// Stats 获取统计信息
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":          cb.state.String(),
		"request_count":  cb.requestCount,
		"failure_count":  cb.failureCount,
		"success_count":  cb.successCount,
		"failure_rate":   cb.failureRate(),
		"retry_after":    cb.retryAfter().String(),
	}
}

// failureRate 计算失败率
func (cb *CircuitBreaker) failureRate() float64 {
	total := cb.failureCount + cb.successCount
	if total == 0 {
		return 0
	}
	return float64(cb.failureCount) / float64(total)
}

// CircuitOpenError 熔断器打开错误
type CircuitOpenError struct {
	RetryAfter time.Duration
}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open, retry after " + e.RetryAfter.String()
}
