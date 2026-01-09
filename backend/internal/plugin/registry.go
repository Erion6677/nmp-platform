package plugin

import (
	"fmt"
	"sync"
	"time"
)

// DefaultPluginRegistry 默认插件注册表实现
type DefaultPluginRegistry struct {
	plugins map[string]Plugin
	infos   map[string]*PluginInfo
	mutex   sync.RWMutex
}

// NewDefaultPluginRegistry 创建新的插件注册表
func NewDefaultPluginRegistry() *DefaultPluginRegistry {
	return &DefaultPluginRegistry{
		plugins: make(map[string]Plugin),
		infos:   make(map[string]*PluginInfo),
	}
}

// Register 注册插件
func (r *DefaultPluginRegistry) Register(plugin Plugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	// 检查插件是否已注册
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// 检查依赖
	for _, dep := range plugin.Dependencies() {
		if _, exists := r.plugins[dep]; !exists {
			return fmt.Errorf("dependency %s not found for plugin %s", dep, name)
		}
	}

	// 注册插件
	r.plugins[name] = plugin
	r.infos[name] = &PluginInfo{
		Name:         name,
		Version:      plugin.Version(),
		Description:  plugin.Description(),
		Dependencies: plugin.Dependencies(),
		Status:       PluginStatusRegistered,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return nil
}

// Unregister 注销插件
func (r *DefaultPluginRegistry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// 检查是否有其他插件依赖此插件
	for _, p := range r.plugins {
		for _, dep := range p.Dependencies() {
			if dep == name {
				return fmt.Errorf("plugin %s is required by %s", name, p.Name())
			}
		}
	}

	// 停止插件
	if info := r.infos[name]; info.Status == PluginStatusStarted {
		if err := plugin.Stop(nil); err != nil {
			return fmt.Errorf("failed to stop plugin %s: %w", name, err)
		}
	}

	delete(r.plugins, name)
	delete(r.infos, name)

	return nil
}

// Get 获取插件
func (r *DefaultPluginRegistry) Get(name string) (Plugin, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugin, exists := r.plugins[name]
	return plugin, exists
}

// List 获取所有插件
func (r *DefaultPluginRegistry) List() []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// GetByStatus 根据状态获取插件
func (r *DefaultPluginRegistry) GetByStatus(status PluginStatus) []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var plugins []Plugin
	for name, info := range r.infos {
		if info.Status == status {
			if plugin, exists := r.plugins[name]; exists {
				plugins = append(plugins, plugin)
			}
		}
	}
	return plugins
}

// GetInfo 获取插件信息
func (r *DefaultPluginRegistry) GetInfo(name string) (*PluginInfo, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	info, exists := r.infos[name]
	return info, exists
}

// UpdateStatus 更新插件状态
func (r *DefaultPluginRegistry) UpdateStatus(name string, status PluginStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	info, exists := r.infos[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	info.Status = status
	info.UpdatedAt = time.Now()
	return nil
}

// GetAllInfos 获取所有插件信息
func (r *DefaultPluginRegistry) GetAllInfos() map[string]*PluginInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	infos := make(map[string]*PluginInfo)
	for name, info := range r.infos {
		// 创建副本避免并发修改
		infoCopy := *info
		infos[name] = &infoCopy
	}
	return infos
}