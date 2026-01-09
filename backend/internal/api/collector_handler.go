package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"nmp-platform/internal/collector"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
)

// CollectorHandler 采集器处理器
type CollectorHandler struct {
	collectorRepo      repository.CollectorRepository
	settingsRepo       repository.SettingsRepository
	deviceService      service.DeviceService
	interfaceRepo      repository.InterfaceRepository
	pingTargetRepo     repository.PingTargetRepository
	dataCleanupService *service.DataCleanupService
	deployer           *collector.Deployer
	generator          *collector.ScriptGenerator
	serverURL          string
}

// NewCollectorHandler 创建新的采集器处理器
func NewCollectorHandler(
	collectorRepo repository.CollectorRepository,
	settingsRepo repository.SettingsRepository,
	deviceService service.DeviceService,
	interfaceRepo repository.InterfaceRepository,
	pingTargetRepo repository.PingTargetRepository,
	dataCleanupService *service.DataCleanupService,
	serverURL string,
) *CollectorHandler {
	return &CollectorHandler{
		collectorRepo:      collectorRepo,
		settingsRepo:       settingsRepo,
		deviceService:      deviceService,
		interfaceRepo:      interfaceRepo,
		pingTargetRepo:     pingTargetRepo,
		dataCleanupService: dataCleanupService,
		deployer:           collector.NewDeployer(serverURL),
		generator:          collector.NewScriptGenerator(serverURL),
		serverURL:          serverURL,
	}
}

// UpdateCollectorConfigRequest 更新采集器配置请求
type UpdateCollectorConfigRequest struct {
	IntervalMs    int  `json:"interval_ms"`
	PushBatchSize int  `json:"push_batch_size"`
	Enabled       *bool `json:"enabled"`
}

// GetCollectorConfig 获取设备的采集器配置
func (h *CollectorHandler) GetCollectorConfig(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 获取或创建采集器配置
	config, err := h.collectorRepo.GetOrCreate(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 获取全局默认配置
	defaultInterval := h.settingsRepo.GetDefaultPushInterval()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"config":           config,
			"default_interval": defaultInterval,
		},
	})
}

// UpdateCollectorConfig 更新设备的采集器配置
func (h *CollectorHandler) UpdateCollectorConfig(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	var req UpdateCollectorConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求格式",
			"details": err.Error(),
		})
		return
	}

	// 获取或创建采集器配置
	config, err := h.collectorRepo.GetOrCreate(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 更新配置
	if req.IntervalMs > 0 {
		config.IntervalMs = req.IntervalMs
	}
	if req.PushBatchSize > 0 {
		config.PushBatchSize = req.PushBatchSize
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}

	if err := h.collectorRepo.Update(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    config,
		"message": "采集器配置已更新",
	})
}

// DeployCollector 部署采集器脚本到设备
func (h *CollectorHandler) DeployCollector(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 获取设备信息
	device, err := h.deviceService.GetDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	// 获取或创建采集器配置
	config, err := h.collectorRepo.GetOrCreate(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 获取监控接口列表
	interfaces, err := h.interfaceRepo.GetMonitoredByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取监控接口失败: " + err.Error(),
		})
		return
	}

	// 构建接口名称列表
	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	// 获取启用的 Ping 目标
	pingTargets, err := h.pingTargetRepo.GetEnabledByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取 Ping 目标失败: " + err.Error(),
		})
		return
	}

	// 构建 Ping 目标配置
	var pingTargetConfigs []collector.PingTargetConfig
	for _, pt := range pingTargets {
		pingTargetConfigs = append(pingTargetConfigs, collector.PingTargetConfig{
			TargetAddress:   pt.TargetAddress,
			TargetName:      pt.TargetName,
			SourceInterface: pt.SourceInterface,
		})
	}

	// 构建脚本配置
	scriptConfig := &collector.ScriptConfig{
		DeviceID:      uint(deviceID),
		DeviceIP:      device.Host,
		ServerURL:     h.serverURL,
		IntervalMs:    config.IntervalMs,
		PushBatchSize: config.PushBatchSize,
		ScriptName:    config.ScriptName,
		SchedulerName: config.SchedulerName,
		Interfaces:    interfaceNames,
		PingTargets:   pingTargetConfigs,
	}

	// 部署脚本
	result := h.deployer.DeployToMikroTik(scriptConfig, device.Host, device.APIPort, device.Port, device.Username, device.Password)

	if !result.Success {
		// 更新状态为错误
		h.collectorRepo.UpdateStatus(uint(deviceID), models.CollectorStatusError, result.ErrorMessage)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "部署失败",
			"details": result.ErrorMessage,
			"method":  result.Method,
		})
		return
	}

	// 更新部署状态
	h.collectorRepo.UpdateDeployedAt(uint(deviceID))
	h.collectorRepo.UpdateEnabled(uint(deviceID), true)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"method":  result.Method,
			"message": result.Message,
		},
		"message": "采集器部署成功",
	})
}


// RemoveCollector 从设备移除采集器脚本
func (h *CollectorHandler) RemoveCollector(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 获取设备信息
	device, err := h.deviceService.GetDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	// 获取采集器配置
	config, err := h.collectorRepo.GetByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	scriptName := "nmp-collector"
	schedulerName := "nmp-scheduler"
	if config != nil {
		scriptName = config.ScriptName
		schedulerName = config.SchedulerName
	}

	// 移除脚本
	result := h.deployer.RemoveFromMikroTik(scriptName, schedulerName, device.Host, device.APIPort, device.Port, device.Username, device.Password)

	if !result.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "移除失败",
			"details": result.ErrorMessage,
		})
		return
	}

	// 更新状态
	if config != nil {
		h.collectorRepo.UpdateStatus(uint(deviceID), models.CollectorStatusNotDeployed, "")
		h.collectorRepo.UpdateEnabled(uint(deviceID), false)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"method":  result.Method,
			"message": result.Message,
		},
		"message": "采集器已移除",
	})
}

// ToggleCollector 开启/关闭采集器推送
func (h *CollectorHandler) ToggleCollector(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求格式",
			"details": err.Error(),
		})
		return
	}

	// 获取设备信息
	device, err := h.deviceService.GetDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	// 获取采集器配置
	config, err := h.collectorRepo.GetByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	schedulerName := "nmp-scheduler"
	if config != nil {
		schedulerName = config.SchedulerName
	}

	// 启用或禁用调度器
	var result *collector.DeployResult
	if req.Enabled {
		result = h.deployer.EnableScheduler(schedulerName, device.Host, device.APIPort, device.Port, device.Username, device.Password)
	} else {
		result = h.deployer.DisableScheduler(schedulerName, device.Host, device.APIPort, device.Port, device.Username, device.Password)
	}

	if !result.Success {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "操作失败",
			"details": result.ErrorMessage,
		})
		return
	}

	// 更新状态
	h.collectorRepo.UpdateEnabled(uint(deviceID), req.Enabled)

	message := "采集器已关闭"
	if req.Enabled {
		message = "采集器已开启"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled": req.Enabled,
			"method":  result.Method,
		},
		"message": message,
	})
}

// GetCollectorStatus 获取采集器状态
func (h *CollectorHandler) GetCollectorStatus(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 获取设备信息
	device, err := h.deviceService.GetDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	// 获取采集器配置
	config, err := h.collectorRepo.GetByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	scriptName := "nmp-collector"
	schedulerName := "nmp-scheduler"
	if config != nil {
		scriptName = config.ScriptName
		schedulerName = config.SchedulerName
	}

	// 从设备获取实际状态
	status, err := h.deployer.GetScriptStatus(scriptName, schedulerName, device.Host, device.APIPort, device.Port, device.Username, device.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取状态失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"config":        config,
			"device_status": status,
		},
	})
}

// PreviewScript 预览生成的脚本
func (h *CollectorHandler) PreviewScript(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 获取设备信息
	device, err := h.deviceService.GetDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	// 获取或创建采集器配置
	config, err := h.collectorRepo.GetOrCreate(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 获取监控接口列表
	interfaces, err := h.interfaceRepo.GetMonitoredByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取监控接口失败: " + err.Error(),
		})
		return
	}

	// 构建接口名称列表
	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	// 获取启用的 Ping 目标
	pingTargets, err := h.pingTargetRepo.GetEnabledByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取 Ping 目标失败: " + err.Error(),
		})
		return
	}

	// 构建 Ping 目标配置
	var pingTargetConfigs []collector.PingTargetConfig
	for _, pt := range pingTargets {
		pingTargetConfigs = append(pingTargetConfigs, collector.PingTargetConfig{
			TargetAddress:   pt.TargetAddress,
			TargetName:      pt.TargetName,
			SourceInterface: pt.SourceInterface,
		})
	}

	// 构建脚本配置
	scriptConfig := &collector.ScriptConfig{
		DeviceID:      uint(deviceID),
		DeviceIP:      device.Host,
		ServerURL:     h.serverURL,
		IntervalMs:    config.IntervalMs,
		PushBatchSize: config.PushBatchSize,
		ScriptName:    config.ScriptName,
		SchedulerName: config.SchedulerName,
		Interfaces:    interfaceNames,
		PingTargets:   pingTargetConfigs,
	}

	// 生成脚本
	script := h.generator.GenerateMikroTikScript(scriptConfig)
	deployCommands := h.generator.GenerateDeployCommands(scriptConfig)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"script":          script,
			"deploy_commands": deployCommands,
			"config":          scriptConfig,
		},
	})
}

// ClearDeviceData 清除设备的所有采集数据
// 只清理指定设备的数据，不影响其他设备
// Requirements: 8.3
func (h *CollectorHandler) ClearDeviceData(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 验证设备是否存在
	device, err := h.deviceService.GetDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	// 检查 DataCleanupService 是否可用
	if h.dataCleanupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "数据清理服务不可用",
		})
		return
	}

	// 创建带超时的 context
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// 清理设备数据
	if err := h.dataCleanupService.CleanupDeviceData(ctx, uint(deviceID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "清理数据失败",
			"details": err.Error(),
		})
		return
	}

	// 禁用采集器推送
	config, err := h.collectorRepo.GetByDeviceID(uint(deviceID))
	if err == nil && config != nil {
		h.collectorRepo.UpdateEnabled(uint(deviceID), false)
		h.collectorRepo.UpdateStatus(uint(deviceID), models.CollectorStatusNotDeployed, "")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"device_id":   deviceID,
			"device_name": device.Name,
			"device_ip":   device.Host,
		},
		"message": "设备数据已清除，采集器已禁用",
	})
}

// RegisterRoutes 注册采集器相关路由
func (h *CollectorHandler) RegisterRoutes(router *gin.RouterGroup) {
	collector := router.Group("/devices/:id/collector")
	{
		collector.GET("", h.GetCollectorConfig)
		collector.PUT("", h.UpdateCollectorConfig)
		collector.POST("/deploy", h.DeployCollector)
		collector.DELETE("/deploy", h.RemoveCollector)
		collector.POST("/toggle", h.ToggleCollector)
		collector.POST("/clear", h.ClearDeviceData)
		collector.GET("/status", h.GetCollectorStatus)
		collector.GET("/preview", h.PreviewScript)
	}
}

// RegisterRoutesWithPermission 注册采集器相关路由（带权限检查）
// readMiddleware: 读取权限中间件
// updateMiddleware: 更新权限中间件
func (h *CollectorHandler) RegisterRoutesWithPermission(router *gin.RouterGroup, readMiddleware, updateMiddleware gin.HandlerFunc) {
	collector := router.Group("/devices/:id/collector")
	{
		// 获取采集器配置 - 需要设备级别读取权限
		collector.GET("", readMiddleware, h.GetCollectorConfig)
		// 更新采集器配置 - 需要设备级别更新权限
		collector.PUT("", updateMiddleware, h.UpdateCollectorConfig)
		// 部署采集器 - 需要设备级别更新权限
		collector.POST("/deploy", updateMiddleware, h.DeployCollector)
		// 移除采集器 - 需要设备级别更新权限
		collector.DELETE("/deploy", updateMiddleware, h.RemoveCollector)
		// 开启/关闭采集器 - 需要设备级别更新权限
		collector.POST("/toggle", updateMiddleware, h.ToggleCollector)
		// 清除设备数据 - 需要设备级别更新权限
		collector.POST("/clear", updateMiddleware, h.ClearDeviceData)
		// 获取采集器状态 - 需要设备级别读取权限
		collector.GET("/status", readMiddleware, h.GetCollectorStatus)
		// 预览脚本 - 需要设备级别读取权限
		collector.GET("/preview", readMiddleware, h.PreviewScript)
	}
}