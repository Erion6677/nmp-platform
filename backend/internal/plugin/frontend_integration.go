package plugin

import (
	"fmt"
	"log"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
)

// FrontendIntegrator 前端集成器
type FrontendIntegrator struct {
	manager *Manager
	logger  *log.Logger
}

// NewFrontendIntegrator 创建前端集成器
func NewFrontendIntegrator(manager *Manager, logger *log.Logger) *FrontendIntegrator {
	return &FrontendIntegrator{
		manager: manager,
		logger:  logger,
	}
}

// MenuResponse 菜单响应结构
type MenuResponse struct {
	Key         string         `json:"key"`
	Label       string         `json:"label"`
	Icon        string         `json:"icon"`
	Path        string         `json:"path"`
	Children    []MenuResponse `json:"children,omitempty"`
	Permission  string         `json:"permission,omitempty"`
	Order       int            `json:"order"`
	Visible     bool           `json:"visible"`
	PluginName  string         `json:"plugin_name,omitempty"`
}

// ModuleInfo 前端模块信息
type ModuleInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	EntryPoint  string                 `json:"entry_point"`
	Assets      []string               `json:"assets"`
	Routes      []FrontendRoute        `json:"routes"`
	Config      map[string]interface{} `json:"config"`
	PluginName  string                 `json:"plugin_name"`
}

// FrontendRoute 前端路由信息
type FrontendRoute struct {
	Path      string `json:"path"`
	Component string `json:"component"`
	Name      string `json:"name"`
	Meta      RouteMetadata `json:"meta"`
}

// RouteMetadata 路由元数据
type RouteMetadata struct {
	Title       string `json:"title"`
	Icon        string `json:"icon"`
	Permission  string `json:"permission"`
	KeepAlive   bool   `json:"keep_alive"`
	Hidden      bool   `json:"hidden"`
}

// RegisterFrontendRoutes 注册前端集成相关的API路由
func (fi *FrontendIntegrator) RegisterFrontendRoutes(router *gin.RouterGroup) {
	// 插件菜单API
	router.GET("/menus", fi.GetPluginMenus)
	
	// 插件模块信息API
	router.GET("/modules", fi.GetPluginModules)
	
	// 插件配置API
	router.GET("/config/:plugin", fi.GetPluginConfig)
	router.PUT("/config/:plugin", fi.UpdatePluginConfig)
	
	// 插件状态API
	router.GET("/status", fi.GetPluginStatus)
	router.POST("/status/:plugin/:action", fi.ControlPlugin)
}

// GetPluginMenus 获取插件菜单
func (fi *FrontendIntegrator) GetPluginMenus(c *gin.Context) {
	// 获取用户权限信息（从JWT或session中）
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的用户ID"})
		return
	}

	// 获取用户有权限的菜单
	authorizedMenus := fi.manager.GetUserAuthorizedMenus(userID)
	
	// 转换为响应格式
	var menuResponses []MenuResponse
	for _, menu := range authorizedMenus {
		menuResp := fi.convertMenuToResponse(menu)
		menuResponses = append(menuResponses, menuResp)
	}
	
	// 按order排序
	sort.Slice(menuResponses, func(i, j int) bool {
		return menuResponses[i].Order < menuResponses[j].Order
	})
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": menuResponses,
		"message": "获取菜单成功",
	})
}

// convertMenuToResponse 转换菜单为响应格式
func (fi *FrontendIntegrator) convertMenuToResponse(menu MenuItem) MenuResponse {
	resp := MenuResponse{
		Key:        menu.Key,
		Label:      menu.Label,
		Icon:       menu.Icon,
		Path:       menu.Path,
		Permission: menu.Permission,
		Order:      menu.Order,
		Visible:    menu.Visible,
		Children:   make([]MenuResponse, 0),
	}
	
	for _, child := range menu.Children {
		childResp := fi.convertMenuToResponse(child)
		resp.Children = append(resp.Children, childResp)
	}
	
	return resp
}

// GetPluginModules 获取插件前端模块信息
func (fi *FrontendIntegrator) GetPluginModules(c *gin.Context) {
	plugins := fi.manager.ListPlugins()
	var modules []ModuleInfo
	
	for _, plugin := range plugins {
		// 获取插件信息
		info, err := fi.manager.GetPluginInfo(plugin.Name())
		if err != nil {
			fi.logger.Printf("Failed to get plugin info for %s: %v", plugin.Name(), err)
			continue
		}
		
		// 检查插件是否启动
		if info.Status != PluginStatusStarted {
			continue
		}
		
		// 构建模块信息
		module := ModuleInfo{
			Name:       plugin.Name(),
			Version:    plugin.Version(),
			PluginName: plugin.Name(),
		}
		
		// 从插件配置中获取前端模块信息
		config, err := fi.manager.config.GetConfig(plugin.Name())
		if err == nil {
			if configMap, ok := config.(map[string]interface{}); ok {
				if frontend, exists := configMap["frontend"]; exists {
					if frontendConfig, ok := frontend.(map[string]interface{}); ok {
						// 解析前端配置
						fi.parseFrontendConfig(&module, frontendConfig)
					}
				}
			}
		}
		
		// 从插件路由生成前端路由信息
		routes := plugin.GetRoutes()
		for _, route := range routes {
			frontendRoute := FrontendRoute{
				Path:      route.Path,
				Component: fmt.Sprintf("plugin-%s-%s", plugin.Name(), route.Path),
				Name:      fmt.Sprintf("%s%s", plugin.Name(), route.Path),
				Meta: RouteMetadata{
					Title:      route.Description,
					Permission: route.Permission,
					Hidden:     false,
				},
			}
			module.Routes = append(module.Routes, frontendRoute)
		}
		
		modules = append(modules, module)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": modules,
		"message": "获取模块信息成功",
	})
}

// parseFrontendConfig 解析前端配置
func (fi *FrontendIntegrator) parseFrontendConfig(module *ModuleInfo, config map[string]interface{}) {
	if entryPoint, ok := config["entry_point"].(string); ok {
		module.EntryPoint = entryPoint
	}
	
	if assets, ok := config["assets"].([]interface{}); ok {
		for _, asset := range assets {
			if assetStr, ok := asset.(string); ok {
				module.Assets = append(module.Assets, assetStr)
			}
		}
	}
	
	if moduleConfig, ok := config["config"].(map[string]interface{}); ok {
		module.Config = moduleConfig
	}
}

// GetPluginConfig 获取插件配置
func (fi *FrontendIntegrator) GetPluginConfig(c *gin.Context) {
	pluginName := c.Param("plugin")
	if pluginName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件名称不能为空"})
		return
	}
	
	// 检查插件是否存在
	_, err := fi.manager.GetPlugin(pluginName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在"})
		return
	}
	
	// 获取插件配置
	config, err := fi.manager.config.GetConfig(pluginName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取配置失败"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": config,
		"message": "获取配置成功",
	})
}

// UpdatePluginConfig 更新插件配置
func (fi *FrontendIntegrator) UpdatePluginConfig(c *gin.Context) {
	pluginName := c.Param("plugin")
	if pluginName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件名称不能为空"})
		return
	}
	
	// 检查插件是否存在
	plugin, err := fi.manager.GetPlugin(pluginName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在"})
		return
	}
	
	// 解析请求体
	var newConfig map[string]interface{}
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置格式错误"})
		return
	}
	
	// 验证配置
	if err := fi.manager.config.ValidateConfig(pluginName, newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置验证失败: " + err.Error()})
		return
	}
	
	// 验证配置模式（如果插件定义了模式）
	schema := plugin.GetConfigSchema()
	if schema != nil {
		if err := fi.validateConfigAgainstSchema(newConfig, schema); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "配置验证失败: " + err.Error()})
			return
		}
	}
	
	// 保存配置
	if err := fi.manager.config.SetConfig(pluginName, newConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"message": "配置更新成功",
	})
}

// GetPluginStatus 获取插件状态
func (fi *FrontendIntegrator) GetPluginStatus(c *gin.Context) {
	allInfos := fi.manager.GetAllPluginInfos()
	
	var statusList []map[string]interface{}
	for name, info := range allInfos {
		status := map[string]interface{}{
			"name":         name,
			"version":      info.Version,
			"description":  info.Description,
			"status":       info.Status,
			"created_at":   info.CreatedAt,
			"updated_at":   info.UpdatedAt,
		}
		
		// 检查健康状态
		if err := fi.manager.CheckPluginHealth(name); err != nil {
			status["health"] = "unhealthy"
			status["health_error"] = err.Error()
		} else {
			status["health"] = "healthy"
		}
		
		statusList = append(statusList, status)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": statusList,
		"message": "获取状态成功",
	})
}

// ControlPlugin 控制插件（启动/停止/重启）
func (fi *FrontendIntegrator) ControlPlugin(c *gin.Context) {
	pluginName := c.Param("plugin")
	action := c.Param("action")
	
	if pluginName == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件名称和操作不能为空"})
		return
	}
	
	// 检查插件是否存在
	_, err := fi.manager.GetPlugin(pluginName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在"})
		return
	}
	
	ctx := c.Request.Context()
	
	switch action {
	case "start":
		err = fi.manager.StartPlugin(ctx, pluginName)
	case "stop":
		err = fi.manager.StopPlugin(ctx, pluginName)
	case "restart":
		err = fi.manager.RestartPlugin(ctx, pluginName)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的操作"})
		return
	}
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("操作失败: %v", err),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"message": fmt.Sprintf("插件%s%s成功", pluginName, action),
	})
}

// GenerateMenuTree 生成菜单树结构
func (fi *FrontendIntegrator) GenerateMenuTree(menus []MenuItem) []MenuResponse {
	menuMap := make(map[string]*MenuResponse)
	var rootMenus []MenuResponse
	
	// 第一遍：创建所有菜单项
	for _, menu := range menus {
		menuResp := &MenuResponse{
			Key:        menu.Key,
			Label:      menu.Label,
			Icon:       menu.Icon,
			Path:       menu.Path,
			Permission: menu.Permission,
			Order:      menu.Order,
			Visible:    menu.Visible,
			Children:   make([]MenuResponse, 0),
		}
		menuMap[menu.Key] = menuResp
	}
	
	// 第二遍：构建树结构
	for _, menu := range menus {
		menuResp := menuMap[menu.Key]
		
		// 处理子菜单
		for _, child := range menu.Children {
			if childResp, exists := menuMap[child.Key]; exists {
				menuResp.Children = append(menuResp.Children, *childResp)
			}
		}
		
		// 如果是根菜单，添加到根菜单列表
		isRoot := true
		for _, otherMenu := range menus {
			for _, child := range otherMenu.Children {
				if child.Key == menu.Key {
					isRoot = false
					break
				}
			}
			if !isRoot {
				break
			}
		}
		
		if isRoot {
			rootMenus = append(rootMenus, *menuResp)
		}
	}
	
	// 排序
	sort.Slice(rootMenus, func(i, j int) bool {
		return rootMenus[i].Order < rootMenus[j].Order
	})
	
	return rootMenus
}

// GetPluginAssets 获取插件静态资源
func (fi *FrontendIntegrator) GetPluginAssets(c *gin.Context) {
	pluginName := c.Param("plugin")
	assetPath := c.Param("path")
	
	if pluginName == "" || assetPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不完整"})
		return
	}
	
	// 检查插件是否存在且已启动
	plugin, err := fi.manager.GetPlugin(pluginName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在"})
		return
	}
	
	info, err := fi.manager.GetPluginInfo(pluginName)
	if err != nil || info.Status != PluginStatusStarted {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "插件未启动"})
		return
	}
	
	// 获取插件配置中的资源路径
	config, err := fi.manager.config.GetConfig(pluginName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取插件配置失败"})
		return
	}
	
	// 构建资源文件路径
	var assetDir string
	if configMap, ok := config.(map[string]interface{}); ok {
		if frontend, exists := configMap["frontend"]; exists {
			if frontendConfig, ok := frontend.(map[string]interface{}); ok {
				if assetsDir, ok := frontendConfig["assets_dir"].(string); ok {
					assetDir = assetsDir
				}
			}
		}
	}
	
	if assetDir == "" {
		assetDir = fmt.Sprintf("./plugins/%s/assets", plugin.Name())
	}
	
	// 提供静态文件服务
	fullPath := fmt.Sprintf("%s/%s", assetDir, assetPath)
	c.File(fullPath)
}


// validateConfigAgainstSchema 验证配置是否符合模式
func (fi *FrontendIntegrator) validateConfigAgainstSchema(config map[string]interface{}, schema interface{}) error {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil // 无法解析模式，跳过验证
	}
	
	// 检查必需字段
	if required, exists := schemaMap["required"]; exists {
		if requiredFields, ok := required.([]interface{}); ok {
			for _, field := range requiredFields {
				fieldName, ok := field.(string)
				if !ok {
					continue
				}
				if _, exists := config[fieldName]; !exists {
					return fmt.Errorf("缺少必需字段: %s", fieldName)
				}
			}
		}
	}
	
	// 检查字段类型
	if properties, exists := schemaMap["properties"]; exists {
		if propsMap, ok := properties.(map[string]interface{}); ok {
			for fieldName, fieldSchema := range propsMap {
				if value, exists := config[fieldName]; exists {
					if err := fi.validateFieldType(fieldName, value, fieldSchema); err != nil {
						return err
					}
				}
			}
		}
	}
	
	return nil
}

// validateFieldType 验证字段类型
func (fi *FrontendIntegrator) validateFieldType(fieldName string, value interface{}, schema interface{}) error {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil
	}
	
	expectedType, exists := schemaMap["type"]
	if !exists {
		return nil
	}
	
	typeStr, ok := expectedType.(string)
	if !ok {
		return nil
	}
	
	// 类型检查
	switch typeStr {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("字段 %s 应为字符串类型", fieldName)
		}
	case "number", "integer":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			// OK
		default:
			return fmt.Errorf("字段 %s 应为数字类型", fieldName)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("字段 %s 应为布尔类型", fieldName)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("字段 %s 应为数组类型", fieldName)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("字段 %s 应为对象类型", fieldName)
		}
	}
	
	// 检查枚举值
	if enum, exists := schemaMap["enum"]; exists {
		if enumValues, ok := enum.([]interface{}); ok {
			found := false
			for _, enumVal := range enumValues {
				if value == enumVal {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("字段 %s 的值不在允许的范围内", fieldName)
			}
		}
	}
	
	// 检查字符串长度
	if typeStr == "string" {
		strValue, _ := value.(string)
		if minLen, exists := schemaMap["minLength"]; exists {
			if minLenFloat, ok := minLen.(float64); ok {
				if len(strValue) < int(minLenFloat) {
					return fmt.Errorf("字段 %s 长度不能小于 %d", fieldName, int(minLenFloat))
				}
			}
		}
		if maxLen, exists := schemaMap["maxLength"]; exists {
			if maxLenFloat, ok := maxLen.(float64); ok {
				if len(strValue) > int(maxLenFloat) {
					return fmt.Errorf("字段 %s 长度不能大于 %d", fieldName, int(maxLenFloat))
				}
			}
		}
	}
	
	// 检查数字范围
	if typeStr == "number" || typeStr == "integer" {
		var numValue float64
		switch v := value.(type) {
		case int:
			numValue = float64(v)
		case int32:
			numValue = float64(v)
		case int64:
			numValue = float64(v)
		case float32:
			numValue = float64(v)
		case float64:
			numValue = v
		}
		
		if min, exists := schemaMap["minimum"]; exists {
			if minFloat, ok := min.(float64); ok {
				if numValue < minFloat {
					return fmt.Errorf("字段 %s 不能小于 %v", fieldName, minFloat)
				}
			}
		}
		if max, exists := schemaMap["maximum"]; exists {
			if maxFloat, ok := max.(float64); ok {
				if numValue > maxFloat {
					return fmt.Errorf("字段 %s 不能大于 %v", fieldName, maxFloat)
				}
			}
		}
	}
	
	return nil
}
