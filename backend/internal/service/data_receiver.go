package service

import (
	"context"
	"fmt"
	"log"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"strconv"
	"time"
)

// InfluxClient InfluxDB客户端接口
type InfluxClient interface {
	WritePoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) error
	Query(query string) (QueryResult, error)
	Delete(start, stop time.Time, predicate string) error
	Health() error
	Close()
	Flush() // 刷新写入缓冲区
}

// QueryResult 查询结果接口
type QueryResult interface {
	Next() bool
	Record() QueryRecord
	Err() error
}

// QueryRecord 查询记录接口
type QueryRecord interface {
	Time() time.Time
	Field() string
	Value() interface{}
	ValueByKey(key string) interface{} // 获取 tag 或其他字段的值
}

// RedisClient Redis客户端接口
type RedisClient interface {
	Exists(ctx context.Context, key string) (bool, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetJSON(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	Health() error
	Close() error
}

// DataReceiverService 数据接收服务
type DataReceiverService struct {
	influxClient    InfluxClient
	redisClient     RedisClient
	deviceRepo      repository.DeviceRepository
	collectorRepo   repository.CollectorRepository
	cacheExpiry     time.Duration
	batchSize       int
	offlineTimeout  time.Duration // 离线超时时间
}

// NewDataReceiverService 创建数据接收服务实例
func NewDataReceiverService(
	influxClient InfluxClient,
	redisClient RedisClient,
	deviceRepo repository.DeviceRepository,
) *DataReceiverService {
	return &DataReceiverService{
		influxClient:   influxClient,
		redisClient:    redisClient,
		deviceRepo:     deviceRepo,
		cacheExpiry:    5 * time.Minute,  // 实时数据缓存5分钟
		batchSize:      100,               // 批处理大小
		offlineTimeout: 10 * time.Minute,  // 10分钟无数据视为离线
	}
}

// NewDataReceiverServiceWithCollector 创建带采集器仓库的数据接收服务实例
func NewDataReceiverServiceWithCollector(
	influxClient InfluxClient,
	redisClient RedisClient,
	deviceRepo repository.DeviceRepository,
	collectorRepo repository.CollectorRepository,
) *DataReceiverService {
	return &DataReceiverService{
		influxClient:   influxClient,
		redisClient:    redisClient,
		deviceRepo:     deviceRepo,
		collectorRepo:  collectorRepo,
		cacheExpiry:    5 * time.Minute,
		batchSize:      100,
		offlineTimeout: 10 * time.Minute,
	}
}

// SetOfflineTimeout 设置离线超时时间
func (s *DataReceiverService) SetOfflineTimeout(timeout time.Duration) {
	s.offlineTimeout = timeout
}

// ValidateDeviceByIP 通过 IP 验证设备身份
func (s *DataReceiverService) ValidateDeviceByIP(ctx context.Context, ip string) (*models.Device, error) {
	// 首先检查 Redis 缓存
	cacheKey := fmt.Sprintf("device:ip:%s", ip)
	var cachedDevice models.Device
	if err := s.redisClient.GetJSON(ctx, cacheKey, &cachedDevice); err == nil && cachedDevice.ID > 0 {
		return &cachedDevice, nil
	}
	
	// 从数据库查询设备
	device, err := s.deviceRepo.GetByHost(ip)
	if err != nil {
		return nil, fmt.Errorf("device not found for IP %s: %w", ip, err)
	}
	
	if device == nil {
		return nil, fmt.Errorf("device not found for IP %s", ip)
	}
	
	// 缓存设备信息（缓存1小时）
	_ = s.redisClient.SetJSON(ctx, cacheKey, device, time.Hour)
	
	return device, nil
}

// ProcessBandwidthData 处理带宽数据并写入 InfluxDB
func (s *DataReceiverService) ProcessBandwidthData(ctx context.Context, deviceID uint, timestamp int64, interfaces map[string]models.InterfaceMetrics) error {
	// 确定时间戳
	var ts time.Time
	if timestamp > 0 {
		ts = time.UnixMilli(timestamp)
	} else {
		ts = time.Now()
	}
	
	// 写入每个接口的带宽数据到 InfluxDB
	writeCount := 0
	for ifaceName, metrics := range interfaces {
		tags := map[string]string{
			"device_id": strconv.FormatUint(uint64(deviceID), 10),
			"interface": ifaceName,
		}
		
		// 确保数值类型为 float64，避免 InfluxDB 类型冲突
		fields := map[string]interface{}{
			"rx_rate": float64(metrics.RxRate),
			"tx_rate": float64(metrics.TxRate),
		}
		
		if err := s.influxClient.WritePoint("bandwidth", tags, fields, ts); err != nil {
			log.Printf("Failed to write bandwidth data for device %d interface %s: %v", deviceID, ifaceName, err)
			// 继续处理其他接口，不中断
		} else {
			writeCount++
		}
	}
	
	// 刷新写入缓冲区确保数据被写入
	s.influxClient.Flush()
	log.Printf("Wrote %d bandwidth points for device %d", writeCount, deviceID)
	
	// 更新 Redis 中的最新带宽数据
	latestKey := fmt.Sprintf("device:bandwidth:%d", deviceID)
	bandwidthData := map[string]interface{}{
		"timestamp":  ts,
		"interfaces": interfaces,
	}
	if err := s.redisClient.SetJSON(ctx, latestKey, bandwidthData, s.cacheExpiry); err != nil {
		log.Printf("Failed to cache bandwidth data for device %d: %v", deviceID, err)
	}
	
	return nil
}

// ProcessPingData 处理 Ping 数据并写入 InfluxDB
func (s *DataReceiverService) ProcessPingData(ctx context.Context, deviceID uint, timestamp int64, pings []models.PingMetric) error {
	// 确定时间戳
	var ts time.Time
	if timestamp > 0 {
		ts = time.UnixMilli(timestamp)
	} else {
		ts = time.Now()
	}
	
	// 写入每个 Ping 目标的数据到 InfluxDB
	writeCount := 0
	for _, ping := range pings {
		tags := map[string]string{
			"device_id":      strconv.FormatUint(uint64(deviceID), 10),
			"target_address": ping.Target,
			"src_interface":  ping.SrcIface,
		}
		
		// 处理延迟值：0 表示丢包
		// 确保 latency 为 float64 类型，避免 InfluxDB 类型冲突
		latency := float64(ping.Latency)
		status := ping.Status
		if status == "" {
			if latency > 0 {
				status = "up"
			} else {
				status = "down"
			}
		}
		
		fields := map[string]interface{}{
			"latency": latency,
			"status":  status,
		}
		
		if err := s.influxClient.WritePoint("ping", tags, fields, ts); err != nil {
			log.Printf("Failed to write ping data for device %d target %s: %v", deviceID, ping.Target, err)
			// 继续处理其他目标，不中断
		} else {
			writeCount++
		}
	}
	
	// 刷新写入缓冲区确保数据被写入
	s.influxClient.Flush()
	log.Printf("Wrote %d ping points for device %d", writeCount, deviceID)
	
	// 更新 Redis 中的最新 Ping 数据
	latestKey := fmt.Sprintf("device:ping:%d", deviceID)
	pingData := map[string]interface{}{
		"timestamp": ts,
		"pings":     pings,
	}
	if err := s.redisClient.SetJSON(ctx, latestKey, pingData, s.cacheExpiry); err != nil {
		log.Printf("Failed to cache ping data for device %d: %v", deviceID, err)
	}
	
	return nil
}

// UpdateDeviceOnlineStatus 更新设备在线状态
func (s *DataReceiverService) UpdateDeviceOnlineStatus(ctx context.Context, deviceID uint) error {
	now := time.Now()
	
	// 更新 Redis 中的设备状态
	statusKey := fmt.Sprintf("device:status:%d", deviceID)
	if err := s.redisClient.Set(ctx, statusKey, "online", 24*time.Hour); err != nil {
		log.Printf("Failed to update device status in Redis for %d: %v", deviceID, err)
	}
	
	// 更新最后在线时间
	lastSeenKey := fmt.Sprintf("device:last_seen:%d", deviceID)
	if err := s.redisClient.Set(ctx, lastSeenKey, now.Format(time.RFC3339), 24*time.Hour); err != nil {
		log.Printf("Failed to update last seen time in Redis for %d: %v", deviceID, err)
	}
	
	// 更新数据库中的设备状态
	if err := s.deviceRepo.UpdateStatus(deviceID, models.DeviceStatusOnline); err != nil {
		log.Printf("Failed to update device status in DB for %d: %v", deviceID, err)
	}
	
	// 更新采集器的最后推送时间
	if s.collectorRepo != nil {
		if err := s.collectorRepo.UpdateLastPushAt(deviceID); err != nil {
			log.Printf("Failed to update collector last push time for %d: %v", deviceID, err)
		}
		if err := s.collectorRepo.IncrementPushCount(deviceID); err != nil {
			log.Printf("Failed to increment push count for %d: %v", deviceID, err)
		}
	}
	
	return nil
}

// GetAllDevicesStatus 获取所有设备的在线状态
func (s *DataReceiverService) GetAllDevicesStatus(ctx context.Context) ([]models.DeviceOnlineStatus, error) {
	// 获取所有设备
	devices, _, err := s.deviceRepo.List(0, 1000, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	
	var statuses []models.DeviceOnlineStatus
	for _, device := range devices {
		status := models.DeviceOnlineStatus{
			DeviceID:   device.ID,
			DeviceName: device.Name,
			DeviceIP:   device.Host,
			LastSeen:   device.LastSeen,
		}
		
		// 从 Redis 获取实时状态
		statusKey := fmt.Sprintf("device:status:%d", device.ID)
		if statusStr, err := s.redisClient.Get(ctx, statusKey); err == nil {
			status.Status = statusStr
		} else {
			// 根据最后在线时间判断状态
			if device.LastSeen != nil && time.Since(*device.LastSeen) < s.offlineTimeout {
				status.Status = "online"
			} else {
				status.Status = "offline"
			}
		}
		
		// 获取最后推送时间
		if s.collectorRepo != nil {
			if collector, err := s.collectorRepo.GetByDeviceID(device.ID); err == nil && collector != nil {
				status.LastPushAt = collector.LastPushAt
			}
		}
		
		statuses = append(statuses, status)
	}
	
	return statuses, nil
}

// ReceiveData 接收单个设备的监控数据
func (s *DataReceiverService) ReceiveData(ctx context.Context, data *models.MetricData) error {
	// 验证数据格式
	if err := s.validateData(data); err != nil {
		return fmt.Errorf("data validation failed: %w", err)
	}

	// 验证设备ID
	if err := s.validateDeviceID(ctx, data.DeviceID); err != nil {
		return fmt.Errorf("device validation failed: %w", err)
	}

	// 存储到InfluxDB
	if err := s.storeToInfluxDB(data); err != nil {
		log.Printf("Failed to store data to InfluxDB: %v", err)
		// 不返回错误，继续缓存到Redis
	}

	// 缓存到Redis
	if err := s.cacheToRedis(ctx, data); err != nil {
		log.Printf("Failed to cache data to Redis: %v", err)
		// Redis缓存失败不影响主要功能
	}

	return nil
}

// ReceiveBatchData 接收批量监控数据
func (s *DataReceiverService) ReceiveBatchData(ctx context.Context, batchData []models.MetricData) (*models.BatchPushDataResponse, error) {
	response := &models.BatchPushDataResponse{
		Success: true,
		Message: "Batch data processed",
	}

	var errors []string
	processedCount := 0

	for i, data := range batchData {
		if err := s.ReceiveData(ctx, &data); err != nil {
			errors = append(errors, fmt.Sprintf("Item %d: %v", i, err))
			response.FailedCount++
		} else {
			processedCount++
		}
	}

	response.ProcessedCount = processedCount

	if len(errors) > 0 {
		response.Errors = errors
		if processedCount == 0 {
			response.Success = false
			response.Message = "All items failed to process"
		} else {
			response.Message = fmt.Sprintf("Partially processed: %d success, %d failed", processedCount, len(errors))
		}
	}

	return response, nil
}

// validateData 验证监控数据格式
func (s *DataReceiverService) validateData(data *models.MetricData) error {
	return data.Validate()
}

// validateDeviceID 验证设备ID是否存在
func (s *DataReceiverService) validateDeviceID(ctx context.Context, deviceID string) error {
	// 首先检查Redis缓存
	cacheKey := fmt.Sprintf("device:exists:%s", deviceID)
	exists, err := s.redisClient.Exists(ctx, cacheKey)
	if err == nil && exists {
		return nil
	}

	// 从数据库查询设备
	device, err := s.deviceRepo.GetByID(parseDeviceID(deviceID))
	if err != nil {
		return models.NewValidationError(fmt.Sprintf("device not found: %s", deviceID))
	}

	if device == nil {
		return models.NewValidationError(fmt.Sprintf("device not found: %s", deviceID))
	}

	// 缓存设备存在状态（缓存1小时）
	_ = s.redisClient.Set(ctx, cacheKey, "1", time.Hour)

	return nil
}

// storeToInfluxDB 存储数据到InfluxDB
func (s *DataReceiverService) storeToInfluxDB(data *models.MetricData) error {
	// 添加设备ID作为标签
	if data.Tags == nil {
		data.Tags = make(map[string]string)
	}
	data.Tags["device_id"] = data.DeviceID

	// 写入监控数据
	err := s.influxClient.WritePoint(
		"device_metrics", // measurement名称
		data.Tags,
		data.Metrics,
		data.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to write to InfluxDB: %w", err)
	}

	return nil
}

// cacheToRedis 缓存实时数据到Redis
func (s *DataReceiverService) cacheToRedis(ctx context.Context, data *models.MetricData) error {
	// 缓存最新的监控数据
	latestKey := fmt.Sprintf("device:latest:%s", data.DeviceID)
	if err := s.redisClient.SetJSON(ctx, latestKey, data, s.cacheExpiry); err != nil {
		return fmt.Errorf("failed to cache latest data: %w", err)
	}

	// 缓存各个指标的最新值
	for metric, value := range data.Metrics {
		metricKey := fmt.Sprintf("device:metric:%s:%s", data.DeviceID, metric)
		metricData := map[string]interface{}{
			"value":     value,
			"timestamp": data.Timestamp,
		}
		if err := s.redisClient.SetJSON(ctx, metricKey, metricData, s.cacheExpiry); err != nil {
			log.Printf("Failed to cache metric %s: %v", metric, err)
		}
	}

	// 更新设备最后活跃时间
	lastSeenKey := fmt.Sprintf("device:last_seen:%s", data.DeviceID)
	if err := s.redisClient.Set(ctx, lastSeenKey, data.Timestamp.Format(time.RFC3339), 24*time.Hour); err != nil {
		log.Printf("Failed to update last seen time: %v", err)
	}

	return nil
}

// GetLatestData 获取设备最新数据
func (s *DataReceiverService) GetLatestData(ctx context.Context, deviceID string) (*models.MetricData, error) {
	key := fmt.Sprintf("device:latest:%s", deviceID)
	var data models.MetricData
	
	err := s.redisClient.GetJSON(ctx, key, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest data: %w", err)
	}

	return &data, nil
}

// GetMetricValue 获取特定指标的最新值
func (s *DataReceiverService) GetMetricValue(ctx context.Context, deviceID, metric string) (interface{}, error) {
	key := fmt.Sprintf("device:metric:%s:%s", deviceID, metric)
	
	var metricData map[string]interface{}
	err := s.redisClient.GetJSON(ctx, key, &metricData)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric value: %w", err)
	}

	return metricData["value"], nil
}

// GetDeviceStatus 获取设备在线状态
func (s *DataReceiverService) GetDeviceStatus(ctx context.Context, deviceID string) (string, error) {
	key := fmt.Sprintf("device:last_seen:%s", deviceID)
	
	lastSeenStr, err := s.redisClient.Get(ctx, key)
	if err != nil {
		return "unknown", nil
	}

	lastSeen, err := time.Parse(time.RFC3339, lastSeenStr)
	if err != nil {
		return "unknown", nil
	}

	// 如果超过10分钟没有数据，认为离线
	if time.Since(lastSeen) > 10*time.Minute {
		return "offline", nil
	}

	return "online", nil
}

// parseDeviceID 解析设备ID为数字ID
func parseDeviceID(deviceID string) uint {
	// 这里简化处理，实际应该根据业务需求解析
	// 可能是数字ID，也可能是字符串标识符
	var id uint
	fmt.Sscanf(deviceID, "%d", &id)
	return id
}

// CleanupExpiredData 清理过期的缓存数据
func (s *DataReceiverService) CleanupExpiredData(ctx context.Context) error {
	// 获取所有设备的最后活跃时间
	pattern := "device:last_seen:*"
	keys, err := s.redisClient.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to get device keys: %w", err)
	}

	expiredCount := 0
	for _, key := range keys {
		lastSeenStr, err := s.redisClient.Get(ctx, key)
		if err != nil {
			continue
		}

		lastSeen, err := time.Parse(time.RFC3339, lastSeenStr)
		if err != nil {
			continue
		}

		// 清理超过24小时没有活动的设备缓存
		if time.Since(lastSeen) > 24*time.Hour {
			deviceID := key[len("device:last_seen:"):]
			
			// 删除相关的缓存键
			keysToDelete := []string{
				key,
				fmt.Sprintf("device:latest:%s", deviceID),
				fmt.Sprintf("device:exists:%s", deviceID),
			}
			
			// 删除所有指标缓存
			metricKeys, _ := s.redisClient.Keys(ctx, fmt.Sprintf("device:metric:%s:*", deviceID))
			keysToDelete = append(keysToDelete, metricKeys...)
			
			if err := s.redisClient.Delete(ctx, keysToDelete...); err != nil {
				log.Printf("Failed to delete expired cache for device %s: %v", deviceID, err)
			} else {
				expiredCount++
			}
		}
	}

	if expiredCount > 0 {
		log.Printf("Cleaned up cache for %d expired devices", expiredCount)
	}

	return nil
}

// DataCompressionService 数据压缩和清理服务
type DataCompressionService struct {
	influxClient InfluxClient
	redisClient  RedisClient
}

// NewDataCompressionService 创建数据压缩服务实例
func NewDataCompressionService(influxClient InfluxClient, redisClient RedisClient) *DataCompressionService {
	return &DataCompressionService{
		influxClient: influxClient,
		redisClient:  redisClient,
	}
}

// CompressHistoricalData 压缩历史数据
func (s *DataCompressionService) CompressHistoricalData(ctx context.Context, olderThan time.Duration) error {
	// 这里实现数据压缩逻辑
	// 例如：将超过指定时间的详细数据聚合为小时/天级别的数据
	
	cutoffTime := time.Now().Add(-olderThan)
	log.Printf("Starting data compression for data older than %v (cutoff: %v)", olderThan, cutoffTime)
	
	// 示例：压缩超过7天的分钟级数据为小时级数据
	if olderThan >= 7*24*time.Hour {
		if err := s.compressToHourly(ctx, cutoffTime); err != nil {
			return fmt.Errorf("failed to compress to hourly data: %w", err)
		}
	}
	
	// 示例：压缩超过30天的小时级数据为天级数据
	if olderThan >= 30*24*time.Hour {
		if err := s.compressToDaily(ctx, cutoffTime); err != nil {
			return fmt.Errorf("failed to compress to daily data: %w", err)
		}
	}
	
	log.Printf("Data compression completed for data older than %v", olderThan)
	return nil
}

// compressToHourly 将分钟级数据压缩为小时级数据
func (s *DataCompressionService) compressToHourly(ctx context.Context, cutoffTime time.Time) error {
	// 这里应该实现具体的InfluxDB查询和聚合逻辑
	// 由于这是一个复杂的操作，这里提供一个框架
	
	log.Printf("Compressing minute-level data to hourly for data before %v", cutoffTime)
	
	// 示例查询（实际实现需要根据具体的数据结构调整）
	/*
	query := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: -30d, stop: %s)
		|> filter(fn: (r) => r._measurement == "device_metrics")
		|> aggregateWindow(every: 1h, fn: mean, createEmpty: false)
		|> to(bucket: "monitoring_hourly")
	`, cutoffTime.Format(time.RFC3339))
	*/
	
	// 这里应该执行查询并处理结果
	// result, err := s.influxClient.Query(query)
	
	return nil
}

// compressToDaily 将小时级数据压缩为天级数据
func (s *DataCompressionService) compressToDaily(ctx context.Context, cutoffTime time.Time) error {
	log.Printf("Compressing hourly data to daily for data before %v", cutoffTime)
	
	// 类似于compressToHourly的实现
	// 这里应该实现将小时级数据聚合为天级数据的逻辑
	
	return nil
}

// CleanupOldData 清理过期数据
func (s *DataCompressionService) CleanupOldData(ctx context.Context, retentionPeriod time.Duration) error {
	cutoffTime := time.Now().Add(-retentionPeriod)
	log.Printf("Starting cleanup of data older than %v (cutoff: %v)", retentionPeriod, cutoffTime)
	
	// 清理InfluxDB中的过期数据
	if err := s.cleanupInfluxData(ctx, cutoffTime); err != nil {
		return fmt.Errorf("failed to cleanup InfluxDB data: %w", err)
	}
	
	// 清理Redis中的过期缓存
	if err := s.cleanupRedisCache(ctx); err != nil {
		return fmt.Errorf("failed to cleanup Redis cache: %w", err)
	}
	
	log.Printf("Data cleanup completed for data older than %v", retentionPeriod)
	return nil
}

// cleanupInfluxData 清理InfluxDB中的过期数据
func (s *DataCompressionService) cleanupInfluxData(ctx context.Context, cutoffTime time.Time) error {
	// 这里应该实现删除InfluxDB中过期数据的逻辑
	// 注意：InfluxDB的删除操作需要谨慎处理
	
	log.Printf("Cleaning up InfluxDB data before %v", cutoffTime)
	
	// 示例删除查询（实际实现需要根据具体需求调整）
	/*
	deleteQuery := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: -365d, stop: %s)
		|> filter(fn: (r) => r._measurement == "device_metrics")
		|> drop()
	`, cutoffTime.Format(time.RFC3339))
	*/
	
	return nil
}

// cleanupRedisCache 清理Redis中的过期缓存
func (s *DataCompressionService) cleanupRedisCache(ctx context.Context) error {
	log.Printf("Cleaning up expired Redis cache")
	
	// 获取所有设备相关的键
	patterns := []string{
		"device:latest:*",
		"device:metric:*",
		"device:last_seen:*",
		"device:exists:*",
	}
	
	totalCleaned := 0
	for _, pattern := range patterns {
		keys, err := s.redisClient.Keys(ctx, pattern)
		if err != nil {
			log.Printf("Failed to get keys for pattern %s: %v", pattern, err)
			continue
		}
		
		// 检查每个键的TTL，清理已过期但未自动删除的键
		expiredKeys := []string{}
		for _, key := range keys {
			// 这里可以添加更复杂的过期逻辑
			// 例如检查最后更新时间等
			expiredKeys = append(expiredKeys, key)
		}
		
		if len(expiredKeys) > 0 {
			if err := s.redisClient.Delete(ctx, expiredKeys...); err != nil {
				log.Printf("Failed to delete expired keys: %v", err)
			} else {
				totalCleaned += len(expiredKeys)
			}
		}
	}
	
	if totalCleaned > 0 {
		log.Printf("Cleaned up %d expired cache entries", totalCleaned)
	}
	
	return nil
}

// ScheduleDataMaintenance 调度数据维护任务
func (s *DataCompressionService) ScheduleDataMaintenance(ctx context.Context) {
	// 创建定时器进行数据维护
	compressionTicker := time.NewTicker(24 * time.Hour) // 每天执行一次压缩
	cleanupTicker := time.NewTicker(6 * time.Hour)      // 每6小时执行一次清理
	
	go func() {
		defer compressionTicker.Stop()
		defer cleanupTicker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				log.Println("Data maintenance scheduler stopped")
				return
				
			case <-compressionTicker.C:
				log.Println("Starting scheduled data compression")
				if err := s.CompressHistoricalData(ctx, 7*24*time.Hour); err != nil {
					log.Printf("Scheduled data compression failed: %v", err)
				}
				
			case <-cleanupTicker.C:
				log.Println("Starting scheduled data cleanup")
				if err := s.CleanupOldData(ctx, 90*24*time.Hour); err != nil {
					log.Printf("Scheduled data cleanup failed: %v", err)
				}
			}
		}
	}()
	
	log.Println("Data maintenance scheduler started")
}

// GetDataStatistics 获取数据统计信息
func (s *DataCompressionService) GetDataStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 获取Redis统计信息
	redisKeys, err := s.redisClient.Keys(ctx, "device:*")
	if err != nil {
		log.Printf("Failed to get Redis keys: %v", err)
	} else {
		stats["redis_keys_count"] = len(redisKeys)
	}
	
	// 这里可以添加更多统计信息
	// 例如InfluxDB中的数据点数量、存储大小等
	
	stats["last_updated"] = time.Now().Format(time.RFC3339)
	
	return stats, nil
}

// OptimizeStorage 优化存储性能
func (s *DataCompressionService) OptimizeStorage(ctx context.Context) error {
	log.Println("Starting storage optimization")
	
	// 执行Redis内存优化
	if err := s.optimizeRedisMemory(ctx); err != nil {
		log.Printf("Redis memory optimization failed: %v", err)
	}
	
	// 执行InfluxDB优化
	if err := s.optimizeInfluxDB(ctx); err != nil {
		log.Printf("InfluxDB optimization failed: %v", err)
	}
	
	log.Println("Storage optimization completed")
	return nil
}

// optimizeRedisMemory 优化Redis内存使用
func (s *DataCompressionService) optimizeRedisMemory(ctx context.Context) error {
	// 这里可以实现Redis内存优化逻辑
	// 例如：清理不必要的键、优化数据结构等
	
	log.Println("Optimizing Redis memory usage")
	
	// 示例：清理空的或无效的缓存项
	patterns := []string{"device:latest:*", "device:metric:*"}
	for _, pattern := range patterns {
		keys, err := s.redisClient.Keys(ctx, pattern)
		if err != nil {
			continue
		}
		
		// 检查并清理空值或无效值
		for _, key := range keys {
			value, err := s.redisClient.Get(ctx, key)
			if err != nil || value == "" || value == "null" {
				s.redisClient.Delete(ctx, key)
			}
		}
	}
	
	return nil
}

// optimizeInfluxDB 优化InfluxDB性能
func (s *DataCompressionService) optimizeInfluxDB(ctx context.Context) error {
	// 这里可以实现InfluxDB优化逻辑
	// 例如：重建索引、压缩数据等
	
	log.Println("Optimizing InfluxDB performance")
	
	// 实际的InfluxDB优化需要根据具体版本和配置来实现
	// 这里提供一个框架
	
	return nil
}