package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nmp-platform/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestInterfaceHandler_GetInterfaces 测试获取接口列表
func TestInterfaceHandler_GetInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceService.On("GetDeviceInterfaces", uint(1)).Return([]*models.Interface{
		{ID: 1, DeviceID: 1, Name: "ether1", Status: models.InterfaceStatusUp, Monitored: true},
		{ID: 2, DeviceID: 1, Name: "ether2", Status: models.InterfaceStatusUp, Monitored: false},
		{ID: 3, DeviceID: 1, Name: "bridge", Status: models.InterfaceStatusDown, Monitored: false},
	}, nil)

	// 准备请求
	req, _ := http.NewRequest("GET", "/api/devices/1/interfaces", nil)

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

	// 验证接口数据只包含名称和状态 (Property 2)
	for _, item := range data {
		iface := item.(map[string]interface{})
		assert.NotEmpty(t, iface["name"], "接口名称不应为空")
		assert.NotEmpty(t, iface["status"], "接口状态不应为空")
		// 确保没有 MAC 地址等其他信息
		_, hasMac := iface["mac_address"]
		assert.False(t, hasMac, "不应包含 MAC 地址")
	}

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestInterfaceHandler_SyncInterfaces 测试同步接口
func TestInterfaceHandler_SyncInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceService.On("SyncInterfacesFromDevice", uint(1)).Return([]*models.Interface{
		{ID: 1, DeviceID: 1, Name: "ether1", Status: models.InterfaceStatusUp, Monitored: false},
		{ID: 2, DeviceID: 1, Name: "ether2", Status: models.InterfaceStatusUp, Monitored: false},
		{ID: 3, DeviceID: 1, Name: "bridge", Status: models.InterfaceStatusUp, Monitored: false},
	}, nil)

	// 准备请求
	req, _ := http.NewRequest("POST", "/api/devices/1/interfaces/sync", nil)

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

	// 验证接口数据只包含名称和状态 (Property 2)
	for _, item := range data {
		iface := item.(map[string]interface{})
		assert.NotEmpty(t, iface["name"], "接口名称不应为空")
		assert.Contains(t, []string{"up", "down", "unknown"}, iface["status"], "接口状态应为有效值")
	}

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestInterfaceHandler_SetMonitoredInterfaces 测试设置监控接口
func TestInterfaceHandler_SetMonitoredInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceService.On("SetMonitoredInterfaces", uint(1), []string{"ether1", "ether2"}).Return(nil)
	mockDeviceService.On("GetDeviceInterfaces", uint(1)).Return([]*models.Interface{
		{ID: 1, DeviceID: 1, Name: "ether1", Status: models.InterfaceStatusUp, Monitored: true},
		{ID: 2, DeviceID: 1, Name: "ether2", Status: models.InterfaceStatusUp, Monitored: true},
		{ID: 3, DeviceID: 1, Name: "bridge", Status: models.InterfaceStatusUp, Monitored: false},
	}, nil)

	// 准备请求
	setReq := SetMonitoredInterfacesRequest{
		InterfaceNames: []string{"ether1", "ether2"},
	}
	reqBody, _ := json.Marshal(setReq)
	req, _ := http.NewRequest("PUT", "/api/devices/1/interfaces/monitored", bytes.NewBuffer(reqBody))
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

	// 验证返回的接口中，只有选中的接口被监控 (Property 2)
	data := response["data"].([]interface{})
	monitoredCount := 0
	for _, item := range data {
		iface := item.(map[string]interface{})
		if iface["monitored"].(bool) {
			monitoredCount++
			// 验证监控的接口是我们选择的
			assert.Contains(t, []string{"ether1", "ether2"}, iface["name"])
		}
	}
	assert.Equal(t, 2, monitoredCount, "应该有2个接口被监控")

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestInterfaceHandler_GetMonitoredInterfaces 测试获取监控接口列表
func TestInterfaceHandler_GetMonitoredInterfaces(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceService.On("GetMonitoredInterfaces", uint(1)).Return([]*models.Interface{
		{ID: 1, DeviceID: 1, Name: "ether1", Status: models.InterfaceStatusUp, Monitored: true},
		{ID: 2, DeviceID: 1, Name: "ether2", Status: models.InterfaceStatusUp, Monitored: true},
	}, nil)

	// 准备请求
	req, _ := http.NewRequest("GET", "/api/devices/1/interfaces/monitored", nil)

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
	assert.Len(t, data, 2)

	// 验证所有返回的接口都是被监控的
	for _, item := range data {
		iface := item.(map[string]interface{})
		assert.True(t, iface["monitored"].(bool), "返回的接口应该都是被监控的")
	}

	// 验证模拟调用
	mockDeviceService.AssertExpectations(t)
}

// TestInterfaceHandler_UpdateInterfaceMonitorStatus 测试更新单个接口监控状态
func TestInterfaceHandler_UpdateInterfaceMonitorStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	// 设置模拟期望
	mockDeviceService.On("UpdateInterfaceMonitorStatus", uint(1), true).Return(nil)

	// 准备请求
	updateReq := map[string]bool{"monitored": true}
	reqBody, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("PUT", "/api/interfaces/1/monitor", bytes.NewBuffer(reqBody))
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
// Property 2: 接口数据管理一致性
// Feature: device-monitoring, Property 2: 接口数据管理一致性
// Validates: Requirements 3.2, 3.3
// ============================================================================

// TestInterfaceDataConsistency_Property 属性测试：接口数据管理一致性
// *For any* 接口同步操作，返回的接口数据应只包含名称和状态字段；
// *For any* 接口保存操作，数据库中应只存储用户选中的接口。
func TestInterfaceDataConsistency_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	t.Run("同步接口只返回名称和状态", func(t *testing.T) {
		// 设置模拟期望
		mockDeviceService.On("SyncInterfacesFromDevice", uint(2)).Return([]*models.Interface{
			{ID: 1, DeviceID: 2, Name: "eth0", Status: models.InterfaceStatusUp, Monitored: false},
			{ID: 2, DeviceID: 2, Name: "eth1", Status: models.InterfaceStatusDown, Monitored: false},
		}, nil)

		req, _ := http.NewRequest("POST", "/api/devices/2/interfaces/sync", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		for _, item := range data {
			iface := item.(map[string]interface{})
			
			// 验证必须有名称和状态
			assert.NotEmpty(t, iface["name"], "必须有接口名称")
			assert.NotEmpty(t, iface["status"], "必须有接口状态")
			
			// 验证状态值有效
			status := iface["status"].(string)
			assert.Contains(t, []string{"up", "down", "unknown"}, status, "状态值应有效")
		}
	})

	t.Run("设置监控接口只保存选中的接口", func(t *testing.T) {
		// 模拟设置监控接口
		selectedInterfaces := []string{"eth0"}
		
		mockDeviceService.On("SetMonitoredInterfaces", uint(3), selectedInterfaces).Return(nil)
		mockDeviceService.On("GetDeviceInterfaces", uint(3)).Return([]*models.Interface{
			{ID: 1, DeviceID: 3, Name: "eth0", Status: models.InterfaceStatusUp, Monitored: true},
			{ID: 2, DeviceID: 3, Name: "eth1", Status: models.InterfaceStatusUp, Monitored: false},
			{ID: 3, DeviceID: 3, Name: "eth2", Status: models.InterfaceStatusUp, Monitored: false},
		}, nil)

		setReq := SetMonitoredInterfacesRequest{
			InterfaceNames: selectedInterfaces,
		}
		reqBody, _ := json.Marshal(setReq)
		req, _ := http.NewRequest("PUT", "/api/devices/3/interfaces/monitored", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		
		// 验证只有选中的接口被监控
		monitoredNames := make([]string, 0)
		for _, item := range data {
			iface := item.(map[string]interface{})
			if iface["monitored"].(bool) {
				monitoredNames = append(monitoredNames, iface["name"].(string))
			}
		}
		
		assert.Equal(t, selectedInterfaces, monitoredNames, "只有选中的接口应被监控")
	})

	t.Run("清空监控接口", func(t *testing.T) {
		// 模拟清空所有监控接口
		emptyInterfaces := []string{}
		
		mockDeviceService.On("SetMonitoredInterfaces", uint(4), emptyInterfaces).Return(nil)
		mockDeviceService.On("GetDeviceInterfaces", uint(4)).Return([]*models.Interface{
			{ID: 1, DeviceID: 4, Name: "eth0", Status: models.InterfaceStatusUp, Monitored: false},
			{ID: 2, DeviceID: 4, Name: "eth1", Status: models.InterfaceStatusUp, Monitored: false},
		}, nil)

		setReq := SetMonitoredInterfacesRequest{
			InterfaceNames: emptyInterfaces,
		}
		reqBody, _ := json.Marshal(setReq)
		req, _ := http.NewRequest("PUT", "/api/devices/4/interfaces/monitored", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data := response["data"].([]interface{})
		
		// 验证没有接口被监控
		for _, item := range data {
			iface := item.(map[string]interface{})
			assert.False(t, iface["monitored"].(bool), "清空后不应有接口被监控")
		}
	})
}

// TestInterfaceHandler_InvalidDeviceID 测试无效设备ID
func TestInterfaceHandler_InvalidDeviceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建模拟服务
	mockDeviceService := new(MockDeviceService)

	// 创建处理器
	handler := NewInterfaceHandler(mockDeviceService)

	// 创建测试路由
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	testCases := []struct {
		name   string
		url    string
		method string
	}{
		{"获取接口-无效ID", "/api/devices/invalid/interfaces", "GET"},
		{"同步接口-无效ID", "/api/devices/abc/interfaces/sync", "POST"},
		{"设置监控-无效ID", "/api/devices/xyz/interfaces/monitored", "PUT"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var req *http.Request
			if tc.method == "PUT" {
				body := []byte(`{"interface_names": ["eth0"]}`)
				req, _ = http.NewRequest(tc.method, tc.url, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tc.method, tc.url, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code, "无效ID应返回400")
		})
	}
}
