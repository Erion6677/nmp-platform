package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试设备配置（真实 MikroTik 设备）
// Feature: device-monitoring, Property 6: 主动采集数据完整性
// Validates: Requirements 7.3, 7.4
var testMikroTikDevice = struct {
	IP       string
	APIPort  int
	SSHPort  int
	Username string
	Password string
}{
	IP:       "10.10.10.254",
	APIPort:  8827,
	SSHPort:  3399,
	Username: "admin",
	Password: "927528",
}

// TestRouterOSCollector_Connect 测试 RouterOS API 连接
func TestRouterOSCollector_Connect(t *testing.T) {
	collector := NewRouterOSCollector(10 * time.Second)
	
	client, err := collector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.APIPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	
	require.NoError(t, err, "连接 RouterOS 设备失败")
	require.NotNil(t, client, "客户端不应为空")
	
	defer client.Close()
}

// TestRouterOSCollector_TestConnection 测试连接测试功能
func TestRouterOSCollector_TestConnection(t *testing.T) {
	collector := NewRouterOSCollector(10 * time.Second)
	
	err := collector.TestConnection(
		testMikroTikDevice.IP,
		testMikroTikDevice.APIPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	
	assert.NoError(t, err, "连接测试应该成功")
}

// TestRouterOSCollector_TestConnection_WrongPassword 测试错误密码
func TestRouterOSCollector_TestConnection_WrongPassword(t *testing.T) {
	collector := NewRouterOSCollector(5 * time.Second)
	
	err := collector.TestConnection(
		testMikroTikDevice.IP,
		testMikroTikDevice.APIPort,
		testMikroTikDevice.Username,
		"wrong_password",
	)
	
	assert.Error(t, err, "错误密码应该返回错误")
	assert.Contains(t, err.Error(), "用户名或密码错误", "应该返回认证错误")
}

// TestRouterOSCollector_TestConnection_WrongPort 测试错误端口
func TestRouterOSCollector_TestConnection_WrongPort(t *testing.T) {
	collector := NewRouterOSCollector(3 * time.Second)
	
	err := collector.TestConnection(
		testMikroTikDevice.IP,
		9999, // 错误端口
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	
	assert.Error(t, err, "错误端口应该返回错误")
}

// TestRouterOSCollector_GetSystemInfo 测试获取系统信息
// Feature: device-monitoring, Property 6: 主动采集数据完整性
// Validates: Requirements 7.3, 7.4
func TestRouterOSCollector_GetSystemInfo(t *testing.T) {
	collector := NewRouterOSCollector(10 * time.Second)
	
	client, err := collector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.APIPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	require.NoError(t, err, "连接失败")
	defer client.Close()
	
	info, err := collector.GetSystemInfo(client)
	require.NoError(t, err, "获取系统信息失败")
	require.NotNil(t, info, "系统信息不应为空")
	
	// 验证 Property 6: 主动采集数据完整性
	// 返回的数据应包含设备名称、IP、CPU核心数、系统版本、授权等级、运行时间、CPU使用率、内存使用率、总内存、可用内存
	t.Logf("设备名称: %s", info.DeviceName)
	t.Logf("CPU 核心数: %d", info.CPUCount)
	t.Logf("系统版本: %s", info.Version)
	t.Logf("授权等级: %s", info.License)
	t.Logf("运行时间: %d 秒", info.Uptime)
	t.Logf("CPU 使用率: %.2f%%", info.CPUUsage)
	t.Logf("内存使用率: %.2f%%", info.MemoryUsage)
	t.Logf("总内存: %d 字节", info.MemoryTotal)
	t.Logf("可用内存: %d 字节", info.MemoryFree)
	
	// 验证必要字段
	assert.NotEmpty(t, info.DeviceName, "设备名称不应为空")
	assert.Greater(t, info.CPUCount, 0, "CPU 核心数应大于 0")
	assert.NotEmpty(t, info.Version, "系统版本不应为空")
	assert.GreaterOrEqual(t, info.Uptime, int64(0), "运行时间应大于等于 0")
	assert.GreaterOrEqual(t, info.CPUUsage, float64(0), "CPU 使用率应大于等于 0")
	assert.LessOrEqual(t, info.CPUUsage, float64(100), "CPU 使用率应小于等于 100")
	assert.Greater(t, info.MemoryTotal, int64(0), "总内存应大于 0")
	assert.GreaterOrEqual(t, info.MemoryFree, int64(0), "可用内存应大于等于 0")
	assert.GreaterOrEqual(t, info.MemoryUsage, float64(0), "内存使用率应大于等于 0")
	assert.LessOrEqual(t, info.MemoryUsage, float64(100), "内存使用率应小于等于 100")
}

// TestRouterOSCollector_GetInterfaces 测试获取接口列表
func TestRouterOSCollector_GetInterfaces(t *testing.T) {
	collector := NewRouterOSCollector(10 * time.Second)
	
	client, err := collector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.APIPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	require.NoError(t, err, "连接失败")
	defer client.Close()
	
	interfaces, err := collector.GetInterfaces(client)
	require.NoError(t, err, "获取接口列表失败")
	require.NotEmpty(t, interfaces, "接口列表不应为空")
	
	t.Logf("接口数量: %d", len(interfaces))
	for _, iface := range interfaces {
		t.Logf("接口: %s, 状态: %s", iface.Name, iface.Status)
		
		// 验证接口数据只包含名称和状态
		assert.NotEmpty(t, iface.Name, "接口名称不应为空")
		assert.Contains(t, []string{"up", "down"}, iface.Status, "接口状态应为 up 或 down")
	}
}

// TestRouterOSCollector_ParseUptime 测试运行时间解析
func TestRouterOSCollector_ParseUptime(t *testing.T) {
	collector := NewRouterOSCollector(0)
	
	tests := []struct {
		input    string
		expected int64
	}{
		{"1s", 1},
		{"1m", 60},
		{"1h", 3600},
		{"1d", 86400},
		{"1w", 604800},
		{"1w2d3h4m5s", 604800 + 2*86400 + 3*3600 + 4*60 + 5},
		{"2d12h30m", 2*86400 + 12*3600 + 30*60},
		{"", 0},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := collector.parseUptime(tt.input)
			assert.Equal(t, tt.expected, result, "解析 %s 应得到 %d", tt.input, tt.expected)
		})
	}
}
