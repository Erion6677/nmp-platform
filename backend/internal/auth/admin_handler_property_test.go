package auth

import (
	"testing"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"nmp-platform/internal/service"

	"github.com/glebarez/sqlite"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Feature: nmp-bugfix-iteration, Property 5: Admin API 数据库实现
// **Validates: Requirements 5.2, 5.3**
// *For any* Admin API 返回的用户/角色/权限列表，数据必须来自数据库查询结果，而非硬编码数据。
// 具体验证：创建新用户后，列表 API 必须返回包含该新用户的结果。

// setupAdminTestDB 创建测试用的内存数据库
func setupAdminTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// 自动迁移所有相关表
	err = db.AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.Permission{},
		&models.UserRole{},
		&models.RolePermission{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// TestUserData 测试用户数据结构
type TestUserData struct {
	Username string
	Email    string
	FullName string
	Password string
}

// TestRoleData 测试角色数据结构
type TestRoleData struct {
	Name        string
	DisplayName string
	Description string
}

func TestAdminAPIDatabaseImplementationProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性1: 用户创建后应该出现在列表中
	// 对于任意有效的用户数据，创建后通过列表 API 应该能查询到该用户
	properties.Property("created user should appear in user list",
		prop.ForAll(
			func(userData TestUserData) bool {
				db := setupAdminTestDB(t)
				userRepo := repository.NewUserRepository(db)
				passwordService := service.NewPasswordService()

				// 哈希密码
				hashedPassword, err := passwordService.Hash(userData.Password)
				if err != nil {
					return false
				}

				// 创建用户
				user := &models.User{
					Username: userData.Username,
					Email:    userData.Email,
					FullName: userData.FullName,
					Password: hashedPassword,
					Status:   models.UserStatusActive,
				}

				err = userRepo.Create(user)
				if err != nil {
					return false
				}

				// 通过列表 API 查询
				users, total, err := userRepo.List(1, 100, "")
				if err != nil {
					return false
				}

				// 验证用户在列表中
				if total < 1 {
					return false
				}

				found := false
				for _, u := range users {
					if u.Username == userData.Username {
						found = true
						break
					}
				}

				return found
			},
			genTestUserData(),
		))

	// 属性2: 角色创建后应该出现在列表中
	// 对于任意有效的角色数据，创建后通过列表 API 应该能查询到该角色
	properties.Property("created role should appear in role list",
		prop.ForAll(
			func(roleData TestRoleData) bool {
				db := setupAdminTestDB(t)
				roleRepo := repository.NewRoleRepository(db)

				// 创建角色
				role := &models.Role{
					Name:        roleData.Name,
					DisplayName: roleData.DisplayName,
					Description: roleData.Description,
					IsSystem:    false,
				}

				err := roleRepo.Create(role)
				if err != nil {
					return false
				}

				// 通过列表 API 查询
				roles, total, err := roleRepo.List(1, 100, "")
				if err != nil {
					return false
				}

				// 验证角色在列表中
				if total < 1 {
					return false
				}

				found := false
				for _, r := range roles {
					if r.Name == roleData.Name {
						found = true
						break
					}
				}

				return found
			},
			genTestRoleData(),
		))

	// 属性3: 用户更新后列表应该反映更新
	// 对于任意用户，更新其信息后，列表 API 应该返回更新后的数据
	properties.Property("updated user should reflect changes in list",
		prop.ForAll(
			func(userData TestUserData, newFullName string) bool {
				if newFullName == "" || newFullName == userData.FullName {
					return true // 跳过无效情况
				}

				db := setupAdminTestDB(t)
				userRepo := repository.NewUserRepository(db)
				passwordService := service.NewPasswordService()

				// 哈希密码
				hashedPassword, err := passwordService.Hash(userData.Password)
				if err != nil {
					return false
				}

				// 创建用户
				user := &models.User{
					Username: userData.Username,
					Email:    userData.Email,
					FullName: userData.FullName,
					Password: hashedPassword,
					Status:   models.UserStatusActive,
				}

				err = userRepo.Create(user)
				if err != nil {
					return false
				}

				// 更新用户
				user.FullName = newFullName
				err = userRepo.Update(user)
				if err != nil {
					return false
				}

				// 通过列表 API 查询
				users, _, err := userRepo.List(1, 100, "")
				if err != nil {
					return false
				}

				// 验证更新后的数据
				for _, u := range users {
					if u.Username == userData.Username {
						return u.FullName == newFullName
					}
				}

				return false
			},
			genTestUserData(),
			gen.Identifier(), // 使用 Identifier 生成新的全名
		))

	// 属性4: 删除用户后不应该出现在列表中
	// 对于任意用户，删除后通过列表 API 不应该能查询到该用户
	properties.Property("deleted user should not appear in list",
		prop.ForAll(
			func(userData TestUserData) bool {
				db := setupAdminTestDB(t)
				userRepo := repository.NewUserRepository(db)
				passwordService := service.NewPasswordService()

				// 哈希密码
				hashedPassword, err := passwordService.Hash(userData.Password)
				if err != nil {
					return false
				}

				// 创建用户
				user := &models.User{
					Username: userData.Username,
					Email:    userData.Email,
					FullName: userData.FullName,
					Password: hashedPassword,
					Status:   models.UserStatusActive,
				}

				err = userRepo.Create(user)
				if err != nil {
					return false
				}

				// 删除用户
				err = userRepo.Delete(user.ID)
				if err != nil {
					return false
				}

				// 通过列表 API 查询
				users, _, err := userRepo.List(1, 100, "")
				if err != nil {
					return false
				}

				// 验证用户不在列表中
				for _, u := range users {
					if u.Username == userData.Username {
						return false // 不应该找到
					}
				}

				return true
			},
			genTestUserData(),
		))

	// 属性5: 密码应该以 bcrypt 哈希存储
	// 对于任意创建的用户，数据库中存储的密码应该是 bcrypt 哈希格式
	properties.Property("password should be stored as bcrypt hash",
		prop.ForAll(
			func(userData TestUserData) bool {
				db := setupAdminTestDB(t)
				userRepo := repository.NewUserRepository(db)
				passwordService := service.NewPasswordService()

				// 哈希密码
				hashedPassword, err := passwordService.Hash(userData.Password)
				if err != nil {
					return false
				}

				// 创建用户
				user := &models.User{
					Username: userData.Username,
					Email:    userData.Email,
					FullName: userData.FullName,
					Password: hashedPassword,
					Status:   models.UserStatusActive,
				}

				err = userRepo.Create(user)
				if err != nil {
					return false
				}

				// 直接从数据库查询密码
				var dbUser models.User
				err = db.Where("username = ?", userData.Username).First(&dbUser).Error
				if err != nil {
					return false
				}

				// 验证密码是 bcrypt 哈希格式
				return service.IsBcryptHash(dbUser.Password)
			},
			genTestUserData(),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genTestUserData 生成有效的测试用户数据
func genTestUserData() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(), // 生成有效的标识符作为用户名
		gen.Identifier(), // 生成有效的标识符作为邮箱前缀
		gen.Identifier(), // 生成有效的标识符作为全名
		gen.Identifier(), // 生成有效的标识符作为密码
	).Map(func(values []interface{}) TestUserData {
		username := values[0].(string)
		if len(username) > 20 {
			username = username[:20]
		}
		emailPrefix := values[1].(string)
		if len(emailPrefix) > 20 {
			emailPrefix = emailPrefix[:20]
		}
		fullName := values[2].(string)
		if len(fullName) > 50 {
			fullName = fullName[:50]
		}
		password := values[3].(string)
		if len(password) < 6 {
			password = password + "123456"
		}
		if len(password) > 20 {
			password = password[:20]
		}
		return TestUserData{
			Username: username,
			Email:    emailPrefix + "@test.com",
			FullName: fullName,
			Password: password,
		}
	})
}

// genTestRoleData 生成有效的测试角色数据
func genTestRoleData() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(), // 生成有效的标识符作为角色名
		gen.Identifier(), // 生成有效的标识符作为显示名
		gen.Identifier(), // 生成有效的标识符作为描述
	).Map(func(values []interface{}) TestRoleData {
		name := values[0].(string)
		if len(name) > 20 {
			name = name[:20]
		}
		displayName := values[1].(string)
		if len(displayName) > 50 {
			displayName = displayName[:50]
		}
		description := values[2].(string)
		if len(description) > 100 {
			description = description[:100]
		}
		return TestRoleData{
			Name:        name,
			DisplayName: displayName,
			Description: description,
		}
	})
}

// 单元测试：验证具体场景
func TestAdminHandlerUnit(t *testing.T) {
	t.Run("create user with valid data should succeed", func(t *testing.T) {
		db := setupAdminTestDB(t)
		userRepo := repository.NewUserRepository(db)
		passwordService := service.NewPasswordService()

		hashedPassword, err := passwordService.Hash("testpassword123")
		assert.NoError(t, err)

		user := &models.User{
			Username: "testuser",
			Email:    "test@example.com",
			FullName: "Test User",
			Password: hashedPassword,
			Status:   models.UserStatusActive,
		}

		err = userRepo.Create(user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)
	})

	t.Run("create duplicate username should fail", func(t *testing.T) {
		db := setupAdminTestDB(t)
		userRepo := repository.NewUserRepository(db)
		passwordService := service.NewPasswordService()

		hashedPassword, err := passwordService.Hash("testpassword123")
		assert.NoError(t, err)

		user1 := &models.User{
			Username: "duplicateuser",
			Email:    "test1@example.com",
			Password: hashedPassword,
			Status:   models.UserStatusActive,
		}

		err = userRepo.Create(user1)
		assert.NoError(t, err)

		user2 := &models.User{
			Username: "duplicateuser",
			Email:    "test2@example.com",
			Password: hashedPassword,
			Status:   models.UserStatusActive,
		}

		err = userRepo.Create(user2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("create role with valid data should succeed", func(t *testing.T) {
		db := setupAdminTestDB(t)
		roleRepo := repository.NewRoleRepository(db)

		role := &models.Role{
			Name:        "testrole",
			DisplayName: "Test Role",
			Description: "A test role",
			IsSystem:    false,
		}

		err := roleRepo.Create(role)
		assert.NoError(t, err)
		assert.NotZero(t, role.ID)
	})

	t.Run("delete system role should fail", func(t *testing.T) {
		db := setupAdminTestDB(t)
		roleRepo := repository.NewRoleRepository(db)

		role := &models.Role{
			Name:        "admin",
			DisplayName: "Administrator",
			Description: "System administrator",
			IsSystem:    true,
		}

		err := roleRepo.Create(role)
		assert.NoError(t, err)

		err = roleRepo.Delete(role.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "system role")
	})

	t.Run("user list should support search", func(t *testing.T) {
		db := setupAdminTestDB(t)
		userRepo := repository.NewUserRepository(db)
		passwordService := service.NewPasswordService()

		hashedPassword, err := passwordService.Hash("testpassword123")
		assert.NoError(t, err)

		// 创建多个用户
		users := []*models.User{
			{Username: "alice", Email: "alice@test.com", FullName: "Alice Smith", Password: hashedPassword, Status: models.UserStatusActive},
			{Username: "bob", Email: "bob@test.com", FullName: "Bob Jones", Password: hashedPassword, Status: models.UserStatusActive},
			{Username: "charlie", Email: "charlie@test.com", FullName: "Charlie Brown", Password: hashedPassword, Status: models.UserStatusActive},
		}

		for _, u := range users {
			err := userRepo.Create(u)
			assert.NoError(t, err)
		}

		// 搜索 "alice"
		results, total, err := userRepo.List(1, 10, "alice")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, results, 1)
		assert.Equal(t, "alice", results[0].Username)
	})

	t.Run("role list should support pagination", func(t *testing.T) {
		db := setupAdminTestDB(t)
		roleRepo := repository.NewRoleRepository(db)

		// 创建多个角色
		for i := 0; i < 15; i++ {
			role := &models.Role{
				Name:        "role" + string(rune('a'+i)),
				DisplayName: "Role " + string(rune('A'+i)),
				IsSystem:    false,
			}
			err := roleRepo.Create(role)
			assert.NoError(t, err)
		}

		// 获取第一页
		page1, total, err := roleRepo.List(1, 10, "")
		assert.NoError(t, err)
		assert.Equal(t, int64(15), total)
		assert.Len(t, page1, 10)

		// 获取第二页
		page2, _, err := roleRepo.List(2, 10, "")
		assert.NoError(t, err)
		assert.Len(t, page2, 5)
	})
}
