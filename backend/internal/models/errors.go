package models

import "fmt"

// ValidationError 数据验证错误
type ValidationError struct {
	Message string
	Field   string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// NewValidationError 创建验证错误
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		Message: message,
	}
}

// NewFieldValidationError 创建字段验证错误
func NewFieldValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// AuthError 认证错误
type AuthError struct {
	Message string
	Code    string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error [%s]: %s", e.Code, e.Message)
}

// NewAuthError 创建认证错误
func NewAuthError(code, message string) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
	}
}

// NotFoundError 资源未找到错误
type NotFoundError struct {
	Resource string
	ID       interface{}
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %v", e.Resource, e.ID)
}

// NewNotFoundError 创建资源未找到错误
func NewNotFoundError(resource string, id interface{}) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		ID:       id,
	}
}

// ConflictError 资源冲突错误
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("conflict error: %s", e.Message)
}

// NewConflictError 创建资源冲突错误
func NewConflictError(message string) *ConflictError {
	return &ConflictError{
		Message: message,
	}
}

// ErrorResponse API错误响应结构
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}