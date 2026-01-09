package backup

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FullBackupInfo 完整备份信息（用于系统备份插件）
type FullBackupInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Size        int64     `json:"size"`
	SizeHuman   string    `json:"size_human"`
	CreatedAt   time.Time `json:"created_at"`
	Type        string    `json:"type"`   // full, database, config
	Status      string    `json:"status"` // completed, failed, in_progress
	FilePath    string    `json:"file_path"`
	Components  []string  `json:"components"` // postgres, influxdb, redis, config
}

// BackupConfig 备份配置
type BackupConfig struct {
	BackupDir     string `json:"backup_dir"`
	MaxBackups    int    `json:"max_backups"`
	PostgresHost  string `json:"postgres_host"`
	PostgresPort  int    `json:"postgres_port"`
	PostgresDB    string `json:"postgres_db"`
	PostgresUser  string `json:"postgres_user"`
	PostgresPass  string `json:"postgres_pass"`
	InfluxDBURL   string `json:"influxdb_url"`
	InfluxDBOrg   string `json:"influxdb_org"`
	InfluxDBToken string `json:"influxdb_token"`
	ConfigDir     string `json:"config_dir"`
	PluginsDir    string `json:"plugins_dir"`
}

// Service 备份服务
type Service struct {
	config     *BackupConfig
	logger     *zap.Logger
	mu         sync.Mutex
	inProgress bool
}

// NewService 创建备份服务
func NewService(config *BackupConfig, logger *zap.Logger) *Service {
	// 确保备份目录存在
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		logger.Error("Failed to create backup directory", zap.Error(err))
	}

	return &Service{
		config: config,
		logger: logger,
	}
}

// CreateBackup 创建完整备份
func (s *Service) CreateBackup(name, description string, components []string) (*FullBackupInfo, error) {
	s.mu.Lock()
	if s.inProgress {
		s.mu.Unlock()
		return nil, fmt.Errorf("另一个备份任务正在进行中")
	}
	s.inProgress = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inProgress = false
		s.mu.Unlock()
	}()

	// 生成备份ID
	backupID := time.Now().Format("20060102_150405")
	if name == "" {
		name = fmt.Sprintf("backup_%s", backupID)
	}

	// 创建临时目录
	tempDir := filepath.Join(s.config.BackupDir, "temp_"+backupID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 如果没有指定组件，默认备份所有
	if len(components) == 0 {
		components = []string{"postgres", "influxdb", "config"}
	}

	backupInfo := &FullBackupInfo{
		ID:          backupID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		Type:        "full",
		Status:      "in_progress",
		Components:  components,
	}

	// 备份各组件
	for _, component := range components {
		var err error
		switch component {
		case "postgres":
			err = s.backupPostgres(tempDir)
		case "influxdb":
			err = s.backupInfluxDB(tempDir)
		case "config":
			err = s.backupConfig(tempDir)
		case "plugins":
			err = s.backupPlugins(tempDir)
		}

		if err != nil {
			s.logger.Error("Backup component failed",
				zap.String("component", component),
				zap.Error(err))
			backupInfo.Status = "failed"
			return backupInfo, fmt.Errorf("备份 %s 失败: %w", component, err)
		}
	}

	// 保存备份元数据
	metaPath := filepath.Join(tempDir, "backup_meta.json")
	metaData, _ := json.MarshalIndent(backupInfo, "", "  ")
	if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
		return nil, fmt.Errorf("保存元数据失败: %w", err)
	}

	// 打包成 tar.gz
	archiveName := fmt.Sprintf("%s.tar.gz", name)
	archivePath := filepath.Join(s.config.BackupDir, archiveName)

	if err := s.createTarGz(tempDir, archivePath); err != nil {
		return nil, fmt.Errorf("打包备份失败: %w", err)
	}

	// 获取文件大小
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return nil, fmt.Errorf("获取备份文件信息失败: %w", err)
	}

	backupInfo.Size = fileInfo.Size()
	backupInfo.SizeHuman = formatSize(fileInfo.Size())
	backupInfo.FilePath = archivePath
	backupInfo.Status = "completed"

	// 清理旧备份
	s.cleanOldBackups()

	s.logger.Info("Backup created successfully",
		zap.String("id", backupID),
		zap.String("path", archivePath),
		zap.Int64("size", backupInfo.Size))

	return backupInfo, nil
}

// backupPostgres 备份 PostgreSQL
func (s *Service) backupPostgres(destDir string) error {
	s.logger.Info("Backing up PostgreSQL...")

	dumpFile := filepath.Join(destDir, "postgres_dump.sql")

	// 设置环境变量
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", s.config.PostgresPass))

	cmd := exec.Command("pg_dump",
		"-h", s.config.PostgresHost,
		"-p", fmt.Sprintf("%d", s.config.PostgresPort),
		"-U", s.config.PostgresUser,
		"-d", s.config.PostgresDB,
		"-F", "p", // plain text format
		"-f", dumpFile,
	)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump 失败: %s, %w", string(output), err)
	}

	s.logger.Info("PostgreSQL backup completed", zap.String("file", dumpFile))
	return nil
}

// backupInfluxDB 备份 InfluxDB
func (s *Service) backupInfluxDB(destDir string) error {
	s.logger.Info("Backing up InfluxDB...")

	influxBackupDir := filepath.Join(destDir, "influxdb")
	if err := os.MkdirAll(influxBackupDir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("influx", "backup",
		influxBackupDir,
		"--org", s.config.InfluxDBOrg,
		"--token", s.config.InfluxDBToken,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// InfluxDB 备份失败不是致命错误
		s.logger.Warn("InfluxDB backup failed",
			zap.String("output", string(output)),
			zap.Error(err))
		return nil
	}

	s.logger.Info("InfluxDB backup completed", zap.String("dir", influxBackupDir))
	return nil
}

// backupConfig 备份配置文件
func (s *Service) backupConfig(destDir string) error {
	s.logger.Info("Backing up config files...")

	configBackupDir := filepath.Join(destDir, "config")
	if err := os.MkdirAll(configBackupDir, 0755); err != nil {
		return err
	}

	// 复制配置目录
	if s.config.ConfigDir != "" {
		if err := copyDir(s.config.ConfigDir, configBackupDir); err != nil {
			s.logger.Warn("Config backup partial failure", zap.Error(err))
		}
	}

	s.logger.Info("Config backup completed")
	return nil
}

// backupPlugins 备份插件
func (s *Service) backupPlugins(destDir string) error {
	s.logger.Info("Backing up plugins...")

	pluginsBackupDir := filepath.Join(destDir, "plugins")
	if err := os.MkdirAll(pluginsBackupDir, 0755); err != nil {
		return err
	}

	if s.config.PluginsDir != "" {
		if err := copyDir(s.config.PluginsDir, pluginsBackupDir); err != nil {
			s.logger.Warn("Plugins backup partial failure", zap.Error(err))
		}
	}

	s.logger.Info("Plugins backup completed")
	return nil
}

// ListBackups 列出所有备份
func (s *Service) ListBackups() ([]*FullBackupInfo, error) {
	var backups []*FullBackupInfo

	files, err := os.ReadDir(s.config.BackupDir)
	if err != nil {
		return nil, fmt.Errorf("读取备份目录失败: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".tar.gz") {
			info, err := file.Info()
			if err != nil {
				continue
			}

			backup := &FullBackupInfo{
				ID:        strings.TrimSuffix(file.Name(), ".tar.gz"),
				Name:      strings.TrimSuffix(file.Name(), ".tar.gz"),
				Size:      info.Size(),
				SizeHuman: formatSize(info.Size()),
				CreatedAt: info.ModTime(),
				FilePath:  filepath.Join(s.config.BackupDir, file.Name()),
				Status:    "completed",
				Type:      "full",
			}

			backups = append(backups, backup)
		}
	}

	// 按时间倒序排列
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// GetBackup 获取备份信息
func (s *Service) GetBackup(id string) (*FullBackupInfo, error) {
	backups, err := s.ListBackups()
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		if backup.ID == id || backup.Name == id {
			return backup, nil
		}
	}

	return nil, fmt.Errorf("备份不存在: %s", id)
}

// DeleteBackup 删除备份
func (s *Service) DeleteBackup(id string) error {
	backup, err := s.GetBackup(id)
	if err != nil {
		return err
	}

	if err := os.Remove(backup.FilePath); err != nil {
		return fmt.Errorf("删除备份文件失败: %w", err)
	}

	s.logger.Info("Backup deleted", zap.String("id", id))
	return nil
}

// GetBackupFilePath 获取备份文件路径（用于下载）
func (s *Service) GetBackupFilePath(id string) (string, error) {
	backup, err := s.GetBackup(id)
	if err != nil {
		return "", err
	}
	return backup.FilePath, nil
}

// RestoreBackup 还原备份
func (s *Service) RestoreBackup(id string, components []string) error {
	s.mu.Lock()
	if s.inProgress {
		s.mu.Unlock()
		return fmt.Errorf("另一个备份/还原任务正在进行中")
	}
	s.inProgress = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inProgress = false
		s.mu.Unlock()
	}()

	backup, err := s.GetBackup(id)
	if err != nil {
		return err
	}

	// 创建临时解压目录
	tempDir := filepath.Join(s.config.BackupDir, "restore_temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 解压备份文件
	if err := s.extractTarGz(backup.FilePath, tempDir); err != nil {
		return fmt.Errorf("解压备份失败: %w", err)
	}

	// 如果没有指定组件，还原所有
	if len(components) == 0 {
		components = []string{"postgres", "config"}
	}

	// 还原各组件
	for _, component := range components {
		var err error
		switch component {
		case "postgres":
			err = s.restorePostgres(tempDir)
		case "influxdb":
			err = s.restoreInfluxDB(tempDir)
		case "config":
			err = s.restoreConfig(tempDir)
		}

		if err != nil {
			s.logger.Error("Restore component failed",
				zap.String("component", component),
				zap.Error(err))
			return fmt.Errorf("还原 %s 失败: %w", component, err)
		}
	}

	s.logger.Info("Backup restored successfully", zap.String("id", id))
	return nil
}

// restorePostgres 还原 PostgreSQL
func (s *Service) restorePostgres(sourceDir string) error {
	s.logger.Info("Restoring PostgreSQL...")

	dumpFile := filepath.Join(sourceDir, "postgres_dump.sql")
	if _, err := os.Stat(dumpFile); os.IsNotExist(err) {
		return fmt.Errorf("PostgreSQL 备份文件不存在")
	}

	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", s.config.PostgresPass))

	cmd := exec.Command("psql",
		"-h", s.config.PostgresHost,
		"-p", fmt.Sprintf("%d", s.config.PostgresPort),
		"-U", s.config.PostgresUser,
		"-d", s.config.PostgresDB,
		"-f", dumpFile,
	)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql 还原失败: %s, %w", string(output), err)
	}

	s.logger.Info("PostgreSQL restored successfully")
	return nil
}

// restoreInfluxDB 还原 InfluxDB
func (s *Service) restoreInfluxDB(sourceDir string) error {
	s.logger.Info("Restoring InfluxDB...")

	influxBackupDir := filepath.Join(sourceDir, "influxdb")
	if _, err := os.Stat(influxBackupDir); os.IsNotExist(err) {
		s.logger.Warn("InfluxDB backup not found, skipping")
		return nil
	}

	cmd := exec.Command("influx", "restore",
		influxBackupDir,
		"--org", s.config.InfluxDBOrg,
		"--token", s.config.InfluxDBToken,
		"--full",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Warn("InfluxDB restore failed",
			zap.String("output", string(output)),
			zap.Error(err))
		return nil
	}

	s.logger.Info("InfluxDB restored successfully")
	return nil
}

// restoreConfig 还原配置文件
func (s *Service) restoreConfig(sourceDir string) error {
	s.logger.Info("Restoring config files...")

	configBackupDir := filepath.Join(sourceDir, "config")
	if _, err := os.Stat(configBackupDir); os.IsNotExist(err) {
		s.logger.Warn("Config backup not found, skipping")
		return nil
	}

	if s.config.ConfigDir != "" {
		if err := copyDir(configBackupDir, s.config.ConfigDir); err != nil {
			return fmt.Errorf("还原配置失败: %w", err)
		}
	}

	s.logger.Info("Config restored successfully")
	return nil
}

// createTarGz 创建 tar.gz 压缩包
func (s *Service) createTarGz(sourceDir, destFile string) error {
	file, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			return err
		}

		return nil
	})
}

// extractTarGz 解压 tar.gz 文件
func (s *Service) extractTarGz(sourceFile, destDir string) error {
	file, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// cleanOldBackups 清理旧备份
func (s *Service) cleanOldBackups() {
	backups, err := s.ListBackups()
	if err != nil {
		return
	}

	if len(backups) <= s.config.MaxBackups {
		return
	}

	// 删除超出数量的旧备份
	for i := s.config.MaxBackups; i < len(backups); i++ {
		if err := os.Remove(backups[i].FilePath); err != nil {
			s.logger.Warn("Failed to delete old backup",
				zap.String("path", backups[i].FilePath),
				zap.Error(err))
		} else {
			s.logger.Info("Old backup deleted", zap.String("id", backups[i].ID))
		}
	}
}

// copyDir 复制目录
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath)
	})
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
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
