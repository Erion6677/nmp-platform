package config

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// ConfigWatcher 配置文件监听器
type ConfigWatcher struct {
	watcher     *fsnotify.Watcher
	configPath  string
	logger      *zap.Logger
	subscribers []ConfigChangeHandler
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// ConfigChangeHandler 配置变更处理器
type ConfigChangeHandler interface {
	OnConfigChange(oldConfig, newConfig *Config) error
	GetName() string
}

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Type      string    `json:"type"`
	Path      string    `json:"path"`
	OldConfig *Config   `json:"old_config,omitempty"`
	NewConfig *Config   `json:"new_config,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
}

// NewConfigWatcher 创建配置监听器
func NewConfigWatcher(configPath string, logger *zap.Logger) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cw := &ConfigWatcher{
		watcher:     watcher,
		configPath:  configPath,
		logger:      logger,
		subscribers: make([]ConfigChangeHandler, 0),
		ctx:         ctx,
		cancel:      cancel,
	}

	return cw, nil
}

// Subscribe 订阅配置变更
func (cw *ConfigWatcher) Subscribe(handler ConfigChangeHandler) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	cw.subscribers = append(cw.subscribers, handler)
	cw.logger.Info("Config change handler subscribed", 
		zap.String("handler", handler.GetName()))
}

// Unsubscribe 取消订阅配置变更
func (cw *ConfigWatcher) Unsubscribe(handlerName string) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	for i, handler := range cw.subscribers {
		if handler.GetName() == handlerName {
			cw.subscribers = append(cw.subscribers[:i], cw.subscribers[i+1:]...)
			cw.logger.Info("Config change handler unsubscribed", 
				zap.String("handler", handlerName))
			break
		}
	}
}

// Start 启动配置监听
func (cw *ConfigWatcher) Start() error {
	// 监听配置文件目录
	configDir := filepath.Dir(cw.configPath)
	if err := cw.watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch config directory: %w", err)
	}

	cw.logger.Info("Config watcher started", 
		zap.String("path", configDir))

	go cw.watchLoop()
	return nil
}

// Stop 停止配置监听
func (cw *ConfigWatcher) Stop() error {
	cw.cancel()
	return cw.watcher.Close()
}

// watchLoop 监听循环
func (cw *ConfigWatcher) watchLoop() {
	// 防抖动：在短时间内的多次变更只处理一次
	debounceTimer := time.NewTimer(0)
	if !debounceTimer.Stop() {
		<-debounceTimer.C
	}

	for {
		select {
		case <-cw.ctx.Done():
			cw.logger.Info("Config watcher stopped")
			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// 只处理配置文件的写入事件
			if event.Has(fsnotify.Write) && filepath.Base(event.Name) == "config.yaml" {
				cw.logger.Debug("Config file change detected", 
					zap.String("file", event.Name),
					zap.String("op", event.Op.String()))

				// 重置防抖动定时器
				debounceTimer.Reset(500 * time.Millisecond)
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			cw.logger.Error("Config watcher error", zap.Error(err))

		case <-debounceTimer.C:
			// 处理配置变更
			cw.handleConfigChange()
		}
	}
}

// handleConfigChange 处理配置变更
func (cw *ConfigWatcher) handleConfigChange() {
	cw.logger.Info("Processing config file change")

	// 获取当前配置
	oldConfig := GetConfig()
	if oldConfig == nil {
		cw.logger.Warn("No current config found, skipping reload")
		return
	}

	// 重新加载配置
	newConfig, err := LoadWithPath(filepath.Dir(cw.configPath))
	if err != nil {
		cw.logger.Error("Failed to reload config", zap.Error(err))
		cw.notifyError("reload_failed", err)
		return
	}

	// 通知订阅者
	cw.notifySubscribers(oldConfig, newConfig)

	// 更新全局配置
	SetConfig(newConfig)

	cw.logger.Info("Config reloaded successfully")
}

// notifySubscribers 通知所有订阅者
func (cw *ConfigWatcher) notifySubscribers(oldConfig, newConfig *Config) {
	cw.mu.RLock()
	subscribers := make([]ConfigChangeHandler, len(cw.subscribers))
	copy(subscribers, cw.subscribers)
	cw.mu.RUnlock()

	for _, handler := range subscribers {
		go func(h ConfigChangeHandler) {
			if err := h.OnConfigChange(oldConfig, newConfig); err != nil {
				cw.logger.Error("Config change handler failed", 
					zap.String("handler", h.GetName()),
					zap.Error(err))
			} else {
				cw.logger.Debug("Config change handler succeeded", 
					zap.String("handler", h.GetName()))
			}
		}(handler)
	}
}

// notifyError 通知错误
func (cw *ConfigWatcher) notifyError(eventType string, err error) {
	event := ConfigChangeEvent{
		Type:      eventType,
		Path:      cw.configPath,
		Timestamp: time.Now(),
		Error:     err.Error(),
	}

	cw.logger.Error("Config change error", 
		zap.String("type", event.Type),
		zap.String("error", event.Error))
}

// HotReloadableConfig 支持热重载的配置项
type HotReloadableConfig struct {
	// 可以热重载的配置项
	LogLevel     string        `json:"log_level"`
	DebugMode    bool          `json:"debug_mode"`
	RateLimit    int           `json:"rate_limit"`
	CacheTimeout time.Duration `json:"cache_timeout"`
	
	// 插件配置（支持热重载）
	PluginConfigs map[string]interface{} `json:"plugin_configs"`
}

// ExtractHotReloadableConfig 提取可热重载的配置
func ExtractHotReloadableConfig(config *Config) *HotReloadableConfig {
	return &HotReloadableConfig{
		LogLevel:      "info", // 从配置中提取
		DebugMode:     config.Server.Mode == "debug",
		RateLimit:     100, // 默认值，可从配置中提取
		CacheTimeout:  5 * time.Minute,
		PluginConfigs: config.Plugins.Configs,
	}
}

// CompareConfigs 比较两个配置的差异
func CompareConfigs(oldConfig, newConfig *Config) map[string]interface{} {
	changes := make(map[string]interface{})

	// 比较服务器配置
	if oldConfig.Server.Mode != newConfig.Server.Mode {
		changes["server.mode"] = map[string]string{
			"old": oldConfig.Server.Mode,
			"new": newConfig.Server.Mode,
		}
	}

	if oldConfig.Server.Port != newConfig.Server.Port {
		changes["server.port"] = map[string]int{
			"old": oldConfig.Server.Port,
			"new": newConfig.Server.Port,
		}
	}

	// 比较认证配置
	if oldConfig.Auth.TokenExpiry != newConfig.Auth.TokenExpiry {
		changes["auth.token_expiry"] = map[string]time.Duration{
			"old": oldConfig.Auth.TokenExpiry,
			"new": newConfig.Auth.TokenExpiry,
		}
	}

	// 比较插件配置
	if len(oldConfig.Plugins.Configs) != len(newConfig.Plugins.Configs) {
		changes["plugins.configs.count"] = map[string]int{
			"old": len(oldConfig.Plugins.Configs),
			"new": len(newConfig.Plugins.Configs),
		}
	}

	return changes
}