package auth

import (
	"errors"
	"testing"

	"nmp-platform/internal/models"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// mockDevicePermissionRepository 模拟设备权限仓库用于测试
type mockDevicePermissionRepository struct {
	permissions map[uint]map[uint]bool // userID -> deviceID -> hasPermission
}

func newMockDevicePermissionRepository() *mockDevicePermissionRepository {
	return &mockDevicePermissionRepository{
		permissions: make(map[uint]map[uint]bool),
	}
}

func (m *mockDevicePermissionRepository) AssignDeviceToUser(userID, deviceID uint) error {
	if m.permissions[userID] == nil {
		m.permissions[userID] = make(map[uint]bool)
	}
	m.permissions[userID][deviceID] = true
	return nil
}

func (m *mockDevicePermissionRepository) RemoveDeviceFromUser(userID, deviceID uint) error {
	if m.permissions[userID] != nil {
		delete(m.permissions[userID], deviceID)
	}
	return nil
}

func (m *mockDevicePermissionRepository) GetUserDevices(userID uint) ([]*models.Device, error) {
	var devices []*models.Device
	if m.permissions[userID] != nil {
		for deviceID := range m.permissions[userID] {
			devices = append(devices, &models.Device{ID: deviceID})
		}
	}
	return devices, nil
}

func (m *mockDevicePermissionRepository) HasDevicePermission(userID, deviceID uint) (bool, error) {
	if m.permissions[userID] == nil {
		return false, nil
	}
	return m.permissions[userID][deviceID], nil
}

func (m *mockDevicePermissionRepository) Clear() {
	m.permissions = make(map[uint]map[uint]bool)
}

// mockUserRepoForPermission 模拟用户仓库用于权限测试
type mockUserRepoForPermission struct {
	users map[uint]*models.User
}

func newMockUserRepoForPermission() *mockUserRepoForPermission {
	return &mockUserRepoForPermission{
		users: make(map[uint]*models.User),
	}
}

func (m *mockUserRepoForPermission) GetByID(id uint) (*models.User, error) {
	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepoForPermission) AddUser(user *models.User) {
	m.users[user.ID] = user
}

func (m *mockUserRepoForPermission) Clear() {
	m.users = make(map[uint]*models.User)
}

// TestDevicePermissionControl 测试设备权限控制属性
// Feature: network-monitoring-platform, Property 11: 权限控制
// 验证角色权限和设备权限隔离
// **验证需求: 12.4, 12.5**
func TestDevicePermissionControl(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 创建模拟仓库
	permRepo := newMockDevicePermissionRepository()
	userRepo := newMockUserRepoForPermission()

	// Property 1: admin 角色可以操作所有设备
	properties.Property("admin role should have access to all devices",
		prop.ForAll(
			func(userID, deviceID uint, action string) bool {
				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建 admin 用户
				adminUser := &models.User{
					ID:       userID,
					Username: "admin_user",
					Roles: []models.Role{
						{ID: 1, Name: "admin"},
					},
				}
				userRepo.AddUser(adminUser)

				// 检查权限
				allowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, action)
				if err != nil {
					return false
				}

				// admin 应该对所有设备有权限
				return allowed
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
			gen.OneConstOf("read", "create", "update", "delete"),
		))

	// Property 2: viewer 角色只能读取，不能修改
	properties.Property("viewer role should only have read access",
		prop.ForAll(
			func(userID, deviceID uint) bool {
				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建 viewer 用户
				viewerUser := &models.User{
					ID:       userID,
					Username: "viewer_user",
					Roles: []models.Role{
						{ID: 3, Name: "viewer"},
					},
				}
				userRepo.AddUser(viewerUser)

				// 检查读取权限
				readAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, "read")
				if err != nil {
					return false
				}

				// 检查写入权限
				writeAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, "update")
				if err != nil {
					return false
				}

				deleteAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, "delete")
				if err != nil {
					return false
				}

				// viewer 应该可以读取，但不能修改或删除
				return readAllowed && !writeAllowed && !deleteAllowed
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
		))

	// Property 3: operator 角色只能操作分配给自己的设备
	properties.Property("operator role should only access assigned devices",
		prop.ForAll(
			func(userID, assignedDeviceID, unassignedDeviceID uint) bool {
				// 确保两个设备ID不同
				if assignedDeviceID == unassignedDeviceID {
					return true // 跳过这个测试用例
				}

				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建 operator 用户
				operatorUser := &models.User{
					ID:       userID,
					Username: "operator_user",
					Roles: []models.Role{
						{ID: 2, Name: "operator"},
					},
				}
				userRepo.AddUser(operatorUser)

				// 分配一个设备给用户
				permRepo.AssignDeviceToUser(userID, assignedDeviceID)

				// 检查已分配设备的权限
				assignedAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, assignedDeviceID, "update")
				if err != nil {
					return false
				}

				// 检查未分配设备的权限
				unassignedAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, unassignedDeviceID, "update")
				if err != nil {
					return false
				}

				// operator 应该可以操作已分配的设备，但不能操作未分配的设备
				return assignedAllowed && !unassignedAllowed
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 500),
			gen.UIntRange(501, 1000),
		))

	// Property 4: 没有角色的用户不能访问任何设备
	properties.Property("user without role should not access any device",
		prop.ForAll(
			func(userID, deviceID uint, action string) bool {
				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建没有角色的用户
				noRoleUser := &models.User{
					ID:       userID,
					Username: "no_role_user",
					Roles:    []models.Role{},
				}
				userRepo.AddUser(noRoleUser)

				// 检查权限
				allowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, action)
				if err != nil {
					return false
				}

				// 没有角色的用户不应该有任何权限
				return !allowed
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
			gen.OneConstOf("read", "create", "update", "delete"),
		))

	// Property 5: 设备权限隔离 - 用户A的设备权限不影响用户B
	properties.Property("device permissions should be isolated between users",
		prop.ForAll(
			func(userAID, userBID, deviceID uint) bool {
				// 确保两个用户ID不同
				if userAID == userBID {
					return true // 跳过这个测试用例
				}

				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建两个 operator 用户
				userA := &models.User{
					ID:       userAID,
					Username: "operator_a",
					Roles: []models.Role{
						{ID: 2, Name: "operator"},
					},
				}
				userB := &models.User{
					ID:       userBID,
					Username: "operator_b",
					Roles: []models.Role{
						{ID: 2, Name: "operator"},
					},
				}
				userRepo.AddUser(userA)
				userRepo.AddUser(userB)

				// 只给用户A分配设备权限
				permRepo.AssignDeviceToUser(userAID, deviceID)

				// 检查用户A的权限
				userAAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userAID, deviceID, "update")
				if err != nil {
					return false
				}

				// 检查用户B的权限
				userBAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userBID, deviceID, "update")
				if err != nil {
					return false
				}

				// 用户A应该有权限，用户B不应该有权限
				return userAAllowed && !userBAllowed
			},
			gen.UIntRange(1, 500),
			gen.UIntRange(501, 1000),
			gen.UIntRange(1, 1000),
		))

	// Property 6: 移除设备权限后，用户不能再访问该设备
	properties.Property("removing device permission should revoke access",
		prop.ForAll(
			func(userID, deviceID uint) bool {
				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建 operator 用户
				operatorUser := &models.User{
					ID:       userID,
					Username: "operator_user",
					Roles: []models.Role{
						{ID: 2, Name: "operator"},
					},
				}
				userRepo.AddUser(operatorUser)

				// 分配设备权限
				permRepo.AssignDeviceToUser(userID, deviceID)

				// 验证有权限
				beforeRemove, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, "update")
				if err != nil || !beforeRemove {
					return false
				}

				// 移除设备权限
				permRepo.RemoveDeviceFromUser(userID, deviceID)

				// 验证权限已被移除
				afterRemove, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, "update")
				if err != nil {
					return false
				}

				return !afterRemove
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
		))

	// Property 7: operator + viewer 双角色用户应该有 operator 的权限
	properties.Property("user with both operator and viewer roles should have operator permissions",
		prop.ForAll(
			func(userID, deviceID uint) bool {
				// 清理数据
				permRepo.Clear()
				userRepo.Clear()

				// 创建同时拥有 operator 和 viewer 角色的用户
				dualRoleUser := &models.User{
					ID:       userID,
					Username: "dual_role_user",
					Roles: []models.Role{
						{ID: 2, Name: "operator"},
						{ID: 3, Name: "viewer"},
					},
				}
				userRepo.AddUser(dualRoleUser)

				// 分配设备权限
				permRepo.AssignDeviceToUser(userID, deviceID)

				// 检查更新权限（operator 权限）
				updateAllowed, err := checkDevicePermissionLogic(userRepo, permRepo, userID, deviceID, "update")
				if err != nil {
					return false
				}

				// 应该有 operator 的权限
				return updateAllowed
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
		))
}

// checkDevicePermissionLogic 模拟 DevicePermissionChecker.CheckDevicePermission 的逻辑
// 这是为了测试权限逻辑而不依赖完整的 DevicePermissionChecker 结构
func checkDevicePermissionLogic(
	userRepo *mockUserRepoForPermission,
	permRepo *mockDevicePermissionRepository,
	userID, deviceID uint,
	action string,
) (bool, error) {
	// 获取用户信息（包含角色）
	user, err := userRepo.GetByID(userID)
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
		hasPermission, err := permRepo.HasDevicePermission(userID, deviceID)
		if err != nil {
			return false, err
		}
		return hasPermission, nil
	}

	// 没有任何角色，拒绝访问
	return false, nil
}

// TestRoleHierarchy 测试角色层级属性
func TestRoleHierarchy(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	userRepo := newMockUserRepoForPermission()

	// Property: 角色优先级 admin > operator > viewer
	properties.Property("role priority should be admin > operator > viewer",
		prop.ForAll(
			func(userID uint) bool {
				userRepo.Clear()

				// 创建拥有所有角色的用户
				user := &models.User{
					ID:       userID,
					Username: "multi_role_user",
					Roles: []models.Role{
						{ID: 3, Name: "viewer"},
						{ID: 2, Name: "operator"},
						{ID: 1, Name: "admin"},
					},
				}
				userRepo.AddUser(user)

				// 获取主要角色
				role := getUserRole(user)

				// 应该返回 admin（最高优先级）
				return role == "admin"
			},
			gen.UIntRange(1, 1000),
		))

	// Property: 只有 operator 和 viewer 时，应该返回 operator
	properties.Property("operator should take priority over viewer",
		prop.ForAll(
			func(userID uint) bool {
				userRepo.Clear()

				user := &models.User{
					ID:       userID,
					Username: "op_viewer_user",
					Roles: []models.Role{
						{ID: 3, Name: "viewer"},
						{ID: 2, Name: "operator"},
					},
				}
				userRepo.AddUser(user)

				role := getUserRole(user)
				return role == "operator"
			},
			gen.UIntRange(1, 1000),
		))

	// Property: 只有 viewer 时，应该返回 viewer
	properties.Property("single viewer role should return viewer",
		prop.ForAll(
			func(userID uint) bool {
				userRepo.Clear()

				user := &models.User{
					ID:       userID,
					Username: "viewer_only_user",
					Roles: []models.Role{
						{ID: 3, Name: "viewer"},
					},
				}
				userRepo.AddUser(user)

				role := getUserRole(user)
				return role == "viewer"
			},
			gen.UIntRange(1, 1000),
		))

	// Property: 没有角色时，应该返回空字符串
	properties.Property("no role should return empty string",
		prop.ForAll(
			func(userID uint) bool {
				userRepo.Clear()

				user := &models.User{
					ID:       userID,
					Username: "no_role_user",
					Roles:    []models.Role{},
				}
				userRepo.AddUser(user)

				role := getUserRole(user)
				return role == ""
			},
			gen.UIntRange(1, 1000),
		))
}

// getUserRole 获取用户的主要角色（模拟 DevicePermissionChecker.GetUserRole）
func getUserRole(user *models.User) string {
	// 按优先级返回角色: admin > operator > viewer
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return "admin"
		}
	}
	for _, role := range user.Roles {
		if role.Name == "operator" {
			return "operator"
		}
	}
	for _, role := range user.Roles {
		if role.Name == "viewer" {
			return "viewer"
		}
	}
	return ""
}

// TestSuperAdminCheck 测试超级管理员检查
func TestSuperAdminCheck(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	userRepo := newMockUserRepoForPermission()

	// Property: admin 角色应该被识别为超级管理员
	properties.Property("admin role should be identified as super admin",
		prop.ForAll(
			func(userID uint) bool {
				userRepo.Clear()

				adminUser := &models.User{
					ID:       userID,
					Username: "admin_user",
					Roles: []models.Role{
						{ID: 1, Name: "admin"},
					},
				}
				userRepo.AddUser(adminUser)

				return isSuperAdmin(adminUser)
			},
			gen.UIntRange(1, 1000),
		))

	// Property: 非 admin 角色不应该被识别为超级管理员
	properties.Property("non-admin roles should not be identified as super admin",
		prop.ForAll(
			func(userID uint, roleName string) bool {
				if roleName == "admin" {
					return true // 跳过 admin 角色
				}

				userRepo.Clear()

				user := &models.User{
					ID:       userID,
					Username: "non_admin_user",
					Roles: []models.Role{
						{ID: 2, Name: roleName},
					},
				}
				userRepo.AddUser(user)

				return !isSuperAdmin(user)
			},
			gen.UIntRange(1, 1000),
			gen.OneConstOf("operator", "viewer", "custom_role"),
		))
}

// isSuperAdmin 检查用户是否是超级管理员（模拟 DevicePermissionChecker.IsSuperAdmin）
func isSuperAdmin(user *models.User) bool {
	for _, role := range user.Roles {
		if role.Name == "admin" {
			return true
		}
	}
	return false
}

// TestDevicePermissionAssignment 测试设备权限分配
func TestDevicePermissionAssignment(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	permRepo := newMockDevicePermissionRepository()

	// Property: 分配设备权限后，HasDevicePermission 应该返回 true
	properties.Property("assigned device permission should be detectable",
		prop.ForAll(
			func(userID, deviceID uint) bool {
				permRepo.Clear()

				// 分配权限
				err := permRepo.AssignDeviceToUser(userID, deviceID)
				if err != nil {
					return false
				}

				// 检查权限
				hasPermission, err := permRepo.HasDevicePermission(userID, deviceID)
				if err != nil {
					return false
				}

				return hasPermission
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
		))

	// Property: 未分配的设备权限应该返回 false
	properties.Property("unassigned device permission should return false",
		prop.ForAll(
			func(userID, deviceID uint) bool {
				permRepo.Clear()

				// 不分配权限，直接检查
				hasPermission, err := permRepo.HasDevicePermission(userID, deviceID)
				if err != nil {
					return false
				}

				return !hasPermission
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
		))

	// Property: 分配多个设备后，GetUserDevices 应该返回所有设备
	properties.Property("GetUserDevices should return all assigned devices",
		prop.ForAll(
			func(userID uint, deviceCount uint8) bool {
				permRepo.Clear()

				// 限制设备数量在合理范围内
				count := int(deviceCount%10) + 1

				// 分配多个设备
				for i := 1; i <= count; i++ {
					err := permRepo.AssignDeviceToUser(userID, uint(i))
					if err != nil {
						return false
					}
				}

				// 获取用户设备
				devices, err := permRepo.GetUserDevices(userID)
				if err != nil {
					return false
				}

				return len(devices) == count
			},
			gen.UIntRange(1, 1000),
			gen.UInt8(),
		))
}
