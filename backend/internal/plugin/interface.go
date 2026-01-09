package plugin

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// Plugin 定义插件的基础接口
type Plugin interface {
	// 基础信息
	Name() string
	Version() string
	Description() string
	Dependencies() []string

	// 生命周期管理
	Initialize(ctx context.Context, config interface{}) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health() error

	// 功能接口
	GetRoutes() []Route
	GetMenus() []MenuItem
	GetPermissions() []Permission
	GetConfigSchema() interface{}
}

// Route 定义插件路由
type Route struct {
	Method      string              // HTTP方法 (GET, POST, PUT, DELETE等)
	Path        string              // 路由路径
	Handler     gin.HandlerFunc     // 处理函数
	Middlewares []gin.HandlerFunc   // 中间件列表
	Permission  string              // 所需权限
	Description string              // 路由描述
}

// MenuItem 定义插件菜单项
type MenuItem struct {
	Key         string      `json:"key"`         // 菜单唯一标识
	Label       string      `json:"label"`       // 菜单显示名称
	Icon        string      `json:"icon"`        // 菜单图标
	Path        string      `json:"path"`        // 前端路由路径
	Children    []MenuItem  `json:"children"`    // 子菜单
	Permission  string      `json:"permission"`  // 所需权限
	Order       int         `json:"order"`       // 排序权重
	Visible     bool        `json:"visible"`     // 是否可见
}

// Permission 定义插件权限
type Permission struct {
	Resource    string `json:"resource"`    // 资源类型
	Action      string `json:"action"`      // 操作类型 (create, read, update, delete)
	Scope       string `json:"scope"`       // 权限范围 (global, group, self)
	Description string `json:"description"` // 权限描述
}

// PluginInfo 插件基础信息
type PluginInfo struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Dependencies []string          `json:"dependencies"`
	Config       map[string]interface{} `json:"config"`
	Status       PluginStatus      `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// PluginStatus 插件状态枚举
type PluginStatus string

const (
	PluginStatusUnknown     PluginStatus = "unknown"
	PluginStatusRegistered  PluginStatus = "registered"
	PluginStatusInitialized PluginStatus = "initialized"
	PluginStatusStarted     PluginStatus = "started"
	PluginStatusStopped     PluginStatus = "stopped"
	PluginStatusError       PluginStatus = "error"
)

// PluginRegistry 插件注册表接口
type PluginRegistry interface {
	Register(plugin Plugin) error
	Unregister(name string) error
	Get(name string) (Plugin, bool)
	List() []Plugin
	GetByStatus(status PluginStatus) []Plugin
}

// PluginConfig 插件配置接口
type PluginConfig interface {
	GetConfig(pluginName string) (interface{}, error)
	SetConfig(pluginName string, config interface{}) error
	ValidateConfig(pluginName string, config interface{}) error
}