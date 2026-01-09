package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// PermissionRepository 权限仓库接口
type PermissionRepository interface {
	Create(permission *models.Permission) error
	GetByID(id uint) (*models.Permission, error)
	GetByResourceAction(resource, action string) (*models.Permission, error)
	Update(permission *models.Permission) error
	Delete(id uint) error
	List(page, size int, search string) ([]*models.Permission, int64, error)
	ListByResource(resource string) ([]*models.Permission, error)
	ListByResourcePrefix(resourcePrefix string) ([]*models.Permission, error)
	DeleteByResourcePrefix(resourcePrefix string) error
	GetByResource(resource string) ([]*models.Permission, error)
	GetRoleCount(permissionID uint) (int, error)
	
	// 设备权限管理方法
	AssignDeviceToUser(userID, deviceID uint) error
	RemoveDeviceFromUser(userID, deviceID uint) error
	GetUserDevices(userID uint) ([]*models.Device, error)
	HasDevicePermission(userID, deviceID uint) (bool, error)
	AssignDevicesToUser(userID uint, deviceIDs []uint) error
	RemoveAllDevicesFromUser(userID uint) error
	GetDeviceUsers(deviceID uint) ([]*models.User, error)
}

// permissionRepository 权限仓库实现
type permissionRepository struct {
	db *gorm.DB
}

// NewPermissionRepository 创建新的权限仓库
func NewPermissionRepository(db *gorm.DB) PermissionRepository {
	return &permissionRepository{db: db}
}

// Create 创建权限
func (r *permissionRepository) Create(permission *models.Permission) error {
	if permission == nil {
		return errors.New("permission cannot be nil")
	}

	// 检查资源和操作组合是否已存在
	var existingPermission models.Permission
	if err := r.db.Where("resource = ? AND action = ? AND scope = ?", 
		permission.Resource, permission.Action, permission.Scope).First(&existingPermission).Error; err == nil {
		return errors.New("permission with same resource, action and scope already exists")
	}

	return r.db.Create(permission).Error
}

// GetByID 根据ID获取权限
func (r *permissionRepository) GetByID(id uint) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.First(&permission, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("permission not found")
		}
		return nil, err
	}
	return &permission, nil
}

// GetByResourceAction 根据资源和操作获取权限
func (r *permissionRepository) GetByResourceAction(resource, action string) (*models.Permission, error) {
	var permission models.Permission
	err := r.db.Where("resource = ? AND action = ?", resource, action).First(&permission).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("permission not found")
		}
		return nil, err
	}
	return &permission, nil
}

// Update 更新权限
func (r *permissionRepository) Update(permission *models.Permission) error {
	if permission == nil {
		return errors.New("permission cannot be nil")
	}

	return r.db.Save(permission).Error
}

// Delete 删除权限
func (r *permissionRepository) Delete(id uint) error {
	return r.db.Delete(&models.Permission{}, id).Error
}

// List 获取权限列表
func (r *permissionRepository) List(page, size int, search string) ([]*models.Permission, int64, error) {
	var permissions []*models.Permission
	var total int64

	query := r.db.Model(&models.Permission{})
	
	// 添加搜索条件
	if search != "" {
		query = query.Where("resource ILIKE ? OR action ILIKE ? OR description ILIKE ?", 
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * size

	// 获取权限列表
	err := query.Offset(offset).Limit(size).Find(&permissions).Error
	if err != nil {
		return nil, 0, err
	}

	return permissions, total, nil
}

// ListByResource 根据资源获取权限列表
func (r *permissionRepository) ListByResource(resource string) ([]*models.Permission, error) {
	var permissions []*models.Permission
	err := r.db.Where("resource = ?", resource).Find(&permissions).Error
	return permissions, err
}

// ListByResourcePrefix 根据资源前缀获取权限列表
func (r *permissionRepository) ListByResourcePrefix(resourcePrefix string) ([]*models.Permission, error) {
	var permissions []*models.Permission
	err := r.db.Where("resource LIKE ?", resourcePrefix+"%").Find(&permissions).Error
	return permissions, err
}

// DeleteByResourcePrefix 根据资源前缀删除权限
func (r *permissionRepository) DeleteByResourcePrefix(resourcePrefix string) error {
	return r.db.Where("resource LIKE ?", resourcePrefix+"%").Delete(&models.Permission{}).Error
}

// GetByResource 根据资源获取权限列表（别名方法）
func (r *permissionRepository) GetByResource(resource string) ([]*models.Permission, error) {
	return r.ListByResource(resource)
}

// GetRoleCount 获取使用该权限的角色数量
func (r *permissionRepository) GetRoleCount(permissionID uint) (int, error) {
	var count int64
	err := r.db.Model(&models.RolePermission{}).Where("permission_id = ?", permissionID).Count(&count).Error
	return int(count), err
}

// AssignDeviceToUser 分配设备权限给用户
func (r *permissionRepository) AssignDeviceToUser(userID, deviceID uint) error {
	// 检查用户是否存在
	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 检查设备是否存在
	var device models.Device
	if err := r.db.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("device not found")
		}
		return err
	}

	// 检查权限是否已存在
	var existingPermission models.UserDevicePermission
	err := r.db.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&existingPermission).Error
	if err == nil {
		// 权限已存在，直接返回成功
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 创建新的权限记录
	permission := &models.UserDevicePermission{
		UserID:   userID,
		DeviceID: deviceID,
	}
	return r.db.Create(permission).Error
}

// RemoveDeviceFromUser 移除用户的设备权限
func (r *permissionRepository) RemoveDeviceFromUser(userID, deviceID uint) error {
	result := r.db.Where("user_id = ? AND device_id = ?", userID, deviceID).Delete(&models.UserDevicePermission{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("device permission not found")
	}
	return nil
}

// GetUserDevices 获取用户的设备列表
func (r *permissionRepository) GetUserDevices(userID uint) ([]*models.Device, error) {
	// 检查用户是否存在
	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	// 获取用户有权限的设备ID列表
	var permissions []models.UserDevicePermission
	if err := r.db.Where("user_id = ?", userID).Find(&permissions).Error; err != nil {
		return nil, err
	}

	if len(permissions) == 0 {
		return []*models.Device{}, nil
	}

	// 获取设备ID列表
	deviceIDs := make([]uint, len(permissions))
	for i, p := range permissions {
		deviceIDs[i] = p.DeviceID
	}

	// 查询设备详情
	var devices []*models.Device
	if err := r.db.Where("id IN ?", deviceIDs).Find(&devices).Error; err != nil {
		return nil, err
	}

	return devices, nil
}

// HasDevicePermission 检查用户是否有设备权限
func (r *permissionRepository) HasDevicePermission(userID, deviceID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.UserDevicePermission{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AssignDevicesToUser 批量分配设备权限给用户
func (r *permissionRepository) AssignDevicesToUser(userID uint, deviceIDs []uint) error {
	if len(deviceIDs) == 0 {
		return nil
	}

	// 检查用户是否存在
	var user models.User
	if err := r.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 开始事务
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if rec := recover(); rec != nil {
			tx.Rollback()
		}
	}()

	for _, deviceID := range deviceIDs {
		// 检查设备是否存在
		var device models.Device
		if err := tx.First(&device, deviceID).Error; err != nil {
			tx.Rollback()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("device not found")
			}
			return err
		}

		// 检查权限是否已存在
		var existingPermission models.UserDevicePermission
		err := tx.Where("user_id = ? AND device_id = ?", userID, deviceID).First(&existingPermission).Error
		if err == nil {
			// 权限已存在，跳过
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return err
		}

		// 创建新的权限记录
		permission := &models.UserDevicePermission{
			UserID:   userID,
			DeviceID: deviceID,
		}
		if err := tx.Create(permission).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RemoveAllDevicesFromUser 移除用户的所有设备权限
func (r *permissionRepository) RemoveAllDevicesFromUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.UserDevicePermission{}).Error
}

// GetDeviceUsers 获取有设备权限的用户列表
func (r *permissionRepository) GetDeviceUsers(deviceID uint) ([]*models.User, error) {
	// 检查设备是否存在
	var device models.Device
	if err := r.db.First(&device, deviceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("device not found")
		}
		return nil, err
	}

	// 获取有权限的用户ID列表
	var permissions []models.UserDevicePermission
	if err := r.db.Where("device_id = ?", deviceID).Find(&permissions).Error; err != nil {
		return nil, err
	}

	if len(permissions) == 0 {
		return []*models.User{}, nil
	}

	// 获取用户ID列表
	userIDs := make([]uint, len(permissions))
	for i, p := range permissions {
		userIDs[i] = p.UserID
	}

	// 查询用户详情
	var users []*models.User
	if err := r.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, err
	}

	return users, nil
}