package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"nmp-platform/internal/config"

	"go.uber.org/zap"
)

// BackupService 备份服务
type BackupService struct {
	config *config.Config
	logger *zap.Logger
}

// NewBackupService 创建新的备份服务
func NewBackupService(cfg *config.Config, logger *zap.Logger) *BackupService {
	return &BackupService{
		config: cfg,
		logger: logger,
	}
}

// BackupOptions 备份选项
type BackupOptions struct {
	BackupDir     string
	IncludeData   bool
	IncludeSchema bool
	Compress      bool
}

// BackupResult 备份结果
type BackupResult struct {
	FilePath  string
	Size      int64
	Duration  time.Duration
	Timestamp time.Time
}

// CreateDatabaseBackup 创建数据库备份
func (bs *BackupService) CreateDatabaseBackup(ctx context.Context, opts BackupOptions) (*BackupResult, error) {
	startTime := time.Now()
	
	// 确保备份目录存在
	if err := os.MkdirAll(opts.BackupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("nmp_backup_%s.sql", timestamp)
	if opts.Compress {
		filename += ".gz"
	}
	
	backupPath := filepath.Join(opts.BackupDir, filename)
	
	// 构建pg_dump命令
	args := []string{
		"pg_dump",
		"-h", bs.config.Database.Host,
		"-p", fmt.Sprintf("%d", bs.config.Database.Port),
		"-U", bs.config.Database.Username,
		"-d", bs.config.Database.Database,
		"-f", backupPath,
		"--verbose",
	}
	
	// 根据选项添加参数
	if !opts.IncludeData {
		args = append(args, "--schema-only")
	}
	if !opts.IncludeSchema {
		args = append(args, "--data-only")
	}
	if opts.Compress {
		args = append(args, "--compress=9")
	}
	
	// 设置环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bs.config.Database.Password))
	
	// 执行备份命令
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	
	bs.logger.Info("Starting database backup",
		zap.String("backup_path", backupPath),
		zap.Bool("include_data", opts.IncludeData),
		zap.Bool("include_schema", opts.IncludeSchema),
		zap.Bool("compress", opts.Compress),
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		bs.logger.Error("Database backup failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return nil, fmt.Errorf("pg_dump failed: %w", err)
	}
	
	// 获取文件信息
	fileInfo, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file info: %w", err)
	}
	
	result := &BackupResult{
		FilePath:  backupPath,
		Size:      fileInfo.Size(),
		Duration:  time.Since(startTime),
		Timestamp: startTime,
	}
	
	bs.logger.Info("Database backup completed successfully",
		zap.String("file_path", result.FilePath),
		zap.Int64("size_bytes", result.Size),
		zap.Duration("duration", result.Duration),
	)
	
	return result, nil
}

// RestoreOptions 恢复选项
type RestoreOptions struct {
	BackupFile    string
	DropExisting  bool
	CreateDB      bool
	IgnoreErrors  bool
}

// RestoreResult 恢复结果
type RestoreResult struct {
	Duration  time.Duration
	Timestamp time.Time
}

// RestoreDatabaseBackup 恢复数据库备份
func (bs *BackupService) RestoreDatabaseBackup(ctx context.Context, opts RestoreOptions) (*RestoreResult, error) {
	startTime := time.Now()
	
	// 检查备份文件是否存在
	if _, err := os.Stat(opts.BackupFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("backup file does not exist: %s", opts.BackupFile)
	}
	
	// 如果需要删除现有数据库
	if opts.DropExisting {
		if err := bs.dropDatabase(ctx); err != nil {
			return nil, fmt.Errorf("failed to drop existing database: %w", err)
		}
	}
	
	// 如果需要创建数据库
	if opts.CreateDB {
		if err := bs.createDatabase(ctx); err != nil {
			return nil, fmt.Errorf("failed to create database: %w", err)
		}
	}
	
	// 构建psql命令
	args := []string{
		"psql",
		"-h", bs.config.Database.Host,
		"-p", fmt.Sprintf("%d", bs.config.Database.Port),
		"-U", bs.config.Database.Username,
		"-d", bs.config.Database.Database,
		"-f", opts.BackupFile,
		"--verbose",
	}
	
	if opts.IgnoreErrors {
		args = append(args, "--set", "ON_ERROR_STOP=off")
	} else {
		args = append(args, "--set", "ON_ERROR_STOP=on")
	}
	
	// 设置环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bs.config.Database.Password))
	
	// 执行恢复命令
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	
	bs.logger.Info("Starting database restore",
		zap.String("backup_file", opts.BackupFile),
		zap.Bool("drop_existing", opts.DropExisting),
		zap.Bool("create_db", opts.CreateDB),
		zap.Bool("ignore_errors", opts.IgnoreErrors),
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		bs.logger.Error("Database restore failed",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return nil, fmt.Errorf("psql restore failed: %w", err)
	}
	
	result := &RestoreResult{
		Duration:  time.Since(startTime),
		Timestamp: startTime,
	}
	
	bs.logger.Info("Database restore completed successfully",
		zap.Duration("duration", result.Duration),
	)
	
	return result, nil
}
// dropDatabase 删除数据库
func (bs *BackupService) dropDatabase(ctx context.Context) error {
	// 连接到postgres数据库来删除目标数据库
	args := []string{
		"psql",
		"-h", bs.config.Database.Host,
		"-p", fmt.Sprintf("%d", bs.config.Database.Port),
		"-U", bs.config.Database.Username,
		"-d", "postgres",
		"-c", fmt.Sprintf("DROP DATABASE IF EXISTS %s;", bs.config.Database.Database),
	}
	
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bs.config.Database.Password))
	
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		bs.logger.Error("Failed to drop database",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return err
	}
	
	bs.logger.Info("Database dropped successfully")
	return nil
}

// createDatabase 创建数据库
func (bs *BackupService) createDatabase(ctx context.Context) error {
	// 连接到postgres数据库来创建目标数据库
	args := []string{
		"psql",
		"-h", bs.config.Database.Host,
		"-p", fmt.Sprintf("%d", bs.config.Database.Port),
		"-U", bs.config.Database.Username,
		"-d", "postgres",
		"-c", fmt.Sprintf("CREATE DATABASE %s;", bs.config.Database.Database),
	}
	
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bs.config.Database.Password))
	
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = env
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		bs.logger.Error("Failed to create database",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return err
	}
	
	bs.logger.Info("Database created successfully")
	return nil
}

// ListBackups 列出备份文件
func (bs *BackupService) ListBackups(backupDir string) ([]BackupInfo, error) {
	files, err := filepath.Glob(filepath.Join(backupDir, "nmp_backup_*.sql*"))
	if err != nil {
		return nil, fmt.Errorf("failed to list backup files: %w", err)
	}
	
	var backups []BackupInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			bs.logger.Warn("Failed to get file info", zap.String("file", file), zap.Error(err))
			continue
		}
		
		backup := BackupInfo{
			FilePath:  file,
			FileName:  filepath.Base(file),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsCompressed: filepath.Ext(file) == ".gz",
		}
		
		backups = append(backups, backup)
	}
	
	return backups, nil
}

// BackupInfo 备份文件信息
type BackupInfo struct {
	FilePath     string
	FileName     string
	Size         int64
	ModTime      time.Time
	IsCompressed bool
}

// CleanupOldBackups 清理旧备份文件
func (bs *BackupService) CleanupOldBackups(backupDir string, keepDays int) error {
	backups, err := bs.ListBackups(backupDir)
	if err != nil {
		return err
	}
	
	cutoffTime := time.Now().AddDate(0, 0, -keepDays)
	deletedCount := 0
	
	for _, backup := range backups {
		if backup.ModTime.Before(cutoffTime) {
			if err := os.Remove(backup.FilePath); err != nil {
				bs.logger.Warn("Failed to delete old backup",
					zap.String("file", backup.FilePath),
					zap.Error(err),
				)
			} else {
				bs.logger.Info("Deleted old backup",
					zap.String("file", backup.FileName),
					zap.Time("mod_time", backup.ModTime),
				)
				deletedCount++
			}
		}
	}
	
	bs.logger.Info("Backup cleanup completed",
		zap.Int("deleted_count", deletedCount),
		zap.Int("keep_days", keepDays),
	)
	
	return nil
}

// ValidateBackup 验证备份文件
func (bs *BackupService) ValidateBackup(backupFile string) error {
	// 检查文件是否存在
	info, err := os.Stat(backupFile)
	if err != nil {
		return fmt.Errorf("backup file not accessible: %w", err)
	}
	
	// 检查文件大小
	if info.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}
	
	// 如果是压缩文件，尝试读取头部
	if filepath.Ext(backupFile) == ".gz" {
		// 可以添加更详细的压缩文件验证
		bs.logger.Info("Backup file appears to be compressed", zap.String("file", backupFile))
	}
	
	bs.logger.Info("Backup file validation passed",
		zap.String("file", backupFile),
		zap.Int64("size", info.Size()),
	)
	
	return nil
}