package auth

import (
	"fmt"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// RBACService RBAC权限服务
type RBACService struct {
	enforcer   *casbin.Enforcer
	userRepo   repository.UserRepository
	roleRepo   repository.RoleRepository
	permRepo   repository.PermissionRepository
}

// NewRBACService 创建新的RBAC服务
func NewRBACService(db *gorm.DB, userRepo repository.UserRepository, roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) (*RBACService, error) {
	// 创建Casbin适配器
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin adapter: %w", err)
	}

	// 创建Casbin执行器
	enforcer, err := casbin.NewEnforcer("configs/rbac_model.conf", adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	// 启用日志
	enforcer.EnableLog(true)

	service := &RBACService{
		enforcer: enforcer,
		userRepo: userRepo,
		roleRepo: roleRepo,
		permRepo: permRepo,
	}

	// 初始化默认权限
	if err := service.initializeDefaultPermissions(); err != nil {
		return nil, fmt.Errorf("failed to initialize default permissions: %w", err)
	}

	return service, nil
}

// CheckPermission 检查用户权限
func (s *RBACService) CheckPermission(userID uint, resource, action string) (bool, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return false, err
	}

	// 检查用户的每个角色是否有权限
	for _, role := range user.Roles {
		allowed, err := s.enforcer.Enforce(role.Name, resource, action)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}

	return false, nil
}

// AssignRoleToUser 为用户分配角色
func (s *RBACService) AssignRoleToUser(userID, roleID uint) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		return err
	}

	// 检查用户是否已有该角色
	for _, existingRole := range user.Roles {
		if existingRole.ID == roleID {
			return fmt.Errorf("user already has role %s", role.Name)
		}
	}

	// 添加角色到用户
	user.Roles = append(user.Roles, *role)
	return s.userRepo.Update(user)
}

// RemoveRoleFromUser 从用户移除角色
func (s *RBACService) RemoveRoleFromUser(userID, roleID uint) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	// 查找并移除角色
	for i, role := range user.Roles {
		if role.ID == roleID {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			return s.userRepo.Update(user)
		}
	}

	return fmt.Errorf("user does not have the specified role")
}

// AssignPermissionToRole 为角色分配权限
func (s *RBACService) AssignPermissionToRole(roleID, permissionID uint) error {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		return err
	}

	permission, err := s.permRepo.GetByID(permissionID)
	if err != nil {
		return err
	}

	// 添加权限到角色
	if err := s.roleRepo.AddPermission(roleID, permissionID); err != nil {
		return err
	}

	// 更新Casbin策略
	_, err = s.enforcer.AddPolicy(role.Name, permission.Resource, permission.Action)
	if err != nil {
		return err
	}

	return s.enforcer.SavePolicy()
}

// RemovePermissionFromRole 从角色移除权限
func (s *RBACService) RemovePermissionFromRole(roleID, permissionID uint) error {
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		return err
	}

	permission, err := s.permRepo.GetByID(permissionID)
	if err != nil {
		return err
	}

	// 从角色移除权限
	if err := s.roleRepo.RemovePermission(roleID, permissionID); err != nil {
		return err
	}

	// 更新Casbin策略
	_, err = s.enforcer.RemovePolicy(role.Name, permission.Resource, permission.Action)
	if err != nil {
		return err
	}

	return s.enforcer.SavePolicy()
}

// GetUserRoles 获取用户角色
func (s *RBACService) GetUserRoles(userID uint) ([]*models.Role, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	roles := make([]*models.Role, len(user.Roles))
	for i := range user.Roles {
		roles[i] = &user.Roles[i]
	}

	return roles, nil
}

// GetRolePermissions 获取角色权限
func (s *RBACService) GetRolePermissions(roleID uint) ([]*models.Permission, error) {
	return s.roleRepo.GetPermissions(roleID)
}

// CreateRole 创建角色
func (s *RBACService) CreateRole(name, displayName, description string, isSystem bool) (*models.Role, error) {
	role := &models.Role{
		Name:        name,
		DisplayName: displayName,
		Description: description,
		IsSystem:    isSystem,
	}

	if err := s.roleRepo.Create(role); err != nil {
		return nil, err
	}

	return role, nil
}

// CreatePermission 创建权限
func (s *RBACService) CreatePermission(resource, action, scope, description string) (*models.Permission, error) {
	permission := &models.Permission{
		Resource:    resource,
		Action:      action,
		Scope:       scope,
		Description: description,
	}

	if err := s.permRepo.Create(permission); err != nil {
		return nil, err
	}

	return permission, nil
}

// initializeDefaultPermissions 初始化默认权限和角色
func (s *RBACService) initializeDefaultPermissions() error {
	// 创建默认权限
	defaultPermissions := []struct {
		Resource    string
		Action      string
		Scope       string
		Description string
	}{
		{"user", "create", "all", "创建用户"},
		{"user", "read", "all", "查看用户"},
		{"user", "update", "all", "更新用户"},
		{"user", "delete", "all", "删除用户"},
		{"role", "create", "all", "创建角色"},
		{"role", "read", "all", "查看角色"},
		{"role", "update", "all", "更新角色"},
		{"role", "delete", "all", "删除角色"},
		{"device", "create", "all", "创建设备"},
		{"device", "read", "all", "查看设备"},
		{"device", "update", "all", "更新设备"},
		{"device", "delete", "all", "删除设备"},
		{"monitoring", "read", "all", "查看监控数据"},
		{"monitoring", "manage", "all", "管理监控配置"},
		{"system", "manage", "all", "系统管理"},
	}

	for _, perm := range defaultPermissions {
		// 检查权限是否已存在
		if _, err := s.permRepo.GetByResourceAction(perm.Resource, perm.Action); err != nil {
			// 权限不存在，创建它
			_, err := s.CreatePermission(perm.Resource, perm.Action, perm.Scope, perm.Description)
			if err != nil {
				return fmt.Errorf("failed to create permission %s:%s: %w", perm.Resource, perm.Action, err)
			}
		}
	}

	// 创建默认角色
	defaultRoles := []struct {
		Name        string
		DisplayName string
		Description string
		IsSystem    bool
		Permissions []string // resource:action格式
	}{
		{
			Name:        "admin",
			DisplayName: "系统管理员",
			Description: "拥有所有权限的系统管理员",
			IsSystem:    true,
			Permissions: []string{
				"user:create", "user:read", "user:update", "user:delete",
				"role:create", "role:read", "role:update", "role:delete",
				"device:create", "device:read", "device:update", "device:delete",
				"monitoring:read", "monitoring:manage",
				"system:manage",
			},
		},
		{
			Name:        "operator",
			DisplayName: "操作员",
			Description: "设备管理和监控操作员",
			IsSystem:    true,
			Permissions: []string{
				"device:create", "device:read", "device:update", "device:delete",
				"monitoring:read", "monitoring:manage",
			},
		},
		{
			Name:        "viewer",
			DisplayName: "查看者",
			Description: "只能查看监控数据",
			IsSystem:    true,
			Permissions: []string{
				"device:read",
				"monitoring:read",
			},
		},
	}

	for _, roleData := range defaultRoles {
		// 检查角色是否已存在
		if _, err := s.roleRepo.GetByName(roleData.Name); err != nil {
			// 角色不存在，创建它
			role, err := s.CreateRole(roleData.Name, roleData.DisplayName, roleData.Description, roleData.IsSystem)
			if err != nil {
				return fmt.Errorf("failed to create role %s: %w", roleData.Name, err)
			}

			// 为角色分配权限
			for _, permStr := range roleData.Permissions {
				// 解析权限字符串
				resource, action := parsePermissionString(permStr)
				permission, err := s.permRepo.GetByResourceAction(resource, action)
				if err != nil {
					continue // 跳过不存在的权限
				}

				// 分配权限到角色
				if err := s.AssignPermissionToRole(role.ID, permission.ID); err != nil {
					return fmt.Errorf("failed to assign permission %s to role %s: %w", permStr, roleData.Name, err)
				}
			}
		}
	}

	return nil
}

// parsePermissionString 解析权限字符串
func parsePermissionString(permStr string) (resource, action string) {
	for i, char := range permStr {
		if char == ':' {
			return permStr[:i], permStr[i+1:]
		}
	}
	return permStr, ""
}

// LoadPolicy 重新加载策略
func (s *RBACService) LoadPolicy() error {
	return s.enforcer.LoadPolicy()
}

// SavePolicy 保存策略
func (s *RBACService) SavePolicy() error {
	return s.enforcer.SavePolicy()
}

// GetPermissionByResourceAction 根据资源和操作获取权限
func (s *RBACService) GetPermissionByResourceAction(resource, action string) (*models.Permission, error) {
	return s.permRepo.GetByResourceAction(resource, action)
}

// GetPermissionsByResourcePrefix 根据资源前缀获取权限列表
func (s *RBACService) GetPermissionsByResourcePrefix(resourcePrefix string) ([]*models.Permission, error) {
	return s.permRepo.ListByResourcePrefix(resourcePrefix)
}

// DeletePermissionsByResourcePrefix 根据资源前缀删除权限
func (s *RBACService) DeletePermissionsByResourcePrefix(resourcePrefix string) error {
	// 先获取要删除的权限
	permissions, err := s.permRepo.ListByResourcePrefix(resourcePrefix)
	if err != nil {
		return err
	}

	// 从所有角色中移除这些权限
	for _, permission := range permissions {
		// 获取所有角色
		roles, _, err := s.roleRepo.List(0, 1000, "") // 假设不会超过1000个角色
		if err != nil {
			continue
		}

		for _, role := range roles {
			// 从Casbin中移除策略
			s.enforcer.RemovePolicy(role.Name, permission.Resource, permission.Action)
		}
	}

	// 保存Casbin策略
	if err := s.enforcer.SavePolicy(); err != nil {
		return err
	}

	// 从数据库中删除权限
	return s.permRepo.DeleteByResourcePrefix(resourcePrefix)
}