package api

import (
	"net/http"

	"nmp-platform/internal/repository"

	"github.com/gin-gonic/gin"
)

// SettingsHandler 系统设置处理器
type SettingsHandler struct {
	settingsRepo repository.SettingsRepository
}

// NewSettingsHandler 创建新的系统设置处理器
func NewSettingsHandler(settingsRepo repository.SettingsRepository) *SettingsHandler {
	return &SettingsHandler{
		settingsRepo: settingsRepo,
	}
}

// CollectionSettingsRequest 采集设置请求
type CollectionSettingsRequest struct {
	DefaultPushInterval     *int  `json:"default_push_interval"`       // 默认推送间隔（毫秒）
	DataRetentionDays       *int  `json:"data_retention_days"`         // 数据保留天数
	FrontendRefreshInterval *int  `json:"frontend_refresh_interval"`   // 前端刷新间隔（秒），0 表示跟随推送间隔
	DeviceOfflineTimeout    *int  `json:"device_offline_timeout"`      // 设备离线超时（秒）
	FollowPushInterval      *bool `json:"follow_push_interval"`        // 前端刷新是否跟随推送间隔
}

// CollectionSettingsResponse 采集设置响应
type CollectionSettingsResponse struct {
	DefaultPushInterval     int  `json:"default_push_interval"`
	DataRetentionDays       int  `json:"data_retention_days"`
	FrontendRefreshInterval int  `json:"frontend_refresh_interval"`
	DeviceOfflineTimeout    int  `json:"device_offline_timeout"`
	FollowPushInterval      bool `json:"follow_push_interval"`         // 前端刷新是否跟随推送间隔
}

// GetCollectionSettings 获取采集设置
func (h *SettingsHandler) GetCollectionSettings(c *gin.Context) {
	settings := CollectionSettingsResponse{
		DefaultPushInterval:     h.settingsRepo.GetDefaultPushInterval(),
		DataRetentionDays:       h.settingsRepo.GetDataRetentionDays(),
		FrontendRefreshInterval: h.settingsRepo.GetFrontendRefreshInterval(),
		DeviceOfflineTimeout:    h.settingsRepo.GetDeviceOfflineTimeout(),
		FollowPushInterval:      h.settingsRepo.GetFollowPushInterval(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
	})
}

// UpdateCollectionSettings 更新采集设置
func (h *SettingsHandler) UpdateCollectionSettings(c *gin.Context) {
	var req CollectionSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 更新各项设置
	if req.DefaultPushInterval != nil {
		if *req.DefaultPushInterval < 100 {
			BadRequest(c, "推送间隔不能小于 100 毫秒")
			return
		}
		if err := h.settingsRepo.SetDefaultPushInterval(*req.DefaultPushInterval); err != nil {
			InternalError(c, "更新推送间隔失败: "+err.Error())
			return
		}
	}

	if req.DataRetentionDays != nil {
		if *req.DataRetentionDays < 1 {
			BadRequest(c, "数据保留天数不能小于 1 天")
			return
		}
		if err := h.settingsRepo.SetDataRetentionDays(*req.DataRetentionDays); err != nil {
			InternalError(c, "更新数据保留天数失败: "+err.Error())
			return
		}
	}

	if req.FrontendRefreshInterval != nil {
		if *req.FrontendRefreshInterval < 3 || *req.FrontendRefreshInterval > 60 {
			BadRequest(c, "前端刷新间隔必须在 3-60 秒之间")
			return
		}
		if err := h.settingsRepo.SetFrontendRefreshInterval(*req.FrontendRefreshInterval); err != nil {
			InternalError(c, "更新前端刷新间隔失败: "+err.Error())
			return
		}
	}

	if req.DeviceOfflineTimeout != nil {
		if *req.DeviceOfflineTimeout < 10 {
			BadRequest(c, "设备离线超时不能小于 10 秒")
			return
		}
		if err := h.settingsRepo.SetDeviceOfflineTimeout(*req.DeviceOfflineTimeout); err != nil {
			InternalError(c, "更新设备离线超时失败: "+err.Error())
			return
		}
	}

	if req.FollowPushInterval != nil {
		if err := h.settingsRepo.SetFollowPushInterval(*req.FollowPushInterval); err != nil {
			InternalError(c, "更新跟随推送间隔设置失败: "+err.Error())
			return
		}
	}

	// 返回更新后的设置
	settings := CollectionSettingsResponse{
		DefaultPushInterval:     h.settingsRepo.GetDefaultPushInterval(),
		DataRetentionDays:       h.settingsRepo.GetDataRetentionDays(),
		FrontendRefreshInterval: h.settingsRepo.GetFrontendRefreshInterval(),
		DeviceOfflineTimeout:    h.settingsRepo.GetDeviceOfflineTimeout(),
		FollowPushInterval:      h.settingsRepo.GetFollowPushInterval(),
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    settings,
		"message": "采集设置已更新",
	})
}

// GetAllSettings 获取所有系统设置
func (h *SettingsHandler) GetAllSettings(c *gin.Context) {
	settings, err := h.settingsRepo.GetAll()
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, settings)
}

// InitDefaults 初始化默认设置
func (h *SettingsHandler) InitDefaults(c *gin.Context) {
	if err := h.settingsRepo.InitDefaults(); err != nil {
		InternalError(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "默认设置已初始化")
}

// RegisterRoutes 注册系统设置相关路由
func (h *SettingsHandler) RegisterRoutes(router *gin.RouterGroup) {
	settings := router.Group("/settings")
	{
		settings.GET("/collection", h.GetCollectionSettings)
		settings.PUT("/collection", h.UpdateCollectionSettings)
		settings.GET("/all", h.GetAllSettings)
		settings.POST("/init", h.InitDefaults)
	}
}
