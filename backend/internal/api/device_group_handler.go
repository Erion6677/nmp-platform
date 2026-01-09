package api

import (
	"net/http"
	"strconv"

	"nmp-platform/internal/models"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
)

// DeviceGroupHandler 设备分组处理器
type DeviceGroupHandler struct {
	deviceGroupService service.DeviceGroupService
	deviceService      service.DeviceService
}

// NewDeviceGroupHandler 创建新的设备分组处理器
func NewDeviceGroupHandler(
	deviceGroupService service.DeviceGroupService,
	deviceService service.DeviceService,
) *DeviceGroupHandler {
	return &DeviceGroupHandler{
		deviceGroupService: deviceGroupService,
		deviceService:      deviceService,
	}
}

// CreateDeviceGroupRequest 创建设备分组请求
type CreateDeviceGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ParentID    *uint  `json:"parent_id"`
}

// UpdateDeviceGroupRequest 更新设备分组请求
type UpdateDeviceGroupRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	ParentID    *uint  `json:"parent_id"`
}

// ListDeviceGroupsRequest 设备分组列表请求
type ListDeviceGroupsRequest struct {
	Page     int  `form:"page,default=1"`
	PageSize int  `form:"page_size,default=20"`
	ParentID *uint `form:"parent_id"`
}

// CreateDeviceGroup 创建设备分组
func (h *DeviceGroupHandler) CreateDeviceGroup(c *gin.Context) {
	var req CreateDeviceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 创建设备分组对象
	group := &models.DeviceGroup{
		Name:        req.Name,
		Description: req.Description,
		ParentID:    req.ParentID,
	}

	err := h.deviceGroupService.CreateGroup(group)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    group,
	})
}

// GetDeviceGroup 获取设备分组详情
func (h *DeviceGroupHandler) GetDeviceGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid group ID",
		})
		return
	}

	group, err := h.deviceGroupService.GetGroup(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    group,
	})
}

// UpdateDeviceGroup 更新设备分组
func (h *DeviceGroupHandler) UpdateDeviceGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid group ID",
		})
		return
	}

	var req UpdateDeviceGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// 获取现有分组
	group, err := h.deviceGroupService.GetGroup(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 更新分组信息
	group.Name = req.Name
	group.Description = req.Description
	group.ParentID = req.ParentID

	err = h.deviceGroupService.UpdateGroup(group)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    group,
	})
}

// DeleteDeviceGroup 删除设备分组
func (h *DeviceGroupHandler) DeleteDeviceGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid group ID",
		})
		return
	}

	err = h.deviceGroupService.DeleteGroup(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device group deleted successfully",
	})
}

// ListDeviceGroups 获取设备分组列表
func (h *DeviceGroupHandler) ListDeviceGroups(c *gin.Context) {
	var req ListDeviceGroupsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize
	if offset < 0 {
		offset = 0
	}

	groups, total, err := h.deviceGroupService.ListGroups(offset, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 计算分页信息
	totalPages := (int(total) + req.PageSize - 1) / req.PageSize

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"groups":      groups,
			"total":       total,
			"page":        req.Page,
			"page_size":   req.PageSize,
			"total_pages": totalPages,
		},
	})
}

// GetAllDeviceGroups 获取所有设备分组（不分页）
func (h *DeviceGroupHandler) GetAllDeviceGroups(c *gin.Context) {
	groups, err := h.deviceGroupService.GetAllGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}

// GetRootDeviceGroups 获取根分组
func (h *DeviceGroupHandler) GetRootDeviceGroups(c *gin.Context) {
	groups, err := h.deviceGroupService.GetRootGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}

// GetChildDeviceGroups 获取子分组
func (h *DeviceGroupHandler) GetChildDeviceGroups(c *gin.Context) {
	idStr := c.Param("id")
	parentID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid parent group ID",
		})
		return
	}

	groups, err := h.deviceGroupService.GetChildGroups(uint(parentID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    groups,
	})
}

// GetDeviceGroupDevices 获取分组中的设备
func (h *DeviceGroupHandler) GetDeviceGroupDevices(c *gin.Context) {
	idStr := c.Param("id")
	groupID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid group ID",
		})
		return
	}

	devices, err := h.deviceService.GetDevicesByGroup(uint(groupID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 获取设备数量
	deviceCount, err := h.deviceGroupService.GetGroupDeviceCount(uint(groupID))
	if err != nil {
		deviceCount = int64(len(devices))
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"devices":      devices,
			"device_count": deviceCount,
		},
	})
}

// RegisterRoutes 注册设备分组相关路由
func (h *DeviceGroupHandler) RegisterRoutes(router *gin.RouterGroup) {
	groups := router.Group("/device-groups")
	{
		groups.POST("", h.CreateDeviceGroup)
		groups.GET("", h.ListDeviceGroups)
		groups.GET("/all", h.GetAllDeviceGroups)
		groups.GET("/root", h.GetRootDeviceGroups)
		groups.GET("/:id", h.GetDeviceGroup)
		groups.PUT("/:id", h.UpdateDeviceGroup)
		groups.DELETE("/:id", h.DeleteDeviceGroup)
		groups.GET("/:id/children", h.GetChildDeviceGroups)
		groups.GET("/:id/devices", h.GetDeviceGroupDevices)
	}
}