package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"nmp-platform/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestBackupService_ListBackups(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	
	bs := NewBackupService(cfg, logger)
	
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 创建一些测试备份文件
	testFiles := []string{
		"nmp_backup_20240101_120000.sql",
		"nmp_backup_20240102_120000.sql.gz",
		"nmp_backup_20240103_120000.sql",
	}
	
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte("test backup content"), 0644)
		require.NoError(t, err)
	}
	
	// 测试列出备份文件
	backups, err := bs.ListBackups(tempDir)
	require.NoError(t, err)
	
	assert.Len(t, backups, 3)
	
	// 验证文件信息
	for _, backup := range backups {
		assert.NotEmpty(t, backup.FileName)
		assert.Greater(t, backup.Size, int64(0))
		assert.False(t, backup.ModTime.IsZero())
		
		if filepath.Ext(backup.FileName) == ".gz" {
			assert.True(t, backup.IsCompressed)
		} else {
			assert.False(t, backup.IsCompressed)
		}
	}
}

func TestBackupService_CleanupOldBackups(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	
	bs := NewBackupService(cfg, logger)
	
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 创建不同时间的测试文件
	now := time.Now()
	testFiles := []struct {
		name string
		age  time.Duration
	}{
		{"nmp_backup_recent.sql", 1 * time.Hour},
		{"nmp_backup_old1.sql", 10 * 24 * time.Hour}, // 10天前
		{"nmp_backup_old2.sql", 40 * 24 * time.Hour}, // 40天前
	}
	
	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		err := os.WriteFile(filePath, []byte("test content"), 0644)
		require.NoError(t, err)
		
		// 设置文件修改时间
		modTime := now.Add(-tf.age)
		err = os.Chtimes(filePath, modTime, modTime)
		require.NoError(t, err)
	}
	
	// 清理30天前的备份
	err := bs.CleanupOldBackups(tempDir, 30)
	require.NoError(t, err)
	
	// 验证结果
	backups, err := bs.ListBackups(tempDir)
	require.NoError(t, err)
	
	// 应该只剩下最近的和10天前的文件
	assert.Len(t, backups, 2)
	
	for _, backup := range backups {
		assert.NotContains(t, backup.FileName, "old2")
	}
}

func TestBackupService_ValidateBackup(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	
	bs := NewBackupService(cfg, logger)
	
	// 创建临时目录
	tempDir := t.TempDir()
	
	// 测试有效的备份文件
	validFile := filepath.Join(tempDir, "valid_backup.sql")
	err := os.WriteFile(validFile, []byte("-- Valid SQL backup content"), 0644)
	require.NoError(t, err)
	
	err = bs.ValidateBackup(validFile)
	assert.NoError(t, err)
	
	// 测试空文件
	emptyFile := filepath.Join(tempDir, "empty_backup.sql")
	err = os.WriteFile(emptyFile, []byte(""), 0644)
	require.NoError(t, err)
	
	err = bs.ValidateBackup(emptyFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
	
	// 测试不存在的文件
	err = bs.ValidateBackup(filepath.Join(tempDir, "nonexistent.sql"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not accessible")
	
	// 测试压缩文件
	compressedFile := filepath.Join(tempDir, "compressed_backup.sql.gz")
	err = os.WriteFile(compressedFile, []byte("compressed content"), 0644)
	require.NoError(t, err)
	
	err = bs.ValidateBackup(compressedFile)
	assert.NoError(t, err)
}

func TestScheduler_StartStop(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	
	bs := NewBackupService(cfg, logger)
	scheduler := NewScheduler(bs, cfg, logger)
	
	// 测试初始状态
	assert.False(t, scheduler.IsRunning())
	
	// 测试启动调度器
	scheduleConfig := ScheduleConfig{
		CronExpression: "0 0 1 * * *", // 每天凌晨1点
		BackupDir:      "./test_backups",
		RetentionDays:  30,
		Compress:       true,
		Timeout:        5 * time.Minute,
	}
	
	err := scheduler.Start(scheduleConfig)
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())
	
	// 测试重复启动
	err = scheduler.Start(scheduleConfig)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
	
	// 测试停止调度器
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())
	
	// 测试重复停止（应该不会出错）
	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())
}

func TestScheduler_GetBackupStatus(t *testing.T) {
	logger := zap.NewNop()
	cfg := &config.Config{}
	
	bs := NewBackupService(cfg, logger)
	scheduler := NewScheduler(bs, cfg, logger)
	
	// 创建临时目录和测试文件
	tempDir := t.TempDir()
	
	testFile := filepath.Join(tempDir, "nmp_backup_20240101_120000.sql")
	err := os.WriteFile(testFile, []byte("test backup content"), 0644)
	require.NoError(t, err)
	
	// 获取备份状态
	status, err := scheduler.GetBackupStatus(tempDir)
	require.NoError(t, err)
	
	assert.False(t, status.IsSchedulerRunning)
	assert.Equal(t, 1, status.TotalBackups)
	assert.Greater(t, status.TotalBackupSize, int64(0))
	assert.NotEmpty(t, status.LastBackupFile)
	assert.False(t, status.LastBackupTime.IsZero())
}

func TestDefaultScheduleConfig(t *testing.T) {
	config := DefaultScheduleConfig()
	
	assert.Equal(t, "0 0 1 * * *", config.CronExpression)
	assert.Equal(t, "./backups", config.BackupDir)
	assert.Equal(t, 30, config.RetentionDays)
	assert.True(t, config.Compress)
	assert.Equal(t, 30*time.Minute, config.Timeout)
}