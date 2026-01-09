package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// DeviceGroupRepository 设备分组仓库接口
type DeviceGroupRepository interface {
	Create(group *models.DeviceGroup) error
	GetByID(id uint) (*models.DeviceGroup, error)
	GetByName(name string) (*models.DeviceGroup, error)
	Update(group *models.DeviceGroup) error
	Delete(id uint) error
	List(offset, limit int) ([]*models.DeviceGroup, int64, error)
	GetAll() ([]*models.DeviceGroup, error)
	GetRootGroups() ([]*models.DeviceGroup, error)
	GetChildren(parentID uint) ([]*models.DeviceGroup, error)
	GetByDeviceID(deviceID uint) ([]*models.DeviceGroup, error)
	GetDeviceCount(groupID uint) (int64, error)
}

// deviceGroupRepository 设备分组仓库实现
type deviceGroupRepository struct {
	db *gorm.DB
}

// NewDeviceGroupRepository 创建新的设备分组仓库
func NewDeviceGroupRepository(db *gorm.DB) DeviceGroupRepository {
	return &deviceGroupRepository{db: db}
}

// Create 创建设备分组
func (r *deviceGroupRepository) Create(group *models.DeviceGroup) error {
	if group == nil {
		return errors.New("device group cannot be nil")
	}

	// 检查分组名是否已存在
	var existingGroup models.DeviceGroup
	if err := r.db.Where("name = ?", group.Name).First(&existingGroup).Error; err == nil {
		return errors.New("device group name already exists")
	}

	// 如果有父分组，检查父分组是否存在
	if group.ParentID != nil {
		var parentGroup models.DeviceGroup
		if err := r.db.First(&parentGroup, *group.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("parent group not found")
			}
			return err
		}
	}

	return r.db.Create(group).Error
}

// GetByID 根据ID获取设备分组
func (r *deviceGroupRepository) GetByID(id uint) (*models.DeviceGroup, error) {
	var group models.DeviceGroup
	err := r.db.Preload("Parent").Preload("Children").Preload("Devices").First(&group, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("device group not found")
		}
		return nil, err
	}
	return &group, nil
}

// GetByName 根据名称获取设备分组
func (r *deviceGroupRepository) GetByName(name string) (*models.DeviceGroup, error) {
	var group models.DeviceGroup
	err := r.db.Preload("Parent").Preload("Children").Preload("Devices").Where("name = ?", name).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("device group not found")
		}
		return nil, err
	}
	return &group, nil
}

// Update 更新设备分组
func (r *deviceGroupRepository) Update(group *models.DeviceGroup) error {
	if group == nil {
		return errors.New("device group cannot be nil")
	}

	// 如果有父分组，检查父分组是否存在且不是自己
	if group.ParentID != nil {
		if *group.ParentID == group.ID {
			return errors.New("group cannot be its own parent")
		}
		
		var parentGroup models.DeviceGroup
		if err := r.db.First(&parentGroup, *group.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("parent group not found")
			}
			return err
		}
	}

	return r.db.Save(group).Error
}

// Delete 删除设备分组
func (r *deviceGroupRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 检查是否有子分组
		var childCount int64
		if err := tx.Model(&models.DeviceGroup{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
			return err
		}
		if childCount > 0 {
			return errors.New("cannot delete group with child groups")
		}

		// 删除分组（会自动删除设备关联）
		return tx.Select("Devices").Delete(&models.DeviceGroup{}, id).Error
	})
}

// List 获取设备分组列表
func (r *deviceGroupRepository) List(offset, limit int) ([]*models.DeviceGroup, int64, error) {
	var groups []*models.DeviceGroup
	var total int64

	// 获取总数
	if err := r.db.Model(&models.DeviceGroup{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分组列表
	err := r.db.Preload("Parent").Offset(offset).Limit(limit).Order("created_at DESC").Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}

	return groups, total, nil
}

// GetAll 获取所有设备分组
func (r *deviceGroupRepository) GetAll() ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	err := r.db.Preload("Parent").Preload("Children").Order("name").Find(&groups).Error
	return groups, err
}

// GetRootGroups 获取根分组（没有父分组的分组）
func (r *deviceGroupRepository) GetRootGroups() ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	err := r.db.Preload("Children").Where("parent_id IS NULL").Order("name").Find(&groups).Error
	return groups, err
}

// GetChildren 获取子分组
func (r *deviceGroupRepository) GetChildren(parentID uint) ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	err := r.db.Preload("Children").Where("parent_id = ?", parentID).Order("name").Find(&groups).Error
	return groups, err
}

// GetByDeviceID 根据设备ID获取分组列表
func (r *deviceGroupRepository) GetByDeviceID(deviceID uint) ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	err := r.db.Joins("JOIN device_group_members ON device_groups.id = device_group_members.device_group_id").
		Where("device_group_members.device_id = ?", deviceID).
		Order("device_groups.name").Find(&groups).Error
	return groups, err
}

// GetDeviceCount 获取分组中的设备数量
func (r *deviceGroupRepository) GetDeviceCount(groupID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.DeviceGroupMember{}).Where("device_group_id = ?", groupID).Count(&count).Error
	return count, err
}