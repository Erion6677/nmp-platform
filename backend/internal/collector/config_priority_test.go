package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Feature: device-monitoring, Property 10: 配置优先级
// Validates: Requirements 4.8, 11.5
//
// Property 10: 配置优先级
// *For any* 设备，如果设备有单独配置的推送间隔，应使用设备配置；否则应使用全局默认配置。

// ConfigPriorityResolver 配置优先级解析器
// 用于测试配置优先级逻辑
type ConfigPriorityResolver struct {
	GlobalDefaultInterval int // 全局默认推送间隔
}

// NewConfigPriorityResolver 创建配置优先级解析器
func NewConfigPriorityResolver(globalDefault int) *ConfigPriorityResolver {
	return &ConfigPriorityResolver{
		GlobalDefaultInterval: globalDefault,
	}
}

// ResolveInterval 解析设备的实际推送间隔
// 如果设备有单独配置（deviceInterval > 0），使用设备配置
// 否则使用全局默认配置
func (r *ConfigPriorityResolver) ResolveInterval(deviceInterval int) int {
	if deviceInterval > 0 {
		return deviceInterval
	}
	return r.GlobalDefaultInterval
}

// TestProperty10_ConfigPriority_DeviceOverridesGlobal 测试设备配置覆盖全局配置
// Feature: device-monitoring, Property 10: 配置优先级
// Validates: Requirements 4.8, 11.5
func TestProperty10_ConfigPriority_DeviceOverridesGlobal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 生成随机的全局默认间隔 (100-10000ms)
		globalDefault := rapid.IntRange(100, 10000).Draw(t, "globalDefault")
		
		// 生成随机的设备间隔 (100-10000ms)
		deviceInterval := rapid.IntRange(100, 10000).Draw(t, "deviceInterval")
		
		resolver := NewConfigPriorityResolver(globalDefault)
		
		// 当设备有单独配置时，应使用设备配置
		result := resolver.ResolveInterval(deviceInterval)
		
		assert.Equal(t, deviceInterval, result, 
			"当设备有单独配置时，应使用设备配置。全局=%d, 设备=%d, 结果=%d",
			globalDefault, deviceInterval, result)
	})
}

// TestProperty10_ConfigPriority_FallbackToGlobal 测试回退到全局配置
// Feature: device-monitoring, Property 10: 配置优先级
// Validates: Requirements 4.8, 11.5
func TestProperty10_ConfigPriority_FallbackToGlobal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 生成随机的全局默认间隔 (100-10000ms)
		globalDefault := rapid.IntRange(100, 10000).Draw(t, "globalDefault")
		
		resolver := NewConfigPriorityResolver(globalDefault)
		
		// 当设备没有单独配置时（interval = 0），应使用全局配置
		result := resolver.ResolveInterval(0)
		
		assert.Equal(t, globalDefault, result,
			"当设备没有单独配置时，应使用全局配置。全局=%d, 结果=%d",
			globalDefault, result)
	})
}

// TestProperty10_ConfigPriority_Consistency 测试配置优先级一致性
// Feature: device-monitoring, Property 10: 配置优先级
// Validates: Requirements 4.8, 11.5
func TestProperty10_ConfigPriority_Consistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 生成随机的全局默认间隔
		globalDefault := rapid.IntRange(100, 10000).Draw(t, "globalDefault")
		
		// 生成随机的设备间隔（可能为0表示未配置）
		deviceInterval := rapid.IntRange(0, 10000).Draw(t, "deviceInterval")
		
		resolver := NewConfigPriorityResolver(globalDefault)
		
		// 多次调用应返回相同结果（一致性）
		result1 := resolver.ResolveInterval(deviceInterval)
		result2 := resolver.ResolveInterval(deviceInterval)
		
		assert.Equal(t, result1, result2,
			"相同输入应返回相同结果。设备间隔=%d, 结果1=%d, 结果2=%d",
			deviceInterval, result1, result2)
		
		// 验证结果的正确性
		if deviceInterval > 0 {
			assert.Equal(t, deviceInterval, result1,
				"设备有配置时应使用设备配置")
		} else {
			assert.Equal(t, globalDefault, result1,
				"设备无配置时应使用全局配置")
		}
	})
}

// TestProperty10_ConfigPriority_MultipleDevices 测试多设备配置优先级
// Feature: device-monitoring, Property 10: 配置优先级
// Validates: Requirements 4.8, 11.5
func TestProperty10_ConfigPriority_MultipleDevices(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// 生成随机的全局默认间隔
		globalDefault := rapid.IntRange(100, 10000).Draw(t, "globalDefault")
		
		// 生成多个设备的配置
		numDevices := rapid.IntRange(1, 10).Draw(t, "numDevices")
		deviceIntervals := make([]int, numDevices)
		for i := 0; i < numDevices; i++ {
			// 50% 概率设备有单独配置
			if rapid.Bool().Draw(t, "hasConfig") {
				deviceIntervals[i] = rapid.IntRange(100, 10000).Draw(t, "interval")
			} else {
				deviceIntervals[i] = 0
			}
		}
		
		resolver := NewConfigPriorityResolver(globalDefault)
		
		// 验证每个设备的配置优先级
		for i, deviceInterval := range deviceIntervals {
			result := resolver.ResolveInterval(deviceInterval)
			
			if deviceInterval > 0 {
				assert.Equal(t, deviceInterval, result,
					"设备 %d 有配置时应使用设备配置", i)
			} else {
				assert.Equal(t, globalDefault, result,
					"设备 %d 无配置时应使用全局配置", i)
			}
		}
	})
}

// TestConfigPriority_EdgeCases 测试边界情况
func TestConfigPriority_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		globalDefault  int
		deviceInterval int
		expected       int
	}{
		{
			name:           "设备配置为0，使用全局",
			globalDefault:  1000,
			deviceInterval: 0,
			expected:       1000,
		},
		{
			name:           "设备配置大于0，使用设备",
			globalDefault:  1000,
			deviceInterval: 500,
			expected:       500,
		},
		{
			name:           "设备配置等于全局",
			globalDefault:  1000,
			deviceInterval: 1000,
			expected:       1000,
		},
		{
			name:           "设备配置大于全局",
			globalDefault:  1000,
			deviceInterval: 5000,
			expected:       5000,
		},
		{
			name:           "最小间隔",
			globalDefault:  100,
			deviceInterval: 100,
			expected:       100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewConfigPriorityResolver(tt.globalDefault)
			result := resolver.ResolveInterval(tt.deviceInterval)
			assert.Equal(t, tt.expected, result)
		})
	}
}
