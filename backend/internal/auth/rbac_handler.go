package auth

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RBACHandler RBAC处理器
type RBACHandler struct {
	rbacService *RBACService
}

// NewRBACHandler 创建新的RBAC处理器
func NewRBACHandler(rbacService *RBACService) *RBACHandler {
	return &RBACHandler{
		rbacService: rbacService,
	}
}

// CreateRole 创建角色
func (h *RBACHandler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		DisplayName string `json:"display_name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	role, err := h.rbacService.CreateRole(req.Name, req.DisplayName, req.Description, false)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    role,
	})
}

// CreatePermission 创建权限
func (h *RBACHandler) CreatePermission(c *gin.Context) {
	var req struct {
		Resource    string `json:"resource" binding:"required"`
		Action      string `json:"action" binding:"required"`
		Scope       string `json:"scope"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	permission, err := h.rbacService.CreatePermission(req.Resource, req.Action, req.Scope, req.Description)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    permission,
	})
}

// AssignRoleToUser 为用户分配角色
func (h *RBACHandler) AssignRoleToUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	var req struct {
		RoleID uint `json:"role_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err = h.rbacService.AssignRoleToUser(uint(userID), req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role assigned successfully",
	})
}

// RemoveRoleFromUser 从用户移除角色
func (h *RBACHandler) RemoveRoleFromUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	roleIDStr := c.Param("role_id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID",
		})
		return
	}

	err = h.rbacService.RemoveRoleFromUser(uint(userID), uint(roleID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Role removed successfully",
	})
}

// AssignPermissionToRole 为角色分配权限
func (h *RBACHandler) AssignPermissionToRole(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID",
		})
		return
	}

	var req struct {
		PermissionID uint `json:"permission_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	err = h.rbacService.AssignPermissionToRole(uint(roleID), req.PermissionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Permission assigned successfully",
	})
}

// RemovePermissionFromRole 从角色移除权限
func (h *RBACHandler) RemovePermissionFromRole(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID",
		})
		return
	}

	permissionIDStr := c.Param("permission_id")
	permissionID, err := strconv.ParseUint(permissionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid permission ID",
		})
		return
	}

	err = h.rbacService.RemovePermissionFromRole(uint(roleID), uint(permissionID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Permission removed successfully",
	})
}

// GetUserRoles 获取用户角色
func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	roles, err := h.rbacService.GetUserRoles(uint(userID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    roles,
	})
}

// GetRolePermissions 获取角色权限
func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	roleIDStr := c.Param("role_id")
	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID",
		})
		return
	}

	permissions, err := h.rbacService.GetRolePermissions(uint(roleID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    permissions,
	})
}

// CheckPermission 检查用户权限
func (h *RBACHandler) CheckPermission(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		Resource string `json:"resource" binding:"required"`
		Action   string `json:"action" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	allowed, err := h.rbacService.CheckPermission(userID.(uint), req.Resource, req.Action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check permission",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"allowed": allowed,
		},
	})
}

// RegisterRoutes 注册RBAC相关路由
func (h *RBACHandler) RegisterRoutes(router *gin.RouterGroup, authService *AuthService) {
	rbac := router.Group("/rbac")
	rbac.Use(AuthMiddleware(authService))
	{
		// 角色管理 - 需要角色管理权限
		roles := rbac.Group("/roles")
		roles.Use(RequirePermission(h.rbacService, "role", "create"))
		{
			roles.POST("", h.CreateRole)
		}
		
		roles.Use(RequirePermission(h.rbacService, "role", "read"))
		{
			roles.GET("/:role_id/permissions", h.GetRolePermissions)
		}

		// 权限管理 - 需要系统管理权限
		permissions := rbac.Group("/permissions")
		permissions.Use(RequirePermission(h.rbacService, "system", "manage"))
		{
			permissions.POST("", h.CreatePermission)
		}

		// 用户角色管理 - 需要用户管理权限
		users := rbac.Group("/users")
		users.Use(RequirePermission(h.rbacService, "user", "update"))
		{
			users.POST("/:user_id/roles", h.AssignRoleToUser)
			users.DELETE("/:user_id/roles/:role_id", h.RemoveRoleFromUser)
			users.GET("/:user_id/roles", h.GetUserRoles)
		}

		// 角色权限管理 - 需要角色管理权限
		rolePerms := rbac.Group("/roles")
		rolePerms.Use(RequirePermission(h.rbacService, "role", "update"))
		{
			rolePerms.POST("/:role_id/permissions", h.AssignPermissionToRole)
			rolePerms.DELETE("/:role_id/permissions/:permission_id", h.RemovePermissionFromRole)
		}

		// 权限检查 - 任何认证用户都可以检查自己的权限
		rbac.POST("/check", h.CheckPermission)
	}
}