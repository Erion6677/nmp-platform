package config

import (
	"context"
	"fmt"
	"path/filepath"

	"go.uber.org/zap"
)

// ConfigManager 配置管理器，集成配置加载和热重载功能
type ConfigManager struct {
	config  *Config
	watcher *ConfigWatcher
	logger  *zap.Logger
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configPath string, logger *zap.Logger) (*ConfigManager, error) {
	// 加载初始配置
	config, err := LoadWithPath(filepath.Dir(configPath))
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	// 设置全局配置
	SetConfig(config)

	// 创建配置监听器
	watcher, err := NewConfigWatcher(configPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create config watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cm := &ConfigManager{
		config:  config,
		watcher: watcher,
		logger:  logger,
		ctx:     ctx,
		cancel:  cancel,
	}

	// 设置全局配置监听器
	SetConfigWatcher(watcher)

	return cm, nil
}

// Start 启动配置管理器
func (cm *ConfigManager) Start() error {
	// 注册默认的配置变更处理器
	cm.registerDefaultHandlers()

	// 启动配置监听
	if err := cm.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start config watcher: %w", err)
	}

	cm.logger.Info("Config manager started")
	return nil
}

// Stop 停止配置管理器
func (cm *ConfigManager) Stop() error {
	cm.cancel()

	if cm.watcher != nil {
		if err := cm.watcher.Stop(); err != nil {
			cm.logger.Error("Failed to stop config watcher", zap.Error(err))
			return err
		}
	}

	cm.logger.Info("Config manager stopped")
	return nil
}

// GetConfig 获取当前配置
func (cm *ConfigManager) GetConfig() *Config {
	return GetConfig()
}

// Subscribe 订阅配置变更
func (cm *ConfigManager) Subscribe(handler ConfigChangeHandler) {
	cm.watcher.Subscribe(handler)
}

// Unsubscribe 取消订阅配置变更
func (cm *ConfigManager) Unsubscribe(handlerName string) {
	cm.watcher.Unsubscribe(handlerName)
}

// registerDefaultHandlers 注册默认的配置变更处理器
func (cm *ConfigManager) registerDefaultHandlers() {
	// 注册日志级别处理器
	logHandler := NewLogLevelHandler(cm.logger)
	cm.watcher.Subscribe(logHandler)

	// 注册服务器配置处理器
	serverHandler := NewServerConfigHandler(cm.logger)
	cm.watcher.Subscribe(serverHandler)

	// 注册数据库配置处理器
	dbHandler := NewDatabaseConfigHandler(cm.logger)
	cm.watcher.Subscribe(dbHandler)

	// 注册配置通知处理器
	notificationHandler := NewConfigNotificationHandler(cm.logger)
	cm.watcher.Subscribe(notificationHandler)

	cm.logger.Info("Default config change handlers registered")
}

// ReloadConfig 手动重新加载配置
func (cm *ConfigManager) ReloadConfig() error {
	newConfig, err := Reload()
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	cm.config = newConfig
	cm.logger.Info("Config manually reloaded")
	return nil
}

// ValidateConfig 验证当前配置
func (cm *ConfigManager) ValidateConfig() error {
	return validateConfig(cm.config)
}

// GetConfigSummary 获取配置摘要信息
func (cm *ConfigManager) GetConfigSummary() map[string]interface{} {
	config := cm.GetConfig()
	if config == nil {
		return nil
	}

	return map[string]interface{}{
		"server": map[string]interface{}{
			"host": config.Server.Host,
			"port": config.Server.Port,
			"mode": config.Server.Mode,
		},
		"database": map[string]interface{}{
			"host":     config.Database.Host,
			"port":     config.Database.Port,
			"database": config.Database.Database,
		},
		"redis": map[string]interface{}{
			"host": config.Redis.Host,
			"port": config.Redis.Port,
		},
		"influxdb": map[string]interface{}{
			"url":    config.InfluxDB.URL,
			"org":    config.InfluxDB.Org,
			"bucket": config.InfluxDB.Bucket,
		},
		"plugins": map[string]interface{}{
			"directory":    config.Plugins.Directory,
			"config_count": len(config.Plugins.Configs),
		},
	}
}

// IsHotReloadSupported 检查指定配置项是否支持热重载
func (cm *ConfigManager) IsHotReloadSupported(configKey string) bool {
	hotReloadableKeys := map[string]bool{
		"server.mode":           true,
		"server.read_timeout":   true,
		"server.write_timeout":  true,
		"auth.token_expiry":     true,
		"auth.refresh_expiry":   true,
		"plugins.configs":       true,
		// 不支持热重载的配置项
		"server.host":           false,
		"server.port":           false,
		"database.host":         false,
		"database.port":         false,
		"database.database":     false,
		"database.username":     false,
		"database.password":     false,
		"redis.host":            false,
		"redis.port":            false,
		"influxdb.url":          false,
		"influxdb.token":        false,
	}

	supported, exists := hotReloadableKeys[configKey]
	return exists && supported
}

// GetHotReloadableConfig 获取支持热重载的配置项
func (cm *ConfigManager) GetHotReloadableConfig() *HotReloadableConfig {
	return ExtractHotReloadableConfig(cm.GetConfig())
}