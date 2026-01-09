package auth

import (
	"errors"
	"testing"

	"nmp-platform/internal/models"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// mockRoleRepository 模拟角色仓库用于测试
type mockRoleRepository struct {
	roles       map[uint]*models.Role
	permissions map[uint]map[uint]bool // roleID -> permissionID -> exists
	nextID      uint
}

func newMockRoleRepository() *mockRoleRepository {
	return &mockRoleRepository{
		roles:       make(map[uint]*models.Role),
		permissions: make(map[uint]map[uint]bool),
		nextID:      1,
	}
}

func (m *mockRoleRepository) Create(role *models.Role) error {
	if role == nil {
		return errors.New("role cannot be nil")
	}
	
	// 检查角色名是否已存在
	for _, existingRole := range m.roles {
		if existingRole.Name == role.Name {
			return errors.New("role name already exists")
		}
	}
	
	role.ID = m.nextID
	m.nextID++
	m.roles[role.ID] = role
	m.permissions[role.ID] = make(map[uint]bool)
	return nil
}

func (m *mockRoleRepository) GetByID(id uint) (*models.Role, error) {
	if role, exists := m.roles[id]; exists {
		// 加载权限
		var permissions []models.Permission
		for permID := range m.permissions[id] {
			permissions = append(permissions, models.Permission{ID: permID})
		}
		role.Permissions = permissions
		return role, nil
	}
	return nil, errors.New("role not found")
}

func (m *mockRoleRepository) GetByName(name string) (*models.Role, error) {
	for _, role := range m.roles {
		if role.Name == name {
			return m.GetByID(role.ID)
		}
	}
	return nil, errors.New("role not found")
}

func (m *mockRoleRepository) Update(role *models.Role) error {
	if role == nil {
		return errors.New("role cannot be nil")
	}
	
	if _, exists := m.roles[role.ID]; !exists {
		return errors.New("role not found")
	}
	
	m.roles[role.ID] = role
	return nil
}

func (m *mockRoleRepository) Delete(id uint) error {
	if role, exists := m.roles[id]; exists {
		if role.IsSystem {
			return errors.New("cannot delete system role")
		}
		delete(m.roles, id)
		delete(m.permissions, id)
		return nil
	}
	return errors.New("role not found")
}

func (m *mockRoleRepository) List(offset, limit int) ([]*models.Role, int64, error) {
	roles := make([]*models.Role, 0, len(m.roles))
	for _, role := range m.roles {
		roles = append(roles, role)
	}
	
	total := int64(len(roles))
	start := offset
	end := offset + limit
	
	if start > len(roles) {
		return []*models.Role{}, total, nil
	}
	if end > len(roles) {
		end = len(roles)
	}
	
	return roles[start:end], total, nil
}

func (m *mockRoleRepository) AddPermission(roleID, permissionID uint) error {
	if _, exists := m.roles[roleID]; !exists {
		return errors.New("role not found")
	}
	
	if m.permissions[roleID] == nil {
		m.permissions[roleID] = make(map[uint]bool)
	}
	
	if m.permissions[roleID][permissionID] {
		return errors.New("permission already assigned to role")
	}
	
	m.permissions[roleID][permissionID] = true
	return nil
}

func (m *mockRoleRepository) RemovePermission(roleID, permissionID uint) error {
	if _, exists := m.roles[roleID]; !exists {
		return errors.New("role not found")
	}
	
	if m.permissions[roleID] != nil {
		delete(m.permissions[roleID], permissionID)
	}
	
	return nil
}

func (m *mockRoleRepository) GetPermissions(roleID uint) ([]*models.Permission, error) {
	if _, exists := m.roles[roleID]; !exists {
		return nil, errors.New("role not found")
	}
	
	var permissions []*models.Permission
	for permID := range m.permissions[roleID] {
		permissions = append(permissions, &models.Permission{ID: permID})
	}
	
	return permissions, nil
}

func (m *mockRoleRepository) Clear() {
	m.roles = make(map[uint]*models.Role)
	m.permissions = make(map[uint]map[uint]bool)
	m.nextID = 1
}

// mockPermissionRepository 模拟权限仓库用于测试
type mockPermissionRepository struct {
	permissions map[uint]*models.Permission
	nextID      uint
}

func newMockPermissionRepository() *mockPermissionRepository {
	return &mockPermissionRepository{
		permissions: make(map[uint]*models.Permission),
		nextID:      1,
	}
}

func (m *mockPermissionRepository) Create(permission *models.Permission) error {
	if permission == nil {
		return errors.New("permission cannot be nil")
	}
	
	// 检查资源和操作组合是否已存在
	for _, existingPerm := range m.permissions {
		if existingPerm.Resource == permission.Resource && 
		   existingPerm.Action == permission.Action && 
		   existingPerm.Scope == permission.Scope {
			return errors.New("permission with same resource, action and scope already exists")
		}
	}
	
	permission.ID = m.nextID
	m.nextID++
	m.permissions[permission.ID] = permission
	return nil
}

func (m *mockPermissionRepository) GetByID(id uint) (*models.Permission, error) {
	if permission, exists := m.permissions[id]; exists {
		return permission, nil
	}
	return nil, errors.New("permission not found")
}

func (m *mockPermissionRepository) GetByResourceAction(resource, action string) (*models.Permission, error) {
	for _, permission := range m.permissions {
		if permission.Resource == resource && permission.Action == action {
			return permission, nil
		}
	}
	return nil, errors.New("permission not found")
}

func (m *mockPermissionRepository) Update(permission *models.Permission) error {
	if permission == nil {
		return errors.New("permission cannot be nil")
	}
	
	if _, exists := m.permissions[permission.ID]; !exists {
		return errors.New("permission not found")
	}
	
	m.permissions[permission.ID] = permission
	return nil
}

func (m *mockPermissionRepository) Delete(id uint) error {
	if _, exists := m.permissions[id]; exists {
		delete(m.permissions, id)
		return nil
	}
	return errors.New("permission not found")
}

func (m *mockPermissionRepository) List(offset, limit int) ([]*models.Permission, int64, error) {
	permissions := make([]*models.Permission, 0, len(m.permissions))
	for _, permission := range m.permissions {
		permissions = append(permissions, permission)
	}
	
	total := int64(len(permissions))
	start := offset
	end := offset + limit
	
	if start > len(permissions) {
		return []*models.Permission{}, total, nil
	}
	if end > len(permissions) {
		end = len(permissions)
	}
	
	return permissions[start:end], total, nil
}

func (m *mockPermissionRepository) ListByResource(resource string) ([]*models.Permission, error) {
	var permissions []*models.Permission
	for _, permission := range m.permissions {
		if permission.Resource == resource {
			permissions = append(permissions, permission)
		}
	}
	return permissions, nil
}

func (m *mockPermissionRepository) Clear() {
	m.permissions = make(map[uint]*models.Permission)
	m.nextID = 1
}

// TestRolePermissionConsistency 测试角色权限一致性属性
// Feature: network-monitoring-platform, Property 2: 角色权限一致性
// 对于任何用户角色分配，系统应该根据角色定义的权限准确控制用户对资源的访问权限
// **验证需求: 1.3, 1.4**
func TestRolePermissionConsistency(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 创建模拟仓库
	userRepo := newMockUserRepository()
	roleRepo := newMockRoleRepository()
	permRepo := newMockPermissionRepository()

	properties.Property("user with role should have all permissions assigned to that role", 
		prop.ForAll(
			func(username, password, roleName, resource, action string) bool {
				// 清理数据
				userRepo.Clear()
				roleRepo.Clear()
				permRepo.Clear()
				
				// 创建权限
				permission := &models.Permission{
					Resource: resource,
					Action:   action,
					Scope:    "all",
				}
				if err := permRepo.Create(permission); err != nil {
					return false
				}
				
				// 创建角色
				role := &models.Role{
					Name:        roleName,
					DisplayName: roleName,
					Description: "Test role",
					IsSystem:    false,
				}
				if err := roleRepo.Create(role); err != nil {
					return false
				}
				
				// 为角色分配权限
				if err := roleRepo.AddPermission(role.ID, permission.ID); err != nil {
					return false
				}
				
				// 创建用户
				user := &models.User{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					Status:   models.UserStatusActive,
					Roles:    []models.Role{*role},
				}
				if err := userRepo.Create(user); err != nil {
					return false
				}
				
				// 检查用户是否有该权限
				// 模拟权限检查逻辑
				userRoles, err := userRepo.GetByID(user.ID)
				if err != nil {
					return false
				}
				
				hasPermission := false
				for _, userRole := range userRoles.Roles {
					rolePermissions, err := roleRepo.GetPermissions(userRole.ID)
					if err != nil {
						continue
					}
					
					for _, rolePerm := range rolePermissions {
						if rolePerm.ID == permission.ID {
							hasPermission = true
							break
						}
					}
					
					if hasPermission {
						break
					}
				}
				
				return hasPermission
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
		))

	properties.Property("user without role should not have any role-specific permissions", 
		prop.ForAll(
			func(username, password, roleName, resource, action string) bool {
				// 清理数据
				userRepo.Clear()
				roleRepo.Clear()
				permRepo.Clear()
				
				// 创建权限
				permission := &models.Permission{
					Resource: resource,
					Action:   action,
					Scope:    "all",
				}
				if err := permRepo.Create(permission); err != nil {
					return false
				}
				
				// 创建角色并分配权限
				role := &models.Role{
					Name:        roleName,
					DisplayName: roleName,
					Description: "Test role",
					IsSystem:    false,
				}
				if err := roleRepo.Create(role); err != nil {
					return false
				}
				
				if err := roleRepo.AddPermission(role.ID, permission.ID); err != nil {
					return false
				}
				
				// 创建用户但不分配角色
				user := &models.User{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					Status:   models.UserStatusActive,
					Roles:    []models.Role{}, // 没有角色
				}
				if err := userRepo.Create(user); err != nil {
					return false
				}
				
				// 检查用户不应该有该权限
				userRoles, err := userRepo.GetByID(user.ID)
				if err != nil {
					return false
				}
				
				// 用户没有角色，所以不应该有任何权限
				return len(userRoles.Roles) == 0
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
		))

	properties.Property("removing permission from role should remove access for users with that role", 
		prop.ForAll(
			func(username, password, roleName, resource, action string) bool {
				// 清理数据
				userRepo.Clear()
				roleRepo.Clear()
				permRepo.Clear()
				
				// 创建权限
				permission := &models.Permission{
					Resource: resource,
					Action:   action,
					Scope:    "all",
				}
				if err := permRepo.Create(permission); err != nil {
					return false
				}
				
				// 创建角色并分配权限
				role := &models.Role{
					Name:        roleName,
					DisplayName: roleName,
					Description: "Test role",
					IsSystem:    false,
				}
				if err := roleRepo.Create(role); err != nil {
					return false
				}
				
				if err := roleRepo.AddPermission(role.ID, permission.ID); err != nil {
					return false
				}
				
				// 创建用户并分配角色
				user := &models.User{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					Status:   models.UserStatusActive,
					Roles:    []models.Role{*role},
				}
				if err := userRepo.Create(user); err != nil {
					return false
				}
				
				// 从角色移除权限
				if err := roleRepo.RemovePermission(role.ID, permission.ID); err != nil {
					return false
				}
				
				// 检查用户不再有该权限
				rolePermissions, err := roleRepo.GetPermissions(role.ID)
				if err != nil {
					return false
				}
				
				// 角色不应该再有该权限
				for _, rolePerm := range rolePermissions {
					if rolePerm.ID == permission.ID {
						return false // 权限仍然存在，测试失败
					}
				}
				
				return true
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
		))

	properties.Property("removing role from user should remove all role permissions for that user", 
		prop.ForAll(
			func(username, password, roleName, resource, action string) bool {
				// 清理数据
				userRepo.Clear()
				roleRepo.Clear()
				permRepo.Clear()
				
				// 创建权限
				permission := &models.Permission{
					Resource: resource,
					Action:   action,
					Scope:    "all",
				}
				if err := permRepo.Create(permission); err != nil {
					return false
				}
				
				// 创建角色并分配权限
				role := &models.Role{
					Name:        roleName,
					DisplayName: roleName,
					Description: "Test role",
					IsSystem:    false,
				}
				if err := roleRepo.Create(role); err != nil {
					return false
				}
				
				if err := roleRepo.AddPermission(role.ID, permission.ID); err != nil {
					return false
				}
				
				// 创建用户并分配角色
				user := &models.User{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					Status:   models.UserStatusActive,
					Roles:    []models.Role{*role},
				}
				if err := userRepo.Create(user); err != nil {
					return false
				}
				
				// 从用户移除角色
				user.Roles = []models.Role{}
				if err := userRepo.Update(user); err != nil {
					return false
				}
				
				// 检查用户不再有任何角色
				updatedUser, err := userRepo.GetByID(user.ID)
				if err != nil {
					return false
				}
				
				return len(updatedUser.Roles) == 0
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
		))
}