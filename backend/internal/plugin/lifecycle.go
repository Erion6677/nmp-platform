package plugin

import (
	"context"
	"fmt"
	"log"
	"time"
)

// LifecycleManager 插件生命周期管理器
type LifecycleManager struct {
	registry PluginRegistry
	config   PluginConfig
	logger   *log.Logger
}

// NewLifecycleManager 创建生命周期管理器
func NewLifecycleManager(registry PluginRegistry, config PluginConfig, logger *log.Logger) *LifecycleManager {
	return &LifecycleManager{
		registry: registry,
		config:   config,
		logger:   logger,
	}
}

// InitializePlugin 初始化插件
func (lm *LifecycleManager) InitializePlugin(ctx context.Context, name string) error {
	plugin, exists := lm.registry.Get(name)
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// 获取插件配置
	config, err := lm.config.GetConfig(name)
	if err != nil {
		lm.logger.Printf("Warning: failed to get config for plugin %s: %v", name, err)
		config = nil
	}

	// 验证配置
	if config != nil {
		if err := lm.config.ValidateConfig(name, config); err != nil {
			return fmt.Errorf("invalid config for plugin %s: %w", name, err)
		}
	}

	// 初始化插件
	if err := plugin.Initialize(ctx, config); err != nil {
		lm.updatePluginStatus(name, PluginStatusError)
		return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
	}

	lm.updatePluginStatus(name, PluginStatusInitialized)
	lm.logger.Printf("Plugin %s initialized successfully", name)
	return nil
}

// StartPlugin 启动插件
func (lm *LifecycleManager) StartPlugin(ctx context.Context, name string) error {
	plugin, exists := lm.registry.Get(name)
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// 检查插件状态
	if info, exists := lm.registry.(*DefaultPluginRegistry).GetInfo(name); exists {
		if info.Status != PluginStatusInitialized && info.Status != PluginStatusStopped {
			return fmt.Errorf("plugin %s is not in a startable state (current: %s)", name, info.Status)
		}
	}

	// 启动插件
	if err := plugin.Start(ctx); err != nil {
		lm.updatePluginStatus(name, PluginStatusError)
		return fmt.Errorf("failed to start plugin %s: %w", name, err)
	}

	lm.updatePluginStatus(name, PluginStatusStarted)
	lm.logger.Printf("Plugin %s started successfully", name)
	return nil
}

// StopPlugin 停止插件
func (lm *LifecycleManager) StopPlugin(ctx context.Context, name string) error {
	plugin, exists := lm.registry.Get(name)
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// 检查插件状态
	if info, exists := lm.registry.(*DefaultPluginRegistry).GetInfo(name); exists {
		if info.Status != PluginStatusStarted {
			return fmt.Errorf("plugin %s is not running (current: %s)", name, info.Status)
		}
	}

	// 停止插件
	if err := plugin.Stop(ctx); err != nil {
		lm.updatePluginStatus(name, PluginStatusError)
		return fmt.Errorf("failed to stop plugin %s: %w", name, err)
	}

	lm.updatePluginStatus(name, PluginStatusStopped)
	lm.logger.Printf("Plugin %s stopped successfully", name)
	return nil
}

// RestartPlugin 重启插件
func (lm *LifecycleManager) RestartPlugin(ctx context.Context, name string) error {
	// 先停止插件
	if err := lm.StopPlugin(ctx, name); err != nil {
		return err
	}

	// 等待一小段时间确保资源释放
	time.Sleep(100 * time.Millisecond)

	// 重新启动插件
	return lm.StartPlugin(ctx, name)
}

// CheckHealth 检查插件健康状态
func (lm *LifecycleManager) CheckHealth(name string) error {
	plugin, exists := lm.registry.Get(name)
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	return plugin.Health()
}

// InitializeAllPlugins 初始化所有插件
func (lm *LifecycleManager) InitializeAllPlugins(ctx context.Context) error {
	plugins := lm.registry.List()
	
	// 按依赖顺序初始化插件
	initialized := make(map[string]bool)
	
	for len(initialized) < len(plugins) {
		progress := false
		
		for _, plugin := range plugins {
			name := plugin.Name()
			if initialized[name] {
				continue
			}
			
			// 检查依赖是否都已初始化
			canInitialize := true
			for _, dep := range plugin.Dependencies() {
				if !initialized[dep] {
					canInitialize = false
					break
				}
			}
			
			if canInitialize {
				if err := lm.InitializePlugin(ctx, name); err != nil {
					lm.logger.Printf("Failed to initialize plugin %s: %v", name, err)
					// 继续初始化其他插件
				} else {
					initialized[name] = true
					progress = true
				}
			}
		}
		
		if !progress {
			return fmt.Errorf("circular dependency detected or missing dependencies")
		}
	}
	
	return nil
}

// StartAllPlugins 启动所有已初始化的插件
func (lm *LifecycleManager) StartAllPlugins(ctx context.Context) error {
	plugins := lm.registry.GetByStatus(PluginStatusInitialized)
	
	for _, plugin := range plugins {
		if err := lm.StartPlugin(ctx, plugin.Name()); err != nil {
			lm.logger.Printf("Failed to start plugin %s: %v", plugin.Name(), err)
			// 继续启动其他插件
		}
	}
	
	return nil
}

// StopAllPlugins 停止所有运行中的插件
func (lm *LifecycleManager) StopAllPlugins(ctx context.Context) error {
	plugins := lm.registry.GetByStatus(PluginStatusStarted)
	
	// 反向停止插件（先停止依赖者）
	for i := len(plugins) - 1; i >= 0; i-- {
		if err := lm.StopPlugin(ctx, plugins[i].Name()); err != nil {
			lm.logger.Printf("Failed to stop plugin %s: %v", plugins[i].Name(), err)
			// 继续停止其他插件
		}
	}
	
	return nil
}

// updatePluginStatus 更新插件状态
func (lm *LifecycleManager) updatePluginStatus(name string, status PluginStatus) {
	if registry, ok := lm.registry.(*DefaultPluginRegistry); ok {
		if err := registry.UpdateStatus(name, status); err != nil {
			lm.logger.Printf("Failed to update status for plugin %s: %v", name, err)
		}
	}
}