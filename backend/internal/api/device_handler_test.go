package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nmp-platform/internal/models"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDeviceService 模拟设备服务
type MockDeviceService struct {
	mock.Mock
}

func (m *MockDeviceService) CreateDevice(device *models.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceService) GetDevice(id uint) (*models.Device, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceService) GetDeviceByHost(host string) (*models.Device, error) {
	args := m.Called(host)
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceService) UpdateDevice(device *models.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceService) DeleteDevice(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceService) ListDevices(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	args := m.Called(offset, limit, filters)
	return args.Get(0).([]*models.Device), args.Get(1).(int64), args.Error(2)
}

func (m *MockDeviceService) UpdateDeviceStatus(id uint, status models.DeviceStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockDeviceService) UpdateDeviceLastSeen(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceService) AddDeviceToGroup(deviceID, groupID uint) error {
	args := m.Called(deviceID, groupID)
	return args.Error(0)
}

func (m *MockDeviceService) RemoveDeviceFromGroup(deviceID, groupID uint) error {
	args := m.Called(deviceID, groupID)
	return args.Error(0)
}

func (m *MockDeviceService) GetDevicesByGroup(groupID uint) ([]*models.Device, error) {
	args := m.Called(groupID)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceService) AddDeviceTag(deviceID, tagID uint) error {
	args := m.Called(deviceID, tagID)
	return args.Error(0)
}

func (m *MockDeviceService) RemoveDeviceTag(deviceID, tagID uint) error {
	args := m.Called(deviceID, tagID)
	return args.Error(0)
}

func (m *MockDeviceService) GetDevicesByTag(tagID uint) ([]*models.Device, error) {
	args := m.Called(tagID)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceService) SyncDeviceInterfaces(deviceID uint, interfaces []*models.Interface) error {
	args := m.Called(deviceID, interfaces)
	return args.Error(0)
}

func (m *MockDeviceService) GetDeviceInterfaces(deviceID uint) ([]*models.Interface, error) {
	args := m.Called(deviceID)
	return args.Get(0).([]*models.Interface), args.Error(1)
}

func (m *MockDeviceService) GetMonitoredInterfaces(deviceID uint) ([]*models.Interface, error) {
	args := m.Called(deviceID)
	return args.Get(0).([]*models.Interface), args.Error(1)
}

func (m *MockDeviceService) UpdateInterfaceMonitorStatus(interfaceID uint, monitor bool) error {
	args := m.Called(interfaceID, monitor)
	return args.Error(0)
}

func (m *MockDeviceService) SetMonitoredInterfaces(deviceID uint, interfaceNames []string) error {
	args := m.Called(deviceID, interfaceNames)
	return args.Error(0)
}

func (m *MockDeviceService) TestConnection(device *models.Device, connectionType string) (*service.ConnectionTestResult, error) {
	args := m.Called(device, connectionType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.ConnectionTestResult), args.Error(1)
}

func (m *MockDeviceService) GetSystemInfo(deviceID uint) (*service.SystemInfoResult, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SystemInfoResult), args.Error(1)
}

func (m *MockDeviceService) SyncInterfacesFromDevice(deviceID uint) ([]*models.Interface, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Interface), args.Error(1)
}

// MockTagService 模拟标签服务
type MockTagService struct {
	mock.Mock
}

func (m *MockTagService) CreateTag(tag *models.Tag) error {
	args := m.Called(tag)
	return args.Error(0)
}

func (m *MockTagService) GetTag(id uint) (*models.Tag, error) {
	args := m.Called(id)
	return args.Get(0).(*models.Tag), args.Error(1)
}

func (m *MockTagService) GetTagByName(name string) (*models.Tag, error) {
	args := m.Called(name)
	return args.Get(0).(*models.Tag), args.Error(1)
}

func (m *MockTagService) UpdateTag(tag *models.Tag) error {
	args := m.Called(tag)
	return args.Error(0)
}

func (m *MockTagService) DeleteTag(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockTagService) ListTags(offset, limit int) ([]*models.Tag, int64, error) {
	args := m.Called(offset, limit)
	return args.Get(0).([]*models.Tag), args.Get(1).(int64), args.Error(2)
}

func (m *MockTagService) GetAllTags() ([]*models.Tag, error) {
	args := m.Called()
	return args.Get(0).([]*models.Tag), args.Error(1)
}

func (m *MockTagService) GetTagsByDevice(deviceID uint) ([]*models.Tag, error) {
	args := m.Called(deviceID)
	return args.Get(0).([]*models.Tag), args.Error(1)
}

// MockDeviceGroupService 模拟设备分组服务
type MockDeviceGroupService struct {
	mock.Mock
}

func (m *MockDeviceGroupService) CreateGroup(group *models.DeviceGroup) error {
	args := m.Called(group)
	return args.Error(0)
}

func (m *MockDeviceGroupService) GetGroup(id uint) (*models.DeviceGroup, error) {
	args := m.Called(id)
	return args.Get(0).(*models.DeviceGroup), args.Error(1)
}

func (m *MockDeviceGroupService) GetGroupByName(name string) (*models.DeviceGroup, error) {
	args := m.Called(name)
	return args.Get(0).(*models.DeviceGroup), args.Error(1)
}

func (m *MockDeviceGroupService) UpdateGroup(group *models.DeviceGroup) error {
	args := m.Called(group)
	return args.Error(0)
}

func (m *MockDeviceGroupService) DeleteGroup(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceGroupService) ListGroups(offset, limit int) ([]*models.DeviceGroup, int64, error) {
	args := m.Called(offset, limit)
	return args.Get(0).([]*models.DeviceGroup), args.Get(1).(int64), args.Error(2)
}

func (m *MockDeviceGroupService) GetAllGroups() ([]*models.DeviceGroup, error) {
	args := m.Called()
	return args.Get(0).([]*models.DeviceGroup), args.Error(1)
}

func (m *MockDeviceGroupService) GetRootGroups() ([]*models.DeviceGroup, error) {
	args := m.Called()
	return args.Get(0).([]*models.DeviceGroup), args.Error(1)
}

func (m *MockDeviceGroupService) GetChildGroups(parentID uint) ([]*models.DeviceGroup, error) {
	args := m.Called(parentID)
	return args.Get(0).([]*models.DeviceGroup), args.Error(1)
}

func (m *MockDeviceGroupService) GetGroupsByDevice(deviceID uint) ([]*models.DeviceGroup, error) {
	args := m.Called(deviceID)
	return args.Get(0).([]*models.DeviceGroup), args.Error(1)
}

func (m *MockDeviceGroupService) GetGroupDeviceCount(groupID uint) (int64, error) {
	args := m.Called(groupID)
	return args.Get(0).(int64), args.Error(1)
}

func TestDeviceHandler_CreateDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.POST("/devices", handler.CreateDevice)

	// 测试数据 - 包含所有必填字段
	createReq := CreateDeviceRequest{
		Name:     "Test Router",
		Type:     models.DeviceTypeRouter,
		OSType:   models.DeviceOSTypeMikroTik,
		Host:     "192.168.1.1",
		Port:     22,
		APIPort:  8728,
		Protocol: "ssh",
		Username: "admin",
		Password: "password123",
	}

	// 设置模拟期望
	mockDeviceService.On("CreateDevice", mock.AnythingOfType("*models.Device")).Return(nil).Run(func(args mock.Arguments) {
		device := args.Get(0).(*models.Device)
		device.ID = 1 // 模拟数据库分配的ID
	})

	mockDeviceService.On("GetDevice", uint(1)).Return(&models.Device{
		ID:       1,
		Name:     "Test Router",
		Type:     models.DeviceTypeRouter,
		OSType:   models.DeviceOSTypeMikroTik,
		Host:     "192.168.1.1",
		Port:     22,
		APIPort:  8728,
		Protocol: "ssh",
		Username: "admin",
		Status:   models.DeviceStatusUnknown,
	}, nil)

	// 准备请求
	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/devices", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.NotNil(t, response["data"])

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

func TestDeviceHandler_GetDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.GET("/devices/:id", handler.GetDevice)

	// 设置模拟期望
	expectedDevice := &models.Device{
		ID:       1,
		Name:     "Test Router",
		Type:     models.DeviceTypeRouter,
		Host:     "192.168.1.1",
		Port:     22,
		Protocol: "ssh",
		Status:   models.DeviceStatusOnline,
	}

	mockDeviceService.On("GetDevice", uint(1)).Return(expectedDevice, nil)

	// 准备请求
	req, _ := http.NewRequest("GET", "/devices/1", nil)

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	deviceData := response["data"].(map[string]interface{})
	assert.Equal(t, float64(1), deviceData["id"])
	assert.Equal(t, "Test Router", deviceData["name"])
	assert.Equal(t, "router", deviceData["type"])

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

func TestDeviceHandler_ListDevices(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.GET("/devices", handler.ListDevices)

	// 设置模拟期望
	expectedDevices := []*models.Device{
		{
			ID:       1,
			Name:     "Router 1",
			Type:     models.DeviceTypeRouter,
			Host:     "192.168.1.1",
			Status:   models.DeviceStatusOnline,
		},
		{
			ID:       2,
			Name:     "Switch 1",
			Type:     models.DeviceTypeSwitch,
			Host:     "192.168.1.2",
			Status:   models.DeviceStatusOffline,
		},
	}

	mockDeviceService.On("ListDevices", 0, 20, mock.AnythingOfType("map[string]interface {}")).Return(expectedDevices, int64(2), nil)

	// 准备请求
	req, _ := http.NewRequest("GET", "/devices?page=1&page_size=20", nil)

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	devices := data["devices"].([]interface{})
	assert.Len(t, devices, 2)
	assert.Equal(t, float64(2), data["total"])
	assert.Equal(t, float64(1), data["page"])

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

func TestDeviceHandler_UpdateDeviceStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.PUT("/devices/:id/status", handler.UpdateDeviceStatus)

	// 设置模拟期望
	mockDeviceService.On("UpdateDeviceStatus", uint(1), models.DeviceStatusOnline).Return(nil)

	// 准备请求
	statusReq := map[string]string{"status": "online"}
	reqBody, _ := json.Marshal(statusReq)
	req, _ := http.NewRequest("PUT", "/devices/1/status", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
	assert.Equal(t, "Device status updated successfully", response["message"])

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}


// TestDeviceHandler_TestConnection 测试连接测试功能
func TestDeviceHandler_TestConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.POST("/devices/test", handler.TestConnection)

	// 设置模拟期望
	mockDeviceService.On("TestConnection", mock.AnythingOfType("*models.Device"), "all").Return(&service.ConnectionTestResult{
		APISuccess: true,
		SSHSuccess: true,
	}, nil)

	// 准备请求
	testReq := TestConnectionRequest{
		Host:           "10.10.10.254",
		Port:           3399,
		APIPort:        8827,
		Username:       "test",
		Password:       "tset@123",
		OSType:         models.DeviceOSTypeMikroTik,
		ConnectionType: "all",
	}
	reqBody, _ := json.Marshal(testReq)
	req, _ := http.NewRequest("POST", "/devices/test", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.True(t, data["api_success"].(bool))
	assert.True(t, data["ssh_success"].(bool))

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestDeviceHandler_GetSystemInfo 测试获取系统信息
func TestDeviceHandler_GetSystemInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.GET("/devices/:id/info", handler.GetSystemInfo)

	// 设置模拟期望
	mockDeviceService.On("GetSystemInfo", uint(1)).Return(&service.SystemInfoResult{
		DeviceName:   "TestRouter",
		DeviceIP:     "10.10.10.254",
		CPUCount:     4,
		Version:      "7.10",
		License:      "level6",
		Uptime:       86400,
		CPUUsage:     15.5,
		MemoryUsage:  45.2,
		MemoryTotal:  1073741824,
		MemoryFree:   589824000,
	}, nil)

	// 准备请求
	req, _ := http.NewRequest("GET", "/devices/1/info", nil)

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "TestRouter", data["device_name"])
	assert.Equal(t, "10.10.10.254", data["device_ip"])
	assert.Equal(t, float64(4), data["cpu_count"])

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestDeviceHandler_SyncInterfaces 测试同步接口
func TestDeviceHandler_SyncInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.POST("/devices/:id/interfaces/sync", handler.SyncInterfaces)

	// 设置模拟期望
	mockDeviceService.On("SyncInterfacesFromDevice", uint(1)).Return([]*models.Interface{
		{ID: 1, DeviceID: 1, Name: "ether1", Status: models.InterfaceStatusUp, Monitored: false},
		{ID: 2, DeviceID: 1, Name: "ether2", Status: models.InterfaceStatusUp, Monitored: false},
		{ID: 3, DeviceID: 1, Name: "bridge", Status: models.InterfaceStatusUp, Monitored: false},
	}, nil)

	// 准备请求
	req, _ := http.NewRequest("POST", "/devices/1/interfaces/sync", nil)

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Len(t, data, 3)

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestDeviceHandler_SetMonitoredInterfaces 测试设置监控接口
func TestDeviceHandler_SetMonitoredInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.PUT("/devices/:id/interfaces/monitored", handler.SetMonitoredInterfaces)

	// 设置模拟期望
	mockDeviceService.On("SetMonitoredInterfaces", uint(1), []string{"ether1", "ether2"}).Return(nil)
	mockDeviceService.On("GetDeviceInterfaces", uint(1)).Return([]*models.Interface{
		{ID: 1, DeviceID: 1, Name: "ether1", Status: models.InterfaceStatusUp, Monitored: true},
		{ID: 2, DeviceID: 1, Name: "ether2", Status: models.InterfaceStatusUp, Monitored: true},
		{ID: 3, DeviceID: 1, Name: "bridge", Status: models.InterfaceStatusUp, Monitored: false},
	}, nil)

	// 准备请求
	setReq := map[string][]string{"interface_names": {"ether1", "ether2"}}
	reqBody, _ := json.Marshal(setReq)
	req, _ := http.NewRequest("PUT", "/devices/1/interfaces/monitored", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// ============================================================================
// Property 1: 设备表单验证完整性
// Feature: device-monitoring, Property 1: 设备表单验证完整性
// Validates: Requirements 1.1, 1.5
// ============================================================================

// TestDeviceFormValidation_Property 属性测试：设备表单验证完整性
// *For any* 设备添加请求，如果缺少必填字段（名称、IP、用户名、密码、设备类型），
// 系统应拒绝请求并返回具体的验证错误信息。
func TestDeviceFormValidation_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.POST("/devices", handler.CreateDevice)

	// 测试用例：缺少各种必填字段
	testCases := []struct {
		name        string
		request     map[string]interface{}
		expectError bool
		errorField  string
	}{
		{
			name: "缺少设备名称",
			request: map[string]interface{}{
				"type":     "router",
				"host":     "192.168.1.1",
				"username": "admin",
				"password": "password123",
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "缺少IP地址",
			request: map[string]interface{}{
				"name":     "TestDevice",
				"type":     "router",
				"username": "admin",
				"password": "password123",
			},
			expectError: true,
			errorField:  "host",
		},
		{
			name: "缺少用户名",
			request: map[string]interface{}{
				"name":     "TestDevice",
				"type":     "router",
				"host":     "192.168.1.1",
				"password": "password123",
			},
			expectError: true,
			errorField:  "username",
		},
		{
			name: "缺少密码",
			request: map[string]interface{}{
				"name":     "TestDevice",
				"type":     "router",
				"host":     "192.168.1.1",
				"username": "admin",
			},
			expectError: true,
			errorField:  "password",
		},
		{
			name: "缺少设备类型",
			request: map[string]interface{}{
				"name":     "TestDevice",
				"host":     "192.168.1.1",
				"username": "admin",
				"password": "password123",
			},
			expectError: true,
			errorField:  "type",
		},
		{
			name: "空设备名称",
			request: map[string]interface{}{
				"name":     "",
				"type":     "router",
				"host":     "192.168.1.1",
				"username": "admin",
				"password": "password123",
			},
			expectError: true,
			errorField:  "name",
		},
		{
			name: "空IP地址",
			request: map[string]interface{}{
				"name":     "TestDevice",
				"type":     "router",
				"host":     "",
				"username": "admin",
				"password": "password123",
			},
			expectError: true,
			errorField:  "host",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody, _ := json.Marshal(tc.request)
			req, _ := http.NewRequest("POST", "/devices", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectError {
				// 应该返回 400 Bad Request
				assert.Equal(t, http.StatusBadRequest, w.Code, "缺少必填字段应返回 400")

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)

				// 应该有错误信息
				_, hasError := response["error"]
				_, hasDetails := response["details"]
				assert.True(t, hasError || hasDetails, "应返回错误信息")
			}
		})
	}
}

// TestDeviceFormValidation_ValidRequest 测试有效请求
func TestDeviceFormValidation_ValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)
	mockTagService := new(MockTagService)
	mockDeviceGroupService := new(MockDeviceGroupService)

	// 创建处理器
	handler := NewDeviceHandler(mockDeviceService, mockTagService, mockDeviceGroupService)

	// 创建测试路由
	router := gin.New()
	router.POST("/devices", handler.CreateDevice)

	// 设置模拟期望
	mockDeviceService.On("CreateDevice", mock.AnythingOfType("*models.Device")).Return(nil).Run(func(args mock.Arguments) {
		device := args.Get(0).(*models.Device)
		device.ID = 1
	})
	mockDeviceService.On("GetDevice", uint(1)).Return(&models.Device{
		ID:       1,
		Name:     "ValidDevice",
		Type:     models.DeviceTypeRouter,
		OSType:   models.DeviceOSTypeMikroTik,
		Host:     "192.168.1.1",
		Port:     22,
		APIPort:  8728,
		Username: "admin",
		Status:   models.DeviceStatusUnknown,
	}, nil)

	// 有效请求
	validReq := map[string]interface{}{
		"name":     "ValidDevice",
		"type":     "router",
		"os_type":  "mikrotik",
		"host":     "192.168.1.1",
		"username": "admin",
		"password": "password123",
	}

	reqBody, _ := json.Marshal(validReq)
	req, _ := http.NewRequest("POST", "/devices", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 应该返回 201 Created
	assert.Equal(t, http.StatusCreated, w.Code, "有效请求应返回 201")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockDeviceService.AssertExpectations(t)
}
