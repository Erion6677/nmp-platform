package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"nmp-platform/internal/backup"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SystemBackupHandler 系统备份处理器
type SystemBackupHandler struct {
	service *backup.Service
	logger  *zap.Logger
}

// NewSystemBackupHandler 创建系统备份处理器
func NewSystemBackupHandler(service *backup.Service, logger *zap.Logger) *SystemBackupHandler {
	return &SystemBackupHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes 注册路由
func (h *SystemBackupHandler) RegisterRoutes(router *gin.RouterGroup) {
	backupGroup := router.Group("/system-backup")
	{
		backupGroup.GET("/list", h.ListBackups)
		backupGroup.POST("/create", h.CreateBackup)
		backupGroup.GET("/download/:id", h.DownloadBackup)
		backupGroup.POST("/restore/:id", h.RestoreBackup)
		backupGroup.DELETE("/:id", h.DeleteBackup)
		backupGroup.GET("/status", h.GetStatus)
	}
}

// SystemCreateBackupRequest 创建系统备份请求
type SystemCreateBackupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Components  []string `json:"components"` // postgres, influxdb, config, plugins
}

// ListBackups 列出所有备份
func (h *SystemBackupHandler) ListBackups(c *gin.Context) {
	backups, err := h.service.ListBackups()
	if err != nil {
		h.logger.Error("Failed to list backups", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取备份列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"backups": backups,
			"total":   len(backups),
		},
	})
}

// CreateBackup 创建备份
func (h *SystemBackupHandler) CreateBackup(c *gin.Context) {
	var req SystemCreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求格式",
		})
		return
	}

	// 默认备份所有组件
	if len(req.Components) == 0 {
		req.Components = []string{"postgres", "influxdb", "config"}
	}

	backupInfo, err := h.service.CreateBackup(req.Name, req.Description, req.Components)
	if err != nil {
		h.logger.Error("Failed to create backup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "创建备份失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    backupInfo,
		"message": "备份创建成功",
	})
}

// DownloadBackup 下载备份文件
func (h *SystemBackupHandler) DownloadBackup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "备份ID不能为空",
		})
		return
	}

	filePath, err := h.service.GetBackupFilePath(id)
	if err != nil {
		h.logger.Error("Failed to get backup file", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "备份文件不存在",
		})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "备份文件不存在",
		})
		return
	}

	// 设置下载头
	fileName := filepath.Base(filePath)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/gzip")
	
	c.File(filePath)
}

// SystemRestoreBackupRequest 还原系统备份请求
type SystemRestoreBackupRequest struct {
	Components []string `json:"components"` // 要还原的组件
}

// RestoreBackup 还原备份
func (h *SystemBackupHandler) RestoreBackup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "备份ID不能为空",
		})
		return
	}

	var req SystemRestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果没有请求体，使用默认值
		req.Components = []string{"postgres", "config"}
	}

	if err := h.service.RestoreBackup(id, req.Components); err != nil {
		h.logger.Error("Failed to restore backup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "还原备份失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "备份还原成功，请重启服务以应用更改",
	})
}

// DeleteBackup 删除备份
func (h *SystemBackupHandler) DeleteBackup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "备份ID不能为空",
		})
		return
	}

	if err := h.service.DeleteBackup(id); err != nil {
		h.logger.Error("Failed to delete backup", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "删除备份失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "备份删除成功",
	})
}

// GetStatus 获取备份状态
func (h *SystemBackupHandler) GetStatus(c *gin.Context) {
	backups, _ := h.service.ListBackups()

	var totalSize int64
	for _, b := range backups {
		totalSize += b.Size
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total_backups":    len(backups),
			"total_size":       totalSize,
			"total_size_human": formatSizeForAPI(totalSize),
			"backup_dir":       "/opt/nmp/backups",
		},
	})
}

// formatSizeForAPI 格式化文件大小
func formatSizeForAPI(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}
