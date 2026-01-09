package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// InterfaceRepository 接口仓库接口
type InterfaceRepository interface {
	Create(iface *models.Interface) error
	GetByID(id uint) (*models.Interface, error)
	GetByDeviceID(deviceID uint) ([]*models.Interface, error)
	Update(iface *models.Interface) error
	Delete(id uint) error
	BatchCreate(interfaces []*models.Interface) error
	BatchUpdate(interfaces []*models.Interface) error
	DeleteByDeviceID(deviceID uint) error
	GetMonitoredByDeviceID(deviceID uint) ([]*models.Interface, error)
	UpdateMonitorStatus(id uint, monitor bool) error
	SyncInterfaces(deviceID uint, interfaces []*models.Interface) error
	// 批量设置监控接口
	SetMonitoredInterfaces(deviceID uint, interfaceNames []string) error
	GetByDeviceIDAndName(deviceID uint, name string) (*models.Interface, error)
}

// interfaceRepository 接口仓库实现
type interfaceRepository struct {
	db *gorm.DB
}

// NewInterfaceRepository 创建新的接口仓库
func NewInterfaceRepository(db *gorm.DB) InterfaceRepository {
	return &interfaceRepository{db: db}
}

// Create 创建接口
func (r *interfaceRepository) Create(iface *models.Interface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}

	return r.db.Create(iface).Error
}

// GetByID 根据ID获取接口
func (r *interfaceRepository) GetByID(id uint) (*models.Interface, error) {
	var iface models.Interface
	err := r.db.First(&iface, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("interface not found")
		}
		return nil, err
	}
	return &iface, nil
}

// GetByDeviceID 根据设备ID获取接口列表
func (r *interfaceRepository) GetByDeviceID(deviceID uint) ([]*models.Interface, error) {
	var interfaces []*models.Interface
	err := r.db.Where("device_id = ?", deviceID).Order("name").Find(&interfaces).Error
	return interfaces, err
}

// Update 更新接口
func (r *interfaceRepository) Update(iface *models.Interface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}

	return r.db.Save(iface).Error
}

// Delete 删除接口
func (r *interfaceRepository) Delete(id uint) error {
	return r.db.Delete(&models.Interface{}, id).Error
}

// BatchCreate 批量创建接口
func (r *interfaceRepository) BatchCreate(interfaces []*models.Interface) error {
	if len(interfaces) == 0 {
		return nil
	}

	return r.db.CreateInBatches(interfaces, 100).Error
}

// BatchUpdate 批量更新接口
func (r *interfaceRepository) BatchUpdate(interfaces []*models.Interface) error {
	if len(interfaces) == 0 {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, iface := range interfaces {
			if err := tx.Save(iface).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteByDeviceID 删除设备的所有接口
func (r *interfaceRepository) DeleteByDeviceID(deviceID uint) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&models.Interface{}).Error
}

// GetMonitoredByDeviceID 获取设备的监控接口
func (r *interfaceRepository) GetMonitoredByDeviceID(deviceID uint) ([]*models.Interface, error) {
	var interfaces []*models.Interface
	err := r.db.Where("device_id = ? AND monitored = ?", deviceID, true).Order("name").Find(&interfaces).Error
	return interfaces, err
}

// UpdateMonitorStatus 更新接口监控状态
func (r *interfaceRepository) UpdateMonitorStatus(id uint, monitored bool) error {
	return r.db.Model(&models.Interface{}).Where("id = ?", id).Update("monitored", monitored).Error
}

// SyncInterfaces 同步设备接口（只同步名称和状态）
func (r *interfaceRepository) SyncInterfaces(deviceID uint, interfaces []*models.Interface) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 获取现有接口
		var existingInterfaces []*models.Interface
		if err := tx.Where("device_id = ?", deviceID).Find(&existingInterfaces).Error; err != nil {
			return err
		}

		// 创建现有接口映射
		existingMap := make(map[string]*models.Interface)
		for _, iface := range existingInterfaces {
			existingMap[iface.Name] = iface
		}

		// 处理新接口
		for _, newIface := range interfaces {
			newIface.DeviceID = deviceID
			if existing, exists := existingMap[newIface.Name]; exists {
				// 更新现有接口（只更新状态，保留监控设置）
				existing.Status = newIface.Status
				if err := tx.Save(existing).Error; err != nil {
					return err
				}
				delete(existingMap, newIface.Name)
			} else {
				// 创建新接口
				if err := tx.Create(newIface).Error; err != nil {
					return err
				}
			}
		}

		// 删除不存在的接口（可选，根据业务需求决定）
		// for _, remaining := range existingMap {
		//     if err := tx.Delete(remaining).Error; err != nil {
		//         return err
		//     }
		// }

		return nil
	})
}

// SetMonitoredInterfaces 批量设置监控接口
// 将指定名称的接口设为监控，其他接口取消监控
func (r *interfaceRepository) SetMonitoredInterfaces(deviceID uint, interfaceNames []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先将该设备所有接口的监控状态设为 false
		if err := tx.Model(&models.Interface{}).
			Where("device_id = ?", deviceID).
			Update("monitored", false).Error; err != nil {
			return err
		}

		// 如果没有指定接口，直接返回
		if len(interfaceNames) == 0 {
			return nil
		}

		// 将指定接口的监控状态设为 true
		if err := tx.Model(&models.Interface{}).
			Where("device_id = ? AND name IN ?", deviceID, interfaceNames).
			Update("monitored", true).Error; err != nil {
			return err
		}

		return nil
	})
}

// GetByDeviceIDAndName 根据设备ID和接口名称获取接口
func (r *interfaceRepository) GetByDeviceIDAndName(deviceID uint, name string) (*models.Interface, error) {
	var iface models.Interface
	err := r.db.Where("device_id = ? AND name = ?", deviceID, name).First(&iface).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("interface not found")
		}
		return nil, err
	}
	return &iface, nil
}