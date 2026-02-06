// =============================================================================
// 熔断器模块单元测试
// =============================================================================
package circuit

import (
	"testing"
	"time"
)

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{Closed, "closed"},
		{Open, "open"},
		{HalfOpen, "half_open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", config.FailureThreshold)
	}
	if config.FailureWindow != time.Minute {
		t.Errorf("FailureWindow = %v, want 1m", config.FailureWindow)
	}
	if config.SuccessThreshold != 3 {
		t.Errorf("SuccessThreshold = %d, want 3", config.SuccessThreshold)
	}
	if config.RecoveryTimeout != 30*time.Second {
		t.Errorf("RecoveryTimeout = %v, want 30s", config.RecoveryTimeout)
	}
	if config.RequestTimeout != 10*time.Second {
		t.Errorf("RequestTimeout = %v, want 10s", config.RequestTimeout)
	}
}

func TestNewCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	if cb.state != Closed {
		t.Errorf("state = %s, want closed", cb.state.String())
	}
	if cb.config == nil {
		t.Error("config is nil")
	}
	if cb.config.FailureThreshold != 5 {
		t.Errorf("config.FailureThreshold = %d, want 5", cb.config.FailureThreshold)
	}
}

func TestNewCircuitBreaker_CustomConfig(t *testing.T) {
	config := &Config{
		FailureThreshold:  10,
		FailureWindow:     2 * time.Minute,
		SuccessThreshold:  5,
		RecoveryTimeout:   time.Minute,
		RequestTimeout:    30 * time.Second,
	}

	cb := NewCircuitBreaker(config)

	if cb.config != config {
		t.Error("config not set correctly")
	}
	if cb.state != Closed {
		t.Errorf("state = %s, want closed", cb.state.String())
	}
}

func TestCircuitBreaker_State(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	if cb.State() != Closed {
		t.Errorf("initial state = %s, want closed", cb.State().String())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	// 触发一些失败
	cb.Execute(func() error { return nil })
	cb.Execute(func() error { return nil })

	// 重置
	cb.Reset()

	if cb.State() != Closed {
		t.Errorf("state after reset = %s, want closed", cb.State().String())
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if cb.State() != Closed {
		t.Errorf("state = %s, want closed", cb.State().String())
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	expectedErr := &testError{msg: "test error"}
	err := cb.Execute(func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Execute() error = %v, want %v", err, expectedErr)
	}
}

func TestCircuitBreaker_Execute_OpensAfterThreshold(t *testing.T) {
	config := &Config{
		FailureThreshold:  3,
		FailureWindow:     time.Minute,
		SuccessThreshold:  2,
		RecoveryTimeout:   time.Second * 1,
		RequestTimeout:    time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 触发 3 次失败，应该打开熔断器
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return &testError{msg: "error"}
		})
	}

	if cb.State() != Open {
		t.Errorf("state after 3 failures = %s, want open", cb.State().String())
	}
}

func TestCircuitBreaker_Execute_OpenStateRejects(t *testing.T) {
	config := &Config{
		FailureThreshold:  1,
		FailureWindow:    time.Minute,
		SuccessThreshold: 1,
		RecoveryTimeout:  time.Hour,
		RequestTimeout:   time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 触发一次失败
	cb.Execute(func() error {
		return &testError{msg: "error"}
	})

	// 熔断器应该打开
	if cb.State() != Open {
		t.Errorf("state = %s, want open", cb.State().String())
	}

	// 再次执行应该返回 CircuitOpenError
	err := cb.Execute(func() error {
		return nil
	})

	if err == nil {
		t.Error("Execute() on open circuit returned nil")
	}
	_, ok := err.(*CircuitOpenError)
	if !ok {
		t.Errorf("error type = %T, want *CircuitOpenError", err)
	}
}

func TestCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	config := &Config{
		FailureThreshold:  2,
		FailureWindow:     time.Minute,
		SuccessThreshold:  2,
		RecoveryTimeout:   10 * time.Millisecond,
		RequestTimeout:    time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 触发打开
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return &testError{msg: "error"}
		})
	}

	// 等待恢复超时
	time.Sleep(20 * time.Millisecond)

	// 应该进入半开状态
	if cb.State() != HalfOpen {
		t.Errorf("state after recovery timeout = %s, want half_open", cb.State().String())
	}

	// 半开状态下成功
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("Execute() error = %v, want nil", err)
		}
	}

	// 应该恢复关闭状态
	if cb.State() != Closed {
		t.Errorf("state after successful recovery = %s, want closed", cb.State().String())
	}
}

func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	config := &Config{
		FailureThreshold:  2,
		FailureWindow:     time.Minute,
		SuccessThreshold:  2,
		RecoveryTimeout:   10 * time.Millisecond,
		RequestTimeout:    time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 触发打开
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return &testError{msg: "error"}
		})
	}

	// 等待恢复超时
	time.Sleep(20 * time.Millisecond)

	// 半开状态下失败，应该重新打开
	err := cb.Execute(func() error {
		return &testError{msg: "error"}
	})

	if err == nil {
		t.Error("Execute() should return error")
	}

	if cb.State() != Open {
		t.Errorf("state after half-open failure = %s, want open", cb.State().String())
	}
}

func TestCircuitBreaker_RetryAfter(t *testing.T) {
	config := &Config{
		FailureThreshold:  1,
		FailureWindow:     time.Minute,
		SuccessThreshold:  1,
		RecoveryTimeout:   5 * time.Second,
		RequestTimeout:    time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 触发失败
	cb.Execute(func() error {
		return &testError{msg: "error"}
	})

	retryAfter := cb.retryAfter()

	// 应该接近恢复超时
	if retryAfter > 5*time.Second || retryAfter < 4*time.Second {
		t.Errorf("retryAfter = %v, want approximately 5s", retryAfter)
	}
}

func TestCircuitBreaker_Stats(t *testing.T) {
	config := &Config{
		FailureThreshold:  5,
		FailureWindow:     time.Minute,
		SuccessThreshold:  3,
		RecoveryTimeout:   30 * time.Second,
		RequestTimeout:    time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 执行一些成功和失败
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return nil
		})
	}
	for i := 0; i < 2; i++ {
		cb.Execute(func() error {
			return &testError{msg: "error"}
		})
	}

	stats := cb.Stats()

	if stats["state"] != "closed" {
		t.Errorf("stats state = %v, want closed", stats["state"])
	}
	if stats["request_count"].(int64) != 5 {
		t.Errorf("stats request_count = %v, want 5", stats["request_count"])
	}
	if stats["success_count"].(int64) != 3 {
		t.Errorf("stats success_count = %v, want 3", stats["success_count"])
	}
	if stats["failure_count"].(int64) != 2 {
		t.Errorf("stats failure_count = %v, want 2", stats["failure_count"])
	}
}

func TestCircuitBreaker_Stats_Empty(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	stats := cb.Stats()

	if stats["request_count"].(int64) != 0 {
		t.Errorf("stats request_count = %v, want 0", stats["request_count"])
	}
	if stats["success_count"].(int64) != 0 {
		t.Errorf("stats success_count = %v, want 0", stats["success_count"])
	}
	if stats["failure_count"].(int64) != 0 {
		t.Errorf("stats failure_count = %v, want 0", stats["failure_count"])
	}
}

func TestCircuitBreaker_failureRate(t *testing.T) {
	tests := []struct {
		name         string
		successCount int64
		failureCount int64
		expected     float64
	}{
		{"all success", 10, 0, 0},
		{"all failure", 0, 10, 1},
		{"half half", 5, 5, 0.5},
		{"3:1 ratio", 3, 1, 0.25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := &CircuitBreaker{
				successCount: tt.successCount,
				failureCount: tt.failureCount,
			}

			rate := cb.failureRate()

			if rate != tt.expected {
				t.Errorf("failureRate() = %f, want %f", rate, tt.expected)
			}
		})
	}
}

func TestCircuitOpenError_Error(t *testing.T) {
	err := &CircuitOpenError{
		RetryAfter: 5 * time.Second,
	}

	expected := "circuit breaker is open, retry after 5s"
	if err.Error() != expected {
		t.Errorf("Error() = %s, want %s", err.Error(), expected)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				cb.Execute(func() error {
					return nil
				})
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// 所有并发执行完成，检查状态
	if cb.State() != Closed {
		t.Errorf("final state = %s, want closed", cb.State().String())
	}
}

// testError 用于测试的错误类型
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestNullLogger(t *testing.T) {
	logger := &nullLogger{}

	// 应该不 panic
	logger.Printf("test %s", "format")
	logger.Println("test")
}

func TestConfig_String(t *testing.T) {
	config := &Config{
		FailureThreshold:  5,
		FailureWindow:     time.Minute,
		SuccessThreshold:  3,
		RecoveryTimeout:   30 * time.Second,
		RequestTimeout:    10 * time.Second,
	}

	// 验证 config 字段可以访问
	if config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", config.FailureThreshold)
	}
}

func TestCircuitBreaker_ExecutePanic(t *testing.T) {
	cb := NewCircuitBreaker(nil)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Panic in Execute was not recovered")
		}
	}()

	cb.Execute(func() error {
		panic("test panic")
	})
}

func TestCircuitBreaker_RecoveryTimeout_ChangesState(t *testing.T) {
	config := &Config{
		FailureThreshold:  1,
		FailureWindow:     time.Minute,
		SuccessThreshold:  1,
		RecoveryTimeout:   50 * time.Millisecond,
		RequestTimeout:    time.Second * 10,
	}
	cb := NewCircuitBreaker(config)

	// 触发失败
	cb.Execute(func() error {
		return &testError{msg: "error"}
	})

	if cb.State() != Open {
		t.Errorf("state = %s, want open", cb.State().String())
	}

	// 等待恢复超时
	time.Sleep(70 * time.Millisecond)

	// 再次执行应该进入半开状态
	_ = cb.Execute(func() error {
		return nil
	})

	if cb.State() != Closed {
		t.Errorf("state after recovery = %s, want closed", cb.State().String())
	}
}
