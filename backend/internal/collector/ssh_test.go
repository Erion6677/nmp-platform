package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSHCollector_Connect 测试 SSH 连接
func TestSSHCollector_Connect(t *testing.T) {
	collector := NewSSHCollector(10 * time.Second)

	client, err := collector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.SSHPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)

	require.NoError(t, err, "连接 SSH 失败")
	require.NotNil(t, client, "客户端不应为空")

	defer client.Close()
}

// TestSSHCollector_TestConnection 测试连接测试功能
func TestSSHCollector_TestConnection(t *testing.T) {
	collector := NewSSHCollector(10 * time.Second)

	err := collector.TestConnection(
		testMikroTikDevice.IP,
		testMikroTikDevice.SSHPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)

	assert.NoError(t, err, "连接测试应该成功")
}

// TestSSHCollector_TestConnection_WrongPassword 测试错误密码
func TestSSHCollector_TestConnection_WrongPassword(t *testing.T) {
	collector := NewSSHCollector(5 * time.Second)

	err := collector.TestConnection(
		testMikroTikDevice.IP,
		testMikroTikDevice.SSHPort,
		testMikroTikDevice.Username,
		"wrong_password",
	)

	assert.Error(t, err, "错误密码应该返回错误")
	assert.Contains(t, err.Error(), "用户名或密码错误", "应该返回认证错误")
}

// TestSSHCollector_TestConnection_WrongPort 测试错误端口
func TestSSHCollector_TestConnection_WrongPort(t *testing.T) {
	collector := NewSSHCollector(3 * time.Second)

	err := collector.TestConnection(
		testMikroTikDevice.IP,
		9999, // 错误端口
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)

	assert.Error(t, err, "错误端口应该返回错误")
}

// TestSSHCollector_GetMikroTikSystemInfo 测试获取 MikroTik 系统信息（通过 SSH）
// Feature: device-monitoring, Property 6: 主动采集数据完整性
// Validates: Requirements 7.3, 7.4
func TestSSHCollector_GetMikroTikSystemInfo(t *testing.T) {
	collector := NewSSHCollector(10 * time.Second)

	client, err := collector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.SSHPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	require.NoError(t, err, "连接失败")
	defer client.Close()

	info, err := collector.GetMikroTikSystemInfo(client)
	require.NoError(t, err, "获取系统信息失败")
	require.NotNil(t, info, "系统信息不应为空")

	// 验证 Property 6: 主动采集数据完整性
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

// TestSSHCollector_GetMikroTikInterfaces 测试获取 MikroTik 接口列表（通过 SSH）
func TestSSHCollector_GetMikroTikInterfaces(t *testing.T) {
	collector := NewSSHCollector(10 * time.Second)

	client, err := collector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.SSHPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	require.NoError(t, err, "连接失败")
	defer client.Close()

	interfaces, err := collector.GetMikroTikInterfaces(client)
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

// TestSSHCollector_CompareWithAPI 比较 SSH 和 API 采集结果
func TestSSHCollector_CompareWithAPI(t *testing.T) {
	// 通过 API 获取系统信息
	rosCollector := NewRouterOSCollector(10 * time.Second)
	apiClient, err := rosCollector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.APIPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	require.NoError(t, err, "API 连接失败")
	defer apiClient.Close()

	apiInfo, err := rosCollector.GetSystemInfo(apiClient)
	require.NoError(t, err, "API 获取系统信息失败")

	// 通过 SSH 获取系统信息
	sshCollector := NewSSHCollector(10 * time.Second)
	sshClient, err := sshCollector.Connect(
		testMikroTikDevice.IP,
		testMikroTikDevice.SSHPort,
		testMikroTikDevice.Username,
		testMikroTikDevice.Password,
	)
	require.NoError(t, err, "SSH 连接失败")
	defer sshClient.Close()

	sshInfo, err := sshCollector.GetMikroTikSystemInfo(sshClient)
	require.NoError(t, err, "SSH 获取系统信息失败")

	// 比较关键字段（允许一定误差）
	t.Logf("API 设备名称: %s, SSH 设备名称: %s", apiInfo.DeviceName, sshInfo.DeviceName)
	t.Logf("API CPU 核心数: %d, SSH CPU 核心数: %d", apiInfo.CPUCount, sshInfo.CPUCount)
	t.Logf("API 系统版本: %s, SSH 系统版本: %s", apiInfo.Version, sshInfo.Version)

	assert.Equal(t, apiInfo.DeviceName, sshInfo.DeviceName, "设备名称应一致")
	assert.Equal(t, apiInfo.CPUCount, sshInfo.CPUCount, "CPU 核心数应一致")
	assert.Equal(t, apiInfo.Version, sshInfo.Version, "系统版本应一致")
	assert.Equal(t, apiInfo.MemoryTotal, sshInfo.MemoryTotal, "总内存应一致")
}
