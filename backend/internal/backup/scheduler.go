package backup

import (
	"context"
	"fmt"
	"time"

	"nmp-platform/internal/config"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler 备份调度器
type Scheduler struct {
	backupService *BackupService
	cron          *cron.Cron
	config        *config.Config
	logger        *zap.Logger
	isRunning     bool
}

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	CronExpression string        // Cron表达式
	BackupDir      string        // 备份目录
	RetentionDays  int           // 保留天数
	Compress       bool          // 是否压缩
	Timeout        time.Duration // 超时时间
}

// NewScheduler 创建新的备份调度器
func NewScheduler(backupService *BackupService, cfg *config.Config, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		backupService: backupService,
		cron:          cron.New(cron.WithSeconds()),
		config:        cfg,
		logger:        logger,
		isRunning:     false,
	}
}

// Start 启动调度器
func (s *Scheduler) Start(scheduleConfig ScheduleConfig) error {
	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}
	
	// 添加备份任务
	_, err := s.cron.AddFunc(scheduleConfig.CronExpression, func() {
		s.executeBackup(scheduleConfig)
	})
	if err != nil {
		return fmt.Errorf("failed to add backup job: %w", err)
	}
	
	// 添加清理任务（每天凌晨2点执行）
	_, err = s.cron.AddFunc("0 0 2 * * *", func() {
		s.executeCleanup(scheduleConfig)
	})
	if err != nil {
		return fmt.Errorf("failed to add cleanup job: %w", err)
	}
	
	s.cron.Start()
	s.isRunning = true
	
	s.logger.Info("Backup scheduler started",
		zap.String("cron_expression", scheduleConfig.CronExpression),
		zap.String("backup_dir", scheduleConfig.BackupDir),
		zap.Int("retention_days", scheduleConfig.RetentionDays),
	)
	
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}
	
	ctx := s.cron.Stop()
	<-ctx.Done()
	
	s.isRunning = false
	s.logger.Info("Backup scheduler stopped")
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	return s.isRunning
}

// executeBackup 执行备份任务
func (s *Scheduler) executeBackup(config ScheduleConfig) {
	s.logger.Info("Starting scheduled backup")
	
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	opts := BackupOptions{
		BackupDir:     config.BackupDir,
		IncludeData:   true,
		IncludeSchema: true,
		Compress:      config.Compress,
	}
	
	result, err := s.backupService.CreateDatabaseBackup(ctx, opts)
	if err != nil {
		s.logger.Error("Scheduled backup failed", zap.Error(err))
		return
	}
	
	s.logger.Info("Scheduled backup completed successfully",
		zap.String("file_path", result.FilePath),
		zap.Int64("size_bytes", result.Size),
		zap.Duration("duration", result.Duration),
	)
}

// executeCleanup 执行清理任务
func (s *Scheduler) executeCleanup(config ScheduleConfig) {
	s.logger.Info("Starting scheduled backup cleanup")
	
	err := s.backupService.CleanupOldBackups(config.BackupDir, config.RetentionDays)
	if err != nil {
		s.logger.Error("Scheduled cleanup failed", zap.Error(err))
		return
	}
	
	s.logger.Info("Scheduled cleanup completed successfully")
}

// TriggerBackup 手动触发备份
func (s *Scheduler) TriggerBackup(config ScheduleConfig) (*BackupResult, error) {
	s.logger.Info("Triggering manual backup")
	
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	
	opts := BackupOptions{
		BackupDir:     config.BackupDir,
		IncludeData:   true,
		IncludeSchema: true,
		Compress:      config.Compress,
	}
	
	result, err := s.backupService.CreateDatabaseBackup(ctx, opts)
	if err != nil {
		s.logger.Error("Manual backup failed", zap.Error(err))
		return nil, err
	}
	
	s.logger.Info("Manual backup completed successfully",
		zap.String("file_path", result.FilePath),
		zap.Int64("size_bytes", result.Size),
		zap.Duration("duration", result.Duration),
	)
	
	return result, nil
}

// GetNextBackupTime 获取下次备份时间
func (s *Scheduler) GetNextBackupTime() time.Time {
	if !s.isRunning {
		return time.Time{}
	}
	
	entries := s.cron.Entries()
	if len(entries) > 0 {
		return entries[0].Next
	}
	
	return time.Time{}
}

// GetBackupStatus 获取备份状态
func (s *Scheduler) GetBackupStatus(backupDir string) (*BackupStatus, error) {
	backups, err := s.backupService.ListBackups(backupDir)
	if err != nil {
		return nil, err
	}
	
	status := &BackupStatus{
		IsSchedulerRunning: s.isRunning,
		NextBackupTime:     s.GetNextBackupTime(),
		TotalBackups:       len(backups),
	}
	
	if len(backups) > 0 {
		// 找到最新的备份
		var latest BackupInfo
		for _, backup := range backups {
			if backup.ModTime.After(latest.ModTime) {
				latest = backup
			}
		}
		status.LastBackupTime = latest.ModTime
		status.LastBackupFile = latest.FileName
		status.LastBackupSize = latest.Size
		
		// 计算总大小
		var totalSize int64
		for _, backup := range backups {
			totalSize += backup.Size
		}
		status.TotalBackupSize = totalSize
	}
	
	return status, nil
}

// BackupStatus 备份状态
type BackupStatus struct {
	IsSchedulerRunning bool      `json:"is_scheduler_running"`
	NextBackupTime     time.Time `json:"next_backup_time"`
	LastBackupTime     time.Time `json:"last_backup_time"`
	LastBackupFile     string    `json:"last_backup_file"`
	LastBackupSize     int64     `json:"last_backup_size"`
	TotalBackups       int       `json:"total_backups"`
	TotalBackupSize    int64     `json:"total_backup_size"`
}

// DefaultScheduleConfig 返回默认调度配置
func DefaultScheduleConfig() ScheduleConfig {
	return ScheduleConfig{
		CronExpression: "0 0 1 * * *", // 每天凌晨1点
		BackupDir:      "./backups",
		RetentionDays:  30, // 保留30天
		Compress:       true,
		Timeout:        30 * time.Minute,
	}
}