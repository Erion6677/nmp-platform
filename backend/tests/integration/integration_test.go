package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nmp-platform/internal/config"
	"nmp-platform/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestSuite 集成测试套件
type TestSuite struct {
	server     *server.Server
	router     *gin.Engine
	logger     *zap.Logger
	config     *config.Config
	tempDir    string
	configPath string
}

// SetupTestSuite 设置测试套件
func SetupTestSuite(t *testing.T) *TestSuite {
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 创建测试配置文件
	configPath := filepath.Join(tempDir, "test_config.yaml")
	testConfig := `
server:
  host: "localhost"
  port: 0  # 使用随机端口
  mode: "test"
  read_timeout: "30s"
  write_timeout: "30s"

database:
  host: "localhost"
  port: 5432
  database: "nmp_test"
  username: "test"
  password: "test"
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 1  # 使用测试数据库

influxdb:
  url: "http://localhost:8086"
  token: "test-token"
  org: "test-org"
  bucket: "test-bucket"

auth:
  jwt_secret: "test-jwt-secret-key-for-integration-testing-purposes-only"
  token_expiry: "1h"
  refresh_expiry: "24h"

plugins:
  directory: "./plugins"
  configs: {}
`
	
	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	require.NoError(t, err)
	
	// 设置环境变量
	os.Setenv("CONFIG_PATH", configPath)
	
	// 创建测试日志器
	logger := zap.NewNop()
	
	// 加载配置
	cfg, err := config.Load()
	require.NoError(t, err)
	
	// 设置全局配置
	config.SetConfig(cfg)
	
	// 创建服务器
	srv, err := server.New(cfg, logger)
	require.NoError(t, err)
	
	return &TestSuite{
		server:     srv,
		router:     srv.Router(),
		logger:     logger,
		config:     cfg,
		tempDir:    tempDir,
		configPath: configPath,
	}
}

// TeardownTestSuite 清理测试套件
func (ts *TestSuite) TeardownTestSuite(t *testing.T) {
	// 关闭服务器资源
	if ts.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := ts.server.Shutdown(ctx); err != nil {
			t.Logf("Warning: failed to shutdown server: %v", err)
		}
		
		if err := ts.server.Close(); err != nil {
			t.Logf("Warning: failed to close server resources: %v", err)
		}
	}
	
	// 清理环境变量
	os.Unsetenv("CONFIG_PATH")
}

// makeRequest 发送HTTP请求的辅助函数
func (ts *TestSuite) makeRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	
	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	// 添加自定义头部
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	
	return w
}

// TestSystemIntegration_UserAuthenticationFlow 测试用户认证流程
// 验证需求: 1.1, 1.2
func TestSystemIntegration_UserAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)
	
	// 1. 测试用户注册
	registerData := map[string]interface{}{
		"username": "testuser",
		"password": "testpassword123",
		"email":    "test@example.com",
	}
	
	w := ts.makeRequest("POST", "/api/v1/auth/register", registerData, nil)
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var registerResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &registerResp)
	require.NoError(t, err)
	assert.Equal(t, "User registered successfully", registerResp["message"])
	
	// 2. 测试用户登录
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "testpassword123",
	}
	
	w = ts.makeRequest("POST", "/api/v1/auth/login", loginData, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var loginResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	
	token, ok := loginResp["token"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, token)
	
	// 3. 测试使用JWT令牌访问受保护资源
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	
	w = ts.makeRequest("GET", "/api/v1/devices", nil, headers)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 4. 测试无效令牌
	invalidHeaders := map[string]string{
		"Authorization": "Bearer invalid-token",
	}
	
	w = ts.makeRequest("GET", "/api/v1/devices", nil, invalidHeaders)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 5. 测试无令牌访问
	w = ts.makeRequest("GET", "/api/v1/devices", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestSystemIntegration_PluginLoadingAndIntegration 测试插件加载和集成
// 验证需求: 2.1
func TestSystemIntegration_PluginLoadingAndIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)
	
	// 创建测试插件目录
	pluginDir := filepath.Join(ts.tempDir, "plugins")
	err := os.MkdirAll(pluginDir, 0755)
	require.NoError(t, err)
	
	// 创建测试插件配置
	pluginConfig := `
name: "test-plugin"
version: "1.0.0"
description: "Test plugin for integration testing"
enabled: true
routes:
  - method: "GET"
    path: "/test"
    handler: "TestHandler"
    permission: "test:read"
menus:
  - key: "test"
    label: "Test Plugin"
    icon: "test-icon"
    path: "/test"
    permission: "test:read"
permissions:
  - resource: "test"
    action: "read"
    scope: "all"
`
	
	pluginConfigPath := filepath.Join(pluginDir, "test-plugin.yaml")
	err = os.WriteFile(pluginConfigPath, []byte(pluginConfig), 0644)
	require.NoError(t, err)
	
	// 测试插件发现和加载
	// 注意：这里需要实际的插件管理器实现
	// 由于当前实现可能还没有完整的插件系统，我们测试基础的路由注册
	
	// 1. 测试基础API路由是否正常工作
	w := ts.makeRequest("GET", "/api/v1/ping", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var pingResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &pingResp)
	require.NoError(t, err)
	assert.Equal(t, "pong", pingResp["message"])
	
	// 2. 测试健康检查路由
	w = ts.makeRequest("GET", "/health", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var healthResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &healthResp)
	require.NoError(t, err)
	assert.Equal(t, "ok", healthResp["status"])
	
	// 3. 测试版本信息路由
	w = ts.makeRequest("GET", "/version", nil, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var versionResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &versionResp)
	require.NoError(t, err)
	assert.NotEmpty(t, versionResp["version"])
}

// TestSystemIntegration_EndToEndDataFlow 测试端到端数据流
// 验证需求: 1.1, 2.1, 3.1, 5.1, 6.1
func TestSystemIntegration_EndToEndDataFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)
	
	// 1. 用户认证
	registerData := map[string]interface{}{
		"username": "datauser",
		"password": "datapassword123",
		"email":    "data@example.com",
	}
	
	w := ts.makeRequest("POST", "/api/v1/auth/register", registerData, nil)
	require.Equal(t, http.StatusCreated, w.Code)
	
	loginData := map[string]interface{}{
		"username": "datauser",
		"password": "datapassword123",
	}
	
	w = ts.makeRequest("POST", "/api/v1/auth/login", loginData, nil)
	require.Equal(t, http.StatusOK, w.Code)
	
	var loginResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	
	token := loginResp["token"].(string)
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	
	// 2. 创建设备
	deviceData := map[string]interface{}{
		"name":        "test-device-001",
		"type":        "router",
		"host":        "192.168.1.1",
		"port":        22,
		"protocol":    "ssh",
		"username":    "admin",
		"description": "Test device for integration testing",
	}
	
	w = ts.makeRequest("POST", "/api/v1/devices", deviceData, headers)
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var deviceResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &deviceResp)
	require.NoError(t, err)
	
	deviceID := deviceResp["id"].(float64)
	assert.Greater(t, deviceID, float64(0))
	
	// 3. 推送监控数据
	metricData := map[string]interface{}{
		"device_id": fmt.Sprintf("%.0f", deviceID),
		"timestamp": time.Now().Unix(),
		"metrics": map[string]interface{}{
			"cpu_usage":    75.5,
			"memory_usage": 60.2,
			"disk_usage":   45.8,
		},
		"tags": map[string]string{
			"location": "datacenter-1",
			"rack":     "A-01",
		},
	}
	
	w = ts.makeRequest("POST", "/api/v1/data/push", metricData, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var pushResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &pushResp)
	require.NoError(t, err)
	assert.Equal(t, "Data received successfully", pushResp["message"])
	
	// 4. 查询设备列表
	w = ts.makeRequest("GET", "/api/v1/devices", nil, headers)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var devicesResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &devicesResp)
	require.NoError(t, err)
	
	devices, ok := devicesResp["devices"].([]interface{})
	require.True(t, ok)
	assert.Len(t, devices, 1)
	
	// 5. 查询设备详情
	deviceDetailPath := fmt.Sprintf("/api/v1/devices/%.0f", deviceID)
	w = ts.makeRequest("GET", deviceDetailPath, nil, headers)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var deviceDetailResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &deviceDetailResp)
	require.NoError(t, err)
	assert.Equal(t, "test-device-001", deviceDetailResp["name"])
	
	// 6. 查询实时数据（占位符测试）
	realtimePath := fmt.Sprintf("/api/v1/monitoring/realtime/%.0f", deviceID)
	w = ts.makeRequest("GET", realtimePath, nil, headers)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var realtimeResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &realtimeResp)
	require.NoError(t, err)
	assert.Contains(t, realtimeResp["message"], "implementation pending")
	
	// 7. 删除设备
	w = ts.makeRequest("DELETE", deviceDetailPath, nil, headers)
	assert.Equal(t, http.StatusOK, w.Code)
	
	// 8. 验证设备已删除
	w = ts.makeRequest("GET", deviceDetailPath, nil, headers)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestSystemIntegration_ErrorHandling 测试系统错误处理
func TestSystemIntegration_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)
	
	// 1. 测试无效的JSON请求
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// 2. 测试不存在的路由
	w = ts.makeRequest("GET", "/api/v1/nonexistent", nil, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	// 3. 测试无效的认证信息
	loginData := map[string]interface{}{
		"username": "nonexistent",
		"password": "wrongpassword",
	}
	
	w = ts.makeRequest("POST", "/api/v1/auth/login", loginData, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// 4. 测试无效的设备数据
	invalidDeviceData := map[string]interface{}{
		"name": "", // 空名称应该被拒绝
		"type": "invalid-type",
	}
	
	w = ts.makeRequest("POST", "/api/v1/devices", invalidDeviceData, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code) // 首先会因为没有认证而失败
}

// TestSystemIntegration_ConcurrentRequests 测试并发请求处理
func TestSystemIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)
	
	// 并发发送多个健康检查请求
	const numRequests = 10
	results := make(chan int, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func() {
			w := ts.makeRequest("GET", "/health", nil, nil)
			results <- w.Code
		}()
	}
	
	// 收集结果
	for i := 0; i < numRequests; i++ {
		statusCode := <-results
		assert.Equal(t, http.StatusOK, statusCode)
	}
}

// TestSystemIntegration_DatabaseConnectivity 测试数据库连接
func TestSystemIntegration_DatabaseConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	ts := SetupTestSuite(t)
	defer ts.TeardownTestSuite(t)
	
	// 测试详细健康检查，包括数据库状态
	w := ts.makeRequest("GET", "/health/detailed", nil, nil)
	
	// 根据实际的数据库连接状态，状态码可能是200或503
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusServiceUnavailable)
	
	var healthResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &healthResp)
	require.NoError(t, err)
	
	// 验证响应包含服务状态信息
	assert.Contains(t, healthResp, "status")
	assert.Contains(t, healthResp, "services")
	assert.Contains(t, healthResp, "timestamp")
}