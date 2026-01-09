package auth

import (
	"errors"
	"log"
	"time"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"email"`
	FullName string `json:"full_name"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt time.Time   `json:"expires_at"`
	User      *UserInfo   `json:"user"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       uint     `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Status   string   `json:"status"`
	Roles    []string `json:"roles"`
}

// AuthService 认证服务
type AuthService struct {
	userRepo        repository.UserRepository
	jwtManager      *JWTManager
	passwordManager *PasswordManager
	logger          *log.Logger
}

// NewAuthService 创建新的认证服务
func NewAuthService(userRepo repository.UserRepository, jwtSecret string, tokenExpiry time.Duration) *AuthService {
	return &AuthService{
		userRepo:        userRepo,
		jwtManager:      NewJWTManager(jwtSecret, tokenExpiry),
		passwordManager: NewPasswordManager(),
		logger:          log.Default(),
	}
}

// SetLogger 设置日志记录器
func (s *AuthService) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// logLoginEvent 记录登录事件
func (s *AuthService) logLoginEvent(userID uint, username, event, detail string) {
	if s.logger == nil {
		return
	}
	
	timestamp := time.Now().Format(time.RFC3339)
	if detail != "" {
		s.logger.Printf("[AUTH] %s | user_id=%d | username=%s | event=%s | detail=%s", 
			timestamp, userID, username, event, detail)
	} else {
		s.logger.Printf("[AUTH] %s | user_id=%d | username=%s | event=%s", 
			timestamp, userID, username, event)
	}
}

// Login 用户登录
func (s *AuthService) Login(req *LoginRequest) (*LoginResponse, error) {
	// 获取用户
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		s.logLoginEvent(0, req.Username, "login_failed", "user not found")
		return nil, errors.New("invalid username or password")
	}

	// 检查用户状态
	if user.Status != models.UserStatusActive {
		s.logLoginEvent(user.ID, user.Username, "login_failed", "account not active")
		return nil, errors.New("user account is not active")
	}

	// 验证密码
	if err := s.passwordManager.VerifyPassword(user.Password, req.Password); err != nil {
		s.logLoginEvent(user.ID, user.Username, "login_failed", "invalid password")
		return nil, errors.New("invalid username or password")
	}

	// 获取用户角色
	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	// 生成JWT令牌
	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, roles)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	// 更新最后登录时间
	if err := s.userRepo.UpdateLastLogin(user.ID); err != nil {
		// 记录日志但不影响登录流程
		s.logLoginEvent(user.ID, user.Username, "login_time_update_failed", err.Error())
	} else {
		s.logLoginEvent(user.ID, user.Username, "login_success", "")
	}

	// 计算过期时间
	expiresAt := time.Now().Add(s.jwtManager.tokenExpiry)

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			FullName: user.FullName,
			Status:   string(user.Status),
			Roles:    roles,
		},
	}, nil
}

// Register 用户注册
func (s *AuthService) Register(req *RegisterRequest) (*models.User, error) {
	// 验证密码强度
	if err := s.passwordManager.IsValidPassword(req.Password); err != nil {
		return nil, err
	}

	// 加密密码
	hashedPassword, err := s.passwordManager.HashPassword(req.Password)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// 创建用户
	user := &models.User{
		Username: req.Username,
		Password: hashedPassword,
		Email:    req.Email,
		FullName: req.FullName,
		Status:   models.UserStatusActive,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// 清除密码字段
	user.Password = ""
	return user, nil
}

// ValidateToken 验证令牌
func (s *AuthService) ValidateToken(tokenString string) (*TokenClaims, error) {
	return s.jwtManager.ValidateToken(tokenString)
}

// RefreshToken 刷新令牌
func (s *AuthService) RefreshToken(tokenString string) (string, error) {
	return s.jwtManager.RefreshToken(tokenString)
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(userID uint) (*UserInfo, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	return &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		Status:   string(user.Status),
		Roles:    roles,
	}, nil
}

// ChangePassword 修改密码
func (s *AuthService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// 获取用户
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	// 验证旧密码
	if err := s.passwordManager.VerifyPassword(user.Password, oldPassword); err != nil {
		return errors.New("invalid old password")
	}

	// 验证新密码强度
	if err := s.passwordManager.IsValidPassword(newPassword); err != nil {
		return err
	}

	// 加密新密码
	hashedPassword, err := s.passwordManager.HashPassword(newPassword)
	if err != nil {
		return errors.New("failed to hash new password")
	}

	// 更新密码
	user.Password = hashedPassword
	return s.userRepo.Update(user)
}

// UpdateProfileRequest 更新个人信息请求
type UpdateProfileRequest struct {
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

// UpdateProfile 更新个人信息
func (s *AuthService) UpdateProfile(userID uint, req *UpdateProfileRequest) (*UserInfo, error) {
	// 获取用户
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	// 保存更新
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	// 返回更新后的用户信息
	roles := make([]string, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = role.Name
	}

	return &UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		Status:   string(user.Status),
		Roles:    roles,
	}, nil
}