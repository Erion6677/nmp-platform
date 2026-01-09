package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// PingTargetRepository Ping 目标仓库接口
type PingTargetRepository interface {
	Create(target *models.PingTarget) error
	GetByID(id uint) (*models.PingTarget, error)
	GetByDeviceID(deviceID uint) ([]*models.PingTarget, error)
	GetEnabledByDeviceID(deviceID uint) ([]*models.PingTarget, error)
	Update(target *models.PingTarget) error
	Delete(id uint) error
	DeleteByDeviceID(deviceID uint) error
	UpdateEnabled(id uint, enabled bool) error
	Exists(deviceID uint, targetAddress string, sourceInterface string) (bool, error)
}

// pingTargetRepository Ping 目标仓库实现
type pingTargetRepository struct {
	db *gorm.DB
}

// NewPingTargetRepository 创建新的 Ping 目标仓库
func NewPingTargetRepository(db *gorm.DB) PingTargetRepository {
	return &pingTargetRepository{db: db}
}

// Create 创建 Ping 目标
func (r *pingTargetRepository) Create(target *models.PingTarget) error {
	if target == nil {
		return errors.New("ping target cannot be nil")
	}

	if target.TargetAddress == "" {
		return errors.New("target address is required")
	}

	if target.TargetName == "" {
		return errors.New("target name is required")
	}

	return r.db.Create(target).Error
}

// GetByID 根据 ID 获取 Ping 目标
func (r *pingTargetRepository) GetByID(id uint) (*models.PingTarget, error) {
	var target models.PingTarget
	err := r.db.First(&target, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("ping target not found")
		}
		return nil, err
	}
	return &target, nil
}

// GetByDeviceID 根据设备 ID 获取 Ping 目标列表
func (r *pingTargetRepository) GetByDeviceID(deviceID uint) ([]*models.PingTarget, error) {
	var targets []*models.PingTarget
	err := r.db.Where("device_id = ?", deviceID).Order("id").Find(&targets).Error
	return targets, err
}

// GetEnabledByDeviceID 获取设备启用的 Ping 目标列表
func (r *pingTargetRepository) GetEnabledByDeviceID(deviceID uint) ([]*models.PingTarget, error) {
	var targets []*models.PingTarget
	err := r.db.Where("device_id = ? AND enabled = ?", deviceID, true).Order("id").Find(&targets).Error
	return targets, err
}

// Update 更新 Ping 目标
func (r *pingTargetRepository) Update(target *models.PingTarget) error {
	if target == nil {
		return errors.New("ping target cannot be nil")
	}

	return r.db.Save(target).Error
}

// Delete 删除 Ping 目标
func (r *pingTargetRepository) Delete(id uint) error {
	return r.db.Delete(&models.PingTarget{}, id).Error
}

// DeleteByDeviceID 删除设备的所有 Ping 目标
func (r *pingTargetRepository) DeleteByDeviceID(deviceID uint) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&models.PingTarget{}).Error
}

// UpdateEnabled 更新 Ping 目标启用状态
func (r *pingTargetRepository) UpdateEnabled(id uint, enabled bool) error {
	return r.db.Model(&models.PingTarget{}).Where("id = ?", id).Update("enabled", enabled).Error
}

// Exists 检查设备是否已存在相同目标地址和源接口的 Ping 目标
// 重复判断条件：同设备 + 同目标IP + 同源接口
func (r *pingTargetRepository) Exists(deviceID uint, targetAddress string, sourceInterface string) (bool, error) {
	var count int64
	err := r.db.Model(&models.PingTarget{}).
		Where("device_id = ? AND target_address = ? AND source_interface = ?", deviceID, targetAddress, sourceInterface).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
