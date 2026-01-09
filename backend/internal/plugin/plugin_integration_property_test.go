package plugin

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestPluginIntegrationIntegrity 测试插件集成完整性属性
// Feature: network-monitoring-platform, Property 3: 插件集成完整性
// 对于任何注册的插件，系统应该正确集成其路由、权限和菜单项到主系统中，并支持启用/禁用控制
// **验证需求: 2.2, 2.3, 2.4, 2.5**
func TestPluginIntegrationIntegrity(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	properties.Property("registered plugin should be properly integrated", 
		prop.ForAll(
			func(pluginName, version, description string, routeCount, menuCount, permCount int) bool {
				// 确保生成的数据是有效的
				if len(pluginName) == 0 || len(version) == 0 || 
				   routeCount < 0 || menuCount < 0 || permCount < 0 {
					return true // 跳过无效数据
				}
				
				// 为每个测试创建独立的管理器
				pluginConfig := NewDefaultPluginConfig("./test_configs")
				defer os.RemoveAll("./test_configs")
				
				testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
				router := gin.New()
				manager := NewManager(router, pluginConfig, testLogger)
				
				// 创建测试插件
				plugin := createTestPlugin(pluginName, version, description, routeCount, menuCount, permCount)
				
				// 注册插件
				err := manager.RegisterPlugin(plugin)
				if err != nil {
					t.Logf("Failed to register plugin %s: %v", pluginName, err)
					return false
				}
				
				// 验证插件注册
				registeredPlugin, err := manager.GetPlugin(pluginName)
				if err != nil || registeredPlugin == nil {
					t.Logf("Plugin %s not found after registration", pluginName)
					return false
				}
				
				// 验证插件信息
				info, err := manager.GetPluginInfo(pluginName)
				if err != nil || info == nil {
					t.Logf("Plugin info not found for %s", pluginName)
					return false
				}
				
				if info.Name != pluginName || info.Version != version {
					t.Logf("Plugin info mismatch: expected %s/%s, got %s/%s", 
						pluginName, version, info.Name, info.Version)
					return false
				}
				
				// 验证路由集成
				routes := plugin.GetRoutes()
				if len(routes) != routeCount {
					t.Logf("Route count mismatch: expected %d, got %d", routeCount, len(routes))
					return false
				}
				
				// 验证菜单集成
				menus := manager.GetPluginMenus()
				pluginMenuCount := 0
				for _, menu := range menus {
					if menu.Key == fmt.Sprintf("%s_menu", pluginName) {
						pluginMenuCount++
					}
				}
				if pluginMenuCount != menuCount {
					t.Logf("Menu integration failed for plugin %s", pluginName)
					return false
				}
				
				// 验证权限集成
				permissions := manager.GetPluginPermissions()
				pluginPermCount := 0
				for _, perm := range permissions {
					// 检查权限资源是否匹配插件的权限格式
					expectedResource := "test"
					if perm.Resource == expectedResource {
						pluginPermCount++
					}
				}
				if pluginPermCount < permCount {
					t.Logf("Permission integration failed for plugin %s: expected %d, got %d", 
						pluginName, permCount, pluginPermCount)
					return false
				}
				
				// 加载插件（包括路由集成）
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				
				if err := manager.LoadPlugins(ctx); err != nil {
					t.Logf("Failed to load plugins: %v", err)
					return false
				}
				
				// 验证启用/禁用控制
				if err := manager.StopPlugin(ctx, pluginName); err != nil {
					t.Logf("Failed to stop plugin %s: %v", pluginName, err)
					return false
				}
				
				// 验证插件状态
				if err := manager.CheckPluginHealth(pluginName); err == nil {
					t.Logf("Plugin %s should be unhealthy after stop", pluginName)
					return false
				}
				
				// 重新启动插件
				if err := manager.StartPlugin(ctx, pluginName); err != nil {
					t.Logf("Failed to restart plugin %s: %v", pluginName, err)
					return false
				}
				
				// 验证插件重新启动后的健康状态
				if err := manager.CheckPluginHealth(pluginName); err != nil {
					t.Logf("Plugin %s should be healthy after restart: %v", pluginName, err)
					return false
				}
				
				return true
			},
			genValidPluginName(),
			genVersion(),
			genDescription(),
			gen.IntRange(0, 5),  // routeCount - 允许0个路由
			gen.IntRange(0, 3),  // menuCount - 允许0个菜单
			gen.IntRange(0, 4),  // permCount - 允许0个权限
		))

	properties.Property("plugin dependencies should be respected during loading",
		prop.ForAll(
			func(baseName, depName string, uniqueId int) bool {
				// 使用 uniqueId 确保名称不同
				actualBaseName := fmt.Sprintf("%s_base_%d", baseName, uniqueId)
				actualDepName := fmt.Sprintf("%s_dep_%d", depName, uniqueId+1)
				
				// 确保生成的名称有效
				if len(baseName) == 0 || len(depName) == 0 {
					return true // 跳过无效数据
				}
				
				// 为每个测试创建独立的管理器
				pluginConfig := NewDefaultPluginConfig("./test_configs_dep")
				defer os.RemoveAll("./test_configs_dep")
				
				testLogger := log.New(os.Stdout, "[TEST] ", log.LstdFlags)
				router := gin.New()
				manager := NewManager(router, pluginConfig, testLogger)
				
				// 创建有依赖关系的插件
				basePlugin := createTestPlugin(actualBaseName, "1.0.0", "Base plugin", 1, 1, 1)
				dependentPlugin := createTestPluginWithDependency(actualDepName, "1.0.0", "Dependent plugin", actualBaseName)
				
				// 先注册依赖插件（应该失败）
				err := manager.RegisterPlugin(dependentPlugin)
				if err == nil {
					t.Logf("Should fail to register dependent plugin %s without base plugin", actualDepName)
					return false
				}
				
				// 注册基础插件
				if err := manager.RegisterPlugin(basePlugin); err != nil {
					t.Logf("Failed to register base plugin %s: %v", actualBaseName, err)
					return false
				}
				
				// 现在注册依赖插件应该成功
				if err := manager.RegisterPlugin(dependentPlugin); err != nil {
					t.Logf("Failed to register dependent plugin %s: %v", actualDepName, err)
					return false
				}
				
				return true
			},
			genValidPluginName(),
			genValidPluginName(),
			gen.IntRange(1, 10000),
		))

	properties.TestingRun(t)
}

// createTestPlugin 创建测试插件
func createTestPlugin(name, version, description string, routeCount, menuCount, permCount int) Plugin {
	plugin := NewBasePlugin(name, version, description)
	
	// 添加路由
	for i := 0; i < routeCount; i++ {
		route := Route{
			Method:      "GET",
			Path:        fmt.Sprintf("/test%d", i),
			Handler:     func(c *gin.Context) { c.JSON(200, gin.H{"message": "test"}) },
			Permission:  fmt.Sprintf("plugin.%s.read", name),
			Description: fmt.Sprintf("Test route %d", i),
		}
		plugin.AddRoute(route)
	}
	
	// 添加菜单
	for i := 0; i < menuCount; i++ {
		menu := MenuItem{
			Key:        fmt.Sprintf("%s_menu", name),
			Label:      fmt.Sprintf("%s Menu", name),
			Icon:       "test-icon",
			Path:       fmt.Sprintf("/%s", name),
			Permission: fmt.Sprintf("plugin.%s.read", name),
			Order:      i,
			Visible:    true,
		}
		plugin.AddMenu(menu)
	}
	
	// 添加权限
	for i := 0; i < permCount; i++ {
		permission := Permission{
			Resource:    "test",
			Action:      fmt.Sprintf("action%d", i),
			Scope:       "all",
			Description: fmt.Sprintf("Test permission %d", i),
		}
		plugin.AddPermission(permission)
	}
	
	return plugin
}

// createTestPluginWithDependency 创建有依赖的测试插件
func createTestPluginWithDependency(name, version, description, dependency string) Plugin {
	plugin := NewBasePlugin(name, version, description)
	plugin.AddDependency(dependency)
	
	// 添加基本的路由、菜单和权限
	route := Route{
		Method:      "GET",
		Path:        "/test",
		Handler:     func(c *gin.Context) { c.JSON(200, gin.H{"message": "test"}) },
		Permission:  fmt.Sprintf("plugin.%s.read", name),
		Description: "Test route",
	}
	plugin.AddRoute(route)
	
	return plugin
}

// genValidPluginName 生成有效的插件名称
func genValidPluginName() gopter.Gen {
	return gen.AlphaString().Map(func(s string) string {
		// 确保至少有3个字符
		if len(s) < 3 {
			s = "abc" + s
		}
		// 截断到最多10个字符
		if len(s) > 10 {
			s = s[:10]
		}
		// 添加时间戳确保唯一性
		return fmt.Sprintf("test_%s_%d", s, time.Now().UnixNano()%10000)
	})
}

// genVersion 生成版本号
func genVersion() gopter.Gen {
	return gen.Const("1.0.0")
}

// genDescription 生成描述
func genDescription() gopter.Gen {
	return gen.AlphaString().Map(func(s string) string {
		// 确保至少有5个字符
		if len(s) < 5 {
			s = "descr" + s
		}
		// 截断到最多30个字符
		if len(s) > 30 {
			s = s[:30]
		}
		return "Test plugin: " + s
	})
}