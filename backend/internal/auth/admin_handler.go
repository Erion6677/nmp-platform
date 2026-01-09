package auth

import (
	"strconv"
	"strings"

	"nmp-platform/internal/api"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理员API处理器
type AdminHandler struct {
	authService     *AuthService
	rbacService     *RBACService
	userRepo        repository.UserRepository
	roleRepo        repository.RoleRepository
	permissionRepo  repository.PermissionRepository
	passwordService *service.PasswordService
}

// NewAdminHandler 创建新的管理员处理器
func NewAdminHandler(authService *AuthService, rbacService *RBACService) *AdminHandler {
	return &AdminHandler{
		authService:     authService,
		rbacService:     rbacService,
		passwordService: service.GetPasswordService(),
	}
}

// NewAdminHandlerWithRepos 创建带仓库依赖的管理员处理器
func NewAdminHandlerWithRepos(
	authService *AuthService,
	rbacService *RBACService,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
	passwordService *service.PasswordService,
) *AdminHandler {
	return &AdminHandler{
		authService:     authService,
		rbacService:     rbacService,
		userRepo:        userRepo,
		roleRepo:        roleRepo,
		permissionRepo:  permissionRepo,
		passwordService: passwordService,
	}
}

// SetRepositories 设置仓库依赖（用于延迟注入）
func (h *AdminHandler) SetRepositories(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	permissionRepo repository.PermissionRepository,
) {
	h.userRepo = userRepo
	h.roleRepo = roleRepo
	h.permissionRepo = permissionRepo
}

// ========== 用户管理 ==========

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required,min=3,max=50"`
	Password string   `json:"password" binding:"required,min=6,max=100"`
	Email    string   `json:"email" binding:"omitempty,email,max=100"`
	FullName string   `json:"full_name" binding:"max=100"`
	RoleIDs  []uint   `json:"role_ids"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    string            `json:"email" binding:"omitempty,email,max=100"`
	FullName string            `json:"full_name" binding:"max=100"`
	Status   models.UserStatus `json:"status" binding:"omitempty,oneof=active inactive blocked"`
	Password string            `json:"password" binding:"omitempty,min=6,max=100"`
	RoleIDs  []uint            `json:"role_ids"`
}

// ListUsers 获取用户列表
func (h *AdminHandler) ListUsers(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	// 检查是否有仓库依赖
	if h.userRepo == nil {
		api.InternalError(c, "用户仓库未初始化")
		return
	}

	// 从数据库获取用户列表
	users, total, err := h.userRepo.List(page, size, search)
	if err != nil {
		api.InternalError(c, "获取用户列表失败")
		return
	}

	// 转换为响应格式
	items := make([]gin.H, len(users))
	for i, user := range users {
		roleNames := make([]string, len(user.Roles))
		for j, role := range user.Roles {
			roleNames[j] = role.Name
		}
		items[i] = gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"full_name":  user.FullName,
			"status":     user.Status,
			"roles":      roleNames,
			"last_login": user.LastLogin,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		}
	}

	api.SuccessPaginated(c, items, total, page, size)
}

// CreateUser 创建用户
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 检查是否有仓库依赖
	if h.userRepo == nil {
		api.InternalError(c, "用户仓库未初始化")
		return
	}

	// 密码哈希
	hashedPassword, err := h.passwordService.Hash(req.Password)
	if err != nil {
		api.InternalError(c, "密码处理失败")
		return
	}

	// 创建用户
	user := &models.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		FullName: req.FullName,
		Status:   models.UserStatusActive,
	}

	if err := h.userRepo.Create(user); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.Conflict(c, err.Error())
			return
		}
		api.InternalError(c, "创建用户失败: "+err.Error())
		return
	}

	// 分配角色
	if len(req.RoleIDs) > 0 {
		if err := h.userRepo.AssignRoles(user.ID, req.RoleIDs); err != nil {
			// 用户已创建，角色分配失败只记录警告
			c.Set("warning", "用户创建成功，但角色分配失败: "+err.Error())
		}
	}

	// 重新获取用户信息（包含角色）
	user, _ = h.userRepo.GetByID(user.ID)

	api.SuccessWithMessage(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"full_name":  user.FullName,
		"status":     user.Status,
		"created_at": user.CreatedAt,
	}, "用户创建成功")
}

// GetUser 获取单个用户
func (h *AdminHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的用户ID")
		return
	}

	if h.userRepo == nil {
		api.InternalError(c, "用户仓库未初始化")
		return
	}

	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "用户不存在")
			return
		}
		api.InternalError(c, "获取用户失败")
		return
	}

	roleNames := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roleNames[i] = role.Name
	}

	api.Success(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"full_name":  user.FullName,
		"status":     user.Status,
		"roles":      roleNames,
		"last_login": user.LastLogin,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}

// UpdateUser 更新用户
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的用户ID")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if h.userRepo == nil {
		api.InternalError(c, "用户仓库未初始化")
		return
	}

	// 获取现有用户
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "用户不存在")
			return
		}
		api.InternalError(c, "获取用户失败")
		return
	}

	// 更新字段
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Status != "" {
		user.Status = req.Status
	}
	if req.Password != "" {
		hashedPassword, err := h.passwordService.Hash(req.Password)
		if err != nil {
			api.InternalError(c, "密码处理失败")
			return
		}
		user.Password = hashedPassword
	}

	// 保存更新
	if err := h.userRepo.Update(user); err != nil {
		api.InternalError(c, "更新用户失败: "+err.Error())
		return
	}

	// 更新角色
	if req.RoleIDs != nil {
		if err := h.userRepo.AssignRoles(user.ID, req.RoleIDs); err != nil {
			c.Set("warning", "用户更新成功，但角色分配失败: "+err.Error())
		}
	}

	api.SuccessWithMessage(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"full_name":  user.FullName,
		"status":     user.Status,
		"updated_at": user.UpdatedAt,
	}, "用户更新成功")
}

// DeleteUser 删除用户
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的用户ID")
		return
	}

	if h.userRepo == nil {
		api.InternalError(c, "用户仓库未初始化")
		return
	}

	// 检查用户是否存在
	user, err := h.userRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "用户不存在")
			return
		}
		api.InternalError(c, "获取用户失败")
		return
	}

	// 防止删除 admin 用户
	if user.Username == "admin" {
		api.Forbidden(c, "不能删除系统管理员账户")
		return
	}

	// 删除用户
	if err := h.userRepo.Delete(uint(id)); err != nil {
		api.InternalError(c, "删除用户失败: "+err.Error())
		return
	}

	api.SuccessWithMessage(c, nil, "用户删除成功")
}


// ========== 角色管理 ==========

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name          string `json:"name" binding:"required,min=2,max=50"`
	DisplayName   string `json:"display_name" binding:"max=100"`
	Description   string `json:"description" binding:"max=255"`
	PermissionIDs []uint `json:"permission_ids"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	DisplayName   string `json:"display_name" binding:"max=100"`
	Description   string `json:"description" binding:"max=255"`
	PermissionIDs []uint `json:"permission_ids"`
}

// ListRoles 获取角色列表
func (h *AdminHandler) ListRoles(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	// 检查是否有仓库依赖
	if h.roleRepo == nil {
		api.InternalError(c, "角色仓库未初始化")
		return
	}

	// 从数据库获取角色列表
	roles, total, err := h.roleRepo.List(page, size, search)
	if err != nil {
		api.InternalError(c, "获取角色列表失败")
		return
	}

	// 转换为响应格式
	items := make([]gin.H, len(roles))
	for i, role := range roles {
		// 获取使用该角色的用户数量
		userCount, _ := h.roleRepo.GetUserCount(role.ID)
		
		permissionIDs := make([]uint, len(role.Permissions))
		for j, perm := range role.Permissions {
			permissionIDs[j] = perm.ID
		}
		
		items[i] = gin.H{
			"id":             role.ID,
			"name":           role.Name,
			"display_name":   role.DisplayName,
			"description":    role.Description,
			"is_system":      role.IsSystem,
			"user_count":     userCount,
			"permission_ids": permissionIDs,
			"created_at":     role.CreatedAt,
			"updated_at":     role.UpdatedAt,
		}
	}

	api.SuccessPaginated(c, items, total, page, size)
}

// CreateRole 创建角色
func (h *AdminHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if h.roleRepo == nil {
		api.InternalError(c, "角色仓库未初始化")
		return
	}

	// 创建角色
	role := &models.Role{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := h.roleRepo.Create(role); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			api.Conflict(c, err.Error())
			return
		}
		api.InternalError(c, "创建角色失败: "+err.Error())
		return
	}

	// 分配权限
	if len(req.PermissionIDs) > 0 {
		if err := h.roleRepo.AssignPermissions(role.ID, req.PermissionIDs); err != nil {
			c.Set("warning", "角色创建成功，但权限分配失败: "+err.Error())
		}
	}

	api.SuccessWithMessage(c, gin.H{
		"id":           role.ID,
		"name":         role.Name,
		"display_name": role.DisplayName,
		"description":  role.Description,
		"created_at":   role.CreatedAt,
	}, "角色创建成功")
}

// GetRole 获取单个角色
func (h *AdminHandler) GetRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的角色ID")
		return
	}

	if h.roleRepo == nil {
		api.InternalError(c, "角色仓库未初始化")
		return
	}

	role, err := h.roleRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "角色不存在")
			return
		}
		api.InternalError(c, "获取角色失败")
		return
	}

	userCount, _ := h.roleRepo.GetUserCount(role.ID)
	permissionIDs := make([]uint, len(role.Permissions))
	for i, perm := range role.Permissions {
		permissionIDs[i] = perm.ID
	}

	api.Success(c, gin.H{
		"id":             role.ID,
		"name":           role.Name,
		"display_name":   role.DisplayName,
		"description":    role.Description,
		"is_system":      role.IsSystem,
		"user_count":     userCount,
		"permission_ids": permissionIDs,
		"permissions":    role.Permissions,
		"created_at":     role.CreatedAt,
		"updated_at":     role.UpdatedAt,
	})
}

// UpdateRole 更新角色
func (h *AdminHandler) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的角色ID")
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if h.roleRepo == nil {
		api.InternalError(c, "角色仓库未初始化")
		return
	}

	// 获取现有角色
	role, err := h.roleRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "角色不存在")
			return
		}
		api.InternalError(c, "获取角色失败")
		return
	}

	// 系统角色不允许修改名称
	if role.IsSystem {
		// 只允许修改显示名称和描述
	}

	// 更新字段
	if req.DisplayName != "" {
		role.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		role.Description = req.Description
	}

	// 保存更新
	if err := h.roleRepo.Update(role); err != nil {
		api.InternalError(c, "更新角色失败: "+err.Error())
		return
	}

	// 更新权限
	if req.PermissionIDs != nil {
		if err := h.roleRepo.AssignPermissions(role.ID, req.PermissionIDs); err != nil {
			c.Set("warning", "角色更新成功，但权限分配失败: "+err.Error())
		}
	}

	api.SuccessWithMessage(c, gin.H{
		"id":           role.ID,
		"name":         role.Name,
		"display_name": role.DisplayName,
		"description":  role.Description,
		"updated_at":   role.UpdatedAt,
	}, "角色更新成功")
}

// DeleteRole 删除角色
func (h *AdminHandler) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的角色ID")
		return
	}

	if h.roleRepo == nil {
		api.InternalError(c, "角色仓库未初始化")
		return
	}

	// 检查角色是否存在
	role, err := h.roleRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "角色不存在")
			return
		}
		api.InternalError(c, "获取角色失败")
		return
	}

	// 系统角色不能删除
	if role.IsSystem {
		api.Forbidden(c, "不能删除系统角色")
		return
	}

	// 检查是否有用户使用该角色
	userCount, _ := h.roleRepo.GetUserCount(role.ID)
	if userCount > 0 {
		api.Conflict(c, "该角色正在被使用，无法删除")
		return
	}

	// 删除角色
	if err := h.roleRepo.Delete(uint(id)); err != nil {
		api.InternalError(c, "删除角色失败: "+err.Error())
		return
	}

	api.SuccessWithMessage(c, nil, "角色删除成功")
}

// ========== 权限管理 ==========

// ListPermissions 获取权限列表
func (h *AdminHandler) ListPermissions(c *gin.Context) {
	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "100"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 1000 {
		size = 100
	}

	// 检查是否有仓库依赖
	if h.permissionRepo == nil {
		api.InternalError(c, "权限仓库未初始化")
		return
	}

	// 从数据库获取权限列表
	permissions, total, err := h.permissionRepo.List(page, size, search)
	if err != nil {
		api.InternalError(c, "获取权限列表失败")
		return
	}

	// 转换为响应格式，按资源分组
	items := make([]gin.H, len(permissions))
	for i, perm := range permissions {
		roleCount, _ := h.permissionRepo.GetRoleCount(perm.ID)
		items[i] = gin.H{
			"id":          perm.ID,
			"resource":    perm.Resource,
			"action":      perm.Action,
			"scope":       perm.Scope,
			"description": perm.Description,
			"role_count":  roleCount,
			"created_at":  perm.CreatedAt,
		}
	}

	api.SuccessPaginated(c, items, total, page, size)
}

// GetPermission 获取单个权限
func (h *AdminHandler) GetPermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		api.BadRequest(c, "无效的权限ID")
		return
	}

	if h.permissionRepo == nil {
		api.InternalError(c, "权限仓库未初始化")
		return
	}

	perm, err := h.permissionRepo.GetByID(uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.NotFound(c, "权限不存在")
			return
		}
		api.InternalError(c, "获取权限失败")
		return
	}

	roleCount, _ := h.permissionRepo.GetRoleCount(perm.ID)

	api.Success(c, gin.H{
		"id":          perm.ID,
		"resource":    perm.Resource,
		"action":      perm.Action,
		"scope":       perm.Scope,
		"description": perm.Description,
		"role_count":  roleCount,
		"created_at":  perm.CreatedAt,
		"updated_at":  perm.UpdatedAt,
	})
}

// RegisterRoutes 注册管理员API路由
func (h *AdminHandler) RegisterRoutes(router *gin.RouterGroup, authService *AuthService) {
	admin := router.Group("/admin")
	admin.Use(AuthMiddleware(authService))
	{
		// 用户管理
		admin.GET("/users", h.ListUsers)
		admin.POST("/users", h.CreateUser)
		admin.GET("/users/:id", h.GetUser)
		admin.PUT("/users/:id", h.UpdateUser)
		admin.DELETE("/users/:id", h.DeleteUser)
		
		// 角色管理
		admin.GET("/roles", h.ListRoles)
		admin.POST("/roles", h.CreateRole)
		admin.GET("/roles/:id", h.GetRole)
		admin.PUT("/roles/:id", h.UpdateRole)
		admin.DELETE("/roles/:id", h.DeleteRole)
		
		// 权限管理
		admin.GET("/permissions", h.ListPermissions)
		admin.GET("/permissions/:id", h.GetPermission)
	}
}
