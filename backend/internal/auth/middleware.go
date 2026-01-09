package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 HttpOnly Cookie 获取 Token
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
		} else {
			// 回退到 Authorization Header（兼容旧客户端）
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Authorization required",
				})
				c.Abort()
				return
			}

			// 检查Bearer前缀
			if !strings.HasPrefix(authHeader, "Bearer ") {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Invalid authorization header format",
				})
				c.Abort()
				return
			}

			tokenString = authHeader[7:] // 移除 "Bearer " 前缀
		}

		// 验证令牌
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// OptionalAuthMiddleware 可选认证中间件（不强制要求认证）
func OptionalAuthMiddleware(authService *AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从 HttpOnly Cookie 获取 Token
		if cookie, err := c.Cookie("auth_token"); err == nil && cookie != "" {
			tokenString = cookie
		} else {
			// 回退到 Authorization Header
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = authHeader[7:]
			}
		}

		if tokenString != "" {
			if claims, err := authService.ValidateToken(tokenString); err == nil {
				c.Set("user_id", claims.UserID)
				c.Set("username", claims.Username)
				c.Set("roles", claims.Roles)
			}
		}
		c.Next()
	}
}

// RequireRoles 要求特定角色的中间件
func RequireRoles(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user roles format",
			})
			c.Abort()
			return
		}

		// 检查用户是否拥有所需角色之一
		hasRequiredRole := false
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					hasRequiredRole = true
					break
				}
			}
			if hasRequiredRole {
				break
			}
		}

		if !hasRequiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}