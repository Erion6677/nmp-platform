package plugin

import (
	"context"
	"fmt"
)

// BasePlugin 提供插件的基础实现
type BasePlugin struct {
	name         string
	version      string
	description  string
	dependencies []string
	routes       []Route
	menus        []MenuItem
	permissions  []Permission
	configSchema interface{}
	initialized  bool
	started      bool
}

// NewBasePlugin 创建基础插件实例
func NewBasePlugin(name, version, description string) *BasePlugin {
	return &BasePlugin{
		name:         name,
		version:      version,
		description:  description,
		dependencies: make([]string, 0),
		routes:       make([]Route, 0),
		menus:        make([]MenuItem, 0),
		permissions:  make([]Permission, 0),
	}
}

// Name 返回插件名称
func (bp *BasePlugin) Name() string {
	return bp.name
}

// Version 返回插件版本
func (bp *BasePlugin) Version() string {
	return bp.version
}

// Description 返回插件描述
func (bp *BasePlugin) Description() string {
	return bp.description
}

// Dependencies 返回插件依赖
func (bp *BasePlugin) Dependencies() []string {
	return bp.dependencies
}

// Initialize 初始化插件（基础实现）
func (bp *BasePlugin) Initialize(ctx context.Context, config interface{}) error {
	if bp.initialized {
		return fmt.Errorf("plugin %s already initialized", bp.name)
	}
	
	bp.initialized = true
	return nil
}

// Start 启动插件（基础实现）
func (bp *BasePlugin) Start(ctx context.Context) error {
	if !bp.initialized {
		return fmt.Errorf("plugin %s not initialized", bp.name)
	}
	
	if bp.started {
		return fmt.Errorf("plugin %s already started", bp.name)
	}
	
	bp.started = true
	return nil
}

// Stop 停止插件（基础实现）
func (bp *BasePlugin) Stop(ctx context.Context) error {
	if !bp.started {
		return fmt.Errorf("plugin %s not started", bp.name)
	}
	
	bp.started = false
	return nil
}

// Health 健康检查（基础实现）
func (bp *BasePlugin) Health() error {
	if !bp.initialized {
		return fmt.Errorf("plugin %s not initialized", bp.name)
	}
	
	if !bp.started {
		return fmt.Errorf("plugin %s not started", bp.name)
	}
	
	return nil
}

// GetRoutes 获取路由
func (bp *BasePlugin) GetRoutes() []Route {
	return bp.routes
}

// GetMenus 获取菜单
func (bp *BasePlugin) GetMenus() []MenuItem {
	return bp.menus
}

// GetPermissions 获取权限
func (bp *BasePlugin) GetPermissions() []Permission {
	return bp.permissions
}

// GetConfigSchema 获取配置模式
func (bp *BasePlugin) GetConfigSchema() interface{} {
	return bp.configSchema
}

// AddDependency 添加依赖
func (bp *BasePlugin) AddDependency(dep string) {
	bp.dependencies = append(bp.dependencies, dep)
}

// AddRoute 添加路由
func (bp *BasePlugin) AddRoute(route Route) {
	bp.routes = append(bp.routes, route)
}

// AddMenu 添加菜单
func (bp *BasePlugin) AddMenu(menu MenuItem) {
	bp.menus = append(bp.menus, menu)
}

// AddPermission 添加权限
func (bp *BasePlugin) AddPermission(permission Permission) {
	bp.permissions = append(bp.permissions, permission)
}

// SetConfigSchema 设置配置模式
func (bp *BasePlugin) SetConfigSchema(schema interface{}) {
	bp.configSchema = schema
}

// IsInitialized 检查是否已初始化
func (bp *BasePlugin) IsInitialized() bool {
	return bp.initialized
}

// IsStarted 检查是否已启动
func (bp *BasePlugin) IsStarted() bool {
	return bp.started
}