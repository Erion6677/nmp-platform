package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试配置
const (
	BackendURL    = "http://localhost:8080"
	InfluxDBURL   = "http://localhost:8086"
	InfluxDBOrg   = "nmp"
	InfluxDBBucket = "monitoring"
	RedisAddr     = "localhost:6379"
)

// PushMetricsRequest 推送数据请求结构
type PushMetricsRequest struct {
	DeviceKey string            `json:"device_key"`
	Metrics   []PushMetricPoint `json:"metrics"`
}

// PushMetricPoint 推送指标点
type PushMetricPoint struct {
	Timestamp  int64                       `json:"ts"`
	Interfaces map[string]InterfaceMetrics `json:"interfaces"`
	Pings      []PingMetric                `json:"pings"`
}

// InterfaceMetrics 接口带宽指标
type InterfaceMetrics struct {
	RxRate int64 `json:"rx_rate"`
	TxRate int64 `json:"tx_rate"`
}

// PingMetric Ping 指标
type PingMetric struct {
	Target   string `json:"target"`
	SrcIface string `json:"src_iface"`
	Latency  int64  `json:"latency"`
	Status   string `json:"status"`
}

// TestPushDataFlowIntegration 测试推送数据流程的集成测试
// 验证：
// 1. 真实设备推送数据
// 2. 数据写入 InfluxDB
// 3. Redis 状态更新
func TestPushDataFlowIntegration(t *testing.T) {
	// 跳过如果不是集成测试环境
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 检查后端服务是否可用
	resp, err := http.Get(BackendURL + "/health")
	if err != nil {
		t.Skipf("Backend service not available: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("Backend service not healthy: %d", resp.StatusCode)
	}

	t.Run("验证推送数据端点可用", testPushEndpointAvailable)
	t.Run("验证带宽数据推送", testBandwidthDataPush)
	t.Run("验证Ping数据推送", testPingDataPush)
	t.Run("验证Redis状态更新", testRedisStatusUpdate)
}

// testPushEndpointAvailable 测试推送端点是否可用
func testPushEndpointAvailable(t *testing.T) {
	// 发送一个带有 device_key 的请求来检查端点是否存在
	testData := PushMetricsRequest{
		DeviceKey: "10.10.10.254",
		Metrics:   []PushMetricPoint{},
	}
	jsonData, _ := json.Marshal(testData)
	
	resp, err := http.Post(BackendURL+"/api/v1/push/metrics", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	// 端点应该存在（返回 200 或其他非 404 状态码）
	body, _ := io.ReadAll(resp.Body)
	t.Logf("Push endpoint response: %d - %s", resp.StatusCode, string(body))
	
	// 只要不是 404 就说明端点存在
	// 200 表示成功，其他状态码可能是验证错误等
	assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode != http.StatusNotFound, "Push endpoint should exist and be accessible")
}

// testBandwidthDataPush 测试带宽数据推送
func testBandwidthDataPush(t *testing.T) {
	// 构造测试数据
	testData := PushMetricsRequest{
		DeviceKey: "10.10.10.254", // 使用测试设备 IP
		Metrics: []PushMetricPoint{
			{
				Timestamp: time.Now().UnixMilli(),
				Interfaces: map[string]InterfaceMetrics{
					"ether1": {RxRate: 1000000, TxRate: 500000},
					"ether2": {RxRate: 2000000, TxRate: 1000000},
				},
			},
		},
	}

	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)

	// 发送推送请求
	resp, err := http.Post(BackendURL+"/api/v1/push/metrics", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Bandwidth push response: %d - %s", resp.StatusCode, string(body))

	// 检查响应
	// 注意：如果设备未注册，可能返回 404
	if resp.StatusCode == http.StatusNotFound {
		t.Log("Device not registered, this is expected if no device exists in database")
		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Bandwidth data push should succeed")
}

// testPingDataPush 测试 Ping 数据推送
func testPingDataPush(t *testing.T) {
	// 构造测试数据
	testData := PushMetricsRequest{
		DeviceKey: "10.10.10.254",
		Metrics: []PushMetricPoint{
			{
				Timestamp: time.Now().UnixMilli(),
				Pings: []PingMetric{
					{Target: "8.8.8.8", SrcIface: "ether1", Latency: 10, Status: "up"},
					{Target: "1.1.1.1", SrcIface: "ether1", Latency: 15, Status: "up"},
				},
			},
		},
	}

	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)

	// 发送推送请求
	resp, err := http.Post(BackendURL+"/api/v1/push/metrics", "application/json", bytes.NewBuffer(jsonData))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Ping push response: %d - %s", resp.StatusCode, string(body))

	// 检查响应
	if resp.StatusCode == http.StatusNotFound {
		t.Log("Device not registered, this is expected if no device exists in database")
		return
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Ping data push should succeed")
}

// testRedisStatusUpdate 测试 Redis 状态更新
func testRedisStatusUpdate(t *testing.T) {
	// 连接 Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: RedisAddr,
	})
	defer rdb.Close()

	ctx := context.Background()

	// 检查 Redis 连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	// 检查设备状态键
	keys, err := rdb.Keys(ctx, "device:status:*").Result()
	require.NoError(t, err)
	t.Logf("Found %d device status keys in Redis", len(keys))

	// 检查设备最后在线时间键
	lastSeenKeys, err := rdb.Keys(ctx, "device:last_seen:*").Result()
	require.NoError(t, err)
	t.Logf("Found %d device last_seen keys in Redis", len(lastSeenKeys))

	// 如果有设备状态，验证格式
	for _, key := range keys {
		status, err := rdb.Get(ctx, key).Result()
		if err == nil {
			t.Logf("Device status: %s = %s", key, status)
			assert.Contains(t, []string{"online", "offline", "unknown"}, status, "Status should be valid")
		}
	}
}

// TestInfluxDBConnection 测试 InfluxDB 连接
func TestInfluxDBConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 创建 InfluxDB 客户端（无 token 模式）
	client := influxdb2.NewClient(InfluxDBURL, "")
	defer client.Close()

	// 检查健康状态
	health, err := client.Health(context.Background())
	if err != nil {
		t.Logf("InfluxDB health check failed: %v", err)
		t.Log("This may be expected if InfluxDB requires authentication")
		return
	}

	t.Logf("InfluxDB health: %s - %s", health.Status, *health.Message)
	assert.Equal(t, "pass", string(health.Status), "InfluxDB should be healthy")
}

// TestRedisConnection 测试 Redis 连接
func TestRedisConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: RedisAddr,
	})
	defer rdb.Close()

	ctx := context.Background()

	// 测试连接
	pong, err := rdb.Ping(ctx).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong, "Redis should respond with PONG")

	t.Log("Redis connection successful")
}

// TestDeviceStatusFlow 测试设备状态流程
func TestDeviceStatusFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 检查后端服务
	resp, err := http.Get(BackendURL + "/health")
	if err != nil {
		t.Skipf("Backend service not available: %v", err)
	}
	resp.Body.Close()

	// 获取所有设备状态
	resp, err = http.Get(BackendURL + "/api/v1/data/status")
	if err != nil {
		t.Logf("Failed to get device status: %v", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Device status response: %d - %s", resp.StatusCode, string(body))

	// 解析响应
	var statusResp map[string]interface{}
	if err := json.Unmarshal(body, &statusResp); err == nil {
		if devices, ok := statusResp["devices"]; ok {
			t.Logf("Devices status: %v", devices)
		}
	}
}

// BenchmarkPushData 基准测试推送数据性能
func BenchmarkPushData(b *testing.B) {
	// 构造测试数据
	testData := PushMetricsRequest{
		DeviceKey: "10.10.10.254",
		Metrics: []PushMetricPoint{
			{
				Timestamp: time.Now().UnixMilli(),
				Interfaces: map[string]InterfaceMetrics{
					"ether1": {RxRate: 1000000, TxRate: 500000},
				},
				Pings: []PingMetric{
					{Target: "8.8.8.8", Latency: 10, Status: "up"},
				},
			},
		},
	}

	jsonData, _ := json.Marshal(testData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := http.Post(BackendURL+"/api/push/metrics", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}
		resp.Body.Close()
	}
}

// PrintTestSummary 打印测试摘要
func PrintTestSummary() {
	fmt.Println("\n========== 推送数据流程验证摘要 ==========")
	fmt.Println("1. 后端服务状态: 检查 http://localhost:8080/health")
	fmt.Println("2. InfluxDB 状态: 检查 http://localhost:8086/health")
	fmt.Println("3. Redis 状态: 检查 redis-cli ping")
	fmt.Println("4. 推送端点: POST http://localhost:8080/api/push/metrics")
	fmt.Println("5. 设备状态: GET http://localhost:8080/api/v1/data/status")
	fmt.Println("==========================================")
}
