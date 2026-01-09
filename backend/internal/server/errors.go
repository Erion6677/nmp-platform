package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorCode 错误代码类型
type ErrorCode string

const (
	// 认证相关错误
	ErrCodeUnauthorized     ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden        ErrorCode = "FORBIDDEN"
	ErrCodeInvalidToken     ErrorCode = "INVALID_TOKEN"
	ErrCodeTokenExpired     ErrorCode = "TOKEN_EXPIRED"
	
	// 验证相关错误
	ErrCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrCodeInvalidInput     ErrorCode = "INVALID_INPUT"
	ErrCodeMissingField     ErrorCode = "MISSING_FIELD"
	
	// 资源相关错误
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	ErrCodeConflict         ErrorCode = "CONFLICT"
	
	// 系统相关错误
	ErrCodeInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeDatabaseError    ErrorCode = "DATABASE_ERROR"
	ErrCodeExternalService  ErrorCode = "EXTERNAL_SERVICE_ERROR"
	
	// 业务相关错误
	ErrCodeDeviceNotFound   ErrorCode = "DEVICE_NOT_FOUND"
	ErrCodeInvalidDeviceID  ErrorCode = "INVALID_DEVICE_ID"
	ErrCodeDataFormatError  ErrorCode = "DATA_FORMAT_ERROR"
	ErrCodeRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
)

// AppError 应用错误结构
type AppError struct {
	Code       ErrorCode `json:"code"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"-"`
	Cause      error     `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap 支持errors.Unwrap
func (e *AppError) Unwrap() error {
	return e.Cause
}

// ErrorResponse HTTP错误响应结构
type ErrorResponse struct {
	Error     ErrorCode `json:"error"`
	Message   string    `json:"message"`
	Details   string    `json:"details,omitempty"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp int64     `json:"timestamp"`
}

// NewAppError 创建新的应用错误
func NewAppError(code ErrorCode, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewAppErrorWithCause 创建带原因的应用错误
func NewAppErrorWithCause(code ErrorCode, message string, statusCode int, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Cause:      cause,
	}
}

// NewAppErrorWithDetails 创建带详细信息的应用错误
func NewAppErrorWithDetails(code ErrorCode, message, details string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		Details:    details,
		StatusCode: statusCode,
	}
}

// 预定义的常用错误
var (
	ErrUnauthorized = NewAppError(ErrCodeUnauthorized, "Authentication required", http.StatusUnauthorized)
	ErrForbidden    = NewAppError(ErrCodeForbidden, "Access denied", http.StatusForbidden)
	ErrNotFound     = NewAppError(ErrCodeNotFound, "Resource not found", http.StatusNotFound)
	ErrInternalError = NewAppError(ErrCodeInternalError, "Internal server error", http.StatusInternalServerError)
	ErrValidationFailed = NewAppError(ErrCodeValidationFailed, "Validation failed", http.StatusBadRequest)
	ErrServiceUnavailable = NewAppError(ErrCodeServiceUnavailable, "Service temporarily unavailable", http.StatusServiceUnavailable)
)

// ErrorHandlerMiddleware 统一错误处理中间件
func ErrorHandlerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		// 处理错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handleError(c, err, logger)
		}
	}
}

// handleError 处理错误并返回适当的HTTP响应
func handleError(c *gin.Context, err error, logger *zap.Logger) {
	requestID := c.GetString("request_id")
	
	var appErr *AppError
	if errors.As(err, &appErr) {
		// 应用错误
		logError(logger, appErr, requestID, c)
		
		response := ErrorResponse{
			Error:     appErr.Code,
			Message:   appErr.Message,
			Details:   appErr.Details,
			RequestID: requestID,
			Timestamp: getCurrentTimestamp(),
		}
		
		c.JSON(appErr.StatusCode, response)
		return
	}
	
	// 处理Gin绑定错误
	if isBindingError(err) {
		validationErr := NewAppErrorWithDetails(
			ErrCodeValidationFailed,
			"Request validation failed",
			err.Error(),
			http.StatusBadRequest,
		)
		
		logError(logger, validationErr, requestID, c)
		
		response := ErrorResponse{
			Error:     validationErr.Code,
			Message:   validationErr.Message,
			Details:   validationErr.Details,
			RequestID: requestID,
			Timestamp: getCurrentTimestamp(),
		}
		
		c.JSON(validationErr.StatusCode, response)
		return
	}
	
	// 未知错误，记录详细信息并返回通用错误
	logger.Error("Unhandled error",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.Error(err),
	)
	
	response := ErrorResponse{
		Error:     ErrCodeInternalError,
		Message:   "An unexpected error occurred",
		RequestID: requestID,
		Timestamp: getCurrentTimestamp(),
	}
	
	c.JSON(http.StatusInternalServerError, response)
}

// logError 记录错误日志
func logError(logger *zap.Logger, appErr *AppError, requestID string, c *gin.Context) {
	fields := []zap.Field{
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("error_code", string(appErr.Code)),
		zap.String("error_message", appErr.Message),
		zap.Int("status_code", appErr.StatusCode),
	}
	
	if appErr.Details != "" {
		fields = append(fields, zap.String("error_details", appErr.Details))
	}
	
	if appErr.Cause != nil {
		fields = append(fields, zap.Error(appErr.Cause))
	}
	
	// 根据错误级别选择日志级别
	switch appErr.StatusCode {
	case http.StatusInternalServerError, http.StatusServiceUnavailable:
		logger.Error("Application error", fields...)
	case http.StatusUnauthorized, http.StatusForbidden:
		logger.Warn("Authentication/Authorization error", fields...)
	default:
		logger.Info("Client error", fields...)
	}
}

// isBindingError 检查是否为Gin绑定错误
func isBindingError(err error) bool {
	// 这里可以根据具体的错误类型进行判断
	// Gin的绑定错误通常包含特定的错误信息
	errStr := err.Error()
	return contains(errStr, "binding") || 
		   contains(errStr, "validation") || 
		   contains(errStr, "required") ||
		   contains(errStr, "invalid")
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     findSubstring(s, substr)))
}

// findSubstring 查找子字符串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() int64 {
	return getCurrentTime().Unix()
}

// getCurrentTime 获取当前时间（便于测试时mock）
var getCurrentTime = func() time.Time {
	return time.Now()
}

// AbortWithError 中止请求并返回错误
func AbortWithError(c *gin.Context, err *AppError) {
	c.Error(err)
	c.Abort()
}

// AbortWithAppError 中止请求并返回应用错误
func AbortWithAppError(c *gin.Context, code ErrorCode, message string, statusCode int) {
	err := NewAppError(code, message, statusCode)
	AbortWithError(c, err)
}

// AbortWithInternalError 中止请求并返回内部错误
func AbortWithInternalError(c *gin.Context, cause error) {
	err := NewAppErrorWithCause(ErrCodeInternalError, "Internal server error", http.StatusInternalServerError, cause)
	AbortWithError(c, err)
}

// AbortWithValidationError 中止请求并返回验证错误
func AbortWithValidationError(c *gin.Context, details string) {
	err := NewAppErrorWithDetails(ErrCodeValidationFailed, "Validation failed", details, http.StatusBadRequest)
	AbortWithError(c, err)
}

// AbortWithNotFoundError 中止请求并返回未找到错误
func AbortWithNotFoundError(c *gin.Context, resource string) {
	message := fmt.Sprintf("%s not found", resource)
	err := NewAppError(ErrCodeNotFound, message, http.StatusNotFound)
	AbortWithError(c, err)
}