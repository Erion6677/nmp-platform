package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// RoleRepository 角色仓库接口
type RoleRepository interface {
	Create(role *models.Role) error
	GetByID(id uint) (*models.Role, error)
	GetByName(name string) (*models.Role, error)
	Update(role *models.Role) error
	Delete(id uint) error
	List(page, size int, search string) ([]*models.Role, int64, error)
	AddPermission(roleID, permissionID uint) error
	RemovePermission(roleID, permissionID uint) error
	GetPermissions(roleID uint) ([]*models.Permission, error)
	AssignPermissions(roleID uint, permissionIDs []uint) error
	GetUserCount(roleID uint) (int, error)
}

// roleRepository 角色仓库实现
type roleRepository struct {
	db *gorm.DB
}

// NewRoleRepository 创建新的角色仓库
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

// Create 创建角色
func (r *roleRepository) Create(role *models.Role) error {
	if role == nil {
		return errors.New("role cannot be nil")
	}

	// 检查角色名是否已存在
	var existingRole models.Role
	if err := r.db.Where("name = ?", role.Name).First(&existingRole).Error; err == nil {
		return errors.New("role name already exists")
	}

	return r.db.Create(role).Error
}

// GetByID 根据ID获取角色
func (r *roleRepository) GetByID(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// GetByName 根据名称获取角色
func (r *roleRepository) GetByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").Where("name = ?", name).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("role not found")
		}
		return nil, err
	}
	return &role, nil
}

// Update 更新角色
func (r *roleRepository) Update(role *models.Role) error {
	if role == nil {
		return errors.New("role cannot be nil")
	}

	return r.db.Save(role).Error
}

// Delete 删除角色
func (r *roleRepository) Delete(id uint) error {
	// 检查是否为系统角色
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		return err
	}

	if role.IsSystem {
		return errors.New("cannot delete system role")
	}

	return r.db.Delete(&models.Role{}, id).Error
}

// List 获取角色列表
func (r *roleRepository) List(page, size int, search string) ([]*models.Role, int64, error) {
	var roles []*models.Role
	var total int64

	query := r.db.Model(&models.Role{})
	
	// 添加搜索条件
	if search != "" {
		query = query.Where("name ILIKE ? OR display_name ILIKE ? OR description ILIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 获取角色列表
	err := query.Preload("Permissions").Offset(offset).Limit(size).Find(&roles).Error
	if err != nil {
		return nil, 0, err
	}

	return roles, total, nil
}

// AddPermission 为角色添加权限
func (r *roleRepository) AddPermission(roleID, permissionID uint) error {
	// 检查角色和权限是否存在
	var role models.Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return errors.New("role not found")
	}

	var permission models.Permission
	if err := r.db.First(&permission, permissionID).Error; err != nil {
		return errors.New("permission not found")
	}

	// 检查关联是否已存在
	var rolePermission models.RolePermission
	if err := r.db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).First(&rolePermission).Error; err == nil {
		return errors.New("permission already assigned to role")
	}

	// 创建关联
	rolePermission = models.RolePermission{
		RoleID:       roleID,
		PermissionID: permissionID,
	}

	return r.db.Create(&rolePermission).Error
}

// RemovePermission 从角色移除权限
func (r *roleRepository) RemovePermission(roleID, permissionID uint) error {
	return r.db.Where("role_id = ? AND permission_id = ?", roleID, permissionID).Delete(&models.RolePermission{}).Error
}

// GetPermissions 获取角色的所有权限
func (r *roleRepository) GetPermissions(roleID uint) ([]*models.Permission, error) {
	var permissions []*models.Permission
	
	err := r.db.Table("permissions").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Where("role_permissions.role_id = ?", roleID).
		Find(&permissions).Error
	
	return permissions, err
}

// AssignPermissions 分配权限给角色
func (r *roleRepository) AssignPermissions(roleID uint, permissionIDs []uint) error {
	// 开始事务
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除现有权限关联
	if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 添加新的权限关联
	if len(permissionIDs) > 0 {
		rolePermissions := make([]models.RolePermission, len(permissionIDs))
		for i, permissionID := range permissionIDs {
			rolePermissions[i] = models.RolePermission{
				RoleID:       roleID,
				PermissionID: permissionID,
			}
		}

		if err := tx.Create(&rolePermissions).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// GetUserCount 获取使用该角色的用户数量
func (r *roleRepository) GetUserCount(roleID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.UserRole{}).Where("role_id = ?", roleID).Count(&count).Error
	return int(count), err
}