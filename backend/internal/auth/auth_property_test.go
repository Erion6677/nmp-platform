package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"nmp-platform/internal/models"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// mockUserRepository 模拟用户仓库用于测试
type mockUserRepository struct {
	users map[string]*models.User
	nextID uint
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users:  make(map[string]*models.User),
		nextID: 1,
	}
}

func (m *mockUserRepository) Create(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	
	// 检查用户名是否已存在
	for _, existingUser := range m.users {
		if existingUser.Username == user.Username {
			return errors.New("username already exists")
		}
		if user.Email != "" && existingUser.Email == user.Email {
			return errors.New("email already exists")
		}
	}
	
	user.ID = m.nextID
	m.nextID++
	m.users[user.Username] = user
	return nil
}

func (m *mockUserRepository) GetByID(id uint) (*models.User, error) {
	for _, user := range m.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) GetByUsername(username string) (*models.User, error) {
	if user, exists := m.users[username]; exists {
		return user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) GetByEmail(email string) (*models.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}

func (m *mockUserRepository) Update(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	
	for username, existingUser := range m.users {
		if existingUser.ID == user.ID {
			m.users[username] = user
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *mockUserRepository) Delete(id uint) error {
	for username, user := range m.users {
		if user.ID == id {
			delete(m.users, username)
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *mockUserRepository) List(page, size int, search string) ([]*models.User, int64, error) {
	users := make([]*models.User, 0, len(m.users))
	for _, user := range m.users {
		// 简单的搜索过滤
		if search != "" {
			if !strings.Contains(strings.ToLower(user.Username), strings.ToLower(search)) &&
			   !strings.Contains(strings.ToLower(user.Email), strings.ToLower(search)) &&
			   !strings.Contains(strings.ToLower(user.FullName), strings.ToLower(search)) {
				continue
			}
		}
		users = append(users, user)
	}
	
	total := int64(len(users))
	offset := (page - 1) * size
	
	if offset > len(users) {
		return []*models.User{}, total, nil
	}
	end := offset + size
	if end > len(users) {
		end = len(users)
	}
	
	return users[offset:end], total, nil
}

func (m *mockUserRepository) UpdateLastLogin(userID uint) error {
	for _, user := range m.users {
		if user.ID == userID {
			now := time.Now()
			user.LastLogin = &now
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *mockUserRepository) AssignRoles(userID uint, roleIDs []uint) error {
	// 在mock实现中，我们简单地返回nil表示成功
	// 实际的角色分配逻辑在真实的repository中实现
	for _, user := range m.users {
		if user.ID == userID {
			return nil
		}
	}
	return errors.New("user not found")
}

func (m *mockUserRepository) Clear() {
	m.users = make(map[string]*models.User)
	m.nextID = 1
}

// createValidUser 创建有效用户用于测试
func createValidUser(username, password string) *models.User {
	return &models.User{
		Username: username,
		Password: password,
		Email:    username + "@example.com",
		FullName: "Test User " + username,
		Status:   models.UserStatusActive,
	}
}

// TestUserAuthenticationIntegrity 测试用户认证完整性属性
// Feature: network-monitoring-platform, Property 1: 用户认证完整性
// 对于任何有效的用户凭据，认证服务应该生成有效的JWT令牌，并且该令牌应该能够通过后续的访问控制验证
// **验证需求: 1.1, 1.2**
func TestUserAuthenticationIntegrity(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 创建模拟仓库和认证服务
	userRepo := newMockUserRepository()
	authService := NewAuthService(userRepo, "test-secret-key", 24*time.Hour)

	properties.Property("valid credentials should generate valid JWT token that passes validation", 
		prop.ForAll(
			func(username, password string) bool {
				// 清理数据
				userRepo.Clear()
				
				// 注册用户（这会加密密码）
				registerReq := &RegisterRequest{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					FullName: "Test User " + username,
				}
				
				registeredUser, err := authService.Register(registerReq)
				if err != nil {
					return false
				}
				
				// 尝试登录
				loginReq := &LoginRequest{
					Username: username,
					Password: password,
				}
				
				loginResponse, err := authService.Login(loginReq)
				if err != nil {
					return false
				}
				
				// 验证令牌有效性
				claims, err := authService.ValidateToken(loginResponse.Token)
				if err != nil {
					return false
				}
				
				// 验证令牌中的用户信息
				return claims.UserID == registeredUser.ID && 
					   claims.Username == username &&
					   loginResponse.User.ID == registeredUser.ID &&
					   loginResponse.User.Username == username
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
		))

	properties.Property("invalid credentials should fail authentication", 
		prop.ForAll(
			func(username, password, wrongPassword string) bool {
				// 确保错误密码与正确密码不同
				if password == wrongPassword {
					return true // 跳过这个测试用例
				}
				
				// 清理数据
				userRepo.Clear()
				
				// 创建用户
				registerReq := &RegisterRequest{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					FullName: "Test User",
				}
				
				_, err := authService.Register(registerReq)
				if err != nil {
					return false
				}
				
				// 尝试使用错误密码登录
				loginReq := &LoginRequest{
					Username: username,
					Password: wrongPassword,
				}
				
				_, err = authService.Login(loginReq)
				// 应该返回错误
				return err != nil
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
		))

	properties.Property("inactive users should not be able to login", 
		prop.ForAll(
			func(username, password string) bool {
				// 清理数据
				userRepo.Clear()
				
				// 创建用户
				registerReq := &RegisterRequest{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					FullName: "Test User",
				}
				
				user, err := authService.Register(registerReq)
				if err != nil {
					return false
				}
				
				// 将用户状态设置为非活跃
				user.Status = models.UserStatusInactive
				err = userRepo.Update(user)
				if err != nil {
					return false
				}
				
				// 尝试登录
				loginReq := &LoginRequest{
					Username: username,
					Password: password,
				}
				
				_, err = authService.Login(loginReq)
				// 应该返回错误
				return err != nil
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
		))

	properties.Property("token validation should be consistent with token generation", 
		prop.ForAll(
			func(username, password string) bool {
				// 清理数据
				userRepo.Clear()
				
				// 创建用户并登录
				registerReq := &RegisterRequest{
					Username: username,
					Password: password,
					Email:    username + "@example.com",
					FullName: "Test User",
				}
				
				user, err := authService.Register(registerReq)
				if err != nil {
					return false
				}
				
				loginReq := &LoginRequest{
					Username: username,
					Password: password,
				}
				
				loginResponse, err := authService.Login(loginReq)
				if err != nil {
					return false
				}
				
				// 验证令牌
				claims, err := authService.ValidateToken(loginResponse.Token)
				if err != nil {
					return false
				}
				
				// 使用令牌获取用户信息
				userInfo, err := authService.GetUserByID(claims.UserID)
				if err != nil {
					return false
				}
				
				// 验证一致性
				return userInfo.ID == user.ID &&
					   userInfo.Username == user.Username &&
					   userInfo.Email == user.Email
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 3 && len(s) <= 20 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 6 && len(s) <= 50 }),
		))
}