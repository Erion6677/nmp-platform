package system_admin

import (
	"fmt"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// PermissionService 权限管理服务
type PermissionService struct {
	permRepo repository.PermissionRepository
}

// NewPermissionService 创建权限服务
func NewPermissionService(permRepo repository.PermissionRepository) *PermissionService {
	return &PermissionService{
		permRepo: permRepo,
	}
}

// CreatePermissionRequest 创建权限请求
type CreatePermissionRequest struct {
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
}

// UpdatePermissionRequest 更新权限请求
type UpdatePermissionRequest struct {
	Resource    string `json:"resource" binding:"required"`
	Action      string `json:"action" binding:"required"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
}

// PermissionListResponse 权限列表响应
type PermissionListResponse struct {
	Permissions []PermissionResponse `json:"permissions"`
	Total       int64                `json:"total"`
	Page        int                  `json:"page"`
	Size        int                  `json:"size"`
}

// PermissionResponse 权限响应
type PermissionResponse struct {
	ID          uint   `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	RoleCount   int    `json:"role_count"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ListPermissions 获取权限列表
func (s *PermissionService) ListPermissions(page, size int, search string) (*PermissionListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	permissions, total, err := s.permRepo.List(page, size, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %v", err)
	}

	permissionResponses := make([]PermissionResponse, len(permissions))
	for i, perm := range permissions {
		// 获取角色数量
		roleCount, _ := s.permRepo.GetRoleCount(perm.ID)

		permissionResponses[i] = PermissionResponse{
			ID:          perm.ID,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Scope:       perm.Scope,
			Description: perm.Description,
			RoleCount:   roleCount,
			CreatedAt:   perm.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   perm.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return &PermissionListResponse{
		Permissions: permissionResponses,
		Total:       total,
		Page:        page,
		Size:        size,
	}, nil
}

// GetPermission 获取权限详情
func (s *PermissionService) GetPermission(id uint) (*PermissionResponse, error) {
	permission, err := s.permRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %v", err)
	}

	// 获取角色数量
	roleCount, _ := s.permRepo.GetRoleCount(permission.ID)

	return &PermissionResponse{
		ID:          permission.ID,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Scope:       permission.Scope,
		Description: permission.Description,
		RoleCount:   roleCount,
		CreatedAt:   permission.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   permission.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// CreatePermission 创建权限
func (s *PermissionService) CreatePermission(req *CreatePermissionRequest) (*PermissionResponse, error) {
	// 检查权限是否已存在
	if _, err := s.permRepo.GetByResourceAction(req.Resource, req.Action); err == nil {
		return nil, fmt.Errorf("permission with resource '%s' and action '%s' already exists", req.Resource, req.Action)
	}

	permission := &models.Permission{
		Resource:    req.Resource,
		Action:      req.Action,
		Scope:       req.Scope,
		Description: req.Description,
	}

	if err := s.permRepo.Create(permission); err != nil {
		return nil, fmt.Errorf("failed to create permission: %v", err)
	}

	return &PermissionResponse{
		ID:          permission.ID,
		Resource:    permission.Resource,
		Action:      permission.Action,
		Scope:       permission.Scope,
		Description: permission.Description,
		RoleCount:   0,
		CreatedAt:   permission.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   permission.UpdatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// UpdatePermission 更新权限
func (s *PermissionService) UpdatePermission(id uint, req *UpdatePermissionRequest) (*PermissionResponse, error) {
	permission, err := s.permRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %v", err)
	}

	// 检查新的资源和操作组合是否已存在（排除当前权限）
	if existingPerm, err := s.permRepo.GetByResourceAction(req.Resource, req.Action); err == nil && existingPerm.ID != id {
		return nil, fmt.Errorf("permission with resource '%s' and action '%s' already exists", req.Resource, req.Action)
	}

	// 更新字段
	permission.Resource = req.Resource
	permission.Action = req.Action
	permission.Scope = req.Scope
	permission.Description = req.Description

	if err := s.permRepo.Update(permission); err != nil {
		return nil, fmt.Errorf("failed to update permission: %v", err)
	}

	return s.GetPermission(id)
}

// DeletePermission 删除权限
func (s *PermissionService) DeletePermission(id uint) error {
	// 检查权限是否存在
	_, err := s.permRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get permission: %v", err)
	}

	// 检查是否有角色使用该权限
	roleCount, err := s.permRepo.GetRoleCount(id)
	if err != nil {
		return fmt.Errorf("failed to check permission usage: %v", err)
	}

	if roleCount > 0 {
		return fmt.Errorf("cannot delete permission that is assigned to roles")
	}

	if err := s.permRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete permission: %v", err)
	}

	return nil
}

// GetPermissionsByResource 根据资源获取权限列表
func (s *PermissionService) GetPermissionsByResource(resource string) ([]PermissionResponse, error) {
	permissions, err := s.permRepo.GetByResource(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions by resource: %v", err)
	}

	responses := make([]PermissionResponse, len(permissions))
	for i, perm := range permissions {
		roleCount, _ := s.permRepo.GetRoleCount(perm.ID)
		responses[i] = PermissionResponse{
			ID:          perm.ID,
			Resource:    perm.Resource,
			Action:      perm.Action,
			Scope:       perm.Scope,
			Description: perm.Description,
			RoleCount:   roleCount,
			CreatedAt:   perm.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   perm.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return responses, nil
}