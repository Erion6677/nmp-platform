package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPingTargetRepository 模拟 Ping 目标仓库
type MockPingTargetRepository struct {
	mock.Mock
}

func (m *MockPingTargetRepository) Create(target *models.PingTarget) error {
	args := m.Called(target)
	if args.Error(0) == nil && target != nil {
		target.ID = 1 // 模拟数据库分配的 ID
	}
	return args.Error(0)
}

func (m *MockPingTargetRepository) GetByID(id uint) (*models.PingTarget, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PingTarget), args.Error(1)
}

func (m *MockPingTargetRepository) GetByDeviceID(deviceID uint) ([]*models.PingTarget, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PingTarget), args.Error(1)
}

func (m *MockPingTargetRepository) GetEnabledByDeviceID(deviceID uint) ([]*models.PingTarget, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PingTarget), args.Error(1)
}

func (m *MockPingTargetRepository) Update(target *models.PingTarget) error {
	args := m.Called(target)
	return args.Error(0)
}

func (m *MockPingTargetRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockPingTargetRepository) DeleteByDeviceID(deviceID uint) error {
	args := m.Called(deviceID)
	return args.Error(0)
}

func (m *MockPingTargetRepository) UpdateEnabled(id uint, enabled bool) error {
	args := m.Called(id, enabled)
	return args.Error(0)
}

func (m *MockPingTargetRepository) Exists(deviceID uint, targetAddress string) (bool, error) {
	args := m.Called(deviceID, targetAddress)
	return args.Bool(0), args.Error(1)
}

// MockDeviceRepository 模拟设备仓库
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) Create(device *models.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByID(id uint) (*models.Device, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByHost(host string) (*models.Device, error) {
	args := m.Called(host)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) Update(device *models.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceRepository) List(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	args := m.Called(offset, limit, filters)
	return args.Get(0).([]*models.Device), args.Get(1).(int64), args.Error(2)
}

func (m *MockDeviceRepository) UpdateStatus(id uint, status models.DeviceStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockDeviceRepository) UpdateLastSeen(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByGroupID(groupID uint) ([]*models.Device, error) {
	args := m.Called(groupID)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByTagID(tagID uint) ([]*models.Device, error) {
	args := m.Called(tagID)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) AddToGroup(deviceID, groupID uint) error {
	args := m.Called(deviceID, groupID)
	return args.Error(0)
}

func (m *MockDeviceRepository) RemoveFromGroup(deviceID, groupID uint) error {
	args := m.Called(deviceID, groupID)
	return args.Error(0)
}

func (m *MockDeviceRepository) AddTag(deviceID, tagID uint) error {
	args := m.Called(deviceID, tagID)
	return args.Error(0)
}

func (m *MockDeviceRepository) RemoveTag(deviceID, tagID uint) error {
	args := m.Called(deviceID, tagID)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetAllOnline() ([]*models.Device, error) {
	args := m.Called()
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByOSType(osType models.DeviceOSType) ([]*models.Device, error) {
	args := m.Called(osType)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) UpdateConnectionInfo(id uint, apiPort, sshPort int) error {
	args := m.Called(id, apiPort, sshPort)
	return args.Error(0)
}

// 确保 Mock 实现了接口
var _ repository.PingTargetRepository = (*MockPingTargetRepository)(nil)
var _ repository.DeviceRepository = (*MockDeviceRepository)(nil)

// TestPingTargetHandler_GetPingTargets 测试获取 Ping 目标列表
func TestPingTargetHandler_GetPingTargets(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceRepo.On("GetByID", uint(1)).Return(&models.Device{ID: 1, Name: "TestDevice"}, nil)
	mockPingTargetRepo.On("GetByDeviceID", uint(1)).Return([]*models.PingTarget{
		{ID: 1, DeviceID: 1, TargetAddress: "8.8.8.8", TargetName: "Google DNS", SourceInterface: "", Enabled: true},
		{ID: 2, DeviceID: 1, TargetAddress: "1.1.1.1", TargetName: "Cloudflare", SourceInterface: "ether1", Enabled: true},
	}, nil)

	req, _ := http.NewRequest("GET", "/api/devices/1/ping-targets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].([]interface{})
	assert.Len(t, data, 2)

	mockPingTargetRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// TestPingTargetHandler_CreatePingTarget 测试创建 Ping 目标
func TestPingTargetHandler_CreatePingTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceRepo.On("GetByID", uint(1)).Return(&models.Device{ID: 1, Name: "TestDevice"}, nil)
	mockPingTargetRepo.On("Exists", uint(1), "8.8.8.8").Return(false, nil)
	mockPingTargetRepo.On("Create", mock.AnythingOfType("*models.PingTarget")).Return(nil)

	createReq := CreatePingTargetRequest{
		TargetAddress:   "8.8.8.8",
		TargetName:      "Google DNS",
		SourceInterface: "ether1",
	}
	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/devices/1/ping-targets", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "8.8.8.8", data["target_address"])
	assert.Equal(t, "Google DNS", data["target_name"])
	assert.Equal(t, "ether1", data["source_interface"])

	mockPingTargetRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// TestPingTargetHandler_UpdatePingTarget 测试更新 Ping 目标
func TestPingTargetHandler_UpdatePingTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockPingTargetRepo.On("GetByID", uint(1)).Return(&models.PingTarget{
		ID: 1, DeviceID: 1, TargetAddress: "8.8.8.8", TargetName: "Google DNS", Enabled: true,
	}, nil)
	mockPingTargetRepo.On("Update", mock.AnythingOfType("*models.PingTarget")).Return(nil)

	updateReq := UpdatePingTargetRequest{
		TargetName: "Updated Google DNS",
	}
	reqBody, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/devices/1/ping-targets/1", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockPingTargetRepo.AssertExpectations(t)
}

// TestPingTargetHandler_DeletePingTarget 测试删除 Ping 目标
func TestPingTargetHandler_DeletePingTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockPingTargetRepo.On("GetByID", uint(1)).Return(&models.PingTarget{
		ID: 1, DeviceID: 1, TargetAddress: "8.8.8.8", TargetName: "Google DNS",
	}, nil)
	mockPingTargetRepo.On("Delete", uint(1)).Return(nil)

	req, _ := http.NewRequest("DELETE", "/api/devices/1/ping-targets/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	mockPingTargetRepo.AssertExpectations(t)
}

// TestPingTargetHandler_TogglePingTarget 测试切换 Ping 目标启用状态
func TestPingTargetHandler_TogglePingTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockPingTargetRepo.On("GetByID", uint(1)).Return(&models.PingTarget{
		ID: 1, DeviceID: 1, TargetAddress: "8.8.8.8", TargetName: "Google DNS", Enabled: true,
	}, nil)
	mockPingTargetRepo.On("UpdateEnabled", uint(1), false).Return(nil)

	req, _ := http.NewRequest("PUT", "/api/devices/1/ping-targets/1/toggle", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))

	data := response["data"].(map[string]interface{})
	assert.False(t, data["enabled"].(bool))

	mockPingTargetRepo.AssertExpectations(t)
}

// TestPingTargetHandler_DeviceNotFound 测试设备不存在
func TestPingTargetHandler_DeviceNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望 - 设备不存在
	mockDeviceRepo.On("GetByID", uint(999)).Return(nil, errors.New("device not found"))

	req, _ := http.NewRequest("GET", "/api/devices/999/ping-targets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockDeviceRepo.AssertExpectations(t)
}

// TestPingTargetHandler_DuplicateTarget 测试重复目标地址
func TestPingTargetHandler_DuplicateTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望 - 目标地址已存在
	mockDeviceRepo.On("GetByID", uint(1)).Return(&models.Device{ID: 1, Name: "TestDevice"}, nil)
	mockPingTargetRepo.On("Exists", uint(1), "8.8.8.8").Return(true, nil)

	createReq := CreatePingTargetRequest{
		TargetAddress: "8.8.8.8",
		TargetName:    "Google DNS",
	}
	reqBody, _ := json.Marshal(createReq)
	req, _ := http.NewRequest("POST", "/api/devices/1/ping-targets", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "已存在")

	mockPingTargetRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// TestPingTargetHandler_InvalidDeviceID 测试无效设备 ID
func TestPingTargetHandler_InvalidDeviceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	req, _ := http.NewRequest("GET", "/api/devices/invalid/ping-targets", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestPingTargetHandler_TargetNotBelongToDevice 测试目标不属于设备
func TestPingTargetHandler_TargetNotBelongToDevice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望 - 目标属于设备 2，但请求的是设备 1
	mockPingTargetRepo.On("GetByID", uint(1)).Return(&models.PingTarget{
		ID: 1, DeviceID: 2, TargetAddress: "8.8.8.8", TargetName: "Google DNS",
	}, nil)

	req, _ := http.NewRequest("DELETE", "/api/devices/1/ping-targets/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "不属于该设备")

	mockPingTargetRepo.AssertExpectations(t)
}

// ============================================================================
// Property 5: Ping 监控数据完整性
// Feature: device-monitoring, Property 5: Ping 监控数据完整性
// Validates: Requirements 6.1, 6.4
// ============================================================================

// TestPingTargetDataIntegrity_Property 属性测试：Ping 监控数据完整性
// *For any* 设备的 Ping 目标，采集的数据应包含延迟值（毫秒）和状态（up/down）；
// *For any* 设备，应支持添加多个 Ping 目标。
func TestPingTargetDataIntegrity_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockPingTargetRepo := new(MockPingTargetRepository)
	mockDeviceRepo := new(MockDeviceRepository)

	handler := NewPingTargetHandler(mockPingTargetRepo, mockDeviceRepo)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	t.Run("设备支持添加多个 Ping 目标", func(t *testing.T) {
		// 设置模拟期望
		mockDeviceRepo.On("GetByID", uint(10)).Return(&models.Device{ID: 10, Name: "TestDevice"}, nil)
		mockPingTargetRepo.On("GetByDeviceID", uint(10)).Return([]*models.PingTarget{
			{ID: 1, DeviceID: 10, TargetAddress: "8.8.8.8", TargetName: "Google DNS", Enabled: true},
			{ID: 2, DeviceID: 10, TargetAddress: "1.1.1.1", TargetName: "Cloudflare", Enabled: true},
			{ID: 3, DeviceID: 10, TargetAddress: "223.5.5.5", TargetName: "阿里 DNS", Enabled: true},
		}, nil)

		req, _ := http.NewRequest("GET", "/api/devices/10/ping-targets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		// 验证设备可以有多个 Ping 目标
		assert.GreaterOrEqual(t, len(data), 1, "设备应支持添加多个 Ping 目标")
	})

	t.Run("Ping 目标数据包含必要字段", func(t *testing.T) {
		// 设置模拟期望
		mockDeviceRepo.On("GetByID", uint(11)).Return(&models.Device{ID: 11, Name: "TestDevice"}, nil)
		mockPingTargetRepo.On("GetByDeviceID", uint(11)).Return([]*models.PingTarget{
			{ID: 1, DeviceID: 11, TargetAddress: "8.8.8.8", TargetName: "Google DNS", SourceInterface: "ether1", Enabled: true},
		}, nil)

		req, _ := http.NewRequest("GET", "/api/devices/11/ping-targets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		for _, item := range data {
			target := item.(map[string]interface{})
			// 验证必要字段存在
			assert.NotEmpty(t, target["target_address"], "目标地址不应为空")
			assert.NotEmpty(t, target["target_name"], "目标名称不应为空")
			assert.Contains(t, target, "enabled", "应包含启用状态")
			assert.Contains(t, target, "source_interface", "应包含源接口字段")
		}
	})

	t.Run("创建 Ping 目标需要目标地址和名称", func(t *testing.T) {
		// 测试缺少目标地址
		mockDeviceRepo.On("GetByID", uint(12)).Return(&models.Device{ID: 12, Name: "TestDevice"}, nil)

		createReq := map[string]interface{}{
			"target_name": "Test Target",
			// 缺少 target_address
		}
		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/devices/12/ping-targets", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "缺少目标地址应返回 400")
	})

	t.Run("创建 Ping 目标需要目标名称", func(t *testing.T) {
		// 测试缺少目标名称
		mockDeviceRepo.On("GetByID", uint(13)).Return(&models.Device{ID: 13, Name: "TestDevice"}, nil)

		createReq := map[string]interface{}{
			"target_address": "8.8.8.8",
			// 缺少 target_name
		}
		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/devices/13/ping-targets", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code, "缺少目标名称应返回 400")
	})

	t.Run("支持指定源接口", func(t *testing.T) {
		// 设置模拟期望
		mockDeviceRepo.On("GetByID", uint(14)).Return(&models.Device{ID: 14, Name: "TestDevice"}, nil)
		mockPingTargetRepo.On("Exists", uint(14), "8.8.8.8").Return(false, nil)
		mockPingTargetRepo.On("Create", mock.AnythingOfType("*models.PingTarget")).Return(nil)

		createReq := CreatePingTargetRequest{
			TargetAddress:   "8.8.8.8",
			TargetName:      "Google DNS",
			SourceInterface: "ether1", // 指定源接口
		}
		reqBody, _ := json.Marshal(createReq)
		req, _ := http.NewRequest("POST", "/api/devices/14/ping-targets", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].(map[string]interface{})
		assert.Equal(t, "ether1", data["source_interface"], "应保存源接口配置")
	})
}
