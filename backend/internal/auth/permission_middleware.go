package auth

import (
	"net/http"
	"strconv"

	"nmp-platform/internal/repository"

	"github.com/gin-gonic/gin"
)

// DevicePermissionChecker 设备权限检查器
type DevicePermissionChecker struct {
	rbacService *RBACService
	permRepo    repository.PermissionRepository
	userRepo    repository.UserRepository
}

// NewDevicePermissionChecker 创建设备权限检查器
func NewDevicePermissionChecker(
	rbacService *RBACService,
	permRepo repository.PermissionRepository,
	userRepo repository.UserRepository,
) *DevicePermissionChecker {
	return &DevicePermissionChecker{
		rbacService: rbacService,
		permRepo:    permRepo,
		userRepo:    userRepo,
	}
}

// CheckDevicePermission 检查用户是否有设备权限
// 返回值: (是否有权限, 错误)
// 权限规则:
// 1. admin 角色可以操作所有设备
// 2. operator 角色只能操作分配给自己的设备
// 3. viewer 角色只能查看，不能修改
func (c *DevicePermissionChecker) CheckDevicePermission(userID, deviceID uint, action string) (bool, error) {
	// 获取用户信息（包含角色）
	user, err := c.userRepo.GetByID(userID)
	if err != nil {
		return false, err
	}

	// 检查用户角色
	isAdmin := false
	isOperator := false
	isViewer := false

	for _, role := range user.Roles {
		switch role.Name {
		case "admin":
			isAdmin = true
		case "operator":
			isOperator = true
		case "viewer":
			isViewer = true
		}
	}

	// admin 角色可以操作所有设备
	if isAdmin {
		return true, nil
	}

	// viewer 角色只能查看
	if isViewer && !isOperator {
		if action == "read" {
			// viewer 可以查看所有设备
			return true, nil
		}
		// viewer 不能进行其他操作
		return false, nil
	}

	// operator 角色需要检查设备权限
	if isOperator {
		// 检查是否有该设备的权限
		hasPermission, err := c.permRepo.HasDevicePermission(userID, deviceID)
		if err != nil {
			return false, err
		}
		return hasPermission, nil
	}

	// 没有任何角色，拒绝访问
	return false, nil
}

// IsSuperAdmin 检查用户是否是超级管理员
func (c *DevicePermissionChecker) IsSuperAdmin(userID uint) (bool, error) {
	user, err := c.userRepo.GetByID(userID)
	if err != nil {
		return false, err
	}

	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true, nil
		}
	}
	return false, nil
}

// GetUserRole 获取用户的主要角色
func (c *DevicePermissionChecker) GetUserRole(userID uint) (string, error) {
	user, err := c.userRepo.GetByID(userID)
	if err != nil {
		return "", err
	}

	// 按优先级返回角色: admin > operator > viewer
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return "admin", nil
		}
	}
	for _, role := range user.Roles {
		if role.Name == "operator" {
			return "operator", nil
		}
	}
	for _, role := range user.Roles {
		if role.Name == "viewer" {
			return "viewer", nil
		}
	}

	return "", nil
}

// DevicePermissionMiddleware 设备权限检查中间件
// action: read, create, update, delete
func DevicePermissionMiddleware(checker *DevicePermissionChecker, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "用户未认证",
			})
			c.Abort()
			return
		}

		// 获取设备ID（从URL参数）
		deviceIDStr := c.Param("id")
		if deviceIDStr == "" {
			// 如果没有设备ID参数，可能是列表或创建操作，跳过设备级别权限检查
			c.Next()
			return
		}

		deviceID, err := strconv.ParseUint(deviceIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "无效的设备ID",
			})
			c.Abort()
			return
		}

		// 检查设备权限
		allowed, err := checker.CheckDevicePermission(userID.(uint), uint(deviceID), action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "权限检查失败",
			})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "无权限操作此设备",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DeviceListFilterMiddleware 设备列表过滤中间件
// 用于过滤用户只能看到有权限的设备
func DeviceListFilterMiddleware(checker *DevicePermissionChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "用户未认证",
			})
			c.Abort()
			return
		}

		// 检查是否是超级管理员
		isAdmin, err := checker.IsSuperAdmin(userID.(uint))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "权限检查失败",
			})
			c.Abort()
			return
		}

		// 设置上下文变量，供后续处理使用
		c.Set("is_admin", isAdmin)
		c.Set("filter_by_permission", !isAdmin)

		c.Next()
	}
}

// PermissionMiddleware 权限检查中间件
func PermissionMiddleware(rbacService *RBACService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			c.Abort()
			return
		}

		// 检查权限
		allowed, err := rbacService.CheckPermission(userID.(uint), resource, action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check permission",
			})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission 要求特定权限的中间件工厂函数
func RequirePermission(rbacService *RBACService, resource, action string) gin.HandlerFunc {
	return PermissionMiddleware(rbacService, resource, action)
}

// RequireAnyPermission 要求任意权限之一的中间件
func RequireAnyPermission(rbacService *RBACService, permissions []Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			c.Abort()
			return
		}

		// 检查是否有任意一个权限
		hasPermission := false
		for _, perm := range permissions {
			allowed, err := rbacService.CheckPermission(userID.(uint), perm.Resource, perm.Action)
			if err != nil {
				continue
			}
			if allowed {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAllPermissions 要求所有权限的中间件
func RequireAllPermissions(rbacService *RBACService, permissions []Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			c.Abort()
			return
		}

		// 检查是否有所有权限
		for _, perm := range permissions {
			allowed, err := rbacService.CheckPermission(userID.(uint), perm.Resource, perm.Action)
			if err != nil || !allowed {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Insufficient permissions",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// Permission 权限结构
type Permission struct {
	Resource string
	Action   string
}