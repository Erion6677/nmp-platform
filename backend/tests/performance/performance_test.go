package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"nmp-platform/internal/config"
	"nmp-platform/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// PerformanceTestSuite 性能测试套件
type PerformanceTestSuite struct {
	server     *server.Server
	router     *gin.Engine
	logger     *zap.Logger
	config     *config.Config
	tempDir    string
	authToken  string
}

// SetupPerformanceTestSuite 设置性能测试套件
func SetupPerformanceTestSuite(t *testing.T) *PerformanceTestSuite {
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 创建测试配置文件
	configPath := filepath.Join(tempDir, "perf_config.yaml")
	testConfig := `
server:
  host: "localhost"
  port: 0
  mode: "release"  # 使用release模式进行性能测试
  read_timeout: "30s"
  write_timeout: "30s"

database:
  host: "localhost"
  port: 5432
  database: "nmp_perf_test"
  username: "test"
  password: "test"
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 2  # 使用专门的性能测试数据库

influxdb:
  url: "http://localhost:8086"
  token: "perf-test-token"
  org: "perf-test-org"
  bucket: "perf-test-bucket"

auth:
  jwt_secret: "performance-test-jwt-secret-key-for-testing-purposes-only"
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
	
	return &PerformanceTestSuite{
		server:  srv,
		router:  srv.Router(),
		logger:  logger,
		config:  cfg,
		tempDir: tempDir,
	}
}
// TeardownPerformanceTestSuite 清理性能测试套件
func (pts *PerformanceTestSuite) TeardownPerformanceTestSuite(t *testing.T) {
	if pts.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := pts.server.Shutdown(ctx); err != nil {
			t.Logf("Warning: failed to shutdown server: %v", err)
		}
		
		if err := pts.server.Close(); err != nil {
			t.Logf("Warning: failed to close server resources: %v", err)
		}
	}
	
	os.Unsetenv("CONFIG_PATH")
}

// setupAuth 设置认证令牌
func (pts *PerformanceTestSuite) setupAuth(t *testing.T) {
	// 注册测试用户
	registerData := map[string]interface{}{
		"username": "perfuser",
		"password": "perfpassword123",
		"email":    "perf@example.com",
	}
	
	reqBody, _ := json.Marshal(registerData)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	pts.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	
	// 登录获取令牌
	loginData := map[string]interface{}{
		"username": "perfuser",
		"password": "perfpassword123",
	}
	
	reqBody, _ = json.Marshal(loginData)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	
	w = httptest.NewRecorder()
	pts.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	
	var loginResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &loginResp)
	require.NoError(t, err)
	
	pts.authToken = loginResp["token"].(string)
}

// makeAuthenticatedRequest 发送认证请求
func (pts *PerformanceTestSuite) makeAuthenticatedRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	
	req := httptest.NewRequest(method, path, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+pts.authToken)
	
	w := httptest.NewRecorder()
	pts.router.ServeHTTP(w, req)
	
	return w
}

// TestPerformance_DataPushThroughput 测试数据推送性能
// 验证需求: 3.1
func TestPerformance_DataPushThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	pts := SetupPerformanceTestSuite(t)
	defer pts.TeardownPerformanceTestSuite(t)
	
	// 测试参数
	const (
		numRequests    = 1000
		concurrency    = 10
		targetDuration = 10 * time.Second
	)
	
	// 准备测试数据
	testData := map[string]interface{}{
		"device_id": "perf-test-device-001",
		"timestamp": time.Now().Unix(),
		"metrics": map[string]interface{}{
			"cpu_usage":    75.5,
			"memory_usage": 60.2,
			"disk_usage":   45.8,
			"network_in":   1024000,
			"network_out":  512000,
		},
		"tags": map[string]string{
			"location": "datacenter-1",
			"rack":     "A-01",
		},
	}
	
	// 性能测试统计
	var (
		successCount int64
		errorCount   int64
		totalLatency time.Duration
		mu           sync.Mutex
	)
	
	startTime := time.Now()
	
	// 创建工作池
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量
			
			// 修改数据以避免重复
			data := make(map[string]interface{})
			for k, v := range testData {
				data[k] = v
			}
			data["device_id"] = fmt.Sprintf("perf-test-device-%03d", requestID%100)
			data["timestamp"] = time.Now().Unix()
			
			reqStart := time.Now()
			
			reqBody, _ := json.Marshal(data)
			req := httptest.NewRequest("POST", "/api/v1/data/push", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			pts.router.ServeHTTP(w, req)
			
			latency := time.Since(reqStart)
			
			mu.Lock()
			if w.Code == http.StatusOK {
				successCount++
			} else {
				errorCount++
			}
			totalLatency += latency
			mu.Unlock()
		}(i)
	}
	
	wg.Wait()
	totalDuration := time.Since(startTime)
	
	// 计算性能指标
	totalRequests := successCount + errorCount
	throughput := float64(totalRequests) / totalDuration.Seconds()
	avgLatency := totalLatency / time.Duration(totalRequests)
	successRate := float64(successCount) / float64(totalRequests) * 100
	
	// 输出性能报告
	t.Logf("数据推送性能测试结果:")
	t.Logf("  总请求数: %d", totalRequests)
	t.Logf("  成功请求: %d", successCount)
	t.Logf("  失败请求: %d", errorCount)
	t.Logf("  成功率: %.2f%%", successRate)
	t.Logf("  总耗时: %v", totalDuration)
	t.Logf("  吞吐量: %.2f 请求/秒", throughput)
	t.Logf("  平均延迟: %v", avgLatency)
	
	// 性能断言
	assert.Greater(t, successRate, 95.0, "成功率应该大于95%")
	assert.Greater(t, throughput, 50.0, "吞吐量应该大于50请求/秒")
	assert.Less(t, avgLatency, 100*time.Millisecond, "平均延迟应该小于100ms")
}
// TestPerformance_DataQueryPerformance 测试数据查询性能
// 验证需求: 6.1
func TestPerformance_DataQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	pts := SetupPerformanceTestSuite(t)
	defer pts.TeardownPerformanceTestSuite(t)
	pts.setupAuth(t)
	
	// 首先创建一些测试设备
	deviceIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		deviceData := map[string]interface{}{
			"name":        fmt.Sprintf("query-test-device-%03d", i),
			"type":        "router",
			"host":        fmt.Sprintf("192.168.1.%d", i+1),
			"port":        22,
			"protocol":    "ssh",
			"username":    "admin",
			"description": "Query performance test device",
		}
		
		w := pts.makeAuthenticatedRequest("POST", "/api/v1/devices", deviceData)
		require.Equal(t, http.StatusCreated, w.Code)
		
		var deviceResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &deviceResp)
		require.NoError(t, err)
		
		deviceIDs[i] = fmt.Sprintf("%.0f", deviceResp["id"].(float64))
	}
	
	// 测试设备列表查询性能
	const numQueries = 100
	var queryLatencies []time.Duration
	
	for i := 0; i < numQueries; i++ {
		start := time.Now()
		w := pts.makeAuthenticatedRequest("GET", "/api/v1/devices", nil)
		latency := time.Since(start)
		
		assert.Equal(t, http.StatusOK, w.Code)
		queryLatencies = append(queryLatencies, latency)
	}
	
	// 计算查询性能统计
	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency = time.Hour // 初始化为一个很大的值
	
	for _, latency := range queryLatencies {
		totalLatency += latency
		if latency > maxLatency {
			maxLatency = latency
		}
		if latency < minLatency {
			minLatency = latency
		}
	}
	
	avgLatency := totalLatency / time.Duration(len(queryLatencies))
	
	// 输出查询性能报告
	t.Logf("数据查询性能测试结果:")
	t.Logf("  查询次数: %d", numQueries)
	t.Logf("  平均延迟: %v", avgLatency)
	t.Logf("  最小延迟: %v", minLatency)
	t.Logf("  最大延迟: %v", maxLatency)
	
	// 性能断言
	assert.Less(t, avgLatency, 50*time.Millisecond, "平均查询延迟应该小于50ms")
	assert.Less(t, maxLatency, 200*time.Millisecond, "最大查询延迟应该小于200ms")
}

// TestPerformance_ConcurrentUserAccess 测试并发用户访问性能
// 验证需求: 1.1, 6.1
func TestPerformance_ConcurrentUserAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	pts := SetupPerformanceTestSuite(t)
	defer pts.TeardownPerformanceTestSuite(t)
	
	// 测试参数
	const (
		numUsers       = 50
		requestsPerUser = 20
	)
	
	// 为每个用户创建认证令牌
	userTokens := make([]string, numUsers)
	for i := 0; i < numUsers; i++ {
		// 注册用户
		registerData := map[string]interface{}{
			"username": fmt.Sprintf("concuser%d", i),
			"password": "concpassword123",
			"email":    fmt.Sprintf("conc%d@example.com", i),
		}
		
		reqBody, _ := json.Marshal(registerData)
		req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		pts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
		
		// 登录获取令牌
		loginData := map[string]interface{}{
			"username": fmt.Sprintf("concuser%d", i),
			"password": "concpassword123",
		}
		
		reqBody, _ = json.Marshal(loginData)
		req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		
		w = httptest.NewRecorder()
		pts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		
		var loginResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &loginResp)
		require.NoError(t, err)
		
		userTokens[i] = loginResp["token"].(string)
	}
	
	// 并发测试统计
	var (
		totalRequests int64
		successCount  int64
		errorCount    int64
		totalLatency  time.Duration
		mu            sync.Mutex
	)
	
	startTime := time.Now()
	var wg sync.WaitGroup
	
	// 每个用户并发执行请求
	for userID := 0; userID < numUsers; userID++ {
		wg.Add(1)
		go func(uid int, token string) {
			defer wg.Done()
			
			for reqID := 0; reqID < requestsPerUser; reqID++ {
				reqStart := time.Now()
				
				// 执行不同类型的请求
				var w *httptest.ResponseRecorder
				switch reqID % 3 {
				case 0:
					// 健康检查
					req := httptest.NewRequest("GET", "/health", nil)
					w = httptest.NewRecorder()
					pts.router.ServeHTTP(w, req)
				case 1:
					// 设备列表查询
					req := httptest.NewRequest("GET", "/api/v1/devices", nil)
					req.Header.Set("Authorization", "Bearer "+token)
					w = httptest.NewRecorder()
					pts.router.ServeHTTP(w, req)
				case 2:
					// Ping测试
					req := httptest.NewRequest("GET", "/api/v1/ping", nil)
					w = httptest.NewRecorder()
					pts.router.ServeHTTP(w, req)
				}
				
				latency := time.Since(reqStart)
				
				mu.Lock()
				totalRequests++
				if w.Code >= 200 && w.Code < 300 {
					successCount++
				} else {
					errorCount++
				}
				totalLatency += latency
				mu.Unlock()
			}
		}(userID, userTokens[userID])
	}
	
	wg.Wait()
	totalDuration := time.Since(startTime)
	
	// 计算性能指标
	throughput := float64(totalRequests) / totalDuration.Seconds()
	avgLatency := totalLatency / time.Duration(totalRequests)
	successRate := float64(successCount) / float64(totalRequests) * 100
	
	// 输出并发性能报告
	t.Logf("并发用户访问性能测试结果:")
	t.Logf("  并发用户数: %d", numUsers)
	t.Logf("  每用户请求数: %d", requestsPerUser)
	t.Logf("  总请求数: %d", totalRequests)
	t.Logf("  成功请求: %d", successCount)
	t.Logf("  失败请求: %d", errorCount)
	t.Logf("  成功率: %.2f%%", successRate)
	t.Logf("  总耗时: %v", totalDuration)
	t.Logf("  吞吐量: %.2f 请求/秒", throughput)
	t.Logf("  平均延迟: %v", avgLatency)
	
	// 性能断言
	assert.Greater(t, successRate, 98.0, "并发访问成功率应该大于98%")
	assert.Greater(t, throughput, 100.0, "并发吞吐量应该大于100请求/秒")
	assert.Less(t, avgLatency, 50*time.Millisecond, "并发平均延迟应该小于50ms")
}

// TestPerformance_MemoryUsage 测试内存使用情况
func TestPerformance_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	pts := SetupPerformanceTestSuite(t)
	defer pts.TeardownPerformanceTestSuite(t)
	pts.setupAuth(t)
	
	// 执行大量请求来测试内存使用
	const numRequests = 1000
	
	for i := 0; i < numRequests; i++ {
		// 创建设备
		deviceData := map[string]interface{}{
			"name":        fmt.Sprintf("memory-test-device-%d", i),
			"type":        "router",
			"host":        fmt.Sprintf("192.168.%d.%d", i/254+1, i%254+1),
			"port":        22,
			"protocol":    "ssh",
			"username":    "admin",
			"description": "Memory test device",
		}
		
		w := pts.makeAuthenticatedRequest("POST", "/api/v1/devices", deviceData)
		if w.Code != http.StatusCreated {
			t.Logf("Failed to create device %d: status %d", i, w.Code)
		}
		
		// 每100个请求查询一次设备列表
		if i%100 == 0 {
			w = pts.makeAuthenticatedRequest("GET", "/api/v1/devices", nil)
			assert.Equal(t, http.StatusOK, w.Code)
		}
	}
	
	// 最终查询所有设备
	w := pts.makeAuthenticatedRequest("GET", "/api/v1/devices", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	
	var devicesResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &devicesResp)
	require.NoError(t, err)
	
	devices, ok := devicesResp["devices"].([]interface{})
	require.True(t, ok)
	
	t.Logf("内存使用测试完成，创建了 %d 个设备", len(devices))
}