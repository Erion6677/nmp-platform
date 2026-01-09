package service

import (
	"context"
	"fmt"
	"log"
	"nmp-platform/internal/repository"
	"strconv"
	"sync"
	"time"
)

// DataCleanupService 数据清理服务
// 负责清理 InfluxDB 中的过期监控数据
type DataCleanupService struct {
	influxClient   InfluxClient
	redisClient    RedisClient
	settingsRepo   repository.SettingsRepository
	deviceRepo     repository.DeviceRepository
	
	// 清理任务控制
	stopChan       chan struct{}
	wg             sync.WaitGroup
	running        bool
	mu             sync.Mutex
	
	// 配置
	cleanupInterval time.Duration // 清理任务执行间隔
	bucket          string        // InfluxDB bucket 名称
	org             string        // InfluxDB org 名称
}

// DataCleanupConfig 数据清理配置
type DataCleanupConfig struct {
	CleanupInterval time.Duration // 清理任务执行间隔，默认 24 小时
	Bucket          string        // InfluxDB bucket 名称
	Org             string        // InfluxDB org 名称
}

// NewDataCleanupService 创建数据清理服务实例
func NewDataCleanupService(
	influxClient InfluxClient,
	settingsRepo repository.SettingsRepository,
	deviceRepo repository.DeviceRepository,
	config *DataCleanupConfig,
) *DataCleanupService {
	return NewDataCleanupServiceWithRedis(influxClient, nil, settingsRepo, deviceRepo, config)
}

// NewDataCleanupServiceWithRedis 创建带 Redis 的数据清理服务实例
func NewDataCleanupServiceWithRedis(
	influxClient InfluxClient,
	redisClient RedisClient,
	settingsRepo repository.SettingsRepository,
	deviceRepo repository.DeviceRepository,
	config *DataCleanupConfig,
) *DataCleanupService {
	if config == nil {
		config = &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
	}
	
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 24 * time.Hour
	}
	if config.Bucket == "" {
		config.Bucket = "monitoring"
	}
	if config.Org == "" {
		config.Org = "nmp"
	}
	
	return &DataCleanupService{
		influxClient:    influxClient,
		redisClient:     redisClient,
		settingsRepo:    settingsRepo,
		deviceRepo:      deviceRepo,
		stopChan:        make(chan struct{}),
		cleanupInterval: config.CleanupInterval,
		bucket:          config.Bucket,
		org:             config.Org,
	}
}

// Start 启动定时清理任务
func (s *DataCleanupService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("cleanup service is already running")
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()
	
	s.wg.Add(1)
	go s.runCleanupLoop(ctx)
	
	log.Printf("Data cleanup service started with interval: %v", s.cleanupInterval)
	return nil
}

// Stop 停止定时清理任务
func (s *DataCleanupService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()
	
	s.wg.Wait()
	log.Println("Data cleanup service stopped")
}

// runCleanupLoop 运行清理循环
func (s *DataCleanupService) runCleanupLoop(ctx context.Context) {
	defer s.wg.Done()
	
	// 启动时立即执行一次清理
	s.executeCleanup(ctx)
	
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup loop stopped due to context cancellation")
			return
		case <-s.stopChan:
			log.Println("Cleanup loop stopped")
			return
		case <-ticker.C:
			s.executeCleanup(ctx)
		}
	}
}

// executeCleanup 执行清理任务
func (s *DataCleanupService) executeCleanup(ctx context.Context) {
	log.Println("Starting scheduled data cleanup...")
	
	// 获取数据保留天数配置
	retentionDays := s.settingsRepo.GetDataRetentionDays()
	if retentionDays <= 0 {
		retentionDays = repository.DefaultDataRetentionDays
	}
	
	// 执行全局清理
	if err := s.CleanupExpiredData(ctx, retentionDays); err != nil {
		log.Printf("Failed to cleanup expired data: %v", err)
	} else {
		log.Printf("Successfully cleaned up data older than %d days", retentionDays)
	}
}

// CleanupExpiredData 清理过期数据（全局清理）
// 删除所有超过保留天数的数据
// Requirements: 8.1, 8.2, 8.4
func (s *DataCleanupService) CleanupExpiredData(ctx context.Context, retentionDays int) error {
	if retentionDays <= 0 {
		return fmt.Errorf("retention days must be positive, got: %d", retentionDays)
	}
	
	// 计算截止时间
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	
	log.Printf("Cleaning up data older than %v (retention: %d days)", cutoffTime, retentionDays)
	
	// 清理带宽数据
	if err := s.deleteDataBefore(ctx, "bandwidth", cutoffTime, nil); err != nil {
		log.Printf("Failed to cleanup bandwidth data: %v", err)
		// 继续清理其他数据
	}
	
	// 清理 Ping 数据
	if err := s.deleteDataBefore(ctx, "ping", cutoffTime, nil); err != nil {
		log.Printf("Failed to cleanup ping data: %v", err)
	}
	
	// 清理设备指标数据
	if err := s.deleteDataBefore(ctx, "device_metrics", cutoffTime, nil); err != nil {
		log.Printf("Failed to cleanup device_metrics data: %v", err)
	}
	
	log.Printf("Global data cleanup completed for data before %v", cutoffTime)
	return nil
}

// CleanupDeviceData 清理单个设备的所有数据
// 只删除指定设备的数据，不影响其他设备
// Requirements: 8.3
func (s *DataCleanupService) CleanupDeviceData(ctx context.Context, deviceID uint) error {
	if deviceID == 0 {
		return fmt.Errorf("device ID must be positive")
	}
	
	deviceIDStr := strconv.FormatUint(uint64(deviceID), 10)
	predicate := fmt.Sprintf(`device_id="%s"`, deviceIDStr)
	
	log.Printf("Cleaning up all data for device %d", deviceID)
	
	// 清理 InfluxDB 带宽数据
	if err := s.deleteDataWithPredicate(ctx, "bandwidth", predicate); err != nil {
		log.Printf("Failed to cleanup bandwidth data for device %d: %v", deviceID, err)
	}
	
	// 清理 InfluxDB Ping 数据
	if err := s.deleteDataWithPredicate(ctx, "ping", predicate); err != nil {
		log.Printf("Failed to cleanup ping data for device %d: %v", deviceID, err)
	}
	
	// 清理 InfluxDB 设备指标数据
	if err := s.deleteDataWithPredicate(ctx, "device_metrics", predicate); err != nil {
		log.Printf("Failed to cleanup device_metrics data for device %d: %v", deviceID, err)
	}
	
	// 清理 Redis 缓存数据
	if s.redisClient != nil {
		// 清理带宽缓存
		bandwidthKey := fmt.Sprintf("device:bandwidth:%d", deviceID)
		if err := s.redisClient.Delete(ctx, bandwidthKey); err != nil {
			log.Printf("Failed to cleanup bandwidth cache for device %d: %v", deviceID, err)
		}
		
		// 清理 Ping 缓存
		pingKey := fmt.Sprintf("device:ping:%d", deviceID)
		if err := s.redisClient.Delete(ctx, pingKey); err != nil {
			log.Printf("Failed to cleanup ping cache for device %d: %v", deviceID, err)
		}
		
		// 清理最新数据缓存
		latestKey := fmt.Sprintf("device:latest:%s", deviceIDStr)
		if err := s.redisClient.Delete(ctx, latestKey); err != nil {
			log.Printf("Failed to cleanup latest data cache for device %d: %v", deviceID, err)
		}
		
		// 清理设备状态缓存
		statusKey := fmt.Sprintf("device:status:%d", deviceID)
		if err := s.redisClient.Delete(ctx, statusKey); err != nil {
			log.Printf("Failed to cleanup status cache for device %d: %v", deviceID, err)
		}
		
		// 清理最后活跃时间缓存
		lastSeenKey := fmt.Sprintf("device:last_seen:%s", deviceIDStr)
		if err := s.redisClient.Delete(ctx, lastSeenKey); err != nil {
			log.Printf("Failed to cleanup last_seen cache for device %d: %v", deviceID, err)
		}
		
		log.Printf("Redis cache cleanup completed for device %d", deviceID)
	}
	
	log.Printf("Device data cleanup completed for device %d", deviceID)
	return nil
}

// CleanupDeviceDataBefore 清理单个设备指定时间之前的数据
func (s *DataCleanupService) CleanupDeviceDataBefore(ctx context.Context, deviceID uint, before time.Time) error {
	if deviceID == 0 {
		return fmt.Errorf("device ID must be positive")
	}
	
	deviceIDStr := strconv.FormatUint(uint64(deviceID), 10)
	tags := map[string]string{"device_id": deviceIDStr}
	
	log.Printf("Cleaning up data for device %d before %v", deviceID, before)
	
	// 清理带宽数据
	if err := s.deleteDataBefore(ctx, "bandwidth", before, tags); err != nil {
		log.Printf("Failed to cleanup bandwidth data for device %d: %v", deviceID, err)
	}
	
	// 清理 Ping 数据
	if err := s.deleteDataBefore(ctx, "ping", before, tags); err != nil {
		log.Printf("Failed to cleanup ping data for device %d: %v", deviceID, err)
	}
	
	// 清理设备指标数据
	if err := s.deleteDataBefore(ctx, "device_metrics", before, tags); err != nil {
		log.Printf("Failed to cleanup device_metrics data for device %d: %v", deviceID, err)
	}
	
	log.Printf("Device data cleanup completed for device %d before %v", deviceID, before)
	return nil
}

// deleteDataBefore 删除指定时间之前的数据
func (s *DataCleanupService) deleteDataBefore(ctx context.Context, measurement string, before time.Time, tags map[string]string) error {
	// 构建删除谓词
	predicate := fmt.Sprintf(`_measurement="%s"`, measurement)
	
	// 添加标签过滤
	for key, value := range tags {
		predicate += fmt.Sprintf(` AND %s="%s"`, key, value)
	}
	
	// 使用 InfluxDB Delete API
	// 注意：InfluxDB 2.x 的删除操作需要使用 Delete API
	// 这里我们通过执行删除查询来实现
	
	// 构建删除时间范围：从很久以前到截止时间
	startTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	
	return s.executeDelete(ctx, startTime, before, predicate)
}

// deleteDataWithPredicate 使用谓词删除数据
func (s *DataCleanupService) deleteDataWithPredicate(ctx context.Context, measurement string, additionalPredicate string) error {
	predicate := fmt.Sprintf(`_measurement="%s"`, measurement)
	if additionalPredicate != "" {
		predicate += " AND " + additionalPredicate
	}
	
	// 删除所有时间范围的数据
	startTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Now()
	
	return s.executeDelete(ctx, startTime, endTime, predicate)
}

// executeDelete 执行删除操作
// 使用 InfluxDB 2.x Delete API
func (s *DataCleanupService) executeDelete(ctx context.Context, start, stop time.Time, predicate string) error {
	log.Printf("Executing delete: start=%v, stop=%v, predicate=%s", start, stop, predicate)
	
	// 调用 InfluxDB Delete API
	err := s.influxClient.Delete(start, stop, predicate)
	if err != nil {
		log.Printf("Failed to delete data: %v", err)
		return err
	}
	
	log.Printf("Delete completed: bucket=%s, start=%v, stop=%v, predicate=%s",
		s.bucket, start, stop, predicate)
	
	return nil
}

// fluxPredicate 将简单谓词转换为 Flux 过滤表达式
func (s *DataCleanupService) fluxPredicate(predicate string) string {
	// 简单转换：将 key="value" 转换为 r.key == "value"
	// 这是一个简化实现，实际可能需要更复杂的解析
	
	// 将 _measurement="xxx" 转换为 r._measurement == "xxx"
	// 将 device_id="xxx" 转换为 r.device_id == "xxx"
	
	// 替换 = 为 ==
	// 替换 AND 为 and
	// 添加 r. 前缀
	
	// 简单实现：假设谓词格式正确
	// 实际应该使用正则表达式或解析器
	
	return fmt.Sprintf(`r._measurement == "%s"`, extractMeasurement(predicate))
}

// extractMeasurement 从谓词中提取 measurement 名称
func extractMeasurement(predicate string) string {
	// 简单实现：查找 _measurement="xxx" 模式
	// 实际应该使用正则表达式
	
	start := len(`_measurement="`)
	if len(predicate) < start {
		return ""
	}
	
	end := start
	for end < len(predicate) && predicate[end] != '"' {
		end++
	}
	
	if end > start && end < len(predicate) {
		return predicate[start:end]
	}
	
	return ""
}

// GetRetentionDays 获取当前配置的数据保留天数
func (s *DataCleanupService) GetRetentionDays() int {
	return s.settingsRepo.GetDataRetentionDays()
}

// SetRetentionDays 设置数据保留天数
func (s *DataCleanupService) SetRetentionDays(days int) error {
	if days <= 0 {
		return fmt.Errorf("retention days must be positive")
	}
	return s.settingsRepo.SetDataRetentionDays(days)
}

// GetCleanupStatus 获取清理服务状态
func (s *DataCleanupService) GetCleanupStatus() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	return map[string]interface{}{
		"running":          s.running,
		"cleanup_interval": s.cleanupInterval.String(),
		"retention_days":   s.settingsRepo.GetDataRetentionDays(),
		"bucket":           s.bucket,
		"org":              s.org,
	}
}

// TriggerCleanup 手动触发一次清理
func (s *DataCleanupService) TriggerCleanup(ctx context.Context) error {
	retentionDays := s.settingsRepo.GetDataRetentionDays()
	if retentionDays <= 0 {
		retentionDays = repository.DefaultDataRetentionDays
	}
	
	return s.CleanupExpiredData(ctx, retentionDays)
}

// CleanupInterfaceData 清理单个接口的带宽数据
// 当取消接口监控时调用
func (s *DataCleanupService) CleanupInterfaceData(ctx context.Context, deviceID uint, interfaceName string) error {
	if deviceID == 0 {
		return fmt.Errorf("device ID must be positive")
	}
	if interfaceName == "" {
		return fmt.Errorf("interface name cannot be empty")
	}
	
	deviceIDStr := strconv.FormatUint(uint64(deviceID), 10)
	predicate := fmt.Sprintf(`device_id="%s" AND interface="%s"`, deviceIDStr, interfaceName)
	
	log.Printf("Cleaning up bandwidth data for device %d interface %s", deviceID, interfaceName)
	
	// 清理 InfluxDB 带宽数据
	if err := s.deleteDataWithPredicate(ctx, "bandwidth", predicate); err != nil {
		log.Printf("Failed to cleanup bandwidth data for device %d interface %s: %v", deviceID, interfaceName, err)
		return err
	}
	
	// 清理 Redis 缓存
	if s.redisClient != nil {
		cacheKey := fmt.Sprintf("device:bandwidth:%d:%s", deviceID, interfaceName)
		if err := s.redisClient.Delete(ctx, cacheKey); err != nil {
			log.Printf("Failed to cleanup bandwidth cache for device %d interface %s: %v", deviceID, interfaceName, err)
		}
	}
	
	log.Printf("Interface data cleanup completed for device %d interface %s", deviceID, interfaceName)
	return nil
}

// CleanupPingTargetData 清理单个 Ping 目标的数据
// 当删除 Ping 目标时调用
func (s *DataCleanupService) CleanupPingTargetData(ctx context.Context, deviceID uint, targetAddress string, sourceInterface string) error {
	if deviceID == 0 {
		return fmt.Errorf("device ID must be positive")
	}
	if targetAddress == "" {
		return fmt.Errorf("target address cannot be empty")
	}
	
	deviceIDStr := strconv.FormatUint(uint64(deviceID), 10)
	
	// 构建谓词，考虑源接口
	var predicate string
	if sourceInterface != "" {
		predicate = fmt.Sprintf(`device_id="%s" AND target="%s" AND source_interface="%s"`, deviceIDStr, targetAddress, sourceInterface)
	} else {
		predicate = fmt.Sprintf(`device_id="%s" AND target="%s"`, deviceIDStr, targetAddress)
	}
	
	log.Printf("Cleaning up ping data for device %d target %s (source: %s)", deviceID, targetAddress, sourceInterface)
	
	// 清理 InfluxDB Ping 数据
	if err := s.deleteDataWithPredicate(ctx, "ping", predicate); err != nil {
		log.Printf("Failed to cleanup ping data for device %d target %s: %v", deviceID, targetAddress, err)
		return err
	}
	
	// 清理 Redis 缓存
	if s.redisClient != nil {
		var cacheKey string
		if sourceInterface != "" {
			cacheKey = fmt.Sprintf("device:ping:%d:%s_%s", deviceID, targetAddress, sourceInterface)
		} else {
			cacheKey = fmt.Sprintf("device:ping:%d:%s", deviceID, targetAddress)
		}
		if err := s.redisClient.Delete(ctx, cacheKey); err != nil {
			log.Printf("Failed to cleanup ping cache for device %d target %s: %v", deviceID, targetAddress, err)
		}
	}
	
	log.Printf("Ping target data cleanup completed for device %d target %s", deviceID, targetAddress)
	return nil
}
