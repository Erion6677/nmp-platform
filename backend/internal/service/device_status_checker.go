package service

import (
	"context"
	"fmt"
	"log"
	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
	"sync"
	"time"
)

// DeviceStatusChecker 设备状态检查器
type DeviceStatusChecker struct {
	deviceRepo     repository.DeviceRepository
	collectorRepo  repository.CollectorRepository
	settingsRepo   repository.SettingsRepository
	redisClient    RedisClient
	offlineTimeout time.Duration
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	running        bool
	mu             sync.Mutex
}

// NewDeviceStatusChecker 创建设备状态检查器实例
func NewDeviceStatusChecker(
	deviceRepo repository.DeviceRepository,
	collectorRepo repository.CollectorRepository,
	redisClient RedisClient,
) *DeviceStatusChecker {
	return &DeviceStatusChecker{
		deviceRepo:     deviceRepo,
		collectorRepo:  collectorRepo,
		redisClient:    redisClient,
		offlineTimeout: 60 * time.Second, // 默认60秒无数据视为离线
		checkInterval:  30 * time.Second, // 默认每30秒检查一次
		stopChan:       make(chan struct{}),
	}
}

// NewDeviceStatusCheckerWithSettings 创建带设置仓库的设备状态检查器
func NewDeviceStatusCheckerWithSettings(
	deviceRepo repository.DeviceRepository,
	collectorRepo repository.CollectorRepository,
	settingsRepo repository.SettingsRepository,
	redisClient RedisClient,
) *DeviceStatusChecker {
	checker := &DeviceStatusChecker{
		deviceRepo:     deviceRepo,
		collectorRepo:  collectorRepo,
		settingsRepo:   settingsRepo,
		redisClient:    redisClient,
		offlineTimeout: 60 * time.Second,
		checkInterval:  30 * time.Second,
		stopChan:       make(chan struct{}),
	}
	// 从设置中加载离线超时配置
	checker.loadSettingsFromDB()
	return checker
}

// loadSettingsFromDB 从数据库加载设置
func (c *DeviceStatusChecker) loadSettingsFromDB() {
	if c.settingsRepo == nil {
		return
	}
	timeout := c.settingsRepo.GetDeviceOfflineTimeout()
	if timeout > 0 {
		c.offlineTimeout = time.Duration(timeout) * time.Second
		log.Printf("Loaded device offline timeout from settings: %v", c.offlineTimeout)
	}
}

// SetOfflineTimeout 设置离线超时时间
func (c *DeviceStatusChecker) SetOfflineTimeout(timeout time.Duration) {
	c.offlineTimeout = timeout
}

// SetCheckInterval 设置检查间隔
func (c *DeviceStatusChecker) SetCheckInterval(interval time.Duration) {
	c.checkInterval = interval
}

// Start 启动定时检查
func (c *DeviceStatusChecker) Start(ctx context.Context) {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.stopChan = make(chan struct{})
	c.mu.Unlock()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.runChecker(ctx)
	}()

	log.Printf("Device status checker started with interval %v, offline timeout %v", c.checkInterval, c.offlineTimeout)
}

// Stop 停止定时检查
func (c *DeviceStatusChecker) Stop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	close(c.stopChan)
	c.mu.Unlock()

	c.wg.Wait()
	log.Println("Device status checker stopped")
}

// runChecker 运行检查循环
func (c *DeviceStatusChecker) runChecker(ctx context.Context) {
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()

	// 启动时立即执行一次检查
	c.CheckAllDevices(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Device status checker context cancelled")
			return
		case <-c.stopChan:
			log.Println("Device status checker received stop signal")
			return
		case <-ticker.C:
			c.CheckAllDevices(ctx)
		}
	}
}

// CheckAllDevices 检查所有设备的在线状态
func (c *DeviceStatusChecker) CheckAllDevices(ctx context.Context) {
	log.Println("Starting device status check...")

	// 获取所有设备
	devices, _, err := c.deviceRepo.List(0, 10000, nil)
	if err != nil {
		log.Printf("Failed to get devices for status check: %v", err)
		return
	}

	onlineCount := 0
	offlineCount := 0
	unknownCount := 0

	for _, device := range devices {
		status := c.checkDeviceStatus(ctx, device)
		
		// 更新设备状态
		if err := c.updateDeviceStatus(ctx, device, status); err != nil {
			log.Printf("Failed to update status for device %d: %v", device.ID, err)
		}

		switch status {
		case models.DeviceStatusOnline:
			onlineCount++
		case models.DeviceStatusOffline:
			offlineCount++
		default:
			unknownCount++
		}
	}

	log.Printf("Device status check completed: %d online, %d offline, %d unknown (total: %d)",
		onlineCount, offlineCount, unknownCount, len(devices))
}

// checkDeviceStatus 检查单个设备的状态
func (c *DeviceStatusChecker) checkDeviceStatus(ctx context.Context, device *models.Device) models.DeviceStatus {
	// 首先检查 Redis 中的最后在线时间
	lastSeenKey := fmt.Sprintf("device:last_seen:%d", device.ID)
	lastSeenStr, err := c.redisClient.Get(ctx, lastSeenKey)
	if err == nil && lastSeenStr != "" {
		lastSeen, err := time.Parse(time.RFC3339, lastSeenStr)
		if err == nil {
			if time.Since(lastSeen) < c.offlineTimeout {
				return models.DeviceStatusOnline
			}
			return models.DeviceStatusOffline
		}
	}

	// 如果 Redis 中没有数据，检查采集器的最后推送时间
	if c.collectorRepo != nil {
		collector, err := c.collectorRepo.GetByDeviceID(device.ID)
		if err == nil && collector != nil && collector.LastPushAt != nil {
			if time.Since(*collector.LastPushAt) < c.offlineTimeout {
				return models.DeviceStatusOnline
			}
			return models.DeviceStatusOffline
		}
	}

	// 检查数据库中的最后在线时间
	if device.LastSeen != nil {
		if time.Since(*device.LastSeen) < c.offlineTimeout {
			return models.DeviceStatusOnline
		}
		return models.DeviceStatusOffline
	}

	// 如果没有任何在线记录，返回未知状态
	return models.DeviceStatusUnknown
}

// updateDeviceStatus 更新设备状态
func (c *DeviceStatusChecker) updateDeviceStatus(ctx context.Context, device *models.Device, status models.DeviceStatus) error {
	// 只有状态发生变化时才更新
	if device.Status == status {
		return nil
	}

	// 更新 Redis 中的状态
	statusKey := fmt.Sprintf("device:status:%d", device.ID)
	if err := c.redisClient.Set(ctx, statusKey, string(status), 24*time.Hour); err != nil {
		log.Printf("Failed to update device status in Redis for %d: %v", device.ID, err)
	}

	// 更新数据库中的状态
	if err := c.deviceRepo.UpdateStatus(device.ID, status); err != nil {
		return fmt.Errorf("failed to update device status in DB: %w", err)
	}

	// 如果设备从在线变为离线，记录日志
	if device.Status == models.DeviceStatusOnline && status == models.DeviceStatusOffline {
		log.Printf("Device %d (%s) went offline", device.ID, device.Name)
	} else if device.Status == models.DeviceStatusOffline && status == models.DeviceStatusOnline {
		log.Printf("Device %d (%s) came online", device.ID, device.Name)
	}

	return nil
}

// CheckSingleDevice 检查单个设备的状态
func (c *DeviceStatusChecker) CheckSingleDevice(ctx context.Context, deviceID uint) (models.DeviceStatus, error) {
	device, err := c.deviceRepo.GetByID(deviceID)
	if err != nil {
		return models.DeviceStatusUnknown, fmt.Errorf("device not found: %w", err)
	}

	status := c.checkDeviceStatus(ctx, device)
	
	// 更新状态
	if err := c.updateDeviceStatus(ctx, device, status); err != nil {
		log.Printf("Failed to update status for device %d: %v", deviceID, err)
	}

	return status, nil
}

// GetDeviceStatus 获取设备状态（不更新）
func (c *DeviceStatusChecker) GetDeviceStatus(ctx context.Context, deviceID uint) (models.DeviceStatus, error) {
	// 首先从 Redis 获取
	statusKey := fmt.Sprintf("device:status:%d", deviceID)
	statusStr, err := c.redisClient.Get(ctx, statusKey)
	if err == nil && statusStr != "" {
		return models.DeviceStatus(statusStr), nil
	}

	// 从数据库获取
	device, err := c.deviceRepo.GetByID(deviceID)
	if err != nil {
		return models.DeviceStatusUnknown, fmt.Errorf("device not found: %w", err)
	}

	return device.Status, nil
}

// GetOfflineDevices 获取所有离线设备
func (c *DeviceStatusChecker) GetOfflineDevices(ctx context.Context) ([]*models.Device, error) {
	devices, _, err := c.deviceRepo.List(0, 10000, map[string]interface{}{
		"status": models.DeviceStatusOffline,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get offline devices: %w", err)
	}
	return devices, nil
}

// GetOnlineDevices 获取所有在线设备
func (c *DeviceStatusChecker) GetOnlineDevices(ctx context.Context) ([]*models.Device, error) {
	return c.deviceRepo.GetAllOnline()
}

// GetDeviceStatusSummary 获取设备状态摘要
func (c *DeviceStatusChecker) GetDeviceStatusSummary(ctx context.Context) (map[string]int, error) {
	devices, _, err := c.deviceRepo.List(0, 10000, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}

	summary := map[string]int{
		"total":   len(devices),
		"online":  0,
		"offline": 0,
		"unknown": 0,
	}

	for _, device := range devices {
		switch device.Status {
		case models.DeviceStatusOnline:
			summary["online"]++
		case models.DeviceStatusOffline:
			summary["offline"]++
		default:
			summary["unknown"]++
		}
	}

	return summary, nil
}

// MarkDeviceOffline 手动标记设备为离线
func (c *DeviceStatusChecker) MarkDeviceOffline(ctx context.Context, deviceID uint) error {
	device, err := c.deviceRepo.GetByID(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	return c.updateDeviceStatus(ctx, device, models.DeviceStatusOffline)
}

// MarkDeviceOnline 手动标记设备为在线
func (c *DeviceStatusChecker) MarkDeviceOnline(ctx context.Context, deviceID uint) error {
	device, err := c.deviceRepo.GetByID(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// 更新最后在线时间
	now := time.Now()
	lastSeenKey := fmt.Sprintf("device:last_seen:%d", deviceID)
	if err := c.redisClient.Set(ctx, lastSeenKey, now.Format(time.RFC3339), 24*time.Hour); err != nil {
		log.Printf("Failed to update last seen time in Redis for %d: %v", deviceID, err)
	}

	return c.updateDeviceStatus(ctx, device, models.DeviceStatusOnline)
}
