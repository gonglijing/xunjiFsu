// =============================================================================
// 错误模块单元测试
// =============================================================================
package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestErrorCodeValues(t *testing.T) {
	tests := []struct {
		code    ErrorCode
		want    ErrorCode
		name    string
	}{
		{ErrCodeSuccess, ErrCodeSuccess, "ErrCodeSuccess"},
		{ErrCodeBadRequest, ErrCodeBadRequest, "ErrCodeBadRequest"},
		{ErrCodeUnauthorized, ErrCodeUnauthorized, "ErrCodeUnauthorized"},
		{ErrCodeForbidden, ErrCodeForbidden, "ErrCodeForbidden"},
		{ErrCodeNotFound, ErrCodeNotFound, "ErrCodeNotFound"},
		{ErrCodeConflict, ErrCodeConflict, "ErrCodeConflict"},
		{ErrCodeInternalError, ErrCodeInternalError, "ErrCodeInternalError"},
		{ErrCodeDatabaseError, ErrCodeDatabaseError, "ErrCodeDatabaseError"},
		{ErrCodeTimeout, ErrCodeTimeout, "ErrCodeTimeout"},
		{ErrCodeRateLimited, ErrCodeRateLimited, "ErrCodeRateLimited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "simple message",
			err:      &AppError{Code: ErrCodeBadRequest, Message: "bad request"},
			expected: "bad request",
		},
		{
			name:     "with underlying error",
			err:      &AppError{Code: ErrCodeDatabaseError, Message: "db error", Err: errors.New("connection failed")},
			expected: "db error: connection failed",
		},
		{
			name:     "with details",
			err:      &AppError{Code: ErrCodeBadRequest, Message: "validation failed", Details: "field X is required"},
			expected: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &AppError{Code: ErrCodeDatabaseError, Message: "db error", Err: underlying}

	result := err.Unwrap()

	if result != underlying {
		t.Errorf("Unwrap() = %v, want %v", result, underlying)
	}
}

func TestAppError_UnwrapNil(t *testing.T) {
	err := &AppError{Code: ErrCodeBadRequest, Message: "bad request"}

	result := err.Unwrap()

	if result != nil {
		t.Errorf("Unwrap() = %v, want nil", result)
	}
}

func TestAppError_HTTPStatus(t *testing.T) {
	tests := []struct {
		name          string
		err           *AppError
		expectedStatus int
	}{
		{
			name:          "ErrCodeBadRequest",
			err:           NewError(ErrCodeBadRequest, "bad request"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:          "ErrCodeUnauthorized",
			err:           NewError(ErrCodeUnauthorized, "unauthorized"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:          "ErrCodeForbidden",
			err:           NewError(ErrCodeForbidden, "forbidden"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:          "ErrCodeNotFound",
			err:           NewError(ErrCodeNotFound, "not found"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:          "ErrCodeConflict",
			err:           NewError(ErrCodeConflict, "conflict"),
			expectedStatus: http.StatusConflict,
		},
		{
			name:          "ErrCodeRateLimited",
			err:           NewError(ErrCodeRateLimited, "rate limited"),
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name:          "ErrCodeDatabaseError",
			err:           NewError(ErrCodeDatabaseError, "db error"),
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:          "ErrCodeInternalError",
			err:           NewError(ErrCodeInternalError, "internal error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:          "ErrCodeSuccess",
			err:           NewError(ErrCodeSuccess, "success"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:          "Unknown code defaults to 500",
			err:           &AppError{Code: ErrorCode(999), Message: "unknown"},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := tt.err.HTTPStatus()
			if status != tt.expectedStatus {
				t.Errorf("HTTPStatus() = %d, want %d", status, tt.expectedStatus)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeNotFound, "resource not found")

	if err.Code != ErrCodeNotFound {
		t.Errorf("Code = %d, want %d", err.Code, ErrCodeNotFound)
	}
	if err.Message != "resource not found" {
		t.Errorf("Message = %s, want 'resource not found'", err.Message)
	}
	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
	}
	if err.Details != "" {
		t.Errorf("Details = %s, want empty", err.Details)
	}
}

func TestNewErrorWithErr(t *testing.T) {
	underlying := errors.New("database connection failed")
	err := NewErrorWithErr(ErrCodeDatabaseError, "database error", underlying)

	if err.Code != ErrCodeDatabaseError {
		t.Errorf("Code = %d, want %d", err.Code, ErrCodeDatabaseError)
	}
	if err.Message != "database error" {
		t.Errorf("Message = %s, want 'database error'", err.Message)
	}
	if err.Err != underlying {
		t.Errorf("Err = %v, want %v", err.Err, underlying)
	}
}

func TestWrapError_Nil(t *testing.T) {
	result := WrapError(nil, ErrCodeBadRequest, "wrapped")

	if result != nil {
		t.Errorf("WrapError(nil) = %v, want nil", result)
	}
}

func TestWrapError_AppError(t *testing.T) {
	original := &AppError{
		Code:    ErrCodeNotFound,
		Message: "original error",
		Details: "some details",
	}

	result := WrapError(original, ErrCodeBadRequest, "wrapped message")

	if result == nil {
		t.Fatal("WrapError returned nil")
	}

	// 验证代码被更新
	if result.Code != ErrCodeBadRequest {
		t.Errorf("Code = %d, want %d", result.Code, ErrCodeBadRequest)
	}

	// 验证消息被更新
	if result.Message != "wrapped message" {
		t.Errorf("Message = %s, want 'wrapped message'", result.Message)
	}

	// 验证保留了原始的Details
	if result.Details != "some details" {
		t.Errorf("Details = %s, want 'some details'", result.Details)
	}
}

func TestWrapError_StandardError(t *testing.T) {
	original := errors.New("standard error")
	result := WrapError(original, ErrCodeInternalError, "internal error")

	if result == nil {
		t.Fatal("WrapError returned nil")
	}

	if result.Code != ErrCodeInternalError {
		t.Errorf("Code = %d, want %d", result.Code, ErrCodeInternalError)
	}
	if result.Message != "internal error" {
		t.Errorf("Message = %s, want 'internal error'", result.Message)
	}
	if result.Err != original {
		t.Errorf("Err = %v, want %v", result.Err, original)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		err      *AppError
		code     ErrorCode
		expected string
	}{
		{ErrNotFound, ErrCodeNotFound, "Resource not found"},
		{ErrUnauthorized, ErrCodeUnauthorized, "Unauthorized"},
		{ErrForbidden, ErrCodeForbidden, "Forbidden"},
		{ErrBadRequest, ErrCodeBadRequest, "Bad request"},
		{ErrInternalError, ErrCodeInternalError, "Internal server error"},
		{ErrDatabaseError, ErrCodeDatabaseError, "Database error"},
		{ErrTimeout, ErrCodeTimeout, "Operation timeout"},
		{ErrRateLimited, ErrCodeRateLimited, "Rate limited"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %d, want %d", tt.err.Code, tt.code)
			}
			if tt.err.Message != tt.expected {
				t.Errorf("Message = %s, want %s", tt.err.Message, tt.expected)
			}
		})
	}
}

func TestIs_AppError(t *testing.T) {
	target := NewError(ErrCodeNotFound, "not found")
	otherError := errors.New("some error")
	_ = NewError(ErrCodeBadRequest, "bad request")

	tests := []struct {
		name     string
		err      error
		target   *AppError
		expected bool
	}{
		{
			name:     "matching AppError",
			err:      NewError(ErrCodeNotFound, "not found"),
			target:   target,
			expected: true,
		},
		{
			name:     "different code",
			err:      NewError(ErrCodeBadRequest, "bad request"),
			target:   target,
			expected: false,
		},
		{
			name:     "standard error",
			err:      otherError,
			target:   target,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			target:   target,
			expected: false,
		},
		{
			name:     "wrapped AppError",
			err:      WrapError(target, ErrCodeInternalError, "wrapped"),
			target:   target,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Is(tt.err, tt.target)
			if result != tt.expected {
				t.Errorf("Is() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIs_NilTarget(t *testing.T) {
	result := Is(errors.New("error"), nil)

	if result != false {
		t.Errorf("Is() = %v, want false", result)
	}
}

func TestAppError_JSONMarshal(t *testing.T) {
	err := &AppError{
		Code:    ErrCodeBadRequest,
		Message: "test error",
		Details: "some details",
	}

	// 验证字段存在且可以访问
	if err.Code != ErrCodeBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, ErrCodeBadRequest)
	}
	if err.Message != "test error" {
		t.Errorf("Message = %s, want 'test error'", err.Message)
	}
	if err.Details != "some details" {
		t.Errorf("Details = %s, want 'some details'", err.Details)
	}
}

func TestErrorCode_Iota(t *testing.T) {
	// 验证 iota 从 0 开始
	if ErrCodeSuccess != 0 {
		t.Errorf("ErrCodeSuccess = %d, want 0", ErrCodeSuccess)
	}
	if ErrCodeBadRequest != 1 {
		t.Errorf("ErrCodeBadRequest = %d, want 1", ErrCodeBadRequest)
	}
}
