package api

import (
	"net/http"

	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ConnectionTestHandler 连接测试处理器
type ConnectionTestHandler struct {
	connectionTestService *service.ConnectionTestService
	logger                *zap.Logger
}

// NewConnectionTestHandler 创建连接测试处理器
func NewConnectionTestHandler(connectionTestService *service.ConnectionTestService, logger *zap.Logger) *ConnectionTestHandler {
	return &ConnectionTestHandler{
		connectionTestService: connectionTestService,
		logger:                logger,
	}
}

// RegisterRoutes 注册路由
func (h *ConnectionTestHandler) RegisterRoutes(router *gin.RouterGroup) {
	devices := router.Group("/devices")
	{
		devices.POST("/test-connection", h.TestConnection)
	}
}

// TestConnection 测试设备连接
// @Summary 测试设备连接
// @Description 测试设备的 SSH 或 API 连接
// @Tags 设备管理
// @Accept json
// @Produce json
// @Param request body service.TestConnectionRequest true "连接测试请求"
// @Success 200 {object} service.TestConnectionResponse "连接测试结果"
// @Failure 400 {object} ErrorResponse "请求参数错误"
// @Failure 500 {object} ErrorResponse "服务器内部错误"
// @Router /api/v1/devices/test-connection [post]
func (h *ConnectionTestHandler) TestConnection(c *gin.Context) {
	var req service.TestConnectionRequest
	
	// 绑定请求参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("连接测试请求参数错误", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证设备类型和连接类型的组合
	if req.Type == "api" && req.DeviceType != "mikrotik" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Message: "只有 MikroTik 设备支持 API 连接",
		})
		return
	}

	// 执行连接测试
	result, err := h.connectionTestService.TestConnection(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("连接测试执行失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Message: "连接测试执行失败: " + err.Error(),
		})
		return
	}

	// 返回测试结果
	c.JSON(http.StatusOK, result)
}