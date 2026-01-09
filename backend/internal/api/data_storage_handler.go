package api

import (
	"net/http"
	"nmp-platform/internal/models"
	"nmp-platform/internal/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// DataStorageHandler 数据存储管理处理器
type DataStorageHandler struct {
	compressionService *service.DataCompressionService
}

// NewDataStorageHandler 创建数据存储管理处理器实例
func NewDataStorageHandler(compressionService *service.DataCompressionService) *DataStorageHandler {
	return &DataStorageHandler{
		compressionService: compressionService,
	}
}

// CompressData 手动触发数据压缩
// @Summary 压缩历史数据
// @Description 手动触发历史数据压缩，将旧数据聚合为更大的时间粒度
// @Tags 数据管理
// @Accept json
// @Produce json
// @Param days query int false "压缩多少天前的数据" default(7)
// @Success 200 {object} map[string]interface{} "压缩成功"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/storage/compress [post]
func (h *DataStorageHandler) CompressData(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid days parameter",
			Details: "Days must be a positive integer",
		})
		return
	}

	olderThan := time.Duration(days) * 24 * time.Hour
	
	if err := h.compressionService.CompressHistoricalData(c.Request.Context(), olderThan); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to compress data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Data compression completed successfully",
		"older_than": olderThan.String(),
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

// CleanupData 手动触发数据清理
// @Summary 清理过期数据
// @Description 手动触发过期数据清理，删除超过保留期的数据
// @Tags 数据管理
// @Accept json
// @Produce json
// @Param retention_days query int false "数据保留天数" default(90)
// @Success 200 {object} map[string]interface{} "清理成功"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/storage/cleanup [post]
func (h *DataStorageHandler) CleanupData(c *gin.Context) {
	retentionDaysStr := c.DefaultQuery("retention_days", "90")
	retentionDays, err := strconv.Atoi(retentionDaysStr)
	if err != nil || retentionDays <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid retention_days parameter",
			Details: "Retention days must be a positive integer",
		})
		return
	}

	retentionPeriod := time.Duration(retentionDays) * 24 * time.Hour
	
	if err := h.compressionService.CleanupOldData(c.Request.Context(), retentionPeriod); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to cleanup data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Data cleanup completed successfully",
		"retention_period": retentionPeriod.String(),
		"timestamp":        time.Now().Format(time.RFC3339),
	})
}

// GetStorageStats 获取存储统计信息
// @Summary 获取存储统计
// @Description 获取数据存储的统计信息，包括数据量、缓存使用等
// @Tags 数据管理
// @Produce json
// @Success 200 {object} map[string]interface{} "统计信息"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/storage/stats [get]
func (h *DataStorageHandler) GetStorageStats(c *gin.Context) {
	stats, err := h.compressionService.GetDataStatistics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get storage statistics",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// OptimizeStorage 优化存储性能
// @Summary 优化存储
// @Description 执行存储优化操作，提高查询和写入性能
// @Tags 数据管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "优化成功"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/storage/optimize [post]
func (h *DataStorageHandler) OptimizeStorage(c *gin.Context) {
	if err := h.compressionService.OptimizeStorage(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to optimize storage",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Storage optimization completed successfully",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetMaintenanceStatus 获取维护任务状态
// @Summary 获取维护状态
// @Description 获取数据维护任务的运行状态
// @Tags 数据管理
// @Produce json
// @Success 200 {object} map[string]interface{} "维护状态"
// @Router /api/v1/storage/maintenance/status [get]
func (h *DataStorageHandler) GetMaintenanceStatus(c *gin.Context) {
	// 这里可以返回维护任务的状态信息
	status := gin.H{
		"compression_enabled": true,
		"cleanup_enabled":     true,
		"last_compression":    "2024-01-01T00:00:00Z", // 实际应该从存储中获取
		"last_cleanup":        "2024-01-01T00:00:00Z", // 实际应该从存储中获取
		"next_compression":    "2024-01-02T00:00:00Z", // 实际应该计算下次执行时间
		"next_cleanup":        "2024-01-01T06:00:00Z", // 实际应该计算下次执行时间
		"status":              "running",
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, status)
}

// RegisterRoutes 注册数据存储管理相关路由
func (h *DataStorageHandler) RegisterRoutes(router *gin.RouterGroup) {
	storageGroup := router.Group("/storage")
	{
		// 数据压缩和清理
		storageGroup.POST("/compress", h.CompressData)
		storageGroup.POST("/cleanup", h.CleanupData)
		storageGroup.POST("/optimize", h.OptimizeStorage)
		
		// 统计和状态
		storageGroup.GET("/stats", h.GetStorageStats)
		storageGroup.GET("/maintenance/status", h.GetMaintenanceStatus)
	}
}