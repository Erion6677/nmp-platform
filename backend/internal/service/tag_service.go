package service

import (
	"errors"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// TagService 标签服务接口
type TagService interface {
	CreateTag(tag *models.Tag) error
	GetTag(id uint) (*models.Tag, error)
	GetTagByName(name string) (*models.Tag, error)
	UpdateTag(tag *models.Tag) error
	DeleteTag(id uint) error
	ListTags(offset, limit int) ([]*models.Tag, int64, error)
	GetAllTags() ([]*models.Tag, error)
	GetTagsByDevice(deviceID uint) ([]*models.Tag, error)
}

// tagService 标签服务实现
type tagService struct {
	tagRepo repository.TagRepository
}

// NewTagService 创建新的标签服务
func NewTagService(tagRepo repository.TagRepository) TagService {
	return &tagService{
		tagRepo: tagRepo,
	}
}

// CreateTag 创建标签
func (s *tagService) CreateTag(tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}

	// 验证必填字段
	if tag.Name == "" {
		return errors.New("tag name is required")
	}

	// 设置默认颜色
	if tag.Color == "" {
		tag.Color = "#007bff"
	}

	return s.tagRepo.Create(tag)
}

// GetTag 获取标签
func (s *tagService) GetTag(id uint) (*models.Tag, error) {
	if id == 0 {
		return nil, errors.New("invalid tag id")
	}

	return s.tagRepo.GetByID(id)
}

// GetTagByName 根据名称获取标签
func (s *tagService) GetTagByName(name string) (*models.Tag, error) {
	if name == "" {
		return nil, errors.New("tag name cannot be empty")
	}

	return s.tagRepo.GetByName(name)
}

// UpdateTag 更新标签
func (s *tagService) UpdateTag(tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}
	if tag.ID == 0 {
		return errors.New("invalid tag id")
	}

	// 验证必填字段
	if tag.Name == "" {
		return errors.New("tag name is required")
	}

	// 检查标签是否存在
	existing, err := s.tagRepo.GetByID(tag.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("tag not found")
	}

	return s.tagRepo.Update(tag)
}

// DeleteTag 删除标签
func (s *tagService) DeleteTag(id uint) error {
	if id == 0 {
		return errors.New("invalid tag id")
	}

	// 检查标签是否存在
	existing, err := s.tagRepo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("tag not found")
	}

	return s.tagRepo.Delete(id)
}

// ListTags 获取标签列表
func (s *tagService) ListTags(offset, limit int) ([]*models.Tag, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页20条
	}

	return s.tagRepo.List(offset, limit)
}

// GetAllTags 获取所有标签
func (s *tagService) GetAllTags() ([]*models.Tag, error) {
	return s.tagRepo.GetAll()
}

// GetTagsByDevice 根据设备获取标签列表
func (s *tagService) GetTagsByDevice(deviceID uint) ([]*models.Tag, error) {
	if deviceID == 0 {
		return nil, errors.New("invalid device id")
	}

	return s.tagRepo.GetByDeviceID(deviceID)
}