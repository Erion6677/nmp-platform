package plugin

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"plugin"
	"strings"

	"nmp-platform/internal/auth"

	"github.com/gin-gonic/gin"
)

// Manager 插件管理器
type Manager struct {
	registry             PluginRegistry
	lifecycleManager     *LifecycleManager
	config               PluginConfig
	router               *gin.Engine
	logger               *log.Logger
	pluginPaths          []string
	routeGroups          map[string]*gin.RouterGroup
	rbacService          *auth.RBACService
	permissionIntegrator *PermissionIntegrator
}

// NewManager 创建插件管理器
func NewManager(router *gin.Engine, config PluginConfig, logger *log.Logger) *Manager {
	registry := NewDefaultPluginRegistry()
	lifecycleManager := NewLifecycleManager(registry, config, logger)
	
	return &Manager{
		registry:         registry,
		lifecycleManager: lifecycleManager,
		config:           config,
		router:           router,
		logger:           logger,
		pluginPaths:      make([]string, 0),
		routeGroups:      make(map[string]*gin.RouterGroup),
	}
}

// SetRBACService 设置RBAC服务（用于权限检查）
func (m *Manager) SetRBACService(rbacService *auth.RBACService) {
	m.rbacService = rbacService
	m.permissionIntegrator = NewPermissionIntegrator(rbacService, m.logger)
}

// AddPluginPath 添加插件搜索路径
func (m *Manager) AddPluginPath(path string) {
	m.pluginPaths = append(m.pluginPaths, path)
}

// DiscoverPlugins 发现插件
func (m *Manager) DiscoverPlugins() error {
	m.logger.Println("Starting plugin discovery...")
	
	for _, path := range m.pluginPaths {
		if err := m.discoverPluginsInPath(path); err != nil {
			m.logger.Printf("Error discovering plugins in path %s: %v", path, err)
			continue
		}
	}
	
	m.logger.Printf("Plugin discovery completed. Found %d plugins", len(m.registry.List()))
	return nil
}

// discoverPluginsInPath 在指定路径发现插件
func (m *Manager) discoverPluginsInPath(path string) error {
	// 这里实现插件文件的发现逻辑
	// 在实际实现中，可以扫描.so文件或其他插件格式
	m.logger.Printf("Scanning for plugins in: %s", path)
	
	// 示例：扫描.so文件
	matches, err := filepath.Glob(filepath.Join(path, "*.so"))
	if err != nil {
		return err
	}
	
	for _, match := range matches {
		if err := m.loadPluginFromFile(match); err != nil {
			m.logger.Printf("Failed to load plugin from %s: %v", match, err)
		}
	}
	
	return nil
}

// loadPluginFromFile 从文件加载插件
func (m *Manager) loadPluginFromFile(filename string) error {
	// 加载动态库
	p, err := plugin.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open plugin file %s: %w", filename, err)
	}
	
	// 查找插件创建函数
	symbol, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export NewPlugin function: %w", filename, err)
	}
	
	// 类型断言
	newPluginFunc, ok := symbol.(func() Plugin)
	if !ok {
		return fmt.Errorf("invalid NewPlugin function signature in %s", filename)
	}
	
	// 创建插件实例
	pluginInstance := newPluginFunc()
	
	// 注册插件
	return m.RegisterPlugin(pluginInstance)
}

// RegisterPlugin 注册插件
func (m *Manager) RegisterPlugin(p Plugin) error {
	name := p.Name()
	m.logger.Printf("Registering plugin: %s (version: %s)", name, p.Version())
	
	// 注册到注册表
	if err := m.registry.Register(p); err != nil {
		return fmt.Errorf("failed to register plugin %s: %w", name, err)
	}
	
	// 集成插件权限到RBAC系统
	if m.permissionIntegrator != nil {
		if err := m.permissionIntegrator.IntegratePluginPermissions(p); err != nil {
			m.logger.Printf("Warning: failed to integrate permissions for plugin %s: %v", name, err)
			// 不阻止插件注册，只记录警告
		}
		
		// 为插件创建默认角色
		if _, err := m.permissionIntegrator.CreatePluginRole(p); err != nil {
			m.logger.Printf("Warning: failed to create role for plugin %s: %v", name, err)
		}
	}
	
	m.logger.Printf("Plugin %s registered successfully", name)
	return nil
}

// LoadPlugins 加载所有插件
func (m *Manager) LoadPlugins(ctx context.Context) error {
	m.logger.Println("Loading plugins...")
	
	// 初始化所有插件
	if err := m.lifecycleManager.InitializeAllPlugins(ctx); err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}
	
	// 集成插件路由
	if err := m.integratePluginRoutes(); err != nil {
		return fmt.Errorf("failed to integrate plugin routes: %w", err)
	}
	
	// 启动所有插件
	if err := m.lifecycleManager.StartAllPlugins(ctx); err != nil {
		return fmt.Errorf("failed to start plugins: %w", err)
	}
	
	m.logger.Println("All plugins loaded successfully")
	return nil
}

// integratePluginRoutes 集成插件路由
func (m *Manager) integratePluginRoutes() error {
	plugins := m.registry.List()
	
	for _, p := range plugins {
		if err := m.integratePluginRoute(p); err != nil {
			m.logger.Printf("Failed to integrate routes for plugin %s: %v", p.Name(), err)
			continue
		}
	}
	
	return nil
}

// integratePluginRoute 集成单个插件的路由
func (m *Manager) integratePluginRoute(p Plugin) error {
	name := p.Name()
	routes := p.GetRoutes()
	
	if len(routes) == 0 {
		return nil
	}
	
	// 为插件创建路由组
	groupPath := fmt.Sprintf("/api/plugins/%s", strings.ToLower(name))
	group := m.router.Group(groupPath)
	m.routeGroups[name] = group
	
	// 注册路由
	for _, route := range routes {
		handlers := make([]gin.HandlerFunc, 0, len(route.Middlewares)+1)
		
		// 添加中间件
		handlers = append(handlers, route.Middlewares...)
		
		// 添加权限检查中间件（如果需要）
		if route.Permission != "" {
			handlers = append(handlers, m.createPermissionMiddleware(route.Permission))
		}
		
		// 添加处理函数
		handlers = append(handlers, route.Handler)
		
		// 注册路由
		switch strings.ToUpper(route.Method) {
		case "GET":
			group.GET(route.Path, handlers...)
		case "POST":
			group.POST(route.Path, handlers...)
		case "PUT":
			group.PUT(route.Path, handlers...)
		case "DELETE":
			group.DELETE(route.Path, handlers...)
		case "PATCH":
			group.PATCH(route.Path, handlers...)
		case "OPTIONS":
			group.OPTIONS(route.Path, handlers...)
		case "HEAD":
			group.HEAD(route.Path, handlers...)
		default:
			return fmt.Errorf("unsupported HTTP method: %s", route.Method)
		}
		
		m.logger.Printf("Registered route: %s %s%s for plugin %s", 
			route.Method, groupPath, route.Path, name)
	}
	
	return nil
}

// createPermissionMiddleware 创建权限检查中间件
func (m *Manager) createPermissionMiddleware(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户ID
		userIDValue, exists := c.Get("user_id")
		if !exists {
			c.JSON(401, gin.H{"error": "Unauthorized", "message": "用户未认证"})
			c.Abort()
			return
		}

		userID, ok := userIDValue.(uint)
		if !ok {
			c.JSON(401, gin.H{"error": "Unauthorized", "message": "无效的用户ID"})
			c.Abort()
			return
		}

		// 如果没有配置RBAC服务，跳过权限检查（开发模式）
		if m.rbacService == nil {
			m.logger.Printf("Warning: RBAC service not configured, skipping permission check for %s", permission)
			c.Next()
			return
		}

		// 解析权限字符串 (格式: resource:action 或 plugin.name.resource:action)
		resource, action := parsePluginPermission(permission)
		
		// 检查权限
		allowed, err := m.rbacService.CheckPermission(userID, resource, action)
		if err != nil {
			m.logger.Printf("Permission check error for user %d, permission %s: %v", userID, permission, err)
			c.JSON(500, gin.H{"error": "Internal Server Error", "message": "权限检查失败"})
			c.Abort()
			return
		}

		if !allowed {
			m.logger.Printf("Permission denied for user %d, permission %s", userID, permission)
			c.JSON(403, gin.H{"error": "Forbidden", "message": "权限不足，无法访问此功能"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// parsePluginPermission 解析插件权限字符串
func parsePluginPermission(permission string) (resource, action string) {
	parts := strings.Split(permission, ":")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}
	return permission, "access"
}

// GetPluginMenus 获取所有插件的菜单项
func (m *Manager) GetPluginMenus() []MenuItem {
	var allMenus []MenuItem
	
	plugins := m.registry.List()
	for _, p := range plugins {
		menus := p.GetMenus()
		allMenus = append(allMenus, menus...)
	}
	
	return allMenus
}

// GetPluginPermissions 获取所有插件的权限
func (m *Manager) GetPluginPermissions() []Permission {
	var allPermissions []Permission
	
	plugins := m.registry.List()
	for _, p := range plugins {
		permissions := p.GetPermissions()
		allPermissions = append(allPermissions, permissions...)
	}
	
	return allPermissions
}

// StartPlugin 启动指定插件
func (m *Manager) StartPlugin(ctx context.Context, name string) error {
	return m.lifecycleManager.StartPlugin(ctx, name)
}

// StopPlugin 停止指定插件
func (m *Manager) StopPlugin(ctx context.Context, name string) error {
	return m.lifecycleManager.StopPlugin(ctx, name)
}

// RestartPlugin 重启指定插件
func (m *Manager) RestartPlugin(ctx context.Context, name string) error {
	return m.lifecycleManager.RestartPlugin(ctx, name)
}

// GetPlugin 获取插件实例
func (m *Manager) GetPlugin(name string) (Plugin, error) {
	plugin, exists := m.registry.Get(name)
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return plugin, nil
}

// ListPlugins 列出所有插件
func (m *Manager) ListPlugins() []Plugin {
	return m.registry.List()
}

// GetPluginInfo 获取插件信息
func (m *Manager) GetPluginInfo(name string) (*PluginInfo, error) {
	if registry, ok := m.registry.(*DefaultPluginRegistry); ok {
		info, exists := registry.GetInfo(name)
		if !exists {
			return nil, fmt.Errorf("plugin %s not found", name)
		}
		return info, nil
	}
	return nil, fmt.Errorf("unsupported registry type")
}

// GetAllPluginInfos 获取所有插件信息
func (m *Manager) GetAllPluginInfos() map[string]*PluginInfo {
	if registry, ok := m.registry.(*DefaultPluginRegistry); ok {
		return registry.GetAllInfos()
	}
	return make(map[string]*PluginInfo)
}

// CheckPluginHealth 检查插件健康状态
func (m *Manager) CheckPluginHealth(name string) error {
	return m.lifecycleManager.CheckHealth(name)
}

// Shutdown 关闭插件管理器
func (m *Manager) Shutdown(ctx context.Context) error {
	m.logger.Println("Shutting down plugin manager...")
	
	// 停止所有插件
	if err := m.lifecycleManager.StopAllPlugins(ctx); err != nil {
		m.logger.Printf("Error stopping plugins: %v", err)
	}
	
	m.logger.Println("Plugin manager shutdown completed")
	return nil
}

// CheckUserPluginPermission 检查用户是否有插件权限
func (m *Manager) CheckUserPluginPermission(userID uint, pluginName, resource, action string) (bool, error) {
	if m.permissionIntegrator == nil {
		return true, nil // 没有配置权限系统时默认允许
	}
	return m.permissionIntegrator.CheckPluginPermission(userID, pluginName, resource, action)
}

// GetUserAuthorizedPlugins 获取用户有权限使用的插件列表
func (m *Manager) GetUserAuthorizedPlugins(userID uint) []Plugin {
	allPlugins := m.registry.List()
	
	if m.rbacService == nil {
		return allPlugins // 没有配置权限系统时返回所有插件
	}
	
	var authorizedPlugins []Plugin
	for _, p := range allPlugins {
		// 检查用户是否有该插件的任意权限
		permissions := p.GetPermissions()
		if len(permissions) == 0 {
			// 没有定义权限的插件，所有人可用
			authorizedPlugins = append(authorizedPlugins, p)
			continue
		}
		
		// 检查是否有任意一个权限
		for _, perm := range permissions {
			resource := fmt.Sprintf("plugin.%s.%s", p.Name(), perm.Resource)
			allowed, err := m.rbacService.CheckPermission(userID, resource, perm.Action)
			if err == nil && allowed {
				authorizedPlugins = append(authorizedPlugins, p)
				break
			}
		}
	}
	
	return authorizedPlugins
}

// GetUserAuthorizedMenus 获取用户有权限的菜单
func (m *Manager) GetUserAuthorizedMenus(userID uint) []MenuItem {
	var authorizedMenus []MenuItem
	
	plugins := m.GetUserAuthorizedPlugins(userID)
	for _, p := range plugins {
		menus := p.GetMenus()
		for _, menu := range menus {
			if m.isMenuAuthorized(userID, p.Name(), menu) {
				authorizedMenu := m.filterAuthorizedChildren(userID, p.Name(), menu)
				authorizedMenus = append(authorizedMenus, authorizedMenu)
			}
		}
	}
	
	return authorizedMenus
}

// isMenuAuthorized 检查菜单是否授权
func (m *Manager) isMenuAuthorized(userID uint, pluginName string, menu MenuItem) bool {
	if !menu.Visible {
		return false
	}
	
	if menu.Permission == "" {
		return true // 没有权限要求的菜单，所有人可见
	}
	
	if m.rbacService == nil {
		return true
	}
	
	resource, action := parsePluginPermission(menu.Permission)
	// 如果权限不包含插件前缀，添加它
	if !strings.HasPrefix(resource, "plugin.") {
		resource = fmt.Sprintf("plugin.%s.%s", pluginName, resource)
	}
	
	allowed, err := m.rbacService.CheckPermission(userID, resource, action)
	return err == nil && allowed
}

// filterAuthorizedChildren 过滤授权的子菜单
func (m *Manager) filterAuthorizedChildren(userID uint, pluginName string, menu MenuItem) MenuItem {
	result := MenuItem{
		Key:        menu.Key,
		Label:      menu.Label,
		Icon:       menu.Icon,
		Path:       menu.Path,
		Permission: menu.Permission,
		Order:      menu.Order,
		Visible:    menu.Visible,
		Children:   make([]MenuItem, 0),
	}
	
	for _, child := range menu.Children {
		if m.isMenuAuthorized(userID, pluginName, child) {
			authorizedChild := m.filterAuthorizedChildren(userID, pluginName, child)
			result.Children = append(result.Children, authorizedChild)
		}
	}
	
	return result
}

// GetRBACService 获取RBAC服务
func (m *Manager) GetRBACService() *auth.RBACService {
	return m.rbacService
}

// GetPermissionIntegrator 获取权限集成器
func (m *Manager) GetPermissionIntegrator() *PermissionIntegrator {
	return m.permissionIntegrator
}