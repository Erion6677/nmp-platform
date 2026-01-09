package service

import (
	"errors"
	"fmt"
	"time"

	"nmp-platform/internal/collector"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// DeviceService 设备服务接口
type DeviceService interface {
	CreateDevice(device *models.Device) error
	GetDevice(id uint) (*models.Device, error)
	GetDeviceByHost(host string) (*models.Device, error)
	UpdateDevice(device *models.Device) error
	DeleteDevice(id uint) error
	ListDevices(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error)
	UpdateDeviceStatus(id uint, status models.DeviceStatus) error
	UpdateDeviceLastSeen(id uint) error
	
	// 分组管理
	AddDeviceToGroup(deviceID, groupID uint) error
	RemoveDeviceFromGroup(deviceID, groupID uint) error
	GetDevicesByGroup(groupID uint) ([]*models.Device, error)
	
	// 标签管理
	AddDeviceTag(deviceID, tagID uint) error
	RemoveDeviceTag(deviceID, tagID uint) error
	GetDevicesByTag(tagID uint) ([]*models.Device, error)
	
	// 接口管理
	SyncDeviceInterfaces(deviceID uint, interfaces []*models.Interface) error
	GetDeviceInterfaces(deviceID uint) ([]*models.Interface, error)
	GetMonitoredInterfaces(deviceID uint) ([]*models.Interface, error)
	UpdateInterfaceMonitorStatus(interfaceID uint, monitor bool) error
	SetMonitoredInterfaces(deviceID uint, interfaceNames []string) error
	
	// 连接测试和系统信息采集
	TestConnection(device *models.Device, connectionType string) (*ConnectionTestResult, error)
	GetSystemInfo(deviceID uint) (*SystemInfoResult, error)
	SyncInterfacesFromDevice(deviceID uint) ([]*models.Interface, error)
}

// ConnectionTestResult 连接测试结果
type ConnectionTestResult struct {
	APISuccess  bool   `json:"api_success,omitempty"`
	APIError    string `json:"api_error,omitempty"`
	SSHSuccess  bool   `json:"ssh_success"`
	SSHError    string `json:"ssh_error,omitempty"`
}

// SystemInfoResult 系统信息结果
type SystemInfoResult struct {
	DeviceName   string  `json:"device_name"`
	DeviceIP     string  `json:"device_ip"`
	CPUCount     int     `json:"cpu_count"`
	Version      string  `json:"version"`
	License      string  `json:"license"`
	Uptime       int64   `json:"uptime"`
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	MemoryTotal  int64   `json:"memory_total"`
	MemoryFree   int64   `json:"memory_free"`
}

// deviceService 设备服务实现
type deviceService struct {
	deviceRepo      repository.DeviceRepository
	interfaceRepo   repository.InterfaceRepository
	tagRepo         repository.TagRepository
	groupRepo       repository.DeviceGroupRepository
	rosCollector    *collector.RouterOSCollector
	sshCollector    *collector.SSHCollector
}

// NewDeviceService 创建新的设备服务
func NewDeviceService(
	deviceRepo repository.DeviceRepository,
	interfaceRepo repository.InterfaceRepository,
	tagRepo repository.TagRepository,
	groupRepo repository.DeviceGroupRepository,
) DeviceService {
	return &deviceService{
		deviceRepo:      deviceRepo,
		interfaceRepo:   interfaceRepo,
		tagRepo:         tagRepo,
		groupRepo:       groupRepo,
		rosCollector:    collector.NewRouterOSCollector(10 * time.Second),
		sshCollector:    collector.NewSSHCollector(10 * time.Second),
	}
}

// CreateDevice 创建设备
func (s *deviceService) CreateDevice(device *models.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}

	// 验证必填字段
	if device.Name == "" {
		return errors.New("device name is required")
	}
	if device.Host == "" {
		return errors.New("device host is required")
	}

	// 设置默认值
	if device.Status == "" {
		device.Status = models.DeviceStatusUnknown
	}
	if device.Port == 0 {
		device.Port = 22 // 默认SSH端口
	}
	if device.Protocol == "" {
		device.Protocol = "ssh"
	}

	return s.deviceRepo.Create(device)
}

// GetDevice 获取设备
func (s *deviceService) GetDevice(id uint) (*models.Device, error) {
	if id == 0 {
		return nil, errors.New("invalid device id")
	}

	return s.deviceRepo.GetByID(id)
}

// GetDeviceByHost 根据主机地址获取设备
func (s *deviceService) GetDeviceByHost(host string) (*models.Device, error) {
	if host == "" {
		return nil, errors.New("host cannot be empty")
	}

	return s.deviceRepo.GetByHost(host)
}

// UpdateDevice 更新设备
func (s *deviceService) UpdateDevice(device *models.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}
	if device.ID == 0 {
		return errors.New("invalid device id")
	}

	// 验证必填字段
	if device.Name == "" {
		return errors.New("device name is required")
	}
	if device.Host == "" {
		return errors.New("device host is required")
	}

	// 检查设备是否存在
	existing, err := s.deviceRepo.GetByID(device.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("device not found")
	}

	return s.deviceRepo.Update(device)
}

// DeleteDevice 删除设备
func (s *deviceService) DeleteDevice(id uint) error {
	if id == 0 {
		return errors.New("invalid device id")
	}

	// 检查设备是否存在
	existing, err := s.deviceRepo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("device not found")
	}

	// 删除设备的所有接口
	if err := s.interfaceRepo.DeleteByDeviceID(id); err != nil {
		return fmt.Errorf("failed to delete device interfaces: %w", err)
	}

	return s.deviceRepo.Delete(id)
}

// ListDevices 获取设备列表
func (s *deviceService) ListDevices(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页20条
	}

	return s.deviceRepo.List(offset, limit, filters)
}

// UpdateDeviceStatus 更新设备状态
func (s *deviceService) UpdateDeviceStatus(id uint, status models.DeviceStatus) error {
	if id == 0 {
		return errors.New("invalid device id")
	}

	// 验证状态值
	validStatuses := map[models.DeviceStatus]bool{
		models.DeviceStatusOnline:  true,
		models.DeviceStatusOffline: true,
		models.DeviceStatusUnknown: true,
		models.DeviceStatusError:   true,
	}
	if !validStatuses[status] {
		return errors.New("invalid device status")
	}

	return s.deviceRepo.UpdateStatus(id, status)
}

// UpdateDeviceLastSeen 更新设备最后在线时间
func (s *deviceService) UpdateDeviceLastSeen(id uint) error {
	if id == 0 {
		return errors.New("invalid device id")
	}

	return s.deviceRepo.UpdateLastSeen(id)
}

// AddDeviceToGroup 将设备添加到分组
func (s *deviceService) AddDeviceToGroup(deviceID, groupID uint) error {
	if deviceID == 0 {
		return errors.New("invalid device id")
	}
	if groupID == 0 {
		return errors.New("invalid group id")
	}

	// 检查设备是否存在
	if _, err := s.deviceRepo.GetByID(deviceID); err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// 检查分组是否存在
	if _, err := s.groupRepo.GetByID(groupID); err != nil {
		return fmt.Errorf("group not found: %w", err)
	}

	return s.deviceRepo.AddToGroup(deviceID, groupID)
}

// RemoveDeviceFromGroup 从分组中移除设备
func (s *deviceService) RemoveDeviceFromGroup(deviceID, groupID uint) error {
	if deviceID == 0 {
		return errors.New("invalid device id")
	}
	if groupID == 0 {
		return errors.New("invalid group id")
	}

	return s.deviceRepo.RemoveFromGroup(deviceID, groupID)
}

// GetDevicesByGroup 根据分组获取设备列表
func (s *deviceService) GetDevicesByGroup(groupID uint) ([]*models.Device, error) {
	if groupID == 0 {
		return nil, errors.New("invalid group id")
	}

	return s.deviceRepo.GetByGroupID(groupID)
}

// AddDeviceTag 为设备添加标签
func (s *deviceService) AddDeviceTag(deviceID, tagID uint) error {
	if deviceID == 0 {
		return errors.New("invalid device id")
	}
	if tagID == 0 {
		return errors.New("invalid tag id")
	}

	// 检查设备是否存在
	if _, err := s.deviceRepo.GetByID(deviceID); err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// 检查标签是否存在
	if _, err := s.tagRepo.GetByID(tagID); err != nil {
		return fmt.Errorf("tag not found: %w", err)
	}

	return s.deviceRepo.AddTag(deviceID, tagID)
}

// RemoveDeviceTag 移除设备标签
func (s *deviceService) RemoveDeviceTag(deviceID, tagID uint) error {
	if deviceID == 0 {
		return errors.New("invalid device id")
	}
	if tagID == 0 {
		return errors.New("invalid tag id")
	}

	return s.deviceRepo.RemoveTag(deviceID, tagID)
}

// GetDevicesByTag 根据标签获取设备列表
func (s *deviceService) GetDevicesByTag(tagID uint) ([]*models.Device, error) {
	if tagID == 0 {
		return nil, errors.New("invalid tag id")
	}

	return s.deviceRepo.GetByTagID(tagID)
}

// SyncDeviceInterfaces 同步设备接口
func (s *deviceService) SyncDeviceInterfaces(deviceID uint, interfaces []*models.Interface) error {
	if deviceID == 0 {
		return errors.New("invalid device id")
	}

	// 检查设备是否存在
	if _, err := s.deviceRepo.GetByID(deviceID); err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	return s.interfaceRepo.SyncInterfaces(deviceID, interfaces)
}

// GetDeviceInterfaces 获取设备接口列表
func (s *deviceService) GetDeviceInterfaces(deviceID uint) ([]*models.Interface, error) {
	if deviceID == 0 {
		return nil, errors.New("invalid device id")
	}

	return s.interfaceRepo.GetByDeviceID(deviceID)
}

// GetMonitoredInterfaces 获取设备的监控接口
func (s *deviceService) GetMonitoredInterfaces(deviceID uint) ([]*models.Interface, error) {
	if deviceID == 0 {
		return nil, errors.New("invalid device id")
	}

	return s.interfaceRepo.GetMonitoredByDeviceID(deviceID)
}

// UpdateInterfaceMonitorStatus 更新接口监控状态
func (s *deviceService) UpdateInterfaceMonitorStatus(interfaceID uint, monitor bool) error {
	if interfaceID == 0 {
		return errors.New("invalid interface id")
	}

	// 检查接口是否存在
	if _, err := s.interfaceRepo.GetByID(interfaceID); err != nil {
		return fmt.Errorf("interface not found: %w", err)
	}

	return s.interfaceRepo.UpdateMonitorStatus(interfaceID, monitor)
}

// SetMonitoredInterfaces 批量设置监控接口
func (s *deviceService) SetMonitoredInterfaces(deviceID uint, interfaceNames []string) error {
	if deviceID == 0 {
		return errors.New("invalid device id")
	}

	// 检查设备是否存在
	if _, err := s.deviceRepo.GetByID(deviceID); err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	return s.interfaceRepo.SetMonitoredInterfaces(deviceID, interfaceNames)
}

// TestConnection 测试设备连接
// connectionType: "api", "ssh", "all"
func (s *deviceService) TestConnection(device *models.Device, connectionType string) (*ConnectionTestResult, error) {
	if device == nil {
		return nil, errors.New("device cannot be nil")
	}

	result := &ConnectionTestResult{}

	// 根据设备类型和连接类型进行测试
	switch device.OSType {
	case models.DeviceOSTypeMikroTik:
		// MikroTik 设备支持 API 和 SSH 两种连接方式
		if connectionType == "api" || connectionType == "all" {
			err := s.rosCollector.TestConnection(device.Host, device.APIPort, device.Username, device.Password)
			if err != nil {
				result.APISuccess = false
				result.APIError = err.Error()
			} else {
				result.APISuccess = true
			}
		}

		if connectionType == "ssh" || connectionType == "all" {
			err := s.sshCollector.TestConnection(device.Host, device.Port, device.Username, device.Password)
			if err != nil {
				result.SSHSuccess = false
				result.SSHError = err.Error()
			} else {
				result.SSHSuccess = true
			}
		}

	case models.DeviceOSTypeLinux:
		// Linux 设备只支持 SSH 连接
		err := s.sshCollector.TestConnection(device.Host, device.Port, device.Username, device.Password)
		if err != nil {
			result.SSHSuccess = false
			result.SSHError = err.Error()
		} else {
			result.SSHSuccess = true
		}

	default:
		return nil, fmt.Errorf("unsupported device os type: %s", device.OSType)
	}

	return result, nil
}

// GetSystemInfo 获取设备系统信息（主动采集）
func (s *deviceService) GetSystemInfo(deviceID uint) (*SystemInfoResult, error) {
	if deviceID == 0 {
		return nil, errors.New("invalid device id")
	}

	device, err := s.deviceRepo.GetByID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	var sysInfo *collector.SystemInfo

	switch device.OSType {
	case models.DeviceOSTypeMikroTik:
		// MikroTik 设备优先使用 API，失败则使用 SSH
		client, err := s.rosCollector.Connect(device.Host, device.APIPort, device.Username, device.Password)
		if err == nil {
			defer client.Close()
			sysInfo, err = s.rosCollector.GetSystemInfo(client)
			if err == nil {
				sysInfo.DeviceIP = device.Host
				return s.convertSystemInfo(sysInfo), nil
			}
		}

		// API 失败，尝试 SSH
		sshClient, err := s.sshCollector.Connect(device.Host, device.Port, device.Username, device.Password)
		if err != nil {
			return nil, fmt.Errorf("无法连接到设备: %w", err)
		}
		defer sshClient.Close()

		sysInfo, err = s.sshCollector.GetMikroTikSystemInfo(sshClient)
		if err != nil {
			return nil, fmt.Errorf("获取系统信息失败: %w", err)
		}

	case models.DeviceOSTypeLinux:
		// Linux 设备使用 SSH
		sshClient, err := s.sshCollector.Connect(device.Host, device.Port, device.Username, device.Password)
		if err != nil {
			return nil, fmt.Errorf("无法连接到设备: %w", err)
		}
		defer sshClient.Close()

		sysInfo, err = s.sshCollector.GetLinuxSystemInfo(sshClient)
		if err != nil {
			return nil, fmt.Errorf("获取系统信息失败: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported device os type: %s", device.OSType)
	}

	sysInfo.DeviceIP = device.Host
	return s.convertSystemInfo(sysInfo), nil
}

// SyncInterfacesFromDevice 从设备同步接口列表
func (s *deviceService) SyncInterfacesFromDevice(deviceID uint) ([]*models.Interface, error) {
	if deviceID == 0 {
		return nil, errors.New("invalid device id")
	}

	device, err := s.deviceRepo.GetByID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	var collectorInterfaces []collector.InterfaceInfo

	switch device.OSType {
	case models.DeviceOSTypeMikroTik:
		// MikroTik 设备优先使用 API，失败则使用 SSH
		client, err := s.rosCollector.Connect(device.Host, device.APIPort, device.Username, device.Password)
		if err == nil {
			defer client.Close()
			collectorInterfaces, err = s.rosCollector.GetInterfaces(client)
			if err == nil {
				return s.syncAndReturnInterfaces(deviceID, collectorInterfaces)
			}
		}

		// API 失败，尝试 SSH
		sshClient, err := s.sshCollector.Connect(device.Host, device.Port, device.Username, device.Password)
		if err != nil {
			return nil, fmt.Errorf("无法连接到设备: %w", err)
		}
		defer sshClient.Close()

		collectorInterfaces, err = s.sshCollector.GetMikroTikInterfaces(sshClient)
		if err != nil {
			return nil, fmt.Errorf("获取接口列表失败: %w", err)
		}

	case models.DeviceOSTypeLinux:
		// Linux 设备使用 SSH
		sshClient, err := s.sshCollector.Connect(device.Host, device.Port, device.Username, device.Password)
		if err != nil {
			return nil, fmt.Errorf("无法连接到设备: %w", err)
		}
		defer sshClient.Close()

		collectorInterfaces, err = s.sshCollector.GetLinuxInterfaces(sshClient)
		if err != nil {
			return nil, fmt.Errorf("获取接口列表失败: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported device os type: %s", device.OSType)
	}

	return s.syncAndReturnInterfaces(deviceID, collectorInterfaces)
}

// convertSystemInfo 转换系统信息
func (s *deviceService) convertSystemInfo(info *collector.SystemInfo) *SystemInfoResult {
	return &SystemInfoResult{
		DeviceName:   info.DeviceName,
		DeviceIP:     info.DeviceIP,
		CPUCount:     info.CPUCount,
		Version:      info.Version,
		License:      info.License,
		Uptime:       info.Uptime,
		CPUUsage:     info.CPUUsage,
		MemoryUsage:  info.MemoryUsage,
		MemoryTotal:  info.MemoryTotal,
		MemoryFree:   info.MemoryFree,
	}
}

// syncAndReturnInterfaces 同步接口并返回结果
func (s *deviceService) syncAndReturnInterfaces(deviceID uint, collectorInterfaces []collector.InterfaceInfo) ([]*models.Interface, error) {
	// 转换为模型接口
	interfaces := make([]*models.Interface, 0, len(collectorInterfaces))
	for _, ci := range collectorInterfaces {
		status := models.InterfaceStatusDown
		if ci.Status == "up" {
			status = models.InterfaceStatusUp
		}
		interfaces = append(interfaces, &models.Interface{
			DeviceID: deviceID,
			Name:     ci.Name,
			Status:   status,
		})
	}

	// 同步到数据库
	if err := s.interfaceRepo.SyncInterfaces(deviceID, interfaces); err != nil {
		return nil, fmt.Errorf("同步接口失败: %w", err)
	}

	// 返回最新的接口列表
	return s.interfaceRepo.GetByDeviceID(deviceID)
}