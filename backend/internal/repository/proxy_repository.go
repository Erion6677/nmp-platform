package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// ProxyRepository 代理仓库接口
type ProxyRepository interface {
	Create(proxy *models.Proxy) error
	GetByID(id uint) (*models.Proxy, error)
	GetByName(name string) (*models.Proxy, error)
	Update(proxy *models.Proxy) error
	Delete(id uint) error
	List(offset, limit int, filters map[string]interface{}) ([]*models.Proxy, int64, error)
	GetAll() ([]*models.Proxy, error)
	GetEnabled() ([]*models.Proxy, error)
	UpdateStatus(id uint, status models.ProxyStatus, lastError string) error
	GetChildren(parentID uint) ([]*models.Proxy, error)
}

// proxyRepository 代理仓库实现
type proxyRepository struct {
	db *gorm.DB
}

// NewProxyRepository 创建新的代理仓库
func NewProxyRepository(db *gorm.DB) ProxyRepository {
	return &proxyRepository{db: db}
}

// Create 创建代理
func (r *proxyRepository) Create(proxy *models.Proxy) error {
	if proxy == nil {
		return errors.New("proxy cannot be nil")
	}

	// 检查名称是否已存在
	var existingProxy models.Proxy
	if err := r.db.Where("name = ?", proxy.Name).First(&existingProxy).Error; err == nil {
		return errors.New("proxy with this name already exists")
	}

	return r.db.Create(proxy).Error
}

// GetByID 根据ID获取代理
func (r *proxyRepository) GetByID(id uint) (*models.Proxy, error) {
	var proxy models.Proxy
	err := r.db.Preload("ParentProxy").First(&proxy, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("proxy not found")
		}
		return nil, err
	}
	return &proxy, nil
}

// GetByName 根据名称获取代理
func (r *proxyRepository) GetByName(name string) (*models.Proxy, error) {
	var proxy models.Proxy
	err := r.db.Preload("ParentProxy").Where("name = ?", name).First(&proxy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("proxy not found")
		}
		return nil, err
	}
	return &proxy, nil
}

// Update 更新代理
func (r *proxyRepository) Update(proxy *models.Proxy) error {
	if proxy == nil {
		return errors.New("proxy cannot be nil")
	}

	return r.db.Save(proxy).Error
}

// Delete 删除代理
func (r *proxyRepository) Delete(id uint) error {
	// 检查是否有子代理依赖此代理
	var count int64
	r.db.Model(&models.Proxy{}).Where("parent_proxy_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("cannot delete proxy with child proxies")
	}

	// 检查是否有设备使用此代理
	r.db.Model(&models.Device{}).Where("proxy_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("cannot delete proxy used by devices")
	}

	return r.db.Delete(&models.Proxy{}, id).Error
}

// List 获取代理列表
func (r *proxyRepository) List(offset, limit int, filters map[string]interface{}) ([]*models.Proxy, int64, error) {
	var proxies []*models.Proxy
	var total int64

	query := r.db.Model(&models.Proxy{})

	// 应用过滤条件
	if filters != nil {
		if proxyType, ok := filters["type"]; ok {
			query = query.Where("type = ?", proxyType)
		}
		if status, ok := filters["status"]; ok {
			query = query.Where("status = ?", status)
		}
		if enabled, ok := filters["enabled"]; ok {
			query = query.Where("enabled = ?", enabled)
		}
		if search, ok := filters["search"]; ok {
			searchStr := "%" + search.(string) + "%"
			query = query.Where("name ILIKE ?", searchStr)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取代理列表
	err := query.Preload("ParentProxy").Offset(offset).Limit(limit).Order("created_at DESC").Find(&proxies).Error
	if err != nil {
		return nil, 0, err
	}

	return proxies, total, nil
}

// GetAll 获取所有代理
func (r *proxyRepository) GetAll() ([]*models.Proxy, error) {
	var proxies []*models.Proxy
	err := r.db.Preload("ParentProxy").Order("created_at DESC").Find(&proxies).Error
	return proxies, err
}

// GetEnabled 获取所有启用的代理
func (r *proxyRepository) GetEnabled() ([]*models.Proxy, error) {
	var proxies []*models.Proxy
	err := r.db.Preload("ParentProxy").Where("enabled = ?", true).Order("created_at DESC").Find(&proxies).Error
	return proxies, err
}

// UpdateStatus 更新代理状态
func (r *proxyRepository) UpdateStatus(id uint, status models.ProxyStatus, lastError string) error {
	updates := map[string]interface{}{
		"status":     status,
		"last_error": lastError,
	}
	return r.db.Model(&models.Proxy{}).Where("id = ?", id).Updates(updates).Error
}

// GetChildren 获取子代理列表
func (r *proxyRepository) GetChildren(parentID uint) ([]*models.Proxy, error) {
	var proxies []*models.Proxy
	err := r.db.Where("parent_proxy_id = ?", parentID).Find(&proxies).Error
	return proxies, err
}
