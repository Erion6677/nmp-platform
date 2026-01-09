package api

import (
	"context"
	"net/http"
	"strconv"

	"nmp-platform/internal/collector"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
)

// InterfaceHandler 接口管理处理器
type InterfaceHandler struct {
	deviceService      service.DeviceService
	dataCleanupService *service.DataCleanupService
	interfaceRepo      repository.InterfaceRepository
	collectorRepo      repository.CollectorRepository
	deviceRepo         repository.DeviceRepository
	pingTargetRepo     repository.PingTargetRepository
	deployer           *collector.Deployer
	generator          *collector.ScriptGenerator
	serverURL          string
}

// NewInterfaceHandler 创建新的接口管理处理器
func NewInterfaceHandler(deviceService service.DeviceService) *InterfaceHandler {
	return &InterfaceHandler{
		deviceService: deviceService,
	}
}

// NewInterfaceHandlerWithCleanup 创建带数据清理服务的接口管理处理器
func NewInterfaceHandlerWithCleanup(deviceService service.DeviceService, dataCleanupService *service.DataCleanupService) *InterfaceHandler {
	return &InterfaceHandler{
		deviceService:      deviceService,
		dataCleanupService: dataCleanupService,
	}
}

// NewInterfaceHandlerFull 创建完整功能的接口管理处理器（支持自动重新部署）
func NewInterfaceHandlerFull(
	deviceService service.DeviceService,
	dataCleanupService *service.DataCleanupService,
	interfaceRepo repository.InterfaceRepository,
	collectorRepo repository.CollectorRepository,
	deviceRepo repository.DeviceRepository,
	pingTargetRepo repository.PingTargetRepository,
	serverURL string,
) *InterfaceHandler {
	return &InterfaceHandler{
		deviceService:      deviceService,
		dataCleanupService: dataCleanupService,
		interfaceRepo:      interfaceRepo,
		collectorRepo:      collectorRepo,
		deviceRepo:         deviceRepo,
		pingTargetRepo:     pingTargetRepo,
		deployer:           collector.NewDeployer(serverURL),
		generator:          collector.NewScriptGenerator(serverURL),
		serverURL:          serverURL,
	}
}

// redeployCollectorScript 自动重新部署采集器脚本
func (h *InterfaceHandler) redeployCollectorScript(deviceID uint) error {
	if h.deployer == nil || h.collectorRepo == nil || h.interfaceRepo == nil {
		return nil
	}

	config, err := h.collectorRepo.GetByDeviceID(deviceID)
	if err != nil || config == nil {
		return nil
	}

	if config.Status != models.CollectorStatusRunning && config.DeployedAt == nil {
		return nil
	}

	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		return err
	}

	interfaces, err := h.interfaceRepo.GetMonitoredByDeviceID(deviceID)
	if err != nil {
		return err
	}

	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	pingTargets, err := h.pingTargetRepo.GetEnabledByDeviceID(deviceID)
	if err != nil {
		return err
	}

	var pingTargetConfigs []collector.PingTargetConfig
	for _, pt := range pingTargets {
		pingTargetConfigs = append(pingTargetConfigs, collector.PingTargetConfig{
			TargetAddress:   pt.TargetAddress,
			TargetName:      pt.TargetName,
			SourceInterface: pt.SourceInterface,
		})
	}

	scriptConfig := &collector.ScriptConfig{
		DeviceID:      deviceID,
		DeviceIP:      device.Host,
		ServerURL:     h.serverURL,
		IntervalMs:    config.IntervalMs,
		PushBatchSize: config.PushBatchSize,
		ScriptName:    config.ScriptName,
		SchedulerName: config.SchedulerName,
		Interfaces:    interfaceNames,
		PingTargets:   pingTargetConfigs,
	}

	h.deployer.DeployToMikroTik(scriptConfig, device.Host, device.APIPort, device.Port, device.Username, device.Password)
	return nil
}

// SyncInterfacesRequest 同步接口请求（可选参数）
type SyncInterfacesRequest struct {
	Force bool `json:"force"` // 是否强制同步（即使已有接口数据）
}

// SetMonitoredInterfacesRequest 设置监控接口请求
type SetMonitoredInterfacesRequest struct {
	InterfaceNames []string `json:"interface_names" binding:"required"`
}

// InterfaceResponse 接口响应
type InterfaceResponse struct {
	ID        uint                   `json:"id"`
	DeviceID  uint                   `json:"device_id"`
	Name      string                 `json:"name"`
	Status    models.InterfaceStatus `json:"status"`
	Monitored bool                   `json:"monitored"`
}

// GetInterfaces 获取设备接口列表
// GET /api/devices/:id/interfaces
func (h *InterfaceHandler) GetInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	interfaces, err := h.deviceService.GetDeviceInterfaces(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 转换为响应格式（只包含名称和状态）
	response := make([]InterfaceResponse, 0, len(interfaces))
	for _, iface := range interfaces {
		response = append(response, InterfaceResponse{
			ID:        iface.ID,
			DeviceID:  iface.DeviceID,
			Name:      iface.Name,
			Status:    iface.Status,
			Monitored: iface.Monitored,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// SyncInterfaces 从设备同步接口列表
// POST /api/devices/:id/interfaces/sync
func (h *InterfaceHandler) SyncInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 从设备同步接口
	interfaces, err := h.deviceService.SyncInterfacesFromDevice(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 转换为响应格式（只包含名称和状态）
	response := make([]InterfaceResponse, 0, len(interfaces))
	for _, iface := range interfaces {
		response = append(response, InterfaceResponse{
			ID:        iface.ID,
			DeviceID:  iface.DeviceID,
			Name:      iface.Name,
			Status:    iface.Status,
			Monitored: iface.Monitored,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "接口同步成功",
	})
}

// SetMonitoredInterfaces 批量设置监控接口
// PUT /api/devices/:id/interfaces/monitored
func (h *InterfaceHandler) SetMonitoredInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	var req SetMonitoredInterfacesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求格式",
			"details": err.Error(),
		})
		return
	}

	// 获取当前监控的接口列表（用于检测被移除的接口）
	currentInterfaces, err := h.deviceService.GetMonitoredInterfaces(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 构建新监控接口名称集合
	newMonitoredSet := make(map[string]bool)
	for _, name := range req.InterfaceNames {
		newMonitoredSet[name] = true
	}

	// 找出被移除监控的接口
	var removedInterfaces []string
	for _, iface := range currentInterfaces {
		if !newMonitoredSet[iface.Name] {
			removedInterfaces = append(removedInterfaces, iface.Name)
		}
	}

	// 设置监控接口
	err = h.deviceService.SetMonitoredInterfaces(uint(deviceID), req.InterfaceNames)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 清理被移除接口的历史数据
	if h.dataCleanupService != nil && len(removedInterfaces) > 0 {
		for _, ifaceName := range removedInterfaces {
			if err := h.dataCleanupService.CleanupInterfaceData(context.Background(), uint(deviceID), ifaceName); err != nil {
				// 记录错误但不中断流程
				c.Writer.Header().Add("X-Cleanup-Warning", err.Error())
			}
		}
	}

	// 返回更新后的接口列表
	interfaces, err := h.deviceService.GetDeviceInterfaces(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 转换为响应格式
	response := make([]InterfaceResponse, 0, len(interfaces))
	for _, iface := range interfaces {
		response = append(response, InterfaceResponse{
			ID:        iface.ID,
			DeviceID:  iface.DeviceID,
			Name:      iface.Name,
			Status:    iface.Status,
			Monitored: iface.Monitored,
		})
	}

	// 自动重新部署采集器脚本
	go h.redeployCollectorScript(uint(deviceID))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "监控接口设置成功",
	})
}

// GetMonitoredInterfaces 获取设备的监控接口列表
// GET /api/devices/:id/interfaces/monitored
func (h *InterfaceHandler) GetMonitoredInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	interfaces, err := h.deviceService.GetMonitoredInterfaces(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 转换为响应格式
	response := make([]InterfaceResponse, 0, len(interfaces))
	for _, iface := range interfaces {
		response = append(response, InterfaceResponse{
			ID:        iface.ID,
			DeviceID:  iface.DeviceID,
			Name:      iface.Name,
			Status:    iface.Status,
			Monitored: iface.Monitored,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// UpdateInterfaceMonitorStatus 更新单个接口的监控状态
// PUT /api/interfaces/:id/monitor
func (h *InterfaceHandler) UpdateInterfaceMonitorStatus(c *gin.Context) {
	idStr := c.Param("id")
	interfaceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的接口ID",
		})
		return
	}

	var req struct {
		Monitored bool `json:"monitored"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求格式",
			"details": err.Error(),
		})
		return
	}

	// 如果是取消监控，需要获取接口信息用于清理数据
	var ifaceInfo *models.Interface
	if !req.Monitored && h.dataCleanupService != nil {
		// 获取接口信息
		interfaces, err := h.deviceService.GetDeviceInterfaces(0) // 需要通过其他方式获取
		if err == nil {
			for _, iface := range interfaces {
				if iface.ID == uint(interfaceID) {
					ifaceInfo = iface
					break
				}
			}
		}
	}

	err = h.deviceService.UpdateInterfaceMonitorStatus(uint(interfaceID), req.Monitored)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 如果取消监控，清理该接口的历史数据
	if !req.Monitored && h.dataCleanupService != nil && ifaceInfo != nil {
		if err := h.dataCleanupService.CleanupInterfaceData(context.Background(), ifaceInfo.DeviceID, ifaceInfo.Name); err != nil {
			c.Writer.Header().Set("X-Cleanup-Warning", err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "接口监控状态更新成功",
	})
}

// RegisterRoutes 注册接口管理相关路由
func (h *InterfaceHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 设备接口管理
	devices := router.Group("/devices")
	{
		devices.GET("/:id/interfaces", h.GetInterfaces)
		devices.POST("/:id/interfaces/sync", h.SyncInterfaces)
		devices.PUT("/:id/interfaces/monitored", h.SetMonitoredInterfaces)
		devices.GET("/:id/interfaces/monitored", h.GetMonitoredInterfaces)
	}

	// 单个接口操作
	interfaces := router.Group("/interfaces")
	{
		interfaces.PUT("/:id/monitor", h.UpdateInterfaceMonitorStatus)
	}
}
