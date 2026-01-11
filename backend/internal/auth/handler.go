package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *AuthService
}

// NewAuthHandler 创建新的认证处理器
func NewAuthHandler(authService *AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login 登录处理
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := h.authService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 设置 HttpOnly Cookie 存储 JWT Token
	// 生产环境应设置 Secure: true
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"auth_token",           // Cookie 名称
		response.Token,         // Token 值
		60*60*24*7,             // 过期时间：7天（秒）
		"/",                    // 路径
		"",                     // 域名（空表示当前域名）
		isSecure,               // Secure（仅 HTTPS）
		true,                   // HttpOnly（禁止 JS 访问）
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// Register 注册处理
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    user,
	})
}

// RefreshToken 刷新令牌处理
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Authorization header is required",
		})
		return
	}

	// 移除 "Bearer " 前缀
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	newToken, err := h.authService.RefreshToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token": newToken,
		},
	})
}

// GetProfile 获取用户资料
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userInfo, err := h.authService.GetUserByID(userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userInfo,
	})
}

// ChangePassword 修改密码处理
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 支持两种参数格式：驼峰命名和下划线命名
	var req struct {
		OldPassword  string `json:"old_password"`
		NewPassword  string `json:"new_password"`
		OldPassword2 string `json:"oldPassword"`
		NewPassword2 string `json:"newPassword"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 兼容两种参数格式
	oldPwd := req.OldPassword
	if oldPwd == "" {
		oldPwd = req.OldPassword2
	}
	newPwd := req.NewPassword
	if newPwd == "" {
		newPwd = req.NewPassword2
	}

	if oldPwd == "" || newPwd == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "old_password and new_password are required",
		})
		return
	}

	err := h.authService.ChangePassword(userID.(uint), oldPwd, newPwd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Password changed successfully",
	})
}

// UpdateProfile 更新个人信息处理
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	userInfo, err := h.authService.UpdateProfile(userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    userInfo,
		"message": "Profile updated successfully",
	})
}

// Logout 登出处理
func (h *AuthHandler) Logout(c *gin.Context) {
	// 清除 HttpOnly Cookie
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		"auth_token",
		"",
		-1,        // 立即过期
		"/",
		"",
		isSecure,
		true,
	)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

// RegisterRoutes 注册认证相关路由
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.GET("/login", func(c *gin.Context) {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error": "Method not allowed. Use POST to login.",
			})
		})
		auth.POST("/register", h.Register)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/refresh-token", h.RefreshToken) // 兼容前端路由别名
		auth.POST("/logout", h.Logout)
		
		// 需要认证的路由
		authenticated := auth.Group("")
		authenticated.Use(AuthMiddleware(h.authService))
		{
			authenticated.GET("/me", h.GetProfile)
			authenticated.GET("/profile", h.GetProfile)
			authenticated.PUT("/profile", h.UpdateProfile)
			authenticated.POST("/change-password", h.ChangePassword)
			authenticated.PUT("/password", h.ChangePassword)
		}
	}
}