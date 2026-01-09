package system_admin

import (
	"errors"
	"fmt"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// RoleService 角色管理服务
type RoleService struct {
	roleRepo repository.RoleRepository
	permRepo repository.PermissionRepository
}

// NewRoleService 创建角色服务
func NewRoleService(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) *RoleService {
	return &RoleService{
		roleRepo: roleRepo,
		permRepo: permRepo,
	}
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name           string `json:"name" binding:"required"`
	DisplayName    string `json:"display_name" binding:"required"`
	Description    string `json:"description"`
	PermissionIDs  []uint `json:"permission_ids"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description"`
}

// UpdateRolePermissionsRequest 更新角色权限请求
type UpdateRolePermissionsRequest struct {
	PermissionIDs []uint `json:"permission_ids" binding:"required"`
}

// RoleListResponse 角色列表响应
type RoleListResponse struct {
	Roles []RoleResponse `json:"roles"`
	Total int64          `json:"total"`
	Page  int            `json:"page"`
	Size  int            `json:"size"`
}

// RoleResponse 角色响应
type RoleResponse struct {
	ID          uint                 `json:"id"`
	Name        string               `json:"name"`
	DisplayName string               `json:"display_name"`
	Description string               `json:"description"`
	IsSystem    bool                 `json:"is_system"`
	Permissions []PermissionResponse `json:"permissions"`
	UserCount   int                  `json:"user_count"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

// ListRoles 获取角色列表
func (s *RoleService) ListRoles(page, size int, search string) (*RoleListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	roles, total, err := s.roleRepo.List(page, size, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %v", err)
	}

	roleResponses := make([]RoleResponse, len(roles))
	for i, role := range roles {
		permissions := make([]PermissionResponse, len(role.Permissions))
		for j, perm := range role.Permissions {
			permissions[j] = PermissionResponse{
				ID:          perm.ID,
				Resource:    perm.Resource,
				Action:      perm.Action,
				Scope:       perm.Scope,
				Description: perm.Description,
			}
		}

		// 获取用户数量
		userCount, _ := s.roleRepo.GetUserCount(role.ID)

		roleResponses[i] = RoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			Permissions: permissions,
			UserCount:   userCount,
			CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   role.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &RoleListResponse{
		Roles: roleResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// GetRole 获取角色详情
func (s *RoleService) GetRole(id uint) (*RoleResponse, error) {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %v", err)
	}

	permissions := make([]PermissionResponse, len(role.Permissions))
	for i, perm := range role.Permissions {
		permissions[i] = PermissionResponse{
			ID:          perm.ID,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Scope:       perm.Scope,
			Description: perm.Description,
		}
	}

	// 获取用户数量
	userCount, _ := s.roleRepo.GetUserCount(role.ID)

	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Permissions: permissions,
		UserCount:   userCount,
		CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   role.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(req *CreateRoleRequest) (*RoleResponse, error) {
	// 检查角色名是否已存在
	if _, err := s.roleRepo.GetByName(req.Name); err == nil {
		return nil, errors.New("role name already exists")
	}

	role := &models.Role{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := s.roleRepo.Create(role); err != nil {
		return nil, fmt.Errorf("failed to create role: %v", err)
	}

	// 分配权限
	if len(req.PermissionIDs) > 0 {
		if err := s.roleRepo.AssignPermissions(role.ID, req.PermissionIDs); err != nil {
			return nil, fmt.Errorf("failed to assign permissions: %v", err)
		}
		
		// 重新获取角色信息（包含权限）
		role, err := s.roleRepo.GetByID(role.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated role: %v", err)
		}
		
		permissions := make([]PermissionResponse, len(role.Permissions))
		for i, perm := range role.Permissions {
			permissions[i] = PermissionResponse{
				ID:          perm.ID,
				Resource:    perm.Resource,
				Action:      perm.Action,
				Scope:       perm.Scope,
				Description: perm.Description,
			}
		}

		return &RoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			DisplayName: role.DisplayName,
			Description: role.Description,
			IsSystem:    role.IsSystem,
			Permissions: permissions,
			UserCount:   0,
			CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   role.UpdatedAt.Format("2006-01-02 15:04:05"),
		}, nil
	}

	return &RoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		DisplayName: role.DisplayName,
		Description: role.Description,
		IsSystem:    role.IsSystem,
		Permissions: []PermissionResponse{},
		UserCount:   0,
		CreatedAt:   role.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   role.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdateRole 更新角色
func (s *RoleService) UpdateRole(id uint, req *UpdateRoleRequest) (*RoleResponse, error) {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %v", err)
	}

	// 检查是否为系统角色
	if role.IsSystem {
		return nil, errors.New("cannot update system role")
	}

	// 更新字段
	role.DisplayName = req.DisplayName
	role.Description = req.Description

	if err := s.roleRepo.Update(role); err != nil {
		return nil, fmt.Errorf("failed to update role: %v", err)
	}

	return s.GetRole(id)
}

// UpdateRolePermissions 更新角色权限
func (s *RoleService) UpdateRolePermissions(id uint, req *UpdateRolePermissionsRequest) error {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get role: %v", err)
	}

	// 检查是否为系统角色
	if role.IsSystem {
		return errors.New("cannot update system role permissions")
	}

	// 更新权限
	if err := s.roleRepo.AssignPermissions(id, req.PermissionIDs); err != nil {
		return fmt.Errorf("failed to update role permissions: %v", err)
	}

	return nil
}

// DeleteRole 删除角色
func (s *RoleService) DeleteRole(id uint) error {
	role, err := s.roleRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get role: %v", err)
	}

	// 检查是否为系统角色
	if role.IsSystem {
		return errors.New("cannot delete system role")
	}

	// 检查是否有用户使用该角色
	userCount, err := s.roleRepo.GetUserCount(id)
	if err != nil {
		return fmt.Errorf("failed to check role usage: %v", err)
	}

	if userCount > 0 {
		return errors.New("cannot delete role that is assigned to users")
	}

	if err := s.roleRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete role: %v", err)
	}

	return nil
}