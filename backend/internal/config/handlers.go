package config

import (
	"fmt"

	"go.uber.org/zap"
)

// LogLevelHandler 日志级别变更处理器
type LogLevelHandler struct {
	logger *zap.Logger
}

// NewLogLevelHandler 创建日志级别处理器
func NewLogLevelHandler(logger *zap.Logger) *LogLevelHandler {
	return &LogLevelHandler{
		logger: logger,
	}
}

// OnConfigChange 处理配置变更
func (h *LogLevelHandler) OnConfigChange(oldConfig, newConfig *Config) error {
	// 检查服务器模式是否变更（影响日志级别）
	if oldConfig.Server.Mode != newConfig.Server.Mode {
		h.logger.Info("Server mode changed, updating log level",
			zap.String("old_mode", oldConfig.Server.Mode),
			zap.String("new_mode", newConfig.Server.Mode))
		
		// 这里可以实际更新日志级别
		// 例如：重新配置 zap logger
	}
	
	return nil
}

// GetName 获取处理器名称
func (h *LogLevelHandler) GetName() string {
	return "log_level_handler"
}

// PluginConfigHandler 插件配置变更处理器
type PluginConfigHandler struct {
	logger        *zap.Logger
	pluginManager PluginManager // 假设有插件管理器接口
}

// PluginManager 插件管理器接口
type PluginManager interface {
	ReloadPluginConfig(pluginName string, config interface{}) error
	GetLoadedPlugins() []string
}

// NewPluginConfigHandler 创建插件配置处理器
func NewPluginConfigHandler(logger *zap.Logger, pluginManager PluginManager) *PluginConfigHandler {
	return &PluginConfigHandler{
		logger:        logger,
		pluginManager: pluginManager,
	}
}

// OnConfigChange 处理插件配置变更
func (h *PluginConfigHandler) OnConfigChange(oldConfig, newConfig *Config) error {
	// 比较插件配置变更
	oldPluginConfigs := oldConfig.Plugins.Configs
	newPluginConfigs := newConfig.Plugins.Configs

	// 检查每个插件的配置变更
	for pluginName, newPluginConfig := range newPluginConfigs {
		oldPluginConfig, exists := oldPluginConfigs[pluginName]
		
		if !exists {
			// 新增插件配置
			h.logger.Info("New plugin config detected",
				zap.String("plugin", pluginName))
			
			if h.pluginManager != nil {
				if err := h.pluginManager.ReloadPluginConfig(pluginName, newPluginConfig); err != nil {
					h.logger.Error("Failed to load new plugin config",
						zap.String("plugin", pluginName),
						zap.Error(err))
					return err
				}
			}
		} else {
			// 检查配置是否变更
			if !comparePluginConfigs(oldPluginConfig, newPluginConfig) {
				h.logger.Info("Plugin config changed",
					zap.String("plugin", pluginName))
				
				if h.pluginManager != nil {
					if err := h.pluginManager.ReloadPluginConfig(pluginName, newPluginConfig); err != nil {
						h.logger.Error("Failed to reload plugin config",
							zap.String("plugin", pluginName),
							zap.Error(err))
						return err
					}
				}
			}
		}
	}

	// 检查删除的插件配置
	for pluginName := range oldPluginConfigs {
		if _, exists := newPluginConfigs[pluginName]; !exists {
			h.logger.Info("Plugin config removed",
				zap.String("plugin", pluginName))
			
			// 这里可以处理插件配置删除逻辑
		}
	}

	return nil
}

// GetName 获取处理器名称
func (h *PluginConfigHandler) GetName() string {
	return "plugin_config_handler"
}

// comparePluginConfigs 比较插件配置是否相同
func comparePluginConfigs(old, new interface{}) bool {
	// 简单的字符串比较，实际应用中可能需要更复杂的比较逻辑
	return fmt.Sprintf("%v", old) == fmt.Sprintf("%v", new)
}

// ServerConfigHandler 服务器配置变更处理器
type ServerConfigHandler struct {
	logger *zap.Logger
}

// NewServerConfigHandler 创建服务器配置处理器
func NewServerConfigHandler(logger *zap.Logger) *ServerConfigHandler {
	return &ServerConfigHandler{
		logger: logger,
	}
}

// OnConfigChange 处理服务器配置变更
func (h *ServerConfigHandler) OnConfigChange(oldConfig, newConfig *Config) error {
	// 检查不能热重载的配置项
	if oldConfig.Server.Host != newConfig.Server.Host {
		h.logger.Warn("Server host changed, requires restart",
			zap.String("old_host", oldConfig.Server.Host),
			zap.String("new_host", newConfig.Server.Host))
		return fmt.Errorf("server host change requires restart")
	}

	if oldConfig.Server.Port != newConfig.Server.Port {
		h.logger.Warn("Server port changed, requires restart",
			zap.Int("old_port", oldConfig.Server.Port),
			zap.Int("new_port", newConfig.Server.Port))
		return fmt.Errorf("server port change requires restart")
	}

	// 可以热重载的配置项
	if oldConfig.Server.ReadTimeout != newConfig.Server.ReadTimeout {
		h.logger.Info("Server read timeout changed",
			zap.Duration("old_timeout", oldConfig.Server.ReadTimeout),
			zap.Duration("new_timeout", newConfig.Server.ReadTimeout))
		
		// 这里可以更新服务器的读取超时配置
	}

	if oldConfig.Server.WriteTimeout != newConfig.Server.WriteTimeout {
		h.logger.Info("Server write timeout changed",
			zap.Duration("old_timeout", oldConfig.Server.WriteTimeout),
			zap.Duration("new_timeout", newConfig.Server.WriteTimeout))
		
		// 这里可以更新服务器的写入超时配置
	}

	return nil
}

// GetName 获取处理器名称
func (h *ServerConfigHandler) GetName() string {
	return "server_config_handler"
}

// DatabaseConfigHandler 数据库配置变更处理器
type DatabaseConfigHandler struct {
	logger *zap.Logger
}

// NewDatabaseConfigHandler 创建数据库配置处理器
func NewDatabaseConfigHandler(logger *zap.Logger) *DatabaseConfigHandler {
	return &DatabaseConfigHandler{
		logger: logger,
	}
}

// OnConfigChange 处理数据库配置变更
func (h *DatabaseConfigHandler) OnConfigChange(oldConfig, newConfig *Config) error {
	// 数据库配置通常不支持热重载
	if oldConfig.Database.Host != newConfig.Database.Host ||
		oldConfig.Database.Port != newConfig.Database.Port ||
		oldConfig.Database.Database != newConfig.Database.Database ||
		oldConfig.Database.Username != newConfig.Database.Username ||
		oldConfig.Database.Password != newConfig.Database.Password {
		
		h.logger.Warn("Database configuration changed, requires restart")
		return fmt.Errorf("database configuration change requires restart")
	}

	return nil
}

// GetName 获取处理器名称
func (h *DatabaseConfigHandler) GetName() string {
	return "database_config_handler"
}

// ConfigNotificationHandler 配置变更通知处理器
type ConfigNotificationHandler struct {
	logger      *zap.Logger
	subscribers []func(ConfigChangeEvent)
}

// NewConfigNotificationHandler 创建配置通知处理器
func NewConfigNotificationHandler(logger *zap.Logger) *ConfigNotificationHandler {
	return &ConfigNotificationHandler{
		logger:      logger,
		subscribers: make([]func(ConfigChangeEvent), 0),
	}
}

// Subscribe 订阅配置变更通知
func (h *ConfigNotificationHandler) Subscribe(callback func(ConfigChangeEvent)) {
	h.subscribers = append(h.subscribers, callback)
}

// OnConfigChange 处理配置变更通知
func (h *ConfigNotificationHandler) OnConfigChange(oldConfig, newConfig *Config) error {
	// 创建变更事件
	event := ConfigChangeEvent{
		Type:      "config_changed",
		OldConfig: oldConfig,
		NewConfig: newConfig,
	}

	// 通知所有订阅者
	for _, callback := range h.subscribers {
		go func(cb func(ConfigChangeEvent)) {
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("Config notification callback panicked",
						zap.Any("panic", r))
				}
			}()
			cb(event)
		}(callback)
	}

	return nil
}

// GetName 获取处理器名称
func (h *ConfigNotificationHandler) GetName() string {
	return "config_notification_handler"
}