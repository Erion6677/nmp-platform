package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse API 错误响应结构
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Code    int    `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse API 成功响应结构
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(message string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Message: message,
		Error:   message,
	}
}

// NewErrorResponseWithCode 创建带错误码的错误响应
func NewErrorResponseWithCode(code int, message string) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Code:    code,
		Message: message,
		Error:   message,
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Data:    data,
	}
}

// NewSuccessResponseWithMessage 创建带消息的成功响应
func NewSuccessResponseWithMessage(message string, data interface{}) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// ========== 便捷响应函数 ==========

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// SuccessWithMessage 成功响应带消息
func SuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
		"message": message,
	})
}

// SuccessPaginated 分页成功响应
func SuccessPaginated(c *gin.Context, items interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":       items,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// Error 错误响应
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   message,
	})
}

// ErrorWithDetails 错误响应带详情
func ErrorWithDetails(c *gin.Context, statusCode int, message string, details string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"error":   message,
		"details": details,
	})
}

// BadRequest 400 错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

// Unauthorized 401 错误
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

// Forbidden 403 错误
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

// NotFound 404 错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

// Conflict 409 错误
func Conflict(c *gin.Context, message string) {
	Error(c, http.StatusConflict, message)
}

// InternalError 500 错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
