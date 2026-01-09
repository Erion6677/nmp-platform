package repository

import (
	"errors"
	"time"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// DeviceRepository 设备仓库接口
type DeviceRepository interface {
	Create(device *models.Device) error
	GetByID(id uint) (*models.Device, error)
	GetByHost(host string) (*models.Device, error)
	Update(device *models.Device) error
	Delete(id uint) error
	List(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error)
	UpdateStatus(id uint, status models.DeviceStatus) error
	UpdateLastSeen(id uint) error
	GetByGroupID(groupID uint) ([]*models.Device, error)
	GetByTagID(tagID uint) ([]*models.Device, error)
	AddToGroup(deviceID, groupID uint) error
	RemoveFromGroup(deviceID, groupID uint) error
	AddTag(deviceID, tagID uint) error
	RemoveTag(deviceID, tagID uint) error
	// 监控相关方法
	GetAllOnline() ([]*models.Device, error)
	GetByOSType(osType models.DeviceOSType) ([]*models.Device, error)
	UpdateConnectionInfo(id uint, apiPort, sshPort int) error
}

// deviceRepository 设备仓库实现
type deviceRepository struct {
	db *gorm.DB
}

// NewDeviceRepository 创建新的设备仓库
func NewDeviceRepository(db *gorm.DB) DeviceRepository {
	return &deviceRepository{db: db}
}

// Create 创建设备
func (r *deviceRepository) Create(device *models.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}

	// 检查主机地址是否已存在
	var existingDevice models.Device
	if err := r.db.Where("host = ?", device.Host).First(&existingDevice).Error; err == nil {
		return errors.New("device with this host already exists")
	}

	return r.db.Create(device).Error
}

// GetByID 根据ID获取设备
func (r *deviceRepository) GetByID(id uint) (*models.Device, error) {
	var device models.Device
	err := r.db.Preload("Tags").Preload("Interfaces").Preload("Groups").First(&device, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("device not found")
		}
		return nil, err
	}
	return &device, nil
}

// GetByHost 根据主机地址获取设备
func (r *deviceRepository) GetByHost(host string) (*models.Device, error) {
	var device models.Device
	err := r.db.Preload("Tags").Preload("Interfaces").Preload("Groups").Where("host = ?", host).First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("device not found")
		}
		return nil, err
	}
	return &device, nil
}

// Update 更新设备
func (r *deviceRepository) Update(device *models.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}

	return r.db.Save(device).Error
}

// Delete 删除设备
func (r *deviceRepository) Delete(id uint) error {
	return r.db.Select("Tags", "Interfaces").Delete(&models.Device{}, id).Error
}

// List 获取设备列表
func (r *deviceRepository) List(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	var devices []*models.Device
	var total int64

	query := r.db.Model(&models.Device{})

	// 应用过滤条件
	if filters != nil {
		if deviceType, ok := filters["type"]; ok {
			query = query.Where("type = ?", deviceType)
		}
		if status, ok := filters["status"]; ok {
			query = query.Where("status = ?", status)
		}
		if groupID, ok := filters["group_id"]; ok {
			query = query.Joins("JOIN device_group_members ON devices.id = device_group_members.device_id").
				Where("device_group_members.device_group_id = ?", groupID)
		}
		if tagID, ok := filters["tag_id"]; ok {
			query = query.Joins("JOIN device_tags ON devices.id = device_tags.device_id").
				Where("device_tags.tag_id = ?", tagID)
		}
		if search, ok := filters["search"]; ok {
			searchStr := "%" + search.(string) + "%"
			query = query.Where("name ILIKE ? OR host ILIKE ? OR description ILIKE ?", searchStr, searchStr, searchStr)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取设备列表
	err := query.Preload("Tags").Preload("Groups").Offset(offset).Limit(limit).Order("created_at DESC").Find(&devices).Error
	if err != nil {
		return nil, 0, err
	}

	return devices, total, nil
}

// UpdateStatus 更新设备状态
func (r *deviceRepository) UpdateStatus(id uint, status models.DeviceStatus) error {
	now := time.Now()
	return r.db.Model(&models.Device{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    status,
		"last_seen": &now,
	}).Error
}

// UpdateLastSeen 更新最后在线时间
func (r *deviceRepository) UpdateLastSeen(id uint) error {
	now := time.Now()
	return r.db.Model(&models.Device{}).Where("id = ?", id).Update("last_seen", &now).Error
}

// GetByGroupID 根据分组ID获取设备列表
func (r *deviceRepository) GetByGroupID(groupID uint) ([]*models.Device, error) {
	var devices []*models.Device
	err := r.db.Joins("JOIN device_group_members ON devices.id = device_group_members.device_id").
		Where("device_group_members.device_group_id = ?", groupID).
		Preload("Tags").Preload("Groups").Find(&devices).Error
	return devices, err
}

// GetByTagID 根据标签ID获取设备列表
func (r *deviceRepository) GetByTagID(tagID uint) ([]*models.Device, error) {
	var devices []*models.Device
	err := r.db.Joins("JOIN device_tags ON devices.id = device_tags.device_id").
		Where("device_tags.tag_id = ?", tagID).
		Preload("Tags").Preload("Groups").Find(&devices).Error
	return devices, err
}

// AddToGroup 将设备添加到分组
func (r *deviceRepository) AddToGroup(deviceID, groupID uint) error {
	// 检查关联是否已存在
	var count int64
	r.db.Model(&models.DeviceGroupMember{}).Where("device_id = ? AND device_group_id = ?", deviceID, groupID).Count(&count)
	if count > 0 {
		return errors.New("device already in group")
	}

	member := &models.DeviceGroupMember{
		DeviceID:      deviceID,
		DeviceGroupID: groupID,
		CreatedAt:     time.Now(),
	}
	return r.db.Create(member).Error
}

// RemoveFromGroup 从分组中移除设备
func (r *deviceRepository) RemoveFromGroup(deviceID, groupID uint) error {
	return r.db.Where("device_id = ? AND device_group_id = ?", deviceID, groupID).Delete(&models.DeviceGroupMember{}).Error
}

// AddTag 为设备添加标签
func (r *deviceRepository) AddTag(deviceID, tagID uint) error {
	// 检查关联是否已存在
	var count int64
	r.db.Model(&models.DeviceTag{}).Where("device_id = ? AND tag_id = ?", deviceID, tagID).Count(&count)
	if count > 0 {
		return errors.New("device already has this tag")
	}

	tag := &models.DeviceTag{
		DeviceID:  deviceID,
		TagID:     tagID,
		CreatedAt: time.Now(),
	}
	return r.db.Create(tag).Error
}

// RemoveTag 移除设备标签
func (r *deviceRepository) RemoveTag(deviceID, tagID uint) error {
	return r.db.Where("device_id = ? AND tag_id = ?", deviceID, tagID).Delete(&models.DeviceTag{}).Error
}

// GetAllOnline 获取所有在线设备
func (r *deviceRepository) GetAllOnline() ([]*models.Device, error) {
	var devices []*models.Device
	err := r.db.Where("status = ?", models.DeviceStatusOnline).
		Preload("Tags").Preload("Interfaces").Preload("Groups").
		Find(&devices).Error
	return devices, err
}

// GetByOSType 根据操作系统类型获取设备列表
func (r *deviceRepository) GetByOSType(osType models.DeviceOSType) ([]*models.Device, error) {
	var devices []*models.Device
	err := r.db.Where("os_type = ?", osType).
		Preload("Tags").Preload("Interfaces").Preload("Groups").
		Find(&devices).Error
	return devices, err
}

// UpdateConnectionInfo 更新设备连接信息（API端口和SSH端口）
func (r *deviceRepository) UpdateConnectionInfo(id uint, apiPort, sshPort int) error {
	updates := make(map[string]interface{})
	if apiPort > 0 {
		updates["api_port"] = apiPort
	}
	if sshPort > 0 {
		updates["port"] = sshPort
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&models.Device{}).Where("id = ?", id).Updates(updates).Error
}