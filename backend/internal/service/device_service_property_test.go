package service

import (
	"errors"
	"testing"
	"time"

	"nmp-platform/internal/models"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// mockDeviceRepository 模拟设备仓库用于测试
type mockDeviceRepository struct {
	devices map[uint]*models.Device
	nextID  uint
}

func newMockDeviceRepository() *mockDeviceRepository {
	return &mockDeviceRepository{
		devices: make(map[uint]*models.Device),
		nextID:  1,
	}
}

func (m *mockDeviceRepository) Create(device *models.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}
	
	// 检查主机地址是否已存在
	for _, existingDevice := range m.devices {
		if existingDevice.Host == device.Host {
			return errors.New("device with this host already exists")
		}
	}
	
	device.ID = m.nextID
	m.nextID++
	m.devices[device.ID] = device
	return nil
}

func (m *mockDeviceRepository) GetByID(id uint) (*models.Device, error) {
	if device, exists := m.devices[id]; exists {
		return device, nil
	}
	return nil, errors.New("device not found")
}

func (m *mockDeviceRepository) GetByHost(host string) (*models.Device, error) {
	for _, device := range m.devices {
		if device.Host == host {
			return device, nil
		}
	}
	return nil, errors.New("device not found")
}

func (m *mockDeviceRepository) Update(device *models.Device) error {
	if device == nil {
		return errors.New("device cannot be nil")
	}
	
	if _, exists := m.devices[device.ID]; !exists {
		return errors.New("device not found")
	}
	
	m.devices[device.ID] = device
	return nil
}

func (m *mockDeviceRepository) Delete(id uint) error {
	if _, exists := m.devices[id]; !exists {
		return errors.New("device not found")
	}
	
	delete(m.devices, id)
	return nil
}

func (m *mockDeviceRepository) List(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	devices := make([]*models.Device, 0, len(m.devices))
	for _, device := range m.devices {
		// 应用过滤器
		if filters != nil {
			if deviceType, ok := filters["type"]; ok && device.Type != deviceType {
				continue
			}
			if status, ok := filters["status"]; ok && device.Status != status {
				continue
			}
		}
		devices = append(devices, device)
	}
	
	total := int64(len(devices))
	start := offset
	end := offset + limit
	
	if start > len(devices) {
		return []*models.Device{}, total, nil
	}
	if end > len(devices) {
		end = len(devices)
	}
	
	return devices[start:end], total, nil
}

func (m *mockDeviceRepository) UpdateStatus(id uint, status models.DeviceStatus) error {
	device, exists := m.devices[id]
	if !exists {
		return errors.New("device not found")
	}
	
	device.Status = status
	now := time.Now()
	device.LastSeen = &now
	return nil
}

func (m *mockDeviceRepository) UpdateLastSeen(id uint) error {
	device, exists := m.devices[id]
	if !exists {
		return errors.New("device not found")
	}
	
	now := time.Now()
	device.LastSeen = &now
	return nil
}

func (m *mockDeviceRepository) GetByGroupID(groupID uint) ([]*models.Device, error) {
	var devices []*models.Device
	for _, device := range m.devices {
		for _, group := range device.Groups {
			if group.ID == groupID {
				devices = append(devices, device)
				break
			}
		}
	}
	return devices, nil
}

func (m *mockDeviceRepository) GetByTagID(tagID uint) ([]*models.Device, error) {
	var devices []*models.Device
	for _, device := range m.devices {
		for _, tag := range device.Tags {
			if tag.ID == tagID {
				devices = append(devices, device)
				break
			}
		}
	}
	return devices, nil
}

func (m *mockDeviceRepository) AddToGroup(deviceID, groupID uint) error {
	device, exists := m.devices[deviceID]
	if !exists {
		return errors.New("device not found")
	}
	
	// 检查是否已在分组中
	for _, group := range device.Groups {
		if group.ID == groupID {
			return errors.New("device already in group")
		}
	}
	
	// 添加到分组
	device.Groups = append(device.Groups, models.DeviceGroup{ID: groupID})
	return nil
}

func (m *mockDeviceRepository) RemoveFromGroup(deviceID, groupID uint) error {
	device, exists := m.devices[deviceID]
	if !exists {
		return errors.New("device not found")
	}
	
	// 移除分组
	for i, group := range device.Groups {
		if group.ID == groupID {
			device.Groups = append(device.Groups[:i], device.Groups[i+1:]...)
			return nil
		}
	}
	
	return errors.New("device not in group")
}

func (m *mockDeviceRepository) AddTag(deviceID, tagID uint) error {
	device, exists := m.devices[deviceID]
	if !exists {
		return errors.New("device not found")
	}
	
	// 检查是否已有标签
	for _, tag := range device.Tags {
		if tag.ID == tagID {
			return errors.New("device already has this tag")
		}
	}
	
	// 添加标签
	device.Tags = append(device.Tags, models.Tag{ID: tagID})
	return nil
}

func (m *mockDeviceRepository) RemoveTag(deviceID, tagID uint) error {
	device, exists := m.devices[deviceID]
	if !exists {
		return errors.New("device not found")
	}
	
	// 移除标签
	for i, tag := range device.Tags {
		if tag.ID == tagID {
			device.Tags = append(device.Tags[:i], device.Tags[i+1:]...)
			return nil
		}
	}
	
	return errors.New("device does not have this tag")
}

func (m *mockDeviceRepository) GetAllOnline() ([]*models.Device, error) {
	var devices []*models.Device
	for _, device := range m.devices {
		if device.Status == models.DeviceStatusOnline {
			devices = append(devices, device)
		}
	}
	return devices, nil
}

func (m *mockDeviceRepository) GetByOSType(osType models.DeviceOSType) ([]*models.Device, error) {
	var devices []*models.Device
	for _, device := range m.devices {
		if device.OSType == osType {
			devices = append(devices, device)
		}
	}
	return devices, nil
}

func (m *mockDeviceRepository) UpdateConnectionInfo(id uint, apiPort, sshPort int) error {
	device, exists := m.devices[id]
	if !exists {
		return errors.New("device not found")
	}
	if apiPort > 0 {
		device.APIPort = apiPort
	}
	if sshPort > 0 {
		device.Port = sshPort
	}
	return nil
}

func (m *mockDeviceRepository) Clear() {
	m.devices = make(map[uint]*models.Device)
	m.nextID = 1
}

// mockInterfaceRepository 模拟接口仓库
type mockInterfaceRepository struct {
	interfaces map[uint]*models.Interface
	nextID     uint
}

func newMockInterfaceRepository() *mockInterfaceRepository {
	return &mockInterfaceRepository{
		interfaces: make(map[uint]*models.Interface),
		nextID:     1,
	}
}

func (m *mockInterfaceRepository) Create(iface *models.Interface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}
	iface.ID = m.nextID
	m.nextID++
	m.interfaces[iface.ID] = iface
	return nil
}

func (m *mockInterfaceRepository) GetByID(id uint) (*models.Interface, error) {
	if iface, exists := m.interfaces[id]; exists {
		return iface, nil
	}
	return nil, errors.New("interface not found")
}

func (m *mockInterfaceRepository) GetByDeviceID(deviceID uint) ([]*models.Interface, error) {
	var interfaces []*models.Interface
	for _, iface := range m.interfaces {
		if iface.DeviceID == deviceID {
			interfaces = append(interfaces, iface)
		}
	}
	return interfaces, nil
}

func (m *mockInterfaceRepository) Update(iface *models.Interface) error {
	if iface == nil {
		return errors.New("interface cannot be nil")
	}
	if _, exists := m.interfaces[iface.ID]; !exists {
		return errors.New("interface not found")
	}
	m.interfaces[iface.ID] = iface
	return nil
}

func (m *mockInterfaceRepository) Delete(id uint) error {
	if _, exists := m.interfaces[id]; !exists {
		return errors.New("interface not found")
	}
	delete(m.interfaces, id)
	return nil
}

func (m *mockInterfaceRepository) BatchCreate(interfaces []*models.Interface) error {
	for _, iface := range interfaces {
		if err := m.Create(iface); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockInterfaceRepository) BatchUpdate(interfaces []*models.Interface) error {
	for _, iface := range interfaces {
		if err := m.Update(iface); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockInterfaceRepository) DeleteByDeviceID(deviceID uint) error {
	for id, iface := range m.interfaces {
		if iface.DeviceID == deviceID {
			delete(m.interfaces, id)
		}
	}
	return nil
}

func (m *mockInterfaceRepository) GetMonitoredByDeviceID(deviceID uint) ([]*models.Interface, error) {
	var interfaces []*models.Interface
	for _, iface := range m.interfaces {
		if iface.DeviceID == deviceID && iface.Monitored {
			interfaces = append(interfaces, iface)
		}
	}
	return interfaces, nil
}

func (m *mockInterfaceRepository) UpdateMonitorStatus(id uint, monitored bool) error {
	iface, exists := m.interfaces[id]
	if !exists {
		return errors.New("interface not found")
	}
	iface.Monitored = monitored
	return nil
}

func (m *mockInterfaceRepository) SetMonitoredInterfaces(deviceID uint, interfaceNames []string) error {
	// 先将该设备所有接口的监控状态设为 false
	for _, iface := range m.interfaces {
		if iface.DeviceID == deviceID {
			iface.Monitored = false
		}
	}
	
	// 将指定接口的监控状态设为 true
	for _, iface := range m.interfaces {
		if iface.DeviceID == deviceID {
			for _, name := range interfaceNames {
				if iface.Name == name {
					iface.Monitored = true
					break
				}
			}
		}
	}
	return nil
}

func (m *mockInterfaceRepository) GetByDeviceIDAndName(deviceID uint, name string) (*models.Interface, error) {
	for _, iface := range m.interfaces {
		if iface.DeviceID == deviceID && iface.Name == name {
			return iface, nil
		}
	}
	return nil, errors.New("interface not found")
}

func (m *mockInterfaceRepository) SyncInterfaces(deviceID uint, interfaces []*models.Interface) error {
	// 删除现有接口
	m.DeleteByDeviceID(deviceID)
	
	// 添加新接口
	for _, iface := range interfaces {
		iface.DeviceID = deviceID
		if err := m.Create(iface); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockInterfaceRepository) Clear() {
	m.interfaces = make(map[uint]*models.Interface)
	m.nextID = 1
}

// mockTagRepository 模拟标签仓库
type mockTagRepository struct {
	tags   map[uint]*models.Tag
	nextID uint
}

func newMockTagRepository() *mockTagRepository {
	return &mockTagRepository{
		tags:   make(map[uint]*models.Tag),
		nextID: 1,
	}
}

func (m *mockTagRepository) Create(tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}
	tag.ID = m.nextID
	m.nextID++
	m.tags[tag.ID] = tag
	return nil
}

func (m *mockTagRepository) GetByID(id uint) (*models.Tag, error) {
	if tag, exists := m.tags[id]; exists {
		return tag, nil
	}
	return nil, errors.New("tag not found")
}

func (m *mockTagRepository) GetByName(name string) (*models.Tag, error) {
	for _, tag := range m.tags {
		if tag.Name == name {
			return tag, nil
		}
	}
	return nil, errors.New("tag not found")
}

func (m *mockTagRepository) Update(tag *models.Tag) error {
	if tag == nil {
		return errors.New("tag cannot be nil")
	}
	if _, exists := m.tags[tag.ID]; !exists {
		return errors.New("tag not found")
	}
	m.tags[tag.ID] = tag
	return nil
}

func (m *mockTagRepository) Delete(id uint) error {
	if _, exists := m.tags[id]; !exists {
		return errors.New("tag not found")
	}
	delete(m.tags, id)
	return nil
}

func (m *mockTagRepository) List(offset, limit int) ([]*models.Tag, int64, error) {
	tags := make([]*models.Tag, 0, len(m.tags))
	for _, tag := range m.tags {
		tags = append(tags, tag)
	}
	
	total := int64(len(tags))
	start := offset
	end := offset + limit
	
	if start > len(tags) {
		return []*models.Tag{}, total, nil
	}
	if end > len(tags) {
		end = len(tags)
	}
	
	return tags[start:end], total, nil
}

func (m *mockTagRepository) GetAll() ([]*models.Tag, error) {
	tags := make([]*models.Tag, 0, len(m.tags))
	for _, tag := range m.tags {
		tags = append(tags, tag)
	}
	return tags, nil
}

func (m *mockTagRepository) GetByDeviceID(deviceID uint) ([]*models.Tag, error) {
	// 这个方法在实际实现中会通过关联表查询，这里简化处理
	return []*models.Tag{}, nil
}

func (m *mockTagRepository) Clear() {
	m.tags = make(map[uint]*models.Tag)
	m.nextID = 1
}

// mockDeviceGroupRepository 模拟设备分组仓库
type mockDeviceGroupRepository struct {
	groups map[uint]*models.DeviceGroup
	nextID uint
}

func newMockDeviceGroupRepository() *mockDeviceGroupRepository {
	return &mockDeviceGroupRepository{
		groups: make(map[uint]*models.DeviceGroup),
		nextID: 1,
	}
}

func (m *mockDeviceGroupRepository) Create(group *models.DeviceGroup) error {
	if group == nil {
		return errors.New("device group cannot be nil")
	}
	group.ID = m.nextID
	m.nextID++
	m.groups[group.ID] = group
	return nil
}

func (m *mockDeviceGroupRepository) GetByID(id uint) (*models.DeviceGroup, error) {
	if group, exists := m.groups[id]; exists {
		return group, nil
	}
	return nil, errors.New("device group not found")
}

func (m *mockDeviceGroupRepository) GetByName(name string) (*models.DeviceGroup, error) {
	for _, group := range m.groups {
		if group.Name == name {
			return group, nil
		}
	}
	return nil, errors.New("device group not found")
}

func (m *mockDeviceGroupRepository) Update(group *models.DeviceGroup) error {
	if group == nil {
		return errors.New("device group cannot be nil")
	}
	if _, exists := m.groups[group.ID]; !exists {
		return errors.New("device group not found")
	}
	m.groups[group.ID] = group
	return nil
}

func (m *mockDeviceGroupRepository) Delete(id uint) error {
	if _, exists := m.groups[id]; !exists {
		return errors.New("device group not found")
	}
	delete(m.groups, id)
	return nil
}

func (m *mockDeviceGroupRepository) List(offset, limit int) ([]*models.DeviceGroup, int64, error) {
	groups := make([]*models.DeviceGroup, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}
	
	total := int64(len(groups))
	start := offset
	end := offset + limit
	
	if start > len(groups) {
		return []*models.DeviceGroup{}, total, nil
	}
	if end > len(groups) {
		end = len(groups)
	}
	
	return groups[start:end], total, nil
}

func (m *mockDeviceGroupRepository) GetAll() ([]*models.DeviceGroup, error) {
	groups := make([]*models.DeviceGroup, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}
	return groups, nil
}

func (m *mockDeviceGroupRepository) GetRootGroups() ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	for _, group := range m.groups {
		if group.ParentID == nil {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func (m *mockDeviceGroupRepository) GetChildren(parentID uint) ([]*models.DeviceGroup, error) {
	var groups []*models.DeviceGroup
	for _, group := range m.groups {
		if group.ParentID != nil && *group.ParentID == parentID {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func (m *mockDeviceGroupRepository) GetByDeviceID(deviceID uint) ([]*models.DeviceGroup, error) {
	// 这个方法在实际实现中会通过关联表查询，这里简化处理
	return []*models.DeviceGroup{}, nil
}

func (m *mockDeviceGroupRepository) GetDeviceCount(groupID uint) (int64, error) {
	// 简化实现
	return 0, nil
}

func (m *mockDeviceGroupRepository) Clear() {
	m.groups = make(map[uint]*models.DeviceGroup)
	m.nextID = 1
}

// 生成器函数
func genValidDeviceName() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) >= 3 && len(s) <= 50
	})
}

func genValidHost() gopter.Gen {
	return gen.OneGenOf(
		// IP地址格式
		gen.RegexMatch(`^192\.168\.\d{1,3}\.\d{1,3}$`),
		// 域名格式
		gen.RegexMatch(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]\.[a-zA-Z]{2,}$`),
	)
}

func genDeviceType() gopter.Gen {
	return gen.OneConstOf(
		models.DeviceTypeRouter,
		models.DeviceTypeSwitch,
		models.DeviceTypeFirewall,
		models.DeviceTypeServer,
		models.DeviceTypeOther,
	)
}

func genDeviceStatus() gopter.Gen {
	return gen.OneConstOf(
		models.DeviceStatusOnline,
		models.DeviceStatusOffline,
		models.DeviceStatusUnknown,
		models.DeviceStatusError,
	)
}

func genValidPort() gopter.Gen {
	return gen.IntRange(1, 65535)
}

// TestDeviceManagementCRUDConsistency 测试设备管理CRUD一致性属性
// Feature: network-monitoring-platform, Property 6: 设备管理CRUD一致性
// 对于任何设备管理操作（添加、编辑、删除、查看），系统应该正确验证设备信息、更新数据库状态，并支持分组和标签管理
// **验证需求: 5.1, 5.2, 5.3, 5.4, 5.5**
func TestDeviceManagementCRUDConsistency(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 创建模拟仓库和服务
	deviceRepo := newMockDeviceRepository()
	interfaceRepo := newMockInterfaceRepository()
	tagRepo := newMockTagRepository()
	groupRepo := newMockDeviceGroupRepository()
	
	deviceService := NewDeviceService(deviceRepo, interfaceRepo, tagRepo, groupRepo)

	// 属性1: 创建设备后应该能够通过ID获取到相同的设备信息
	properties.Property("created device should be retrievable with consistent data", 
		prop.ForAll(
			func(name, host string, deviceType models.DeviceType, port int) bool {
				// 清理数据
				deviceRepo.Clear()
				
				// 创建设备
				device := &models.Device{
					Name:     name,
					Host:     host,
					Type:     deviceType,
					Port:     port,
					Protocol: "ssh",
					Status:   models.DeviceStatusUnknown,
				}
				
				err := deviceService.CreateDevice(device)
				if err != nil {
					return false
				}
				
				// 获取设备
				retrievedDevice, err := deviceService.GetDevice(device.ID)
				if err != nil {
					return false
				}
				
				// 验证数据一致性
				return retrievedDevice.Name == name &&
					   retrievedDevice.Host == host &&
					   retrievedDevice.Type == deviceType &&
					   retrievedDevice.Port == port &&
					   retrievedDevice.Protocol == "ssh" &&
					   retrievedDevice.Status == models.DeviceStatusUnknown
			},
			genValidDeviceName(),
			genValidHost(),
			genDeviceType(),
			genValidPort(),
		))

	// 属性2: 更新设备信息后应该能够获取到更新后的信息
	properties.Property("updated device should reflect changes consistently", 
		prop.ForAll(
			func(name, host, newName string, deviceType models.DeviceType, port int) bool {
				// 确保新名称与原名称不同
				if name == newName {
					return true // 跳过这个测试用例
				}
				
				// 清理数据
				deviceRepo.Clear()
				
				// 创建设备
				device := &models.Device{
					Name:     name,
					Host:     host,
					Type:     deviceType,
					Port:     port,
					Protocol: "ssh",
					Status:   models.DeviceStatusUnknown,
				}
				
				err := deviceService.CreateDevice(device)
				if err != nil {
					return false
				}
				
				// 更新设备
				device.Name = newName
				err = deviceService.UpdateDevice(device)
				if err != nil {
					return false
				}
				
				// 获取更新后的设备
				updatedDevice, err := deviceService.GetDevice(device.ID)
				if err != nil {
					return false
				}
				
				// 验证更新是否生效
				return updatedDevice.Name == newName &&
					   updatedDevice.Host == host &&
					   updatedDevice.Type == deviceType
			},
			genValidDeviceName(),
			genValidHost(),
			genValidDeviceName(),
			genDeviceType(),
			genValidPort(),
		))

	// 属性3: 删除设备后应该无法再获取到该设备
	properties.Property("deleted device should not be retrievable", 
		prop.ForAll(
			func(name, host string, deviceType models.DeviceType, port int) bool {
				// 清理数据
				deviceRepo.Clear()
				interfaceRepo.Clear()
				
				// 创建设备
				device := &models.Device{
					Name:     name,
					Host:     host,
					Type:     deviceType,
					Port:     port,
					Protocol: "ssh",
					Status:   models.DeviceStatusUnknown,
				}
				
				err := deviceService.CreateDevice(device)
				if err != nil {
					return false
				}
				
				deviceID := device.ID
				
				// 删除设备
				err = deviceService.DeleteDevice(deviceID)
				if err != nil {
					return false
				}
				
				// 尝试获取已删除的设备
				_, err = deviceService.GetDevice(deviceID)
				// 应该返回错误
				return err != nil
			},
			genValidDeviceName(),
			genValidHost(),
			genDeviceType(),
			genValidPort(),
		))

	// 属性4: 设备状态更新应该正确反映在获取的设备信息中
	properties.Property("device status update should be consistent", 
		prop.ForAll(
			func(name, host string, deviceType models.DeviceType, port int, status models.DeviceStatus) bool {
				// 清理数据
				deviceRepo.Clear()
				
				// 创建设备
				device := &models.Device{
					Name:     name,
					Host:     host,
					Type:     deviceType,
					Port:     port,
					Protocol: "ssh",
					Status:   models.DeviceStatusUnknown,
				}
				
				err := deviceService.CreateDevice(device)
				if err != nil {
					return false
				}
				
				// 更新设备状态
				err = deviceService.UpdateDeviceStatus(device.ID, status)
				if err != nil {
					return false
				}
				
				// 获取设备并验证状态
				updatedDevice, err := deviceService.GetDevice(device.ID)
				if err != nil {
					return false
				}
				
				return updatedDevice.Status == status && updatedDevice.LastSeen != nil
			},
			genValidDeviceName(),
			genValidHost(),
			genDeviceType(),
			genValidPort(),
			genDeviceStatus(),
		))

	// 属性5: 设备分组管理应该保持一致性
	properties.Property("device group management should be consistent", 
		prop.ForAll(
			func(deviceName, host, groupName string, deviceType models.DeviceType, port int) bool {
				// 清理数据
				deviceRepo.Clear()
				groupRepo.Clear()
				
				// 创建设备
				device := &models.Device{
					Name:     deviceName,
					Host:     host,
					Type:     deviceType,
					Port:     port,
					Protocol: "ssh",
					Status:   models.DeviceStatusUnknown,
				}
				
				err := deviceService.CreateDevice(device)
				if err != nil {
					return false
				}
				
				// 创建分组
				group := &models.DeviceGroup{
					Name:        groupName,
					Description: "Test group",
				}
				
				err = groupRepo.Create(group)
				if err != nil {
					return false
				}
				
				// 将设备添加到分组
				err = deviceService.AddDeviceToGroup(device.ID, group.ID)
				if err != nil {
					return false
				}
				
				// 验证设备在分组中
				devicesInGroup, err := deviceService.GetDevicesByGroup(group.ID)
				if err != nil {
					return false
				}
				
				// 检查设备是否在分组中
				found := false
				for _, d := range devicesInGroup {
					if d.ID == device.ID {
						found = true
						break
					}
				}
				
				if !found {
					return false
				}
				
				// 从分组中移除设备
				err = deviceService.RemoveDeviceFromGroup(device.ID, group.ID)
				if err != nil {
					return false
				}
				
				// 验证设备不再在分组中
				devicesInGroup, err = deviceService.GetDevicesByGroup(group.ID)
				if err != nil {
					return false
				}
				
				// 检查设备是否已从分组中移除
				for _, d := range devicesInGroup {
					if d.ID == device.ID {
						return false // 设备仍在分组中，测试失败
					}
				}
				
				return true
			},
			genValidDeviceName(),
			genValidHost(),
			genValidDeviceName(), // 用作分组名
			genDeviceType(),
			genValidPort(),
		))

	// 属性6: 设备标签管理应该保持一致性
	properties.Property("device tag management should be consistent", 
		prop.ForAll(
			func(deviceName, host, tagName string, deviceType models.DeviceType, port int) bool {
				// 清理数据
				deviceRepo.Clear()
				tagRepo.Clear()
				
				// 创建设备
				device := &models.Device{
					Name:     deviceName,
					Host:     host,
					Type:     deviceType,
					Port:     port,
					Protocol: "ssh",
					Status:   models.DeviceStatusUnknown,
				}
				
				err := deviceService.CreateDevice(device)
				if err != nil {
					return false
				}
				
				// 创建标签
				tag := &models.Tag{
					Name:        tagName,
					Color:       "#007bff",
					Description: "Test tag",
				}
				
				err = tagRepo.Create(tag)
				if err != nil {
					return false
				}
				
				// 为设备添加标签
				err = deviceService.AddDeviceTag(device.ID, tag.ID)
				if err != nil {
					return false
				}
				
				// 验证设备有该标签
				devicesWithTag, err := deviceService.GetDevicesByTag(tag.ID)
				if err != nil {
					return false
				}
				
				// 检查设备是否有该标签
				found := false
				for _, d := range devicesWithTag {
					if d.ID == device.ID {
						found = true
						break
					}
				}
				
				if !found {
					return false
				}
				
				// 移除设备标签
				err = deviceService.RemoveDeviceTag(device.ID, tag.ID)
				if err != nil {
					return false
				}
				
				// 验证设备不再有该标签
				devicesWithTag, err = deviceService.GetDevicesByTag(tag.ID)
				if err != nil {
					return false
				}
				
				// 检查设备是否已移除该标签
				for _, d := range devicesWithTag {
					if d.ID == device.ID {
						return false // 设备仍有该标签，测试失败
					}
				}
				
				return true
			},
			genValidDeviceName(),
			genValidHost(),
			genValidDeviceName(), // 用作标签名
			genDeviceType(),
			genValidPort(),
		))

	// 属性7: 设备列表过滤应该返回符合条件的设备
	properties.Property("device list filtering should return matching devices", 
		prop.ForAll(
			func(name1, host1, name2, host2 string, type1, type2 models.DeviceType, port1, port2 int) bool {
				// 确保主机地址不同
				if host1 == host2 {
					return true // 跳过这个测试用例
				}
				
				// 清理数据
				deviceRepo.Clear()
				
				// 创建两个不同类型的设备
				device1 := &models.Device{
					Name:     name1,
					Host:     host1,
					Type:     type1,
					Port:     port1,
					Protocol: "ssh",
					Status:   models.DeviceStatusOnline,
				}
				
				device2 := &models.Device{
					Name:     name2,
					Host:     host2,
					Type:     type2,
					Port:     port2,
					Protocol: "ssh",
					Status:   models.DeviceStatusOffline,
				}
				
				err := deviceService.CreateDevice(device1)
				if err != nil {
					return false
				}
				
				err = deviceService.CreateDevice(device2)
				if err != nil {
					return false
				}
				
				// 按类型过滤
				filters := map[string]interface{}{
					"type": type1,
				}
				
				devices, total, err := deviceService.ListDevices(0, 10, filters)
				if err != nil {
					return false
				}
				
				// 验证过滤结果
				if type1 == type2 {
					// 如果两个设备类型相同，应该返回2个设备
					return len(devices) == 2 && total == 2
				} else {
					// 如果类型不同，应该只返回1个设备
					return len(devices) == 1 && total == 1 && devices[0].Type == type1
				}
			},
			genValidDeviceName(),
			genValidHost(),
			genValidDeviceName(),
			genValidHost(),
			genDeviceType(),
			genDeviceType(),
			genValidPort(),
			genValidPort(),
		))
}