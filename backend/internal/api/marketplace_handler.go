package api

import (
	"net/http"

	"nmp-platform/internal/marketplace"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MarketplaceHandler 插件市场处理器
type MarketplaceHandler struct {
	marketplace *marketplace.Marketplace
	logger      *zap.Logger
}

// NewMarketplaceHandler 创建插件市场处理器
func NewMarketplaceHandler(mp *marketplace.Marketplace, logger *zap.Logger) *MarketplaceHandler {
	return &MarketplaceHandler{
		marketplace: mp,
		logger:      logger,
	}
}

// RegisterRoutes 注册路由
func (h *MarketplaceHandler) RegisterRoutes(router *gin.RouterGroup) {
	marketGroup := router.Group("/marketplace")
	{
		marketGroup.GET("/plugins", h.ListAvailablePlugins)
		marketGroup.GET("/plugins/:name", h.GetPluginInfo)
		marketGroup.GET("/installed", h.ListInstalledPlugins)
		marketGroup.GET("/menus", h.GetPluginMenus)
		marketGroup.POST("/install/:name", h.InstallPlugin)
		marketGroup.POST("/uninstall/:name", h.UninstallPlugin)
		marketGroup.POST("/update/:name", h.UpdatePlugin)
		marketGroup.POST("/refresh", h.RefreshRegistry)
	}
}

// ListAvailablePlugins 获取可用插件列表
func (h *MarketplaceHandler) ListAvailablePlugins(c *gin.Context) {
	plugins, err := h.marketplace.GetAvailablePlugins()
	if err != nil {
		h.logger.Error("Failed to get available plugins", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取插件列表失败: " + err.Error(),
		})
		return
	}

	// 标记已安装的插件
	type PluginWithStatus struct {
		*marketplace.RegistryPlugin
		Installed bool `json:"installed"`
	}

	var result []PluginWithStatus
	for _, p := range plugins {
		result = append(result, PluginWithStatus{
			RegistryPlugin: p,
			Installed:      h.marketplace.IsPluginInstalled(p.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plugins": result,
			"total":   len(result),
		},
	})
}

// GetPluginInfo 获取插件详情
func (h *MarketplaceHandler) GetPluginInfo(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "插件名称不能为空",
		})
		return
	}

	info, err := h.marketplace.GetPluginInfo(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plugin":    info,
			"installed": h.marketplace.IsPluginInstalled(name),
		},
	})
}

// ListInstalledPlugins 获取已安装插件列表
func (h *MarketplaceHandler) ListInstalledPlugins(c *gin.Context) {
	plugins, err := h.marketplace.GetInstalledPlugins()
	if err != nil {
		h.logger.Error("Failed to get installed plugins", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取已安装插件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"plugins": plugins,
			"total":   len(plugins),
		},
	})
}

// InstallPlugin 安装插件
func (h *MarketplaceHandler) InstallPlugin(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "插件名称不能为空",
		})
		return
	}

	// 检查是否已安装
	if h.marketplace.IsPluginInstalled(name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "插件已安装",
		})
		return
	}

	if err := h.marketplace.InstallPlugin(name); err != nil {
		h.logger.Error("Failed to install plugin", zap.String("name", name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "安装插件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "插件安装成功，请重启服务以加载插件",
	})
}

// UninstallPlugin 卸载插件
func (h *MarketplaceHandler) UninstallPlugin(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "插件名称不能为空",
		})
		return
	}

	if err := h.marketplace.UninstallPlugin(name); err != nil {
		h.logger.Error("Failed to uninstall plugin", zap.String("name", name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "卸载插件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "插件卸载成功，请重启服务以应用更改",
	})
}

// UpdatePlugin 更新插件
func (h *MarketplaceHandler) UpdatePlugin(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "插件名称不能为空",
		})
		return
	}

	if err := h.marketplace.UpdatePlugin(name); err != nil {
		h.logger.Error("Failed to update plugin", zap.String("name", name), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "更新插件失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "插件更新成功，请重启服务以加载新版本",
	})
}

// RefreshRegistry 刷新插件注册表
func (h *MarketplaceHandler) RefreshRegistry(c *gin.Context) {
	registry, err := h.marketplace.FetchRegistry()
	if err != nil {
		h.logger.Error("Failed to refresh registry", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "刷新插件列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "插件列表已刷新",
		"data": gin.H{
			"version":   registry.Version,
			"updated":   registry.UpdatedAt,
			"count":     len(registry.Plugins),
		},
	})
}


// GetPluginMenus 获取已安装插件的菜单
func (h *MarketplaceHandler) GetPluginMenus(c *gin.Context) {
	menus, err := h.marketplace.GetInstalledPluginMenus()
	if err != nil {
		h.logger.Error("Failed to get plugin menus", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取插件菜单失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"menus": menus,
		},
	})
}
