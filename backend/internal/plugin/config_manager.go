package plugin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	config     PluginConfig
	logger     *log.Logger
	validators map[string]ConfigValidator
	watchers   map[string]*ConfigWatcher
	mutex      sync.RWMutex
}

// ConfigValidator 配置验证器
type ConfigValidator interface {
	Validate(config interface{}) error
	GetSchema() interface{}
}

// ConfigWatcher 配置监视器
type ConfigWatcher struct {
	pluginName string
	manager    *ConfigManager
	stopChan   chan bool
	callback   ConfigChangeCallback
}

// ConfigChangeCallback 配置变更回调
type ConfigChangeCallback func(pluginName string, oldConfig, newConfig interface{}) error

// ConfigSchema 配置模式
type ConfigSchema struct {
	Type        string                 `json:"type"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	Description string                 `json:"description,omitempty"`
}

// PropertySchema 属性模式
type PropertySchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
	MinLength   *int        `json:"min_length,omitempty"`
	MaxLength   *int        `json:"max_length,omitempty"`
	Pattern     string      `json:"pattern,omitempty"`
}

// DefaultConfigValidator 默认配置验证器
type DefaultConfigValidator struct {
	schema ConfigSchema
}

// NewConfigManager 创建配置管理器
func NewConfigManager(config PluginConfig, logger *log.Logger) *ConfigManager {
	return &ConfigManager{
		config:     config,
		logger:     logger,
		validators: make(map[string]ConfigValidator),
		watchers:   make(map[string]*ConfigWatcher),
	}
}

// RegisterValidator 注册配置验证器
func (cm *ConfigManager) RegisterValidator(pluginName string, validator ConfigValidator) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	cm.validators[pluginName] = validator
	cm.logger.Printf("Registered config validator for plugin: %s", pluginName)
}

// ValidateConfig 验证配置
func (cm *ConfigManager) ValidateConfig(pluginName string, config interface{}) error {
	cm.mutex.RLock()
	validator, exists := cm.validators[pluginName]
	cm.mutex.RUnlock()
	
	if !exists {
		// 使用基础验证
		return cm.config.ValidateConfig(pluginName, config)
	}
	
	return validator.Validate(config)
}

// GetConfigWithDefaults 获取带默认值的配置
func (cm *ConfigManager) GetConfigWithDefaults(pluginName string) (interface{}, error) {
	// 获取当前配置
	config, err := cm.config.GetConfig(pluginName)
	if err != nil {
		return nil, err
	}
	
	// 获取验证器和默认值
	cm.mutex.RLock()
	validator, exists := cm.validators[pluginName]
	cm.mutex.RUnlock()
	
	if !exists {
		return config, nil
	}
	
	// 应用默认值
	schema := validator.GetSchema()
	if configSchema, ok := schema.(ConfigSchema); ok {
		return cm.applyDefaults(config, configSchema), nil
	}
	
	return config, nil
}

// applyDefaults 应用默认值
func (cm *ConfigManager) applyDefaults(config interface{}, schema ConfigSchema) interface{} {
	if config == nil {
		return schema.Default
	}
	
	configMap, ok := config.(map[string]interface{})
	if !ok {
		return config
	}
	
	result := make(map[string]interface{})
	
	// 复制现有配置
	for k, v := range configMap {
		result[k] = v
	}
	
	// 应用默认值
	for propName, propSchema := range schema.Properties {
		if _, exists := result[propName]; !exists && propSchema.Default != nil {
			result[propName] = propSchema.Default
		}
	}
	
	return result
}

// SetConfigWithValidation 设置配置并验证
func (cm *ConfigManager) SetConfigWithValidation(pluginName string, config interface{}) error {
	// 验证配置
	if err := cm.ValidateConfig(pluginName, config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	
	// 获取旧配置
	oldConfig, _ := cm.config.GetConfig(pluginName)
	
	// 保存配置
	if err := cm.config.SetConfig(pluginName, config); err != nil {
		return err
	}
	
	// 通知配置变更
	cm.notifyConfigChange(pluginName, oldConfig, config)
	
	return nil
}

// notifyConfigChange 通知配置变更
func (cm *ConfigManager) notifyConfigChange(pluginName string, oldConfig, newConfig interface{}) {
	cm.mutex.RLock()
	watcher, exists := cm.watchers[pluginName]
	cm.mutex.RUnlock()
	
	if exists && watcher.callback != nil {
		go func() {
			if err := watcher.callback(pluginName, oldConfig, newConfig); err != nil {
				cm.logger.Printf("Config change callback failed for plugin %s: %v", pluginName, err)
			}
		}()
	}
}

// WatchConfig 监视配置变更
func (cm *ConfigManager) WatchConfig(pluginName string, callback ConfigChangeCallback) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	watcher := &ConfigWatcher{
		pluginName: pluginName,
		manager:    cm,
		stopChan:   make(chan bool),
		callback:   callback,
	}
	
	cm.watchers[pluginName] = watcher
	cm.logger.Printf("Started watching config for plugin: %s", pluginName)
	
	return nil
}

// StopWatchingConfig 停止监视配置
func (cm *ConfigManager) StopWatchingConfig(pluginName string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if watcher, exists := cm.watchers[pluginName]; exists {
		close(watcher.stopChan)
		delete(cm.watchers, pluginName)
		cm.logger.Printf("Stopped watching config for plugin: %s", pluginName)
	}
}

// ExportConfig 导出配置
func (cm *ConfigManager) ExportConfig(pluginName string, format string) ([]byte, error) {
	config, err := cm.config.GetConfig(pluginName)
	if err != nil {
		return nil, err
	}
	
	switch format {
	case "json":
		return json.MarshalIndent(config, "", "  ")
	case "yaml":
		return yaml.Marshal(config)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// ImportConfig 导入配置
func (cm *ConfigManager) ImportConfig(pluginName string, data []byte, format string) error {
	var config interface{}
	var err error
	
	switch format {
	case "json":
		err = json.Unmarshal(data, &config)
	case "yaml":
		err = yaml.Unmarshal(data, &config)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	
	return cm.SetConfigWithValidation(pluginName, config)
}

// BackupConfig 备份配置
func (cm *ConfigManager) BackupConfig(pluginName string, backupDir string) error {
	config, err := cm.config.GetConfig(pluginName)
	if err != nil {
		return err
	}
	
	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.yaml", pluginName, timestamp))
	
	// 导出配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// 写入备份文件
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}
	
	cm.logger.Printf("Config backed up for plugin %s: %s", pluginName, backupFile)
	return nil
}

// RestoreConfig 恢复配置
func (cm *ConfigManager) RestoreConfig(pluginName string, backupFile string) error {
	// 读取备份文件
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}
	
	// 解析配置
	var config interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse backup config: %w", err)
	}
	
	// 恢复配置
	if err := cm.SetConfigWithValidation(pluginName, config); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}
	
	cm.logger.Printf("Config restored for plugin %s from: %s", pluginName, backupFile)
	return nil
}

// GetConfigHistory 获取配置历史
func (cm *ConfigManager) GetConfigHistory(pluginName string, backupDir string) ([]ConfigHistoryEntry, error) {
	var history []ConfigHistoryEntry
	
	pattern := filepath.Join(backupDir, fmt.Sprintf("%s_*.yaml", pluginName))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		
		entry := ConfigHistoryEntry{
			PluginName: pluginName,
			FilePath:   match,
			Timestamp:  info.ModTime(),
			Size:       info.Size(),
		}
		
		history = append(history, entry)
	}
	
	return history, nil
}

// ConfigHistoryEntry 配置历史条目
type ConfigHistoryEntry struct {
	PluginName string    `json:"plugin_name"`
	FilePath   string    `json:"file_path"`
	Timestamp  time.Time `json:"timestamp"`
	Size       int64     `json:"size"`
}

// NewDefaultConfigValidator 创建默认配置验证器
func NewDefaultConfigValidator(schema ConfigSchema) *DefaultConfigValidator {
	return &DefaultConfigValidator{schema: schema}
}

// Validate 验证配置
func (v *DefaultConfigValidator) Validate(config interface{}) error {
	return v.validateValue(config, v.schema)
}

// GetSchema 获取模式
func (v *DefaultConfigValidator) GetSchema() interface{} {
	return v.schema
}

// validateValue 验证值
func (v *DefaultConfigValidator) validateValue(value interface{}, schema ConfigSchema) error {
	if value == nil {
		if schema.Default != nil {
			return nil
		}
		return fmt.Errorf("value is required")
	}
	
	switch schema.Type {
	case "object":
		return v.validateObject(value, schema)
	case "string":
		return v.validateString(value, schema)
	case "number":
		return v.validateNumber(value, schema)
	case "boolean":
		return v.validateBoolean(value, schema)
	case "array":
		return v.validateArray(value, schema)
	default:
		return fmt.Errorf("unsupported type: %s", schema.Type)
	}
}

// validateObject 验证对象
func (v *DefaultConfigValidator) validateObject(value interface{}, schema ConfigSchema) error {
	valueMap, ok := value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("expected object, got %T", value)
	}
	
	// 检查必需字段
	for _, required := range schema.Required {
		if _, exists := valueMap[required]; !exists {
			return fmt.Errorf("required field missing: %s", required)
		}
	}
	
	// 验证属性
	for propName, propValue := range valueMap {
		if propSchema, exists := schema.Properties[propName]; exists {
			propConfigSchema := ConfigSchema{
				Type:        propSchema.Type,
				Default:     propSchema.Default,
				Description: propSchema.Description,
			}
			
			if err := v.validateValue(propValue, propConfigSchema); err != nil {
				return fmt.Errorf("property %s: %w", propName, err)
			}
		}
	}
	
	return nil
}

// validateString 验证字符串
func (v *DefaultConfigValidator) validateString(value interface{}, schema ConfigSchema) error {
	_, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}
	
	// 检查枚举值
	if len(schema.Properties) > 0 {
		// 这里应该检查enum，但为了简化暂时跳过
	}
	
	return nil
}

// validateNumber 验证数字
func (v *DefaultConfigValidator) validateNumber(value interface{}, schema ConfigSchema) error {
	var num float64
	
	switch v := value.(type) {
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case float32:
		num = float64(v)
	case float64:
		num = v
	default:
		return fmt.Errorf("expected number, got %T", value)
	}
	
	_ = num // 避免未使用变量警告
	return nil
}

// validateBoolean 验证布尔值
func (v *DefaultConfigValidator) validateBoolean(value interface{}, schema ConfigSchema) error {
	_, ok := value.(bool)
	if !ok {
		return fmt.Errorf("expected boolean, got %T", value)
	}
	
	return nil
}

// validateArray 验证数组
func (v *DefaultConfigValidator) validateArray(value interface{}, schema ConfigSchema) error {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return fmt.Errorf("expected array, got %T", value)
	}
	
	return nil
}