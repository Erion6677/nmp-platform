package repository

import (
	"errors"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// TagRepository 标签仓库接口
type TagRepository interface {
	Create(tag *models.Tag) error
	GetByID(id uint) (*models.Tag, error)
	GetByName(name string) (*models.Tag, error)
	Update(tag *models.Tag) error
	Delete(id uint) error
	List(offset, limit int) ([]*models.Tag, int64, error)
	GetAll() ([]*models.Tag, error)
	GetByDeviceID(deviceID uint) ([]*models.Tag, error)
}

// tagRepository 标签仓库实现
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository 创建新的标签仓库
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// Create 创建标签
func (r *tagRepository) Create(tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}

	// 检查标签名是否已存在
	var existingTag models.Tag
	if err := r.db.Where("name = ?", tag.Name).First(&existingTag).Error; err == nil {
		return errors.New("tag name already exists")
	}

	return r.db.Create(tag).Error
}

// GetByID 根据ID获取标签
func (r *tagRepository) GetByID(id uint) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Preload("Devices").First(&tag, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// GetByName 根据名称获取标签
func (r *tagRepository) GetByName(name string) (*models.Tag, error) {
	var tag models.Tag
	err := r.db.Preload("Devices").Where("name = ?", name).First(&tag).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("tag not found")
		}
		return nil, err
	}
	return &tag, nil
}

// Update 更新标签
func (r *tagRepository) Update(tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}

	return r.db.Save(tag).Error
}

// Delete 删除标签
func (r *tagRepository) Delete(id uint) error {
	return r.db.Select("Devices").Delete(&models.Tag{}, id).Error
}

// List 获取标签列表
func (r *tagRepository) List(offset, limit int) ([]*models.Tag, int64, error) {
	var tags []*models.Tag
	var total int64

	// 获取总数
	if err := r.db.Model(&models.Tag{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取标签列表
	err := r.db.Offset(offset).Limit(limit).Order("created_at DESC").Find(&tags).Error
	if err != nil {
		return nil, 0, err
	}

	return tags, total, nil
}

// GetAll 获取所有标签
func (r *tagRepository) GetAll() ([]*models.Tag, error) {
	var tags []*models.Tag
	err := r.db.Order("name").Find(&tags).Error
	return tags, err
}

// GetByDeviceID 根据设备ID获取标签列表
func (r *tagRepository) GetByDeviceID(deviceID uint) ([]*models.Tag, error) {
	var tags []*models.Tag
	err := r.db.Joins("JOIN device_tags ON tags.id = device_tags.tag_id").
		Where("device_tags.device_id = ?", deviceID).
		Order("tags.name").Find(&tags).Error
	return tags, err
}