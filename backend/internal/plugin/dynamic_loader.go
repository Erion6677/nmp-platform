package plugin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// DynamicLoader 动态加载器
type DynamicLoader struct {
	manager    *Manager
	logger     *log.Logger
	pluginDirs []string
	watchers   map[string]*DirectoryWatcher
}

// DirectoryWatcher 目录监视器
type DirectoryWatcher struct {
	path     string
	loader   *DynamicLoader
	stopChan chan bool
}

// PluginManifest 插件清单文件
type PluginManifest struct {
	Name         string                 `json:"name"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	Dependencies []string               `json:"dependencies"`
	EntryPoint   string                 `json:"entry_point"`
	Frontend     *FrontendManifest      `json:"frontend,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Permissions  []PermissionManifest   `json:"permissions,omitempty"`
	Routes       []RouteManifest        `json:"routes,omitempty"`
	Menus        []MenuManifest         `json:"menus,omitempty"`
}

// FrontendManifest 前端清单
type FrontendManifest struct {
	EntryPoint string            `json:"entry_point"`
	AssetsDir  string            `json:"assets_dir"`
	Routes     []FrontendRoute   `json:"routes"`
	Config     map[string]interface{} `json:"config"`
}

// PermissionManifest 权限清单
type PermissionManifest struct {
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
}

// RouteManifest 路由清单
type RouteManifest struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	Handler     string   `json:"handler"`
	Middlewares []string `json:"middlewares"`
	Permission  string   `json:"permission"`
	Description string   `json:"description"`
}

// MenuManifest 菜单清单
type MenuManifest struct {
	Key        string         `json:"key"`
	Label      string         `json:"label"`
	Icon       string         `json:"icon"`
	Path       string         `json:"path"`
	Children   []MenuManifest `json:"children"`
	Permission string         `json:"permission"`
	Order      int            `json:"order"`
	Visible    bool           `json:"visible"`
}

// NewDynamicLoader 创建动态加载器
func NewDynamicLoader(manager *Manager, logger *log.Logger) *DynamicLoader {
	return &DynamicLoader{
		manager:    manager,
		logger:     logger,
		pluginDirs: make([]string, 0),
		watchers:   make(map[string]*DirectoryWatcher),
	}
}

// AddPluginDirectory 添加插件目录
func (dl *DynamicLoader) AddPluginDirectory(dir string) error {
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("plugin directory does not exist: %s", dir)
	}
	
	dl.pluginDirs = append(dl.pluginDirs, dir)
	dl.logger.Printf("Added plugin directory: %s", dir)
	return nil
}

// ScanAndLoadPlugins 扫描并加载插件
func (dl *DynamicLoader) ScanAndLoadPlugins() error {
	dl.logger.Println("Scanning for plugins...")
	
	for _, dir := range dl.pluginDirs {
		if err := dl.scanDirectory(dir); err != nil {
			dl.logger.Printf("Error scanning directory %s: %v", dir, err)
			continue
		}
	}
	
	return nil
}

// scanDirectory 扫描目录
func (dl *DynamicLoader) scanDirectory(dir string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// 查找插件清单文件
		if d.IsDir() {
			manifestPath := filepath.Join(path, "plugin.json")
			if _, err := os.Stat(manifestPath); err == nil {
				if err := dl.loadPluginFromManifest(manifestPath); err != nil {
					dl.logger.Printf("Failed to load plugin from %s: %v", manifestPath, err)
				}
			}
		}
		
		return nil
	})
}

// loadPluginFromManifest 从清单文件加载插件
func (dl *DynamicLoader) loadPluginFromManifest(manifestPath string) error {
	dl.logger.Printf("Loading plugin from manifest: %s", manifestPath)
	
	// 读取清单文件
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	
	// 解析清单
	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	// 验证清单
	if err := dl.validateManifest(&manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}
	
	// 创建插件实例
	plugin := dl.createPluginFromManifest(&manifest, filepath.Dir(manifestPath))
	
	// 注册插件
	if err := dl.manager.RegisterPlugin(plugin); err != nil {
		return fmt.Errorf("failed to register plugin: %w", err)
	}
	
	// 保存插件配置
	if manifest.Config != nil {
		if err := dl.manager.config.SetConfig(manifest.Name, manifest.Config); err != nil {
			dl.logger.Printf("Failed to save config for plugin %s: %v", manifest.Name, err)
		}
	}
	
	dl.logger.Printf("Successfully loaded plugin: %s (version: %s)", manifest.Name, manifest.Version)
	return nil
}

// validateManifest 验证清单
func (dl *DynamicLoader) validateManifest(manifest *PluginManifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	
	if manifest.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	
	if manifest.EntryPoint == "" {
		return fmt.Errorf("plugin entry point is required")
	}
	
	// 验证插件名称格式
	if !isValidPluginName(manifest.Name) {
		return fmt.Errorf("invalid plugin name format: %s", manifest.Name)
	}
	
	return nil
}

// createPluginFromManifest 从清单创建插件
func (dl *DynamicLoader) createPluginFromManifest(manifest *PluginManifest, pluginDir string) Plugin {
	plugin := NewBasePlugin(manifest.Name, manifest.Version, manifest.Description)
	
	// 添加依赖
	for _, dep := range manifest.Dependencies {
		plugin.AddDependency(dep)
	}
	
	// 添加权限
	for _, perm := range manifest.Permissions {
		permission := Permission{
			Resource:    perm.Resource,
			Action:      perm.Action,
			Scope:       perm.Scope,
			Description: perm.Description,
		}
		plugin.AddPermission(permission)
	}
	
	// 添加菜单
	for _, menu := range manifest.Menus {
		menuItem := dl.convertMenuManifest(menu)
		plugin.AddMenu(menuItem)
	}
	
	// 添加路由 - 创建动态处理器
	for _, route := range manifest.Routes {
		handler := dl.createDynamicRouteHandler(manifest.Name, pluginDir, route)
		pluginRoute := Route{
			Method:      route.Method,
			Path:        route.Path,
			Handler:     handler,
			Permission:  route.Permission,
			Description: route.Description,
		}
		plugin.AddRoute(pluginRoute)
		dl.logger.Printf("Registered dynamic route for plugin %s: %s %s", 
			manifest.Name, route.Method, route.Path)
	}
	
	// 设置配置模式
	if manifest.Config != nil {
		plugin.SetConfigSchema(manifest.Config)
	}
	
	return plugin
}

// createDynamicRouteHandler 创建动态路由处理器
func (dl *DynamicLoader) createDynamicRouteHandler(pluginName, pluginDir string, route RouteManifest) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 根据handler类型决定处理方式
		handlerType := dl.parseHandlerType(route.Handler)
		
		switch handlerType {
		case "script":
			// 执行脚本处理器
			dl.executeScriptHandler(c, pluginName, pluginDir, route)
		case "http":
			// HTTP代理到外部服务
			dl.executeHTTPProxyHandler(c, pluginName, route)
		case "static":
			// 静态文件服务
			dl.executeStaticHandler(c, pluginName, pluginDir, route)
		default:
			// 默认返回插件信息
			c.JSON(200, gin.H{
				"plugin":      pluginName,
				"route":       route.Path,
				"method":      route.Method,
				"description": route.Description,
				"message":     "Dynamic route handler",
			})
		}
	}
}

// HandlerType 处理器类型
type HandlerType string

const (
	HandlerTypeScript  HandlerType = "script"
	HandlerTypeHTTP    HandlerType = "http"
	HandlerTypeStatic  HandlerType = "static"
	HandlerTypeDefault HandlerType = "default"
)

// parseHandlerType 解析处理器类型
func (dl *DynamicLoader) parseHandlerType(handler string) HandlerType {
	if strings.HasPrefix(handler, "script:") {
		return HandlerTypeScript
	}
	if strings.HasPrefix(handler, "http://") || strings.HasPrefix(handler, "https://") {
		return HandlerTypeHTTP
	}
	if strings.HasPrefix(handler, "static:") {
		return HandlerTypeStatic
	}
	return HandlerTypeDefault
}

// executeScriptHandler 执行脚本处理器
func (dl *DynamicLoader) executeScriptHandler(c *gin.Context, pluginName, pluginDir string, route RouteManifest) {
	// 从handler中提取脚本路径
	scriptPath := strings.TrimPrefix(route.Handler, "script:")
	fullPath := filepath.Join(pluginDir, scriptPath)
	
	// 检查脚本是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(500, gin.H{
			"error":   "Script not found",
			"message": fmt.Sprintf("Handler script not found: %s", scriptPath),
		})
		return
	}
	
	// 这里可以扩展支持不同类型的脚本执行
	// 目前返回脚本信息
	c.JSON(200, gin.H{
		"plugin":  pluginName,
		"handler": "script",
		"script":  scriptPath,
		"message": "Script handler registered (execution not implemented)",
	})
}

// executeHTTPProxyHandler HTTP代理处理器
func (dl *DynamicLoader) executeHTTPProxyHandler(c *gin.Context, pluginName string, route RouteManifest) {
	// 代理请求到外部HTTP服务
	targetURL := route.Handler
	
	// 创建代理请求
	// 这里简化实现，实际应该使用 httputil.ReverseProxy
	c.JSON(200, gin.H{
		"plugin":    pluginName,
		"handler":   "http_proxy",
		"target":    targetURL,
		"message":   "HTTP proxy handler (full implementation pending)",
	})
}

// executeStaticHandler 静态文件处理器
func (dl *DynamicLoader) executeStaticHandler(c *gin.Context, pluginName, pluginDir string, route RouteManifest) {
	// 从handler中提取静态文件目录
	staticDir := strings.TrimPrefix(route.Handler, "static:")
	fullPath := filepath.Join(pluginDir, staticDir)
	
	// 获取请求的文件路径
	requestPath := c.Param("filepath")
	if requestPath == "" {
		requestPath = "index.html"
	}
	
	filePath := filepath.Join(fullPath, requestPath)
	
	// 安全检查：确保路径在插件目录内
	if !strings.HasPrefix(filePath, pluginDir) {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}
	
	c.File(filePath)
}

// convertMenuManifest 转换菜单清单
func (dl *DynamicLoader) convertMenuManifest(manifest MenuManifest) MenuItem {
	menu := MenuItem{
		Key:        manifest.Key,
		Label:      manifest.Label,
		Icon:       manifest.Icon,
		Path:       manifest.Path,
		Permission: manifest.Permission,
		Order:      manifest.Order,
		Visible:    manifest.Visible,
	}
	
	// 转换子菜单
	for _, child := range manifest.Children {
		childMenu := dl.convertMenuManifest(child)
		menu.Children = append(menu.Children, childMenu)
	}
	
	return menu
}

// isValidPluginName 检查插件名称是否有效
func isValidPluginName(name string) bool {
	if len(name) < 3 || len(name) > 50 {
		return false
	}
	
	// 只允许字母、数字、下划线和连字符
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_' || char == '-') {
			return false
		}
	}
	
	return true
}

// StartWatching 开始监视插件目录
func (dl *DynamicLoader) StartWatching() error {
	for _, dir := range dl.pluginDirs {
		watcher := &DirectoryWatcher{
			path:     dir,
			loader:   dl,
			stopChan: make(chan bool),
		}
		
		dl.watchers[dir] = watcher
		go watcher.watch()
		
		dl.logger.Printf("Started watching plugin directory: %s", dir)
	}
	
	return nil
}

// StopWatching 停止监视
func (dl *DynamicLoader) StopWatching() {
	for dir, watcher := range dl.watchers {
		close(watcher.stopChan)
		delete(dl.watchers, dir)
		dl.logger.Printf("Stopped watching plugin directory: %s", dir)
	}
}

// watch 监视目录变化
func (dw *DirectoryWatcher) watch() {
	// 这里应该实现文件系统监视
	// 由于简化实现，这里只是一个占位符
	dw.loader.logger.Printf("Directory watcher started for: %s", dw.path)
	
	// 等待停止信号
	<-dw.stopChan
	dw.loader.logger.Printf("Directory watcher stopped for: %s", dw.path)
}

// ReloadPlugin 重新加载插件
func (dl *DynamicLoader) ReloadPlugin(pluginName string) error {
	dl.logger.Printf("Reloading plugin: %s", pluginName)
	
	// 查找插件清单文件
	var manifestPath string
	for _, dir := range dl.pluginDirs {
		path := filepath.Join(dir, pluginName, "plugin.json")
		if _, err := os.Stat(path); err == nil {
			manifestPath = path
			break
		}
	}
	
	if manifestPath == "" {
		return fmt.Errorf("plugin manifest not found for: %s", pluginName)
	}
	
	// 停止现有插件
	if err := dl.manager.StopPlugin(nil, pluginName); err != nil {
		dl.logger.Printf("Failed to stop plugin %s: %v", pluginName, err)
	}
	
	// 重新加载插件
	if err := dl.loadPluginFromManifest(manifestPath); err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}
	
	// 启动插件
	if err := dl.manager.StartPlugin(nil, pluginName); err != nil {
		return fmt.Errorf("failed to start reloaded plugin: %w", err)
	}
	
	dl.logger.Printf("Successfully reloaded plugin: %s", pluginName)
	return nil
}

// GetPluginManifest 获取插件清单
func (dl *DynamicLoader) GetPluginManifest(pluginName string) (*PluginManifest, error) {
	for _, dir := range dl.pluginDirs {
		manifestPath := filepath.Join(dir, pluginName, "plugin.json")
		if _, err := os.Stat(manifestPath); err == nil {
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				return nil, err
			}
			
			var manifest PluginManifest
			if err := json.Unmarshal(data, &manifest); err != nil {
				return nil, err
			}
			
			return &manifest, nil
		}
	}
	
	return nil, fmt.Errorf("manifest not found for plugin: %s", pluginName)
}

// ListAvailablePlugins 列出可用的插件
func (dl *DynamicLoader) ListAvailablePlugins() ([]PluginManifest, error) {
	var manifests []PluginManifest
	
	for _, dir := range dl.pluginDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			
			if d.IsDir() {
				manifestPath := filepath.Join(path, "plugin.json")
				if _, err := os.Stat(manifestPath); err == nil {
					data, err := os.ReadFile(manifestPath)
					if err != nil {
						return err
					}
					
					var manifest PluginManifest
					if err := json.Unmarshal(data, &manifest); err != nil {
						return err
					}
					
					manifests = append(manifests, manifest)
				}
			}
			
			return nil
		})
		
		if err != nil {
			dl.logger.Printf("Error scanning directory %s: %v", dir, err)
		}
	}
	
	return manifests, nil
}