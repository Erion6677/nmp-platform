package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nmp-platform/internal/config"

	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// 创建配置管理器
	configPath := "./configs/config.yaml"
	configManager, err := config.NewConfigManager(configPath, logger)
	if err != nil {
		logger.Fatal("Failed to create config manager", zap.Error(err))
	}

	// 启动配置管理器
	if err := configManager.Start(); err != nil {
		logger.Fatal("Failed to start config manager", zap.Error(err))
	}
	defer configManager.Stop()

	// 创建自定义配置变更处理器
	customHandler := &CustomConfigHandler{logger: logger}
	configManager.Subscribe(customHandler)

	// 显示当前配置摘要
	summary := configManager.GetConfigSummary()
	logger.Info("Current configuration", zap.Any("summary", summary))

	// 检查热重载支持
	hotReloadableKeys := []string{
		"server.mode",
		"server.host",
		"server.port",
		"auth.token_expiry",
		"plugins.configs",
	}

	for _, key := range hotReloadableKeys {
		supported := configManager.IsHotReloadSupported(key)
		logger.Info("Hot reload support", 
			zap.String("key", key), 
			zap.Bool("supported", supported))
	}

	// 获取可热重载配置
	hotConfig := configManager.GetHotReloadableConfig()
	logger.Info("Hot reloadable config", zap.Any("config", hotConfig))

	logger.Info("Config example started. Modify ./configs/config.yaml to see hot reload in action.")
	logger.Info("Press Ctrl+C to exit")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 定期显示配置状态
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-quit:
			logger.Info("Shutting down...")
			return
		case <-ticker.C:
			currentConfig := configManager.GetConfig()
			logger.Info("Current server mode", 
				zap.String("mode", currentConfig.Server.Mode),
				zap.Int("port", currentConfig.Server.Port))
		}
	}
}

// CustomConfigHandler 自定义配置变更处理器
type CustomConfigHandler struct {
	logger *zap.Logger
}

func (h *CustomConfigHandler) OnConfigChange(oldConfig, newConfig *config.Config) error {
	h.logger.Info("Configuration changed!")

	// 比较配置变更
	changes := config.CompareConfigs(oldConfig, newConfig)
	for key, change := range changes {
		h.logger.Info("Config change detected", 
			zap.String("key", key), 
			zap.Any("change", change))
	}

	// 处理特定的配置变更
	if oldConfig.Server.Mode != newConfig.Server.Mode {
		h.logger.Info("Server mode changed", 
			zap.String("old", oldConfig.Server.Mode),
			zap.String("new", newConfig.Server.Mode))
		
		// 这里可以添加模式变更的处理逻辑
		// 例如：调整日志级别、启用/禁用调试功能等
	}

	if oldConfig.Auth.TokenExpiry != newConfig.Auth.TokenExpiry {
		h.logger.Info("Token expiry changed", 
			zap.Duration("old", oldConfig.Auth.TokenExpiry),
			zap.Duration("new", newConfig.Auth.TokenExpiry))
		
		// 这里可以添加令牌过期时间变更的处理逻辑
	}

	// 处理插件配置变更
	if len(oldConfig.Plugins.Configs) != len(newConfig.Plugins.Configs) {
		h.logger.Info("Plugin configuration count changed",
			zap.Int("old_count", len(oldConfig.Plugins.Configs)),
			zap.Int("new_count", len(newConfig.Plugins.Configs)))
	}

	return nil
}

func (h *CustomConfigHandler) GetName() string {
	return "custom_example_handler"
}