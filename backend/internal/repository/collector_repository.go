package repository

import (
	"errors"
	"time"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// CollectorRepository 采集器配置仓库接口
type CollectorRepository interface {
	Create(collector *models.CollectorScript) error
	GetByID(id uint) (*models.CollectorScript, error)
	GetByDeviceID(deviceID uint) (*models.CollectorScript, error)
	Update(collector *models.CollectorScript) error
	Delete(id uint) error
	DeleteByDeviceID(deviceID uint) error
	
	// 状态更新
	UpdateStatus(deviceID uint, status models.CollectorStatus, errorMsg string) error
	UpdateDeployedAt(deviceID uint) error
	UpdateLastPushAt(deviceID uint) error
	IncrementPushCount(deviceID uint) error
	
	// 配置更新
	UpdateEnabled(deviceID uint, enabled bool) error
	UpdateInterval(deviceID uint, intervalMs int) error
	UpdateConfig(deviceID uint, intervalMs, pushBatchSize int) error
	
	// 查询
	GetAllEnabled() ([]*models.CollectorScript, error)
	GetByStatus(status models.CollectorStatus) ([]*models.CollectorScript, error)
	
	// 获取或创建
	GetOrCreate(deviceID uint) (*models.CollectorScript, error)
}

// collectorRepository 采集器配置仓库实现
type collectorRepository struct {
	db *gorm.DB
}

// NewCollectorRepository 创建新的采集器配置仓库
func NewCollectorRepository(db *gorm.DB) CollectorRepository {
	return &collectorRepository{db: db}
}

// Create 创建采集器配置
func (r *collectorRepository) Create(collector *models.CollectorScript) error {
	if collector == nil {
		return errors.New("collector cannot be nil")
	}
	return r.db.Create(collector).Error
}

// GetByID 根据ID获取采集器配置
func (r *collectorRepository) GetByID(id uint) (*models.CollectorScript, error) {
	var collector models.CollectorScript
	err := r.db.First(&collector, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("collector not found")
		}
		return nil, err
	}
	return &collector, nil
}

// GetByDeviceID 根据设备ID获取采集器配置
func (r *collectorRepository) GetByDeviceID(deviceID uint) (*models.CollectorScript, error) {
	var collector models.CollectorScript
	err := r.db.Where("device_id = ?", deviceID).First(&collector).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 返回 nil 表示不存在，而不是错误
		}
		return nil, err
	}
	return &collector, nil
}

// Update 更新采集器配置
func (r *collectorRepository) Update(collector *models.CollectorScript) error {
	if collector == nil {
		return errors.New("collector cannot be nil")
	}
	return r.db.Save(collector).Error
}

// Delete 删除采集器配置
func (r *collectorRepository) Delete(id uint) error {
	return r.db.Delete(&models.CollectorScript{}, id).Error
}

// DeleteByDeviceID 根据设备ID删除采集器配置
func (r *collectorRepository) DeleteByDeviceID(deviceID uint) error {
	return r.db.Where("device_id = ?", deviceID).Delete(&models.CollectorScript{}).Error
}

// UpdateStatus 更新采集器状态
func (r *collectorRepository) UpdateStatus(deviceID uint, status models.CollectorStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status":        status,
		"error_message": errorMsg,
		"updated_at":    time.Now(),
	}
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).Updates(updates).Error
}

// UpdateDeployedAt 更新部署时间
func (r *collectorRepository) UpdateDeployedAt(deviceID uint) error {
	now := time.Now()
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).Updates(map[string]interface{}{
		"deployed_at": &now,
		"status":      models.CollectorStatusDeployed,
		"updated_at":  now,
	}).Error
}

// UpdateLastPushAt 更新最后推送时间
func (r *collectorRepository) UpdateLastPushAt(deviceID uint) error {
	now := time.Now()
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).Updates(map[string]interface{}{
		"last_push_at": &now,
		"status":       models.CollectorStatusRunning,
		"updated_at":   now,
	}).Error
}

// IncrementPushCount 增加推送计数
func (r *collectorRepository) IncrementPushCount(deviceID uint) error {
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).
		UpdateColumn("push_count", gorm.Expr("push_count + ?", 1)).Error
}

// UpdateEnabled 更新启用状态
func (r *collectorRepository) UpdateEnabled(deviceID uint, enabled bool) error {
	status := models.CollectorStatusStopped
	if enabled {
		status = models.CollectorStatusRunning
	}
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).Updates(map[string]interface{}{
		"enabled":    enabled,
		"status":     status,
		"updated_at": time.Now(),
	}).Error
}

// UpdateInterval 更新采集间隔
func (r *collectorRepository) UpdateInterval(deviceID uint, intervalMs int) error {
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).Updates(map[string]interface{}{
		"interval_ms": intervalMs,
		"updated_at":  time.Now(),
	}).Error
}

// UpdateConfig 更新配置
func (r *collectorRepository) UpdateConfig(deviceID uint, intervalMs, pushBatchSize int) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if intervalMs > 0 {
		updates["interval_ms"] = intervalMs
	}
	if pushBatchSize > 0 {
		updates["push_batch_size"] = pushBatchSize
	}
	return r.db.Model(&models.CollectorScript{}).Where("device_id = ?", deviceID).Updates(updates).Error
}

// GetAllEnabled 获取所有启用的采集器配置
func (r *collectorRepository) GetAllEnabled() ([]*models.CollectorScript, error) {
	var collectors []*models.CollectorScript
	err := r.db.Where("enabled = ?", true).Find(&collectors).Error
	return collectors, err
}

// GetByStatus 根据状态获取采集器配置
func (r *collectorRepository) GetByStatus(status models.CollectorStatus) ([]*models.CollectorScript, error) {
	var collectors []*models.CollectorScript
	err := r.db.Where("status = ?", status).Find(&collectors).Error
	return collectors, err
}

// GetOrCreate 获取或创建采集器配置
func (r *collectorRepository) GetOrCreate(deviceID uint) (*models.CollectorScript, error) {
	collector, err := r.GetByDeviceID(deviceID)
	if err != nil {
		return nil, err
	}
	
	if collector != nil {
		return collector, nil
	}
	
	// 创建默认配置
	collector = &models.CollectorScript{
		DeviceID:      deviceID,
		Enabled:       false,
		IntervalMs:    1000,
		PushBatchSize: 10,
		ScriptName:    "nmp-collector",
		SchedulerName: "nmp-scheduler",
		Status:        models.CollectorStatusNotDeployed,
	}
	
	if err := r.Create(collector); err != nil {
		return nil, err
	}
	
	return collector, nil
}
