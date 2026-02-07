package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode 错误代码
type ErrorCode int

const (
	ErrCodeSuccess ErrorCode = iota
	ErrCodeBadRequest
	ErrCodeUnauthorized
	ErrCodeForbidden
	ErrCodeNotFound
	ErrCodeConflict
	ErrCodeInternalError
	ErrCodeDatabaseError
	ErrCodeTimeout
	ErrCodeRateLimited
)

// AppError 应用错误
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Err     error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// HTTPStatus 返回对应的HTTP状态码
func (e *AppError) HTTPStatus() int {
	switch e.Code {
	case ErrCodeBadRequest:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeForbidden:
		return http.StatusForbidden
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeConflict:
		return http.StatusConflict
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeDatabaseError:
		return http.StatusServiceUnavailable
	case ErrCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// NewError 创建新错误
func NewError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithErr 创建带底层错误的错误
func NewErrorWithErr(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// WrapError 包装错误
func WrapError(err error, code ErrorCode, message string) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		// 已经是AppError，只更新消息
		return &AppError{
			Code:    code,
			Message: message,
			Details: appErr.Details,
			Err:     appErr,
		}
	}

	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// 预定义错误
var (
	ErrNotFound      = NewError(ErrCodeNotFound, "Resource not found")
	ErrUnauthorized  = NewError(ErrCodeUnauthorized, "Unauthorized")
	ErrForbidden     = NewError(ErrCodeForbidden, "Forbidden")
	ErrBadRequest    = NewError(ErrCodeBadRequest, "Bad request")
	ErrInternalError = NewError(ErrCodeInternalError, "Internal server error")
	ErrDatabaseError = NewError(ErrCodeDatabaseError, "Database error")
	ErrTimeout       = NewError(ErrCodeTimeout, "Operation timeout")
	ErrRateLimited   = NewError(ErrCodeRateLimited, "Rate limited")
)

// Is 检查错误是否为指定类型
func Is(err error, target *AppError) bool {
	if err == nil || target == nil {
		return false
	}

	for current := err; current != nil; current = errors.Unwrap(current) {
		appErr, ok := current.(*AppError)
		if !ok || appErr == nil {
			continue
		}
		if appErr.Code == target.Code {
			return true
		}
	}
	return false
}
