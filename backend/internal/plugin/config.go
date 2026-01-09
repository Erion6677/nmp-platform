package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// DefaultPluginConfig 默认插件配置实现
type DefaultPluginConfig struct {
	configDir string
	configs   map[string]interface{}
	mutex     sync.RWMutex
}

// NewDefaultPluginConfig 创建默认插件配置
func NewDefaultPluginConfig(configDir string) *DefaultPluginConfig {
	return &DefaultPluginConfig{
		configDir: configDir,
		configs:   make(map[string]interface{}),
	}
}

// GetConfig 获取插件配置
func (pc *DefaultPluginConfig) GetConfig(pluginName string) (interface{}, error) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	// 先从内存中查找
	if config, exists := pc.configs[pluginName]; exists {
		return config, nil
	}

	// 从文件加载配置
	config, err := pc.loadConfigFromFile(pluginName)
	if err != nil {
		return nil, err
	}

	// 缓存到内存
	pc.configs[pluginName] = config
	return config, nil
}

// SetConfig 设置插件配置
func (pc *DefaultPluginConfig) SetConfig(pluginName string, config interface{}) error {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	// 保存到内存
	pc.configs[pluginName] = config

	// 保存到文件
	return pc.saveConfigToFile(pluginName, config)
}

// ValidateConfig 验证插件配置
func (pc *DefaultPluginConfig) ValidateConfig(pluginName string, config interface{}) error {
	// 基础验证：检查配置是否为nil
	if config == nil {
		return nil // nil配置是允许的
	}

	// 检查配置是否可以序列化
	_, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("config is not serializable: %w", err)
	}

	return nil
}

// loadConfigFromFile 从文件加载配置
func (pc *DefaultPluginConfig) loadConfigFromFile(pluginName string) (interface{}, error) {
	// 尝试加载YAML配置文件
	yamlPath := filepath.Join(pc.configDir, pluginName+".yaml")
	if _, err := os.Stat(yamlPath); err == nil {
		return pc.loadYAMLConfig(yamlPath)
	}

	// 尝试加载JSON配置文件
	jsonPath := filepath.Join(pc.configDir, pluginName+".json")
	if _, err := os.Stat(jsonPath); err == nil {
		return pc.loadJSONConfig(jsonPath)
	}

	// 配置文件不存在，返回空配置
	return make(map[string]interface{}), nil
}

// loadYAMLConfig 加载YAML配置
func (pc *DefaultPluginConfig) loadYAMLConfig(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config %s: %w", path, err)
	}

	return config, nil
}

// loadJSONConfig 加载JSON配置
func (pc *DefaultPluginConfig) loadJSONConfig(path string) (interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config %s: %w", path, err)
	}

	return config, nil
}

// saveConfigToFile 保存配置到文件
func (pc *DefaultPluginConfig) saveConfigToFile(pluginName string, config interface{}) error {
	// 确保配置目录存在
	if err := os.MkdirAll(pc.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 保存为YAML格式
	yamlPath := filepath.Join(pc.configDir, pluginName+".yaml")
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile(yamlPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", yamlPath, err)
	}

	return nil
}

// ReloadConfig 重新加载插件配置
func (pc *DefaultPluginConfig) ReloadConfig(pluginName string) error {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	// 从文件重新加载配置
	config, err := pc.loadConfigFromFile(pluginName)
	if err != nil {
		return err
	}

	// 更新内存中的配置
	pc.configs[pluginName] = config
	return nil
}

// ListConfigs 列出所有插件配置
func (pc *DefaultPluginConfig) ListConfigs() map[string]interface{} {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()

	configs := make(map[string]interface{})
	for name, config := range pc.configs {
		configs[name] = config
	}
	return configs
}

// DeleteConfig 删除插件配置
func (pc *DefaultPluginConfig) DeleteConfig(pluginName string) error {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()

	// 从内存中删除
	delete(pc.configs, pluginName)

	// 删除配置文件
	yamlPath := filepath.Join(pc.configDir, pluginName+".yaml")
	jsonPath := filepath.Join(pc.configDir, pluginName+".json")

	// 尝试删除YAML文件
	if _, err := os.Stat(yamlPath); err == nil {
		if err := os.Remove(yamlPath); err != nil {
			return fmt.Errorf("failed to delete YAML config file: %w", err)
		}
	}

	// 尝试删除JSON文件
	if _, err := os.Stat(jsonPath); err == nil {
		if err := os.Remove(jsonPath); err != nil {
			return fmt.Errorf("failed to delete JSON config file: %w", err)
		}
	}

	return nil
}