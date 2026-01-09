package api

import (
	"path/filepath"
	"strconv"
	"time"

	"nmp-platform/internal/backup"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// BackupHandler 备份API处理器
type BackupHandler struct {
	backupService *backup.BackupService
	scheduler     *backup.Scheduler
	logger        *zap.Logger
}

// NewBackupHandler 创建新的备份处理器
func NewBackupHandler(backupService *backup.BackupService, scheduler *backup.Scheduler, logger *zap.Logger) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		scheduler:     scheduler,
		logger:        logger,
	}
}

// RegisterRoutes 注册备份相关路由
func (bh *BackupHandler) RegisterRoutes(router *gin.RouterGroup) {
	backupGroup := router.Group("/backup")
	{
		backupGroup.POST("/create", bh.CreateBackup)
		backupGroup.POST("/restore", bh.RestoreBackup)
		backupGroup.GET("/list", bh.ListBackups)
		backupGroup.GET("/status", bh.GetBackupStatus)
		backupGroup.POST("/trigger", bh.TriggerBackup)
		backupGroup.DELETE("/cleanup", bh.CleanupBackups)
		backupGroup.POST("/validate", bh.ValidateBackup)
	}
}

// CreateBackupRequest 创建备份请求
type CreateBackupRequest struct {
	BackupDir     string `json:"backup_dir" binding:"required"`
	IncludeData   bool   `json:"include_data"`
	IncludeSchema bool   `json:"include_schema"`
	Compress      bool   `json:"compress"`
}

// CreateBackup 创建备份
func (bh *BackupHandler) CreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求")
		return
	}
	
	// 设置默认值
	if !req.IncludeData && !req.IncludeSchema {
		req.IncludeData = true
		req.IncludeSchema = true
	}
	
	opts := backup.BackupOptions{
		BackupDir:     req.BackupDir,
		IncludeData:   req.IncludeData,
		IncludeSchema: req.IncludeSchema,
		Compress:      req.Compress,
	}
	
	result, err := bh.backupService.CreateDatabaseBackup(c.Request.Context(), opts)
	if err != nil {
		bh.logger.Error("Failed to create backup", zap.Error(err))
		InternalError(c, "创建备份失败: "+err.Error())
		return
	}
	
	SuccessWithMessage(c, gin.H{
		"file_path": result.FilePath,
		"size":      result.Size,
		"duration":  result.Duration.String(),
		"timestamp": result.Timestamp,
	}, "备份创建成功")
}

// RestoreBackupRequest 恢复备份请求
type RestoreBackupRequest struct {
	BackupFile   string `json:"backup_file" binding:"required"`
	DropExisting bool   `json:"drop_existing"`
	CreateDB     bool   `json:"create_db"`
	IgnoreErrors bool   `json:"ignore_errors"`
}

// RestoreBackup 恢复备份
func (bh *BackupHandler) RestoreBackup(c *gin.Context) {
	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求")
		return
	}
	
	opts := backup.RestoreOptions{
		BackupFile:   req.BackupFile,
		DropExisting: req.DropExisting,
		CreateDB:     req.CreateDB,
		IgnoreErrors: req.IgnoreErrors,
	}
	
	result, err := bh.backupService.RestoreDatabaseBackup(c.Request.Context(), opts)
	if err != nil {
		bh.logger.Error("Failed to restore backup", zap.Error(err))
		InternalError(c, "恢复备份失败: "+err.Error())
		return
	}
	
	SuccessWithMessage(c, gin.H{
		"duration":  result.Duration.String(),
		"timestamp": result.Timestamp,
	}, "备份恢复成功")
}

// ListBackups 列出备份文件
func (bh *BackupHandler) ListBackups(c *gin.Context) {
	backupDir := c.Query("backup_dir")
	if backupDir == "" {
		backupDir = "./backups" // 默认备份目录
	}
	
	backups, err := bh.backupService.ListBackups(backupDir)
	if err != nil {
		bh.logger.Error("Failed to list backups", zap.Error(err))
		InternalError(c, "获取备份列表失败: "+err.Error())
		return
	}
	
	Success(c, gin.H{
		"backups": backups,
		"count":   len(backups),
	})
}

// GetBackupStatus 获取备份状态
func (bh *BackupHandler) GetBackupStatus(c *gin.Context) {
	backupDir := c.Query("backup_dir")
	if backupDir == "" {
		backupDir = "./backups"
	}
	
	status, err := bh.scheduler.GetBackupStatus(backupDir)
	if err != nil {
		bh.logger.Error("Failed to get backup status", zap.Error(err))
		InternalError(c, "获取备份状态失败: "+err.Error())
		return
	}
	
	Success(c, status)
}

// TriggerBackup 手动触发备份
func (bh *BackupHandler) TriggerBackup(c *gin.Context) {
	var req struct {
		BackupDir     string `json:"backup_dir"`
		RetentionDays int    `json:"retention_days"`
		Compress      bool   `json:"compress"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求")
		return
	}
	
	// 设置默认值
	if req.BackupDir == "" {
		req.BackupDir = "./backups"
	}
	if req.RetentionDays == 0 {
		req.RetentionDays = 30
	}
	
	config := backup.ScheduleConfig{
		BackupDir:     req.BackupDir,
		RetentionDays: req.RetentionDays,
		Compress:      req.Compress,
		Timeout:       30 * time.Minute,
	}
	
	result, err := bh.scheduler.TriggerBackup(config)
	if err != nil {
		bh.logger.Error("Failed to trigger backup", zap.Error(err))
		InternalError(c, "触发备份失败: "+err.Error())
		return
	}
	
	SuccessWithMessage(c, gin.H{
		"file_path": result.FilePath,
		"size":      result.Size,
		"duration":  result.Duration.String(),
		"timestamp": result.Timestamp,
	}, "备份触发成功")
}

// CleanupBackups 清理旧备份
func (bh *BackupHandler) CleanupBackups(c *gin.Context) {
	backupDir := c.Query("backup_dir")
	if backupDir == "" {
		backupDir = "./backups"
	}
	
	keepDaysStr := c.Query("keep_days")
	keepDays := 30 // 默认保留30天
	if keepDaysStr != "" {
		if days, err := strconv.Atoi(keepDaysStr); err == nil && days > 0 {
			keepDays = days
		}
	}
	
	err := bh.backupService.CleanupOldBackups(backupDir, keepDays)
	if err != nil {
		bh.logger.Error("Failed to cleanup backups", zap.Error(err))
		InternalError(c, "清理备份失败: "+err.Error())
		return
	}
	
	SuccessWithMessage(c, gin.H{
		"keep_days": keepDays,
	}, "备份清理完成")
}

// ValidateBackup 验证备份文件
func (bh *BackupHandler) ValidateBackup(c *gin.Context) {
	var req struct {
		BackupFile string `json:"backup_file" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求")
		return
	}
	
	err := bh.backupService.ValidateBackup(req.BackupFile)
	if err != nil {
		bh.logger.Error("Backup validation failed", zap.Error(err))
		BadRequest(c, "备份验证失败: "+err.Error())
		return
	}
	
	SuccessWithMessage(c, gin.H{
		"file": filepath.Base(req.BackupFile),
	}, "备份文件有效")
}