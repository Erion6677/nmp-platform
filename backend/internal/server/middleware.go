package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// LoggerMiddleware 请求日志中间件
func LoggerMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 记录请求开始时间
		start := time.Now()

		// 记录请求体（仅用于调试，生产环境应谨慎使用）
		var requestBody []byte
		if c.Request.Body != nil && shouldLogRequestBody(c.Request.Header.Get("Content-Type")) {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 处理请求
		c.Next()

		// 计算处理时间
		duration := time.Since(start)

		// 记录请求日志
		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.Int("response_size", c.Writer.Size()),
		}

		// 添加请求体日志（仅在调试模式下）
		if gin.Mode() == gin.DebugMode && len(requestBody) > 0 && len(requestBody) < 1024 {
			fields = append(fields, zap.String("request_body", string(requestBody)))
		}

		// 根据状态码选择日志级别
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("HTTP Request", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("HTTP Request", fields...)
		default:
			logger.Info("HTTP Request", fields...)
		}
	}
}

// shouldLogRequestBody 判断是否应该记录请求体
func shouldLogRequestBody(contentType string) bool {
	// 只记录JSON和表单数据
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/x-www-form-urlencoded") ||
		strings.Contains(contentType, "multipart/form-data")
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 允许的来源列表（开发环境）
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:80",
			"http://localhost",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:3001",
			"http://127.0.0.1:80",
			"http://127.0.0.1",
			"http://10.10.10.2:3000",
			"http://10.10.10.2:3001",
			"http://10.10.10.2:80",
			"http://10.10.10.2",
		}
		
		// 检查请求来源是否在允许列表中
		allowOrigin := ""
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				allowOrigin = origin
				break
			}
		}
		
		// 如果没有匹配的来源，在开发模式下允许所有来源
		if allowOrigin == "" && gin.Mode() == gin.DebugMode {
			allowOrigin = origin
		}
		
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecurityMiddleware 安全中间件
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 安全头
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// 在生产环境中启用HSTS
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// RateLimitMiddleware 简单的速率限制中间件
func RateLimitMiddleware() gin.HandlerFunc {
	// 这里使用简单的内存存储，生产环境应该使用Redis
	clients := make(map[string][]time.Time)
	const maxRequests = 100
	const timeWindow = time.Minute

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// 清理过期的请求记录
		if requests, exists := clients[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < timeWindow {
					validRequests = append(validRequests, reqTime)
				}
			}
			clients[clientIP] = validRequests
		}

		// 检查请求数量
		if len(clients[clientIP]) >= maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"message": fmt.Sprintf("Maximum %d requests per minute allowed", maxRequests),
			})
			c.Abort()
			return
		}

		// 记录当前请求
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}

// TimeoutMiddleware 请求超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		// 替换请求上下文
		c.Request = c.Request.WithContext(ctx)
		
		// 检查是否已经超时
		select {
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "Request timeout",
				"message": "Request processing time exceeded the allowed limit",
			})
			c.Abort()
			return
		default:
			c.Next()
		}
	}
}

// RecoveryMiddleware 自定义恢复中间件
func RecoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := c.GetString("request_id")
		
		logger.Error("Panic recovered",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Any("panic", recovered),
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"message": "An unexpected error occurred",
			"request_id": requestID,
		})
	})
}