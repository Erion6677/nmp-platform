package service

import (
	"errors"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// DeviceGroupService 设备分组服务接口
type DeviceGroupService interface {
	CreateGroup(group *models.DeviceGroup) error
	GetGroup(id uint) (*models.DeviceGroup, error)
	GetGroupByName(name string) (*models.DeviceGroup, error)
	UpdateGroup(group *models.DeviceGroup) error
	DeleteGroup(id uint) error
	ListGroups(offset, limit int) ([]*models.DeviceGroup, int64, error)
	GetAllGroups() ([]*models.DeviceGroup, error)
	GetRootGroups() ([]*models.DeviceGroup, error)
	GetChildGroups(parentID uint) ([]*models.DeviceGroup, error)
	GetGroupsByDevice(deviceID uint) ([]*models.DeviceGroup, error)
	GetGroupDeviceCount(groupID uint) (int64, error)
}

// deviceGroupService 设备分组服务实现
type deviceGroupService struct {
	groupRepo repository.DeviceGroupRepository
}

// NewDeviceGroupService 创建新的设备分组服务
func NewDeviceGroupService(groupRepo repository.DeviceGroupRepository) DeviceGroupService {
	return &deviceGroupService{
		groupRepo: groupRepo,
	}
}

// CreateGroup 创建设备分组
func (s *deviceGroupService) CreateGroup(group *models.DeviceGroup) error {
	if group == nil {
		return errors.New("device group cannot be nil")
	}

	// 验证必填字段
	if group.Name == "" {
		return errors.New("group name is required")
	}

	return s.groupRepo.Create(group)
}

// GetGroup 获取设备分组
func (s *deviceGroupService) GetGroup(id uint) (*models.DeviceGroup, error) {
	if id == 0 {
		return nil, errors.New("invalid group id")
	}

	return s.groupRepo.GetByID(id)
}

// GetGroupByName 根据名称获取设备分组
func (s *deviceGroupService) GetGroupByName(name string) (*models.DeviceGroup, error) {
	if name == "" {
		return nil, errors.New("group name cannot be empty")
	}

	return s.groupRepo.GetByName(name)
}

// UpdateGroup 更新设备分组
func (s *deviceGroupService) UpdateGroup(group *models.DeviceGroup) error {
	if group == nil {
		return errors.New("device group cannot be nil")
	}
	if group.ID == 0 {
		return errors.New("invalid group id")
	}

	// 验证必填字段
	if group.Name == "" {
		return errors.New("group name is required")
	}

	// 检查分组是否存在
	existing, err := s.groupRepo.GetByID(group.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("group not found")
	}

	return s.groupRepo.Update(group)
}

// DeleteGroup 删除设备分组
func (s *deviceGroupService) DeleteGroup(id uint) error {
	if id == 0 {
		return errors.New("invalid group id")
	}

	// 检查分组是否存在
	existing, err := s.groupRepo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("group not found")
	}

	return s.groupRepo.Delete(id)
}

// ListGroups 获取设备分组列表
func (s *deviceGroupService) ListGroups(offset, limit int) ([]*models.DeviceGroup, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页20条
	}

	return s.groupRepo.List(offset, limit)
}

// GetAllGroups 获取所有设备分组
func (s *deviceGroupService) GetAllGroups() ([]*models.DeviceGroup, error) {
	return s.groupRepo.GetAll()
}

// GetRootGroups 获取根分组
func (s *deviceGroupService) GetRootGroups() ([]*models.DeviceGroup, error) {
	return s.groupRepo.GetRootGroups()
}

// GetChildGroups 获取子分组
func (s *deviceGroupService) GetChildGroups(parentID uint) ([]*models.DeviceGroup, error) {
	if parentID == 0 {
		return nil, errors.New("invalid parent group id")
	}

	return s.groupRepo.GetChildren(parentID)
}

// GetGroupsByDevice 根据设备获取分组列表
func (s *deviceGroupService) GetGroupsByDevice(deviceID uint) ([]*models.DeviceGroup, error) {
	if deviceID == 0 {
		return nil, errors.New("invalid device id")
	}

	return s.groupRepo.GetByDeviceID(deviceID)
}

// GetGroupDeviceCount 获取分组中的设备数量
func (s *deviceGroupService) GetGroupDeviceCount(groupID uint) (int64, error) {
	if groupID == 0 {
		return 0, errors.New("invalid group id")
	}

	return s.groupRepo.GetDeviceCount(groupID)
}