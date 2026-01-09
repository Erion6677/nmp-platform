package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试设备配置（真实 MikroTik 设备）
// Feature: device-monitoring
// Validates: Requirements 4.1, 4.2, 4.3
var testDevice = struct {
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

// TestScriptGenerator_GenerateMikroTikSimpleScript 测试脚本生成
func TestScriptGenerator_GenerateMikroTikSimpleScript(t *testing.T) {
	generator := NewScriptGenerator("http://localhost:8080")

	config := &ScriptConfig{
		DeviceID:      1,
		DeviceIP:      "10.10.10.254",
		ServerURL:     "http://localhost:8080/api/push/metrics",
		IntervalMs:    1000,
		PushBatchSize: 10,
		ScriptName:    "nmp-collector",
		SchedulerName: "nmp-scheduler",
		Interfaces:    []string{"ether1", "ether2"},
		PingTargets: []PingTargetConfig{
			{TargetAddress: "8.8.8.8", TargetName: "Google DNS", SourceInterface: ""},
			{TargetAddress: "1.1.1.1", TargetName: "Cloudflare", SourceInterface: "ether1"},
		},
	}

	script := generator.GenerateMikroTikSimpleScript(config)

	assert.NotEmpty(t, script, "生成的脚本不应为空")
	assert.Contains(t, script, "10.10.10.254", "脚本应包含设备IP")
	assert.Contains(t, script, "http://localhost:8080/api/push/metrics", "脚本应包含服务器URL")
	assert.Contains(t, script, "ether1", "脚本应包含接口名称")
	assert.Contains(t, script, "ether2", "脚本应包含接口名称")
	assert.Contains(t, script, "8.8.8.8", "脚本应包含Ping目标")
	assert.Contains(t, script, "1.1.1.1", "脚本应包含Ping目标")

	t.Logf("生成的脚本:\n%s", script)
}

// TestScriptGenerator_GenerateMikroTikScheduler 测试调度器命令生成
func TestScriptGenerator_GenerateMikroTikScheduler(t *testing.T) {
	generator := NewScriptGenerator("http://localhost:8080")

	config := &ScriptConfig{
		IntervalMs:    1000,
		ScriptName:    "nmp-collector",
		SchedulerName: "nmp-scheduler",
	}

	cmd := generator.GenerateMikroTikScheduler(config)

	assert.NotEmpty(t, cmd, "调度器命令不应为空")
	assert.Contains(t, cmd, "nmp-scheduler", "命令应包含调度器名称")
	assert.Contains(t, cmd, "nmp-collector", "命令应包含脚本名称")
	assert.Contains(t, cmd, "00:00:01", "命令应包含间隔时间")

	t.Logf("调度器命令: %s", cmd)
}

// TestScriptGenerator_msToRouterOSInterval 测试时间间隔转换
func TestScriptGenerator_msToRouterOSInterval(t *testing.T) {
	generator := NewScriptGenerator("")

	tests := []struct {
		ms       int
		expected string
	}{
		{500, "00:00:01"},    // 小于1秒，返回1秒
		{1000, "00:00:01"},   // 1秒
		{5000, "00:00:05"},   // 5秒
		{60000, "00:01:00"},  // 1分钟
		{3600000, "01:00:00"}, // 1小时
		{3661000, "01:01:01"}, // 1小时1分1秒
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := generator.msToRouterOSInterval(tt.ms)
			assert.Equal(t, tt.expected, result, "%d ms 应转换为 %s", tt.ms, tt.expected)
		})
	}
}

// TestDeployer_DeployToMikroTik 测试部署脚本到真实设备
// Feature: device-monitoring
// Validates: Requirements 4.1, 4.2, 4.3
func TestDeployer_DeployToMikroTik(t *testing.T) {
	deployer := NewDeployer("http://localhost:8080")

	config := &ScriptConfig{
		DeviceID:      1,
		DeviceIP:      testDevice.IP,
		ServerURL:     "http://localhost:8080/api/push/metrics",
		IntervalMs:    5000, // 5秒间隔，避免频繁执行
		PushBatchSize: 1,
		ScriptName:    "nmp-test-collector",
		SchedulerName: "nmp-test-scheduler",
		Interfaces:    []string{"ether1"},
		PingTargets:   []PingTargetConfig{},
	}

	// 部署脚本
	result := deployer.DeployToMikroTik(
		config,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)

	require.True(t, result.Success, "部署应该成功: %s", result.ErrorMessage)
	t.Logf("部署方式: %s, 消息: %s", result.Method, result.Message)

	// 验证脚本存在
	exists, err := deployer.CheckScriptExists(
		config.ScriptName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	require.NoError(t, err, "检查脚本存在失败")
	assert.True(t, exists, "脚本应该存在")

	// 清理：移除测试脚本
	removeResult := deployer.RemoveFromMikroTik(
		config.ScriptName,
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	assert.True(t, removeResult.Success, "移除脚本应该成功: %s", removeResult.ErrorMessage)
}

// TestDeployer_RemoveFromMikroTik 测试从设备移除脚本
// Feature: device-monitoring
// Validates: Requirements 4.4
func TestDeployer_RemoveFromMikroTik(t *testing.T) {
	deployer := NewDeployer("http://localhost:8080")

	// 先部署一个测试脚本
	config := &ScriptConfig{
		DeviceID:      1,
		DeviceIP:      testDevice.IP,
		ServerURL:     "http://localhost:8080/api/push/metrics",
		IntervalMs:    5000,
		PushBatchSize: 1,
		ScriptName:    "nmp-remove-test",
		SchedulerName: "nmp-remove-test-scheduler",
		Interfaces:    []string{},
		PingTargets:   []PingTargetConfig{},
	}

	deployResult := deployer.DeployToMikroTik(
		config,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	require.True(t, deployResult.Success, "部署应该成功")

	// 移除脚本
	removeResult := deployer.RemoveFromMikroTik(
		config.ScriptName,
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)

	assert.True(t, removeResult.Success, "移除应该成功: %s", removeResult.ErrorMessage)
	t.Logf("移除方式: %s, 消息: %s", removeResult.Method, removeResult.Message)

	// 验证脚本已被移除
	time.Sleep(500 * time.Millisecond) // 等待设备处理
	exists, _ := deployer.CheckScriptExists(
		config.ScriptName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	assert.False(t, exists, "脚本应该已被移除")
}

// TestDeployer_EnableDisableScheduler 测试启用/禁用调度器
// Feature: device-monitoring
// Validates: Requirements 4.6, 4.7
func TestDeployer_EnableDisableScheduler(t *testing.T) {
	deployer := NewDeployer("http://localhost:8080")

	// 先部署一个测试脚本
	config := &ScriptConfig{
		DeviceID:      1,
		DeviceIP:      testDevice.IP,
		ServerURL:     "http://localhost:8080/api/push/metrics",
		IntervalMs:    5000,
		PushBatchSize: 1,
		ScriptName:    "nmp-toggle-test",
		SchedulerName: "nmp-toggle-test-scheduler",
		Interfaces:    []string{},
		PingTargets:   []PingTargetConfig{},
	}

	deployResult := deployer.DeployToMikroTik(
		config,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	require.True(t, deployResult.Success, "部署应该成功")

	// 禁用调度器
	disableResult := deployer.DisableScheduler(
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	assert.True(t, disableResult.Success, "禁用应该成功: %s", disableResult.ErrorMessage)

	// 检查状态
	status, err := deployer.GetScriptStatus(
		config.ScriptName,
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	require.NoError(t, err, "获取状态失败")
	assert.False(t, status.SchedulerEnabled, "调度器应该被禁用")

	// 启用调度器
	enableResult := deployer.EnableScheduler(
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	assert.True(t, enableResult.Success, "启用应该成功: %s", enableResult.ErrorMessage)

	// 检查状态
	status, err = deployer.GetScriptStatus(
		config.ScriptName,
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
	require.NoError(t, err, "获取状态失败")
	assert.True(t, status.SchedulerEnabled, "调度器应该被启用")

	// 清理
	deployer.RemoveFromMikroTik(
		config.ScriptName,
		config.SchedulerName,
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)
}

// TestDeployer_GetScriptStatus 测试获取脚本状态
func TestDeployer_GetScriptStatus(t *testing.T) {
	deployer := NewDeployer("http://localhost:8080")

	// 测试不存在的脚本
	status, err := deployer.GetScriptStatus(
		"non-existent-script",
		"non-existent-scheduler",
		testDevice.IP,
		testDevice.APIPort,
		testDevice.SSHPort,
		testDevice.Username,
		testDevice.Password,
	)

	require.NoError(t, err, "获取状态不应返回错误")
	assert.False(t, status.ScriptExists, "不存在的脚本应返回 false")
	assert.False(t, status.SchedulerExists, "不存在的调度器应返回 false")
}
