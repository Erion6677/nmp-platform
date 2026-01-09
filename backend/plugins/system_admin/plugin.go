package system_admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"nmp-platform/internal/auth"
	"nmp-platform/internal/plugin"
	"nmp-platform/internal/repository"
)

// SystemAdminPlugin 系统管理插件
type SystemAdminPlugin struct {
	db          *gorm.DB
	authService *auth.AuthService
	userRepo    repository.UserRepository
	roleRepo    repository.RoleRepository
	permRepo    repository.PermissionRepository
	userService *UserService
	roleService *RoleService
	permService *PermissionService
}

// NewSystemAdminPlugin 创建系统管理插件实例
func NewSystemAdminPlugin(db *gorm.DB, authService *auth.AuthService) *SystemAdminPlugin {
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	return &SystemAdminPlugin{
		db:          db,
		authService: authService,
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		permRepo:    permRepo,
		userService: NewUserService(userRepo, authService),
		roleService: NewRoleService(roleRepo, permRepo),
		permService: NewPermissionService(permRepo),
	}
}

// Name 返回插件名称
func (p *SystemAdminPlugin) Name() string {
	return "system_admin"
}

// Version 返回插件版本
func (p *SystemAdminPlugin) Version() string {
	return "1.0.0"
}

// Description 返回插件描述
func (p *SystemAdminPlugin) Description() string {
	return "系统管理插件，提供用户管理、角色权限管理等功能"
}

// Dependencies 返回插件依赖
func (p *SystemAdminPlugin) Dependencies() []string {
	return []string{}
}

// Initialize 初始化插件
func (p *SystemAdminPlugin) Initialize(ctx context.Context, config interface{}) error {
	// 插件初始化逻辑
	return nil
}

// Start 启动插件
func (p *SystemAdminPlugin) Start(ctx context.Context) error {
	// 插件启动逻辑
	return nil
}

// Stop 停止插件
func (p *SystemAdminPlugin) Stop(ctx context.Context) error {
	// 插件停止逻辑
	return nil
}

// Health 健康检查
func (p *SystemAdminPlugin) Health() error {
	// 检查数据库连接
	if p.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	sqlDB, err := p.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}
	
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %v", err)
	}
	
	return nil
}

// GetRoutes 返回插件路由
func (p *SystemAdminPlugin) GetRoutes() []plugin.Route {
	return []plugin.Route{
		// 用户管理路由
		{
			Method:      "GET",
			Path:        "/api/admin/users",
			Handler:     p.listUsers,
			Permission:  "user:read",
			Description: "获取用户列表",
		},
		{
			Method:      "POST",
			Path:        "/api/admin/users",
			Handler:     p.createUser,
			Permission:  "user:create",
			Description: "创建用户",
		},
		{
			Method:      "GET",
			Path:        "/api/admin/users/:id",
			Handler:     p.getUser,
			Permission:  "user:read",
			Description: "获取用户详情",
		},
		{
			Method:      "PUT",
			Path:        "/api/admin/users/:id",
			Handler:     p.updateUser,
			Permission:  "user:update",
			Description: "更新用户",
		},
		{
			Method:      "DELETE",
			Path:        "/api/admin/users/:id",
			Handler:     p.deleteUser,
			Permission:  "user:delete",
			Description: "删除用户",
		},
		{
			Method:      "PUT",
			Path:        "/api/admin/users/:id/status",
			Handler:     p.updateUserStatus,
			Permission:  "user:update",
			Description: "更新用户状态",
		},
		{
			Method:      "PUT",
			Path:        "/api/admin/users/:id/roles",
			Handler:     p.updateUserRoles,
			Permission:  "user:update",
			Description: "更新用户角色",
		},

		// 角色管理路由
		{
			Method:      "GET",
			Path:        "/api/admin/roles",
			Handler:     p.listRoles,
			Permission:  "role:read",
			Description: "获取角色列表",
		},
		{
			Method:      "POST",
			Path:        "/api/admin/roles",
			Handler:     p.createRole,
			Permission:  "role:create",
			Description: "创建角色",
		},
		{
			Method:      "GET",
			Path:        "/api/admin/roles/:id",
			Handler:     p.getRole,
			Permission:  "role:read",
			Description: "获取角色详情",
		},
		{
			Method:      "PUT",
			Path:        "/api/admin/roles/:id",
			Handler:     p.updateRole,
			Permission:  "role:update",
			Description: "更新角色",
		},
		{
			Method:      "DELETE",
			Path:        "/api/admin/roles/:id",
			Handler:     p.deleteRole,
			Permission:  "role:delete",
			Description: "删除角色",
		},
		{
			Method:      "PUT",
			Path:        "/api/admin/roles/:id/permissions",
			Handler:     p.updateRolePermissions,
			Permission:  "role:update",
			Description: "更新角色权限",
		},

		// 权限管理路由
		{
			Method:      "GET",
			Path:        "/api/admin/permissions",
			Handler:     p.listPermissions,
			Permission:  "permission:read",
			Description: "获取权限列表",
		},
		{
			Method:      "POST",
			Path:        "/api/admin/permissions",
			Handler:     p.createPermission,
			Permission:  "permission:create",
			Description: "创建权限",
		},
		{
			Method:      "GET",
			Path:        "/api/admin/permissions/:id",
			Handler:     p.getPermission,
			Permission:  "permission:read",
			Description: "获取权限详情",
		},
		{
			Method:      "PUT",
			Path:        "/api/admin/permissions/:id",
			Handler:     p.updatePermission,
			Permission:  "permission:update",
			Description: "更新权限",
		},
		{
			Method:      "DELETE",
			Path:        "/api/admin/permissions/:id",
			Handler:     p.deletePermission,
			Permission:  "permission:delete",
			Description: "删除权限",
		},
	}
}

// GetMenus 返回插件菜单
func (p *SystemAdminPlugin) GetMenus() []plugin.MenuItem {
	return []plugin.MenuItem{
		{
			Key:        "system_admin",
			Label:      "系统管理",
			Icon:       "mdi:cog",
			Path:       "/admin",
			Permission: "admin:access",
			Order:      100,
			Visible:    true,
			Children: []plugin.MenuItem{
				{
					Key:        "user_management",
					Label:      "用户管理",
					Icon:       "mdi:account-multiple",
					Path:       "/admin/users",
					Permission: "user:read",
					Order:      1,
					Visible:    true,
				},
				{
					Key:        "role_management",
					Label:      "角色管理",
					Icon:       "mdi:account-group",
					Path:       "/admin/roles",
					Permission: "role:read",
					Order:      2,
					Visible:    true,
				},
				{
					Key:        "permission_management",
					Label:      "权限管理",
					Icon:       "mdi:shield-account",
					Path:       "/admin/permissions",
					Permission: "permission:read",
					Order:      3,
					Visible:    true,
				},
			},
		},
	}
}

// GetPermissions 返回插件权限
func (p *SystemAdminPlugin) GetPermissions() []plugin.Permission {
	return []plugin.Permission{
		// 管理员访问权限
		{
			Resource:    "admin",
			Action:      "access",
			Scope:       "global",
			Description: "访问系统管理功能",
		},

		// 用户管理权限
		{
			Resource:    "user",
			Action:      "create",
			Scope:       "global",
			Description: "创建用户",
		},
		{
			Resource:    "user",
			Action:      "read",
			Scope:       "global",
			Description: "查看用户",
		},
		{
			Resource:    "user",
			Action:      "update",
			Scope:       "global",
			Description: "更新用户",
		},
		{
			Resource:    "user",
			Action:      "delete",
			Scope:       "global",
			Description: "删除用户",
		},

		// 角色管理权限
		{
			Resource:    "role",
			Action:      "create",
			Scope:       "global",
			Description: "创建角色",
		},
		{
			Resource:    "role",
			Action:      "read",
			Scope:       "global",
			Description: "查看角色",
		},
		{
			Resource:    "role",
			Action:      "update",
			Scope:       "global",
			Description: "更新角色",
		},
		{
			Resource:    "role",
			Action:      "delete",
			Scope:       "global",
			Description: "删除角色",
		},

		// 权限管理权限
		{
			Resource:    "permission",
			Action:      "create",
			Scope:       "global",
			Description: "创建权限",
		},
		{
			Resource:    "permission",
			Action:      "read",
			Scope:       "global",
			Description: "查看权限",
		},
		{
			Resource:    "permission",
			Action:      "update",
			Scope:       "global",
			Description: "更新权限",
		},
		{
			Resource:    "permission",
			Action:      "delete",
			Scope:       "global",
			Description: "删除权限",
		},
	}
}

// GetConfigSchema 返回插件配置模式
func (p *SystemAdminPlugin) GetConfigSchema() interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"max_users": map[string]interface{}{
				"type":        "integer",
				"description": "最大用户数量",
				"default":     1000,
			},
			"password_policy": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"min_length": map[string]interface{}{
						"type":        "integer",
						"description": "密码最小长度",
						"default":     8,
					},
					"require_uppercase": map[string]interface{}{
						"type":        "boolean",
						"description": "是否需要大写字母",
						"default":     true,
					},
					"require_lowercase": map[string]interface{}{
						"type":        "boolean",
						"description": "是否需要小写字母",
						"default":     true,
					},
					"require_numbers": map[string]interface{}{
						"type":        "boolean",
						"description": "是否需要数字",
						"default":     true,
					},
					"require_symbols": map[string]interface{}{
						"type":        "boolean",
						"description": "是否需要特殊字符",
						"default":     false,
					},
				},
			},
		},
	}
}

// 响应结构体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 成功响应
func (p *SystemAdminPlugin) success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// 错误响应
func (p *SystemAdminPlugin) error(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}