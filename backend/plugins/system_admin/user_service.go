package system_admin

import (
	"errors"
	"fmt"

	"nmp-platform/internal/auth"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// UserService 用户管理服务
type UserService struct {
	userRepo    repository.UserRepository
	authService *auth.AuthService
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository, authService *auth.AuthService) *UserService {
	return &UserService{
		userRepo:    userRepo,
		authService: authService,
	}
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required"`
	Email    string   `json:"email" binding:"email"`
	FullName string   `json:"full_name"`
	Status   string   `json:"status"`
	RoleIDs  []uint   `json:"role_ids"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    string `json:"email" binding:"email"`
	FullName string `json:"full_name"`
	Status   string `json:"status"`
}

// UpdateUserStatusRequest 更新用户状态请求
type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateUserRolesRequest 更新用户角色请求
type UpdateUserRolesRequest struct {
	RoleIDs []uint `json:"role_ids" binding:"required"`
}

// UserListResponse 用户列表响应
type UserListResponse struct {
	Users []UserResponse `json:"users"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID        uint         `json:"id"`
	Username  string       `json:"username"`
	Email     string       `json:"email"`
	FullName  string       `json:"full_name"`
	Status    string       `json:"status"`
	Roles     []RoleInfo   `json:"roles"`
	CreatedAt string       `json:"created_at"`
	UpdatedAt string       `json:"updated_at"`
}

// RoleInfo 角色信息
type RoleInfo struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// ListUsers 获取用户列表
func (s *UserService) ListUsers(page, size int, search string) (*UserListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	users, total, err := s.userRepo.List(page, size, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %v", err)
	}

	userResponses := make([]UserResponse, len(users))
	for i, user := range users {
		roles := make([]RoleInfo, len(user.Roles))
		for j, role := range user.Roles {
			roles[j] = RoleInfo{
				ID:          role.ID,
				Name:        role.Name,
				DisplayName: role.DisplayName,
			}
		}

		userResponses[i] = UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			Status:    string(user.Status),
			Roles:     roles,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &UserListResponse{
		Users: userResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetUser 获取用户详情
func (s *UserService) GetUser(id uint) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	roles := make([]RoleInfo, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = RoleInfo{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
		}
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Status:    string(user.Status),
		Roles:     roles,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateUser 创建用户
func (s *UserService) CreateUser(req *CreateUserRequest) (*UserResponse, error) {
	// 验证状态
	status := models.UserStatusActive
	if req.Status != "" {
		switch models.UserStatus(req.Status) {
		case models.UserStatusActive, models.UserStatusInactive, models.UserStatusBlocked:
			status = models.UserStatus(req.Status)
		default:
			return nil, errors.New("invalid user status")
		}
	}

	// 使用认证服务创建用户
	registerReq := &auth.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		FullName: req.FullName,
	}

	user, err := s.authService.Register(registerReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	// 更新用户状态
	if status != models.UserStatusActive {
		user.Status = status
		if err := s.userRepo.Update(user); err != nil {
			return nil, fmt.Errorf("failed to update user status: %v", err)
		}
	}

	// 分配角色
	if len(req.RoleIDs) > 0 {
		if err := s.userRepo.AssignRoles(user.ID, req.RoleIDs); err != nil {
			return nil, fmt.Errorf("failed to assign roles: %v", err)
		}
		
		// 重新获取用户信息（包含角色）
		user, err = s.userRepo.GetByID(user.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated user: %v", err)
		}
	}

	roles := make([]RoleInfo, len(user.Roles))
	for i, role := range user.Roles {
		roles[i] = RoleInfo{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
		}
	}

	return &UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Status:    string(user.Status),
		Roles:     roles,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: user.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdateUser 更新用户
func (s *UserService) UpdateUser(id uint, req *UpdateUserRequest) (*UserResponse, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	// 更新字段
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Status != "" {
		switch models.UserStatus(req.Status) {
		case models.UserStatusActive, models.UserStatusInactive, models.UserStatusBlocked:
			user.Status = models.UserStatus(req.Status)
		default:
			return nil, errors.New("invalid user status")
		}
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	return s.GetUser(id)
}

// UpdateUserStatus 更新用户状态
func (s *UserService) UpdateUserStatus(id uint, req *UpdateUserStatusRequest) error {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	switch models.UserStatus(req.Status) {
	case models.UserStatusActive, models.UserStatusInactive, models.UserStatusBlocked:
		user.Status = models.UserStatus(req.Status)
	default:
		return errors.New("invalid user status")
	}

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update user status: %v", err)
	}

	return nil
}

// UpdateUserRoles 更新用户角色
func (s *UserService) UpdateUserRoles(id uint, req *UpdateUserRolesRequest) error {
	// 检查用户是否存在
	_, err := s.userRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	// 更新角色
	if err := s.userRepo.AssignRoles(id, req.RoleIDs); err != nil {
		return fmt.Errorf("failed to update user roles: %v", err)
	}

	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id uint) error {
	// 检查用户是否存在
	_, err := s.userRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	if err := s.userRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	return nil
}