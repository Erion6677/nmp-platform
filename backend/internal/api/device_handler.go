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

// DeviceHandler 设备处理器
type DeviceHandler struct {
	deviceService      service.DeviceService
	tagService         service.TagService
	deviceGroupService service.DeviceGroupService
	dataCleanupService *service.DataCleanupService
	// 用于自动重新部署采集器脚本
	interfaceRepo  repository.InterfaceRepository
	collectorRepo  repository.CollectorRepository
	deviceRepo     repository.DeviceRepository
	pingTargetRepo repository.PingTargetRepository
	deployer       *collector.Deployer
	serverURL      string
}

// NewDeviceHandler 创建新的设备处理器
func NewDeviceHandler(
	deviceService service.DeviceService,
	tagService service.TagService,
	deviceGroupService service.DeviceGroupService,
) *DeviceHandler {
	return &DeviceHandler{
		deviceService:      deviceService,
		tagService:         tagService,
		deviceGroupService: deviceGroupService,
	}
}

// NewDeviceHandlerWithCleanup 创建带数据清理服务的设备处理器
func NewDeviceHandlerWithCleanup(
	deviceService service.DeviceService,
	tagService service.TagService,
	deviceGroupService service.DeviceGroupService,
	dataCleanupService *service.DataCleanupService,
) *DeviceHandler {
	return &DeviceHandler{
		deviceService:      deviceService,
		tagService:         tagService,
		deviceGroupService: deviceGroupService,
		dataCleanupService: dataCleanupService,
	}
}

// NewDeviceHandlerFull 创建完整功能的设备处理器（支持自动重新部署）
func NewDeviceHandlerFull(
	deviceService service.DeviceService,
	tagService service.TagService,
	deviceGroupService service.DeviceGroupService,
	dataCleanupService *service.DataCleanupService,
	interfaceRepo repository.InterfaceRepository,
	collectorRepo repository.CollectorRepository,
	deviceRepo repository.DeviceRepository,
	pingTargetRepo repository.PingTargetRepository,
	serverURL string,
) *DeviceHandler {
	return &DeviceHandler{
		deviceService:      deviceService,
		tagService:         tagService,
		deviceGroupService: deviceGroupService,
		dataCleanupService: dataCleanupService,
		interfaceRepo:      interfaceRepo,
		collectorRepo:      collectorRepo,
		deviceRepo:         deviceRepo,
		pingTargetRepo:     pingTargetRepo,
		deployer:           collector.NewDeployer(serverURL),
		serverURL:          serverURL,
	}
}

// redeployCollectorScript 自动重新部署采集器脚本
func (h *DeviceHandler) redeployCollectorScript(deviceID uint) error {
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

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	Name        string             `json:"name" binding:"required"`
	Type        models.DeviceType  `json:"type" binding:"required"`
	OSType      models.DeviceOSType `json:"os_type"`
	Host        string             `json:"host" binding:"required"`
	Port        int                `json:"port"`
	APIPort     int                `json:"api_port"`
	Protocol    string             `json:"protocol"`
	Username    string             `json:"username" binding:"required"`
	Password    string             `json:"password" binding:"required"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	ProxyID     *uint              `json:"proxy_id"`
	GroupIDs    []uint             `json:"group_ids"`
	TagIDs      []uint             `json:"tag_ids"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name        string             `json:"name" binding:"required"`
	Type        models.DeviceType  `json:"type" binding:"required"`
	OSType      models.DeviceOSType `json:"os_type"`
	Host        string             `json:"host" binding:"required"`
	Port        int                `json:"port"`
	APIPort     int                `json:"api_port"`
	Protocol    string             `json:"protocol"`
	Username    string             `json:"username"`
	Password    string             `json:"password"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	ProxyID     *uint              `json:"proxy_id"`
}

// TestConnectionRequest 连接测试请求
type TestConnectionRequest struct {
	// 可以直接传设备信息进行测试（不需要先保存设备）
	Name           string              `json:"name"`
	Host           string              `json:"host" binding:"required"`
	Port           int                 `json:"port"`
	APIPort        int                 `json:"api_port"`
	Username       string              `json:"username" binding:"required"`
	Password       string              `json:"password" binding:"required"`
	OSType         models.DeviceOSType `json:"os_type" binding:"required"`
	ConnectionType string              `json:"connection_type"` // api, ssh, all（默认 all）
}

// ListDevicesRequest 设备列表请求
type ListDevicesRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Type     string `form:"type"`
	Status   string `form:"status"`
	GroupID  uint   `form:"group_id"`
	TagID    uint   `form:"tag_id"`
	Search   string `form:"search"`
}

// CreateDevice 创建设备
func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 验证必填字段
	if req.Name == "" {
		BadRequest(c, "设备名称不能为空")
		return
	}
	if req.Host == "" {
		BadRequest(c, "设备IP地址不能为空")
		return
	}
	if req.Username == "" {
		BadRequest(c, "用户名不能为空")
		return
	}
	if req.Password == "" {
		BadRequest(c, "密码不能为空")
		return
	}

	// 设置默认 OS 类型
	osType := req.OSType
	if osType == "" {
		osType = models.DeviceOSTypeMikroTik
	}

	// 创建设备对象
	device := &models.Device{
		Name:        req.Name,
		Type:        req.Type,
		OSType:      osType,
		Host:        req.Host,
		Port:        req.Port,
		APIPort:     req.APIPort,
		Protocol:    req.Protocol,
		Username:    req.Username,
		Password:    req.Password,
		Version:     req.Version,
		Description: req.Description,
		ProxyID:     req.ProxyID,
		Status:      models.DeviceStatusUnknown,
	}

	// 设置默认值
	if device.Port == 0 {
		device.Port = 22
	}
	if device.APIPort == 0 && device.OSType == models.DeviceOSTypeMikroTik {
		device.APIPort = 8728
	}
	if device.Protocol == "" {
		device.Protocol = "ssh"
	}

	// 创建设备
	err := h.deviceService.CreateDevice(device)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 添加到分组
	for _, groupID := range req.GroupIDs {
		if err := h.deviceService.AddDeviceToGroup(device.ID, groupID); err != nil {
			// 记录错误但不中断流程
			continue
		}
	}

	// 添加标签
	for _, tagID := range req.TagIDs {
		if err := h.deviceService.AddDeviceTag(device.ID, tagID); err != nil {
			// 记录错误但不中断流程
			continue
		}
	}

	// 重新获取设备信息（包含关联数据）
	createdDevice, err := h.deviceService.GetDevice(device.ID)
	if err != nil {
		InternalError(c, "获取创建的设备失败")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    createdDevice,
	})
}

// GetDevice 获取设备详情
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	device, err := h.deviceService.GetDevice(uint(id))
	if err != nil {
		NotFound(c, err.Error())
		return
	}

	Success(c, device)
}

// UpdateDevice 更新设备
func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	var req UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 获取现有设备
	device, err := h.deviceService.GetDevice(uint(id))
	if err != nil {
		NotFound(c, err.Error())
		return
	}

	// 更新设备信息
	device.Name = req.Name
	device.Type = req.Type
	if req.OSType != "" {
		device.OSType = req.OSType
	}
	device.Host = req.Host
	if req.Port > 0 {
		device.Port = req.Port
	}
	if req.APIPort > 0 {
		device.APIPort = req.APIPort
	}
	device.Protocol = req.Protocol
	if req.Username != "" {
		device.Username = req.Username
	}
	if req.Password != "" {
		device.Password = req.Password
	}
	device.Version = req.Version
	device.Description = req.Description
	device.ProxyID = req.ProxyID

	err = h.deviceService.UpdateDevice(device)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, device)
}

// DeleteDevice 删除设备
func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	// 先清理 InfluxDB 中的监控数据
	if h.dataCleanupService != nil {
		if err := h.dataCleanupService.CleanupDeviceData(context.Background(), uint(id)); err != nil {
			// 记录错误但不中断删除流程
			c.Writer.Header().Set("X-Cleanup-Warning", err.Error())
		}
	}

	err = h.deviceService.DeleteDevice(uint(id))
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "设备及其所有监控数据已删除")
}

// ListDevices 获取设备列表
func (h *DeviceHandler) ListDevices(c *gin.Context) {
	var req ListDevicesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "无效的查询参数")
		return
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize
	if offset < 0 {
		offset = 0
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if req.Type != "" {
		filters["type"] = models.DeviceType(req.Type)
	}
	if req.Status != "" {
		filters["status"] = models.DeviceStatus(req.Status)
	}
	if req.GroupID > 0 {
		filters["group_id"] = req.GroupID
	}
	if req.TagID > 0 {
		filters["tag_id"] = req.TagID
	}
	if req.Search != "" {
		filters["search"] = req.Search
	}

	devices, total, err := h.deviceService.ListDevices(offset, req.PageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 计算分页信息
	totalPages := (int(total) + req.PageSize - 1) / req.PageSize

	Success(c, gin.H{
		"devices":     devices,
		"total":       total,
		"page":        req.Page,
		"page_size":   req.PageSize,
		"total_pages": totalPages,
	})
}

// UpdateDeviceStatus 更新设备状态
func (h *DeviceHandler) UpdateDeviceStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	var req struct {
		Status models.DeviceStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	err = h.deviceService.UpdateDeviceStatus(uint(id), req.Status)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "设备状态更新成功")
}

// AddDeviceToGroup 将设备添加到分组
func (h *DeviceHandler) AddDeviceToGroup(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	var req struct {
		GroupID uint `json:"group_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	err = h.deviceService.AddDeviceToGroup(uint(deviceID), req.GroupID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "设备已添加到分组")
}

// RemoveDeviceFromGroup 从分组中移除设备
func (h *DeviceHandler) RemoveDeviceFromGroup(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	groupIDStr := c.Param("group_id")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的分组 ID")
		return
	}

	err = h.deviceService.RemoveDeviceFromGroup(uint(deviceID), uint(groupID))
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "设备已从分组移除")
}

// AddDeviceTag 为设备添加标签
func (h *DeviceHandler) AddDeviceTag(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	var req struct {
		TagID uint `json:"tag_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	err = h.deviceService.AddDeviceTag(uint(deviceID), req.TagID)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "标签已添加到设备")
}

// RemoveDeviceTag 移除设备标签
func (h *DeviceHandler) RemoveDeviceTag(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的标签 ID")
		return
	}

	err = h.deviceService.RemoveDeviceTag(uint(deviceID), uint(tagID))
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "标签已从设备移除")
}

// GetDeviceInterfaces 获取设备接口列表
func (h *DeviceHandler) GetDeviceInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	deviceID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	interfaces, err := h.deviceService.GetDeviceInterfaces(uint(deviceID))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, interfaces)
}

// UpdateInterfaceMonitorStatus 更新接口监控状态
func (h *DeviceHandler) UpdateInterfaceMonitorStatus(c *gin.Context) {
	interfaceIDStr := c.Param("interface_id")
	interfaceID, err := strconv.ParseUint(interfaceIDStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的接口 ID")
		return
	}

	var req struct {
		Monitor bool `json:"monitor"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	err = h.deviceService.UpdateInterfaceMonitorStatus(uint(interfaceID), req.Monitor)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "接口监控状态更新成功")
}

// TestConnection 测试设备连接
func (h *DeviceHandler) TestConnection(c *gin.Context) {
	var req TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 设置默认值
	if req.Port == 0 {
		req.Port = 22
	}
	if req.APIPort == 0 && req.OSType == models.DeviceOSTypeMikroTik {
		req.APIPort = 8728
	}
	if req.ConnectionType == "" {
		req.ConnectionType = "all"
	}

	// 创建临时设备对象用于测试
	device := &models.Device{
		Name:     req.Name,
		Host:     req.Host,
		Port:     req.Port,
		APIPort:  req.APIPort,
		Username: req.Username,
		Password: req.Password,
		OSType:   req.OSType,
	}

	result, err := h.deviceService.TestConnection(device, req.ConnectionType)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// TestConnectionByID 测试已保存设备的连接
func (h *DeviceHandler) TestConnectionByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	// 获取连接类型参数
	connectionType := c.DefaultQuery("type", "all")

	device, err := h.deviceService.GetDevice(uint(id))
	if err != nil {
		NotFound(c, err.Error())
		return
	}

	result, err := h.deviceService.TestConnection(device, connectionType)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, result)
}

// GetSystemInfo 获取设备系统信息（主动采集）
func (h *DeviceHandler) GetSystemInfo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	info, err := h.deviceService.GetSystemInfo(uint(id))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, info)
}

// SyncInterfaces 从设备同步接口列表
func (h *DeviceHandler) SyncInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	interfaces, err := h.deviceService.SyncInterfacesFromDevice(uint(id))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, interfaces)
}

// SetMonitoredInterfaces 批量设置监控接口
func (h *DeviceHandler) SetMonitoredInterfaces(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的设备 ID")
		return
	}

	var req struct {
		InterfaceNames []string `json:"interface_names"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 获取当前监控的接口列表（用于检测被移除的接口）
	currentInterfaces, err := h.deviceService.GetMonitoredInterfaces(uint(id))
	if err != nil {
		InternalError(c, err.Error())
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

	err = h.deviceService.SetMonitoredInterfaces(uint(id), req.InterfaceNames)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 清理被移除接口的历史数据
	if h.dataCleanupService != nil && len(removedInterfaces) > 0 {
		for _, ifaceName := range removedInterfaces {
			if err := h.dataCleanupService.CleanupInterfaceData(context.Background(), uint(id), ifaceName); err != nil {
				// 记录错误但不中断流程
				c.Writer.Header().Add("X-Cleanup-Warning", err.Error())
			}
		}
	}

	// 返回更新后的接口列表
	interfaces, err := h.deviceService.GetDeviceInterfaces(uint(id))
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 自动重新部署采集器脚本
	go h.redeployCollectorScript(uint(id))

	SuccessWithMessage(c, interfaces, "监控接口更新成功")
}

// RegisterRoutes 注册设备相关路由
func (h *DeviceHandler) RegisterRoutes(router *gin.RouterGroup) {
	devices := router.Group("/devices")
	{
		devices.POST("", h.CreateDevice)
		devices.GET("", h.ListDevices)
		devices.GET("/:id", h.GetDevice)
		devices.PUT("/:id", h.UpdateDevice)
		devices.DELETE("/:id", h.DeleteDevice)
		devices.PUT("/:id/status", h.UpdateDeviceStatus)
		
		// 连接测试
		devices.POST("/test", h.TestConnection)           // 测试新设备连接（不需要先保存）
		devices.POST("/:id/test", h.TestConnectionByID)   // 测试已保存设备连接
		
		// 系统信息（主动采集）
		devices.GET("/:id/info", h.GetSystemInfo)
		
		// 分组管理
		devices.POST("/:id/groups", h.AddDeviceToGroup)
		devices.DELETE("/:id/groups/:group_id", h.RemoveDeviceFromGroup)
		
		// 标签管理
		devices.POST("/:id/tags", h.AddDeviceTag)
		devices.DELETE("/:id/tags/:tag_id", h.RemoveDeviceTag)
		
		// 接口管理
		devices.GET("/:id/interfaces", h.GetDeviceInterfaces)
		devices.POST("/:id/interfaces/sync", h.SyncInterfaces)           // 从设备同步接口
		devices.PUT("/:id/interfaces/monitored", h.SetMonitoredInterfaces) // 批量设置监控接口
		devices.PUT("/interfaces/:interface_id/monitor", h.UpdateInterfaceMonitorStatus)
	}
}

// RegisterRoutesWithPermission 注册设备相关路由（带权限检查）
// permChecker: 设备权限检查器
// 权限规则:
// - admin 角色可以操作所有设备
// - operator 角色只能操作分配给自己的设备
// - viewer 角色只能查看，不能修改
func (h *DeviceHandler) RegisterRoutesWithPermission(router *gin.RouterGroup, readMiddleware, updateMiddleware, deleteMiddleware gin.HandlerFunc) {
	devices := router.Group("/devices")
	{
		// 创建设备 - 需要 device:create 权限（由 RBAC 中间件控制）
		devices.POST("", h.CreateDevice)
		
		// 列表查询 - 需要 device:read 权限
		devices.GET("", h.ListDevices)
		
		// 单设备查询 - 需要设备级别读取权限
		devices.GET("/:id", readMiddleware, h.GetDevice)
		
		// 更新设备 - 需要设备级别更新权限
		devices.PUT("/:id", updateMiddleware, h.UpdateDevice)
		
		// 删除设备 - 需要设备级别删除权限
		devices.DELETE("/:id", deleteMiddleware, h.DeleteDevice)
		
		// 更新设备状态 - 需要设备级别更新权限
		devices.PUT("/:id/status", updateMiddleware, h.UpdateDeviceStatus)
		
		// 连接测试
		devices.POST("/test", h.TestConnection)                              // 测试新设备连接（不需要先保存）
		devices.POST("/:id/test", readMiddleware, h.TestConnectionByID)      // 测试已保存设备连接
		
		// 系统信息（主动采集）- 需要设备级别读取权限
		devices.GET("/:id/info", readMiddleware, h.GetSystemInfo)
		
		// 分组管理 - 需要设备级别更新权限
		devices.POST("/:id/groups", updateMiddleware, h.AddDeviceToGroup)
		devices.DELETE("/:id/groups/:group_id", updateMiddleware, h.RemoveDeviceFromGroup)
		
		// 标签管理 - 需要设备级别更新权限
		devices.POST("/:id/tags", updateMiddleware, h.AddDeviceTag)
		devices.DELETE("/:id/tags/:tag_id", updateMiddleware, h.RemoveDeviceTag)
		
		// 接口管理 - 需要设备级别读取/更新权限
		devices.GET("/:id/interfaces", readMiddleware, h.GetDeviceInterfaces)
		devices.POST("/:id/interfaces/sync", updateMiddleware, h.SyncInterfaces)
		devices.PUT("/:id/interfaces/monitored", updateMiddleware, h.SetMonitoredInterfaces)
		devices.PUT("/interfaces/:interface_id/monitor", h.UpdateInterfaceMonitorStatus)
	}
}