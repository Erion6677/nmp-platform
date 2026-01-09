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

// PingTargetHandler Ping 目标管理处理器
type PingTargetHandler struct {
	pingTargetRepo     repository.PingTargetRepository
	deviceRepo         repository.DeviceRepository
	interfaceRepo      repository.InterfaceRepository
	collectorRepo      repository.CollectorRepository
	dataCleanupService *service.DataCleanupService
	deployer           *collector.Deployer
	generator          *collector.ScriptGenerator
	serverURL          string
}

// NewPingTargetHandler 创建新的 Ping 目标管理处理器
func NewPingTargetHandler(pingTargetRepo repository.PingTargetRepository, deviceRepo repository.DeviceRepository) *PingTargetHandler {
	return &PingTargetHandler{
		pingTargetRepo: pingTargetRepo,
		deviceRepo:     deviceRepo,
	}
}

// NewPingTargetHandlerWithCleanup 创建带数据清理服务的 Ping 目标管理处理器
func NewPingTargetHandlerWithCleanup(pingTargetRepo repository.PingTargetRepository, deviceRepo repository.DeviceRepository, dataCleanupService *service.DataCleanupService) *PingTargetHandler {
	return &PingTargetHandler{
		pingTargetRepo:     pingTargetRepo,
		deviceRepo:         deviceRepo,
		dataCleanupService: dataCleanupService,
	}
}

// NewPingTargetHandlerFull 创建完整功能的 Ping 目标管理处理器（支持自动重新部署）
func NewPingTargetHandlerFull(
	pingTargetRepo repository.PingTargetRepository,
	deviceRepo repository.DeviceRepository,
	interfaceRepo repository.InterfaceRepository,
	collectorRepo repository.CollectorRepository,
	dataCleanupService *service.DataCleanupService,
	serverURL string,
) *PingTargetHandler {
	return &PingTargetHandler{
		pingTargetRepo:     pingTargetRepo,
		deviceRepo:         deviceRepo,
		interfaceRepo:      interfaceRepo,
		collectorRepo:      collectorRepo,
		dataCleanupService: dataCleanupService,
		deployer:           collector.NewDeployer(serverURL),
		generator:          collector.NewScriptGenerator(serverURL),
		serverURL:          serverURL,
	}
}

// CreatePingTargetRequest 创建 Ping 目标请求
type CreatePingTargetRequest struct {
	TargetAddress   string `json:"target_address" binding:"required"`
	TargetName      string `json:"target_name" binding:"required"`
	SourceInterface string `json:"source_interface"`
	Enabled         *bool  `json:"enabled"`
}

// UpdatePingTargetRequest 更新 Ping 目标请求
type UpdatePingTargetRequest struct {
	TargetAddress   string `json:"target_address"`
	TargetName      string `json:"target_name"`
	SourceInterface string `json:"source_interface"`
	Enabled         *bool  `json:"enabled"`
}

// PingTargetResponse Ping 目标响应
type PingTargetResponse struct {
	ID              uint   `json:"id"`
	DeviceID        uint   `json:"device_id"`
	TargetAddress   string `json:"target_address"`
	TargetName      string `json:"target_name"`
	SourceInterface string `json:"source_interface"`
	Enabled         bool   `json:"enabled"`
}

// redeployCollectorScript 自动重新部署采集器脚本
// 当 ping 目标发生变化时调用
func (h *PingTargetHandler) redeployCollectorScript(deviceID uint) error {
	// 检查是否有部署能力
	if h.deployer == nil || h.collectorRepo == nil || h.interfaceRepo == nil {
		return nil // 没有部署能力，跳过
	}

	// 获取采集器配置
	config, err := h.collectorRepo.GetByDeviceID(deviceID)
	if err != nil || config == nil {
		return nil // 没有采集器配置，跳过
	}

	// 检查采集器是否已部署
	if config.Status != models.CollectorStatusRunning && config.DeployedAt == nil {
		return nil // 未部署，跳过
	}

	// 获取设备信息
	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		return err
	}

	// 获取监控接口列表
	interfaces, err := h.interfaceRepo.GetMonitoredByDeviceID(deviceID)
	if err != nil {
		return err
	}

	var interfaceNames []string
	for _, iface := range interfaces {
		interfaceNames = append(interfaceNames, iface.Name)
	}

	// 获取启用的 Ping 目标
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

	// 构建脚本配置
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

	// 重新部署脚本
	result := h.deployer.DeployToMikroTik(scriptConfig, device.Host, device.APIPort, device.Port, device.Username, device.Password)
	if !result.Success {
		return nil // 部署失败不影响主流程
	}

	return nil
}

// GetPingTargets 获取设备的 Ping 目标列表
// GET /api/devices/:id/ping-targets
func (h *PingTargetHandler) GetPingTargets(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 检查设备是否存在
	_, err = h.deviceRepo.GetByID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	targets, err := h.pingTargetRepo.GetByDeviceID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 转换为响应格式
	response := make([]PingTargetResponse, 0, len(targets))
	for _, target := range targets {
		response = append(response, PingTargetResponse{
			ID:              target.ID,
			DeviceID:        target.DeviceID,
			TargetAddress:   target.TargetAddress,
			TargetName:      target.TargetName,
			SourceInterface: target.SourceInterface,
			Enabled:         target.Enabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// CreatePingTarget 创建 Ping 目标
// POST /api/devices/:id/ping-targets
func (h *PingTargetHandler) CreatePingTarget(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	// 检查设备是否存在
	_, err = h.deviceRepo.GetByID(uint(deviceID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "设备不存在",
		})
		return
	}

	var req CreatePingTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求格式",
			"details": err.Error(),
		})
		return
	}

	// 检查是否已存在相同目标地址和源接口的组合
	// 重复判断条件：同设备 + 同目标IP + 同源接口
	exists, err := h.pingTargetRepo.Exists(uint(deviceID), req.TargetAddress, req.SourceInterface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	if exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "该设备已存在相同目标地址和源接口的 Ping 目标",
		})
		return
	}

	// 创建 Ping 目标
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	target := &models.PingTarget{
		DeviceID:        uint(deviceID),
		TargetAddress:   req.TargetAddress,
		TargetName:      req.TargetName,
		SourceInterface: req.SourceInterface,
		Enabled:         enabled,
	}

	if err := h.pingTargetRepo.Create(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 自动重新部署采集器脚本
	go h.redeployCollectorScript(uint(deviceID))

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": PingTargetResponse{
			ID:              target.ID,
			DeviceID:        target.DeviceID,
			TargetAddress:   target.TargetAddress,
			TargetName:      target.TargetName,
			SourceInterface: target.SourceInterface,
			Enabled:         target.Enabled,
		},
		"message": "Ping 目标创建成功",
	})
}

// UpdatePingTarget 更新 Ping 目标
// PUT /api/devices/:id/ping-targets/:tid
func (h *PingTargetHandler) UpdatePingTarget(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	tidStr := c.Param("tid")
	targetID, err := strconv.ParseUint(tidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的目标ID",
		})
		return
	}

	// 获取现有目标
	target, err := h.pingTargetRepo.GetByID(uint(targetID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ping 目标不存在",
		})
		return
	}

	// 验证目标属于指定设备
	if target.DeviceID != uint(deviceID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ping 目标不属于该设备",
		})
		return
	}

	var req UpdatePingTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求格式",
			"details": err.Error(),
		})
		return
	}

	// 如果更新目标地址或源接口，检查是否与其他目标冲突
	newTargetAddress := target.TargetAddress
	newSourceInterface := target.SourceInterface
	
	if req.TargetAddress != "" {
		newTargetAddress = req.TargetAddress
	}
	if req.SourceInterface != "" || c.Request.ContentLength > 0 {
		newSourceInterface = req.SourceInterface
	}
	
	// 只有当目标地址或源接口发生变化时才检查重复
	if newTargetAddress != target.TargetAddress || newSourceInterface != target.SourceInterface {
		exists, err := h.pingTargetRepo.Exists(uint(deviceID), newTargetAddress, newSourceInterface)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		if exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "该设备已存在相同目标地址和源接口的 Ping 目标",
			})
			return
		}
	}
	
	target.TargetAddress = newTargetAddress
	target.SourceInterface = newSourceInterface

	// 更新字段
	if req.TargetName != "" {
		target.TargetName = req.TargetName
	}
	if req.Enabled != nil {
		target.Enabled = *req.Enabled
	}

	if err := h.pingTargetRepo.Update(target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 自动重新部署采集器脚本
	go h.redeployCollectorScript(uint(deviceID))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": PingTargetResponse{
			ID:              target.ID,
			DeviceID:        target.DeviceID,
			TargetAddress:   target.TargetAddress,
			TargetName:      target.TargetName,
			SourceInterface: target.SourceInterface,
			Enabled:         target.Enabled,
		},
		"message": "Ping 目标更新成功",
	})
}

// DeletePingTarget 删除 Ping 目标
// DELETE /api/devices/:id/ping-targets/:tid
func (h *PingTargetHandler) DeletePingTarget(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	tidStr := c.Param("tid")
	targetID, err := strconv.ParseUint(tidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的目标ID",
		})
		return
	}

	// 获取现有目标
	target, err := h.pingTargetRepo.GetByID(uint(targetID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ping 目标不存在",
		})
		return
	}

	// 验证目标属于指定设备
	if target.DeviceID != uint(deviceID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ping 目标不属于该设备",
		})
		return
	}

	// 删除 InfluxDB 中的历史数据
	if h.dataCleanupService != nil {
		if err := h.dataCleanupService.CleanupPingTargetData(context.Background(), uint(deviceID), target.TargetAddress, target.SourceInterface); err != nil {
			// 记录错误但不中断删除流程
			c.Writer.Header().Set("X-Cleanup-Warning", err.Error())
		}
	}

	if err := h.pingTargetRepo.Delete(uint(targetID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 自动重新部署采集器脚本
	go h.redeployCollectorScript(uint(deviceID))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Ping 目标及其历史数据已删除",
	})
}

// TogglePingTarget 切换 Ping 目标启用状态
// PUT /api/devices/:id/ping-targets/:tid/toggle
func (h *PingTargetHandler) TogglePingTarget(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的设备ID",
		})
		return
	}

	tidStr := c.Param("tid")
	targetID, err := strconv.ParseUint(tidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的目标ID",
		})
		return
	}

	// 获取现有目标
	target, err := h.pingTargetRepo.GetByID(uint(targetID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ping 目标不存在",
		})
		return
	}

	// 验证目标属于指定设备
	if target.DeviceID != uint(deviceID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ping 目标不属于该设备",
		})
		return
	}

	// 切换启用状态
	newEnabled := !target.Enabled
	if err := h.pingTargetRepo.UpdateEnabled(uint(targetID), newEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 自动重新部署采集器脚本
	go h.redeployCollectorScript(uint(deviceID))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"enabled": newEnabled,
		},
		"message": "Ping 目标状态切换成功",
	})
}

// RegisterRoutes 注册 Ping 目标管理相关路由
func (h *PingTargetHandler) RegisterRoutes(router *gin.RouterGroup) {
	devices := router.Group("/devices")
	{
		devices.GET("/:id/ping-targets", h.GetPingTargets)
		devices.POST("/:id/ping-targets", h.CreatePingTarget)
		devices.PUT("/:id/ping-targets/:tid", h.UpdatePingTarget)
		devices.DELETE("/:id/ping-targets/:tid", h.DeletePingTarget)
		devices.PUT("/:id/ping-targets/:tid/toggle", h.TogglePingTarget)
	}
}

// RegisterRoutesWithPermission 注册 Ping 目标管理相关路由（带权限检查）
// readMiddleware: 读取权限中间件
// updateMiddleware: 更新权限中间件
func (h *PingTargetHandler) RegisterRoutesWithPermission(router *gin.RouterGroup, readMiddleware, updateMiddleware gin.HandlerFunc) {
	devices := router.Group("/devices")
	{
		// 获取 Ping 目标列表 - 需要设备级别读取权限
		devices.GET("/:id/ping-targets", readMiddleware, h.GetPingTargets)
		// 创建 Ping 目标 - 需要设备级别更新权限
		devices.POST("/:id/ping-targets", updateMiddleware, h.CreatePingTarget)
		// 更新 Ping 目标 - 需要设备级别更新权限
		devices.PUT("/:id/ping-targets/:tid", updateMiddleware, h.UpdatePingTarget)
		// 删除 Ping 目标 - 需要设备级别更新权限
		devices.DELETE("/:id/ping-targets/:tid", updateMiddleware, h.DeletePingTarget)
		// 切换 Ping 目标状态 - 需要设备级别更新权限
		devices.PUT("/:id/ping-targets/:tid/toggle", updateMiddleware, h.TogglePingTarget)
	}
}