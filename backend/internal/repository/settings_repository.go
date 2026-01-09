package repository

import (
	"errors"
	"strconv"
	"time"

	"nmp-platform/internal/models"

	"gorm.io/gorm"
)

// 系统设置键常量
const (
	SettingKeyDefaultPushInterval     = "default_push_interval"      // 默认推送间隔（毫秒）
	SettingKeyDataRetentionDays       = "data_retention_days"        // 数据保留天数
	SettingKeyFrontendRefreshInterval = "frontend_refresh_interval"  // 前端刷新间隔（秒）
	SettingKeyDeviceOfflineTimeout    = "device_offline_timeout"     // 设备离线超时（秒）
	SettingKeyFollowPushInterval      = "follow_push_interval"       // 前端刷新是否跟随推送间隔
)

// 默认值
const (
	DefaultPushInterval      = 1000  // 1秒
	DefaultDataRetentionDays = 10    // 10天
	DefaultFrontendRefresh   = 10    // 10秒
	DefaultOfflineTimeout    = 60    // 60秒
	DefaultFollowPush        = false // 默认不跟随推送间隔
)

// SettingsRepository 系统设置仓库接口
type SettingsRepository interface {
	Get(key string) (string, error)
	Set(key, value, description string) error
	GetInt(key string, defaultValue int) int
	SetInt(key string, value int, description string) error
	GetBool(key string, defaultValue bool) bool
	SetBool(key string, value bool, description string) error
	GetAll() (map[string]string, error)
	Delete(key string) error
	
	// 便捷方法
	GetDefaultPushInterval() int
	SetDefaultPushInterval(intervalMs int) error
	GetDataRetentionDays() int
	SetDataRetentionDays(days int) error
	GetFrontendRefreshInterval() int
	SetFrontendRefreshInterval(seconds int) error
	GetDeviceOfflineTimeout() int
	SetDeviceOfflineTimeout(seconds int) error
	GetFollowPushInterval() bool
	SetFollowPushInterval(follow bool) error
	
	// 初始化默认设置
	InitDefaults() error
}

// settingsRepository 系统设置仓库实现
type settingsRepository struct {
	db *gorm.DB
}

// NewSettingsRepository 创建新的系统设置仓库
func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

// Get 获取设置值
func (r *settingsRepository) Get(key string) (string, error) {
	var setting models.SystemSetting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return setting.Value, nil
}

// Set 设置值
func (r *settingsRepository) Set(key, value, description string) error {
	setting := models.SystemSetting{
		Key:         key,
		Value:       value,
		Description: description,
		UpdatedAt:   time.Now(),
	}
	
	// 使用 upsert
	return r.db.Save(&setting).Error
}

// GetInt 获取整数设置值
func (r *settingsRepository) GetInt(key string, defaultValue int) int {
	value, err := r.Get(key)
	if err != nil || value == "" {
		return defaultValue
	}
	
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	
	return intValue
}

// SetInt 设置整数值
func (r *settingsRepository) SetInt(key string, value int, description string) error {
	return r.Set(key, strconv.Itoa(value), description)
}

// GetBool 获取布尔设置值
func (r *settingsRepository) GetBool(key string, defaultValue bool) bool {
	value, err := r.Get(key)
	if err != nil || value == "" {
		return defaultValue
	}
	
	return value == "true" || value == "1"
}

// SetBool 设置布尔值
func (r *settingsRepository) SetBool(key string, value bool, description string) error {
	strValue := "false"
	if value {
		strValue = "true"
	}
	return r.Set(key, strValue, description)
}

// GetAll 获取所有设置
func (r *settingsRepository) GetAll() (map[string]string, error) {
	var settings []models.SystemSetting
	err := r.db.Find(&settings).Error
	if err != nil {
		return nil, err
	}
	
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	
	return result, nil
}

// Delete 删除设置
func (r *settingsRepository) Delete(key string) error {
	return r.db.Where("key = ?", key).Delete(&models.SystemSetting{}).Error
}

// GetDefaultPushInterval 获取默认推送间隔
func (r *settingsRepository) GetDefaultPushInterval() int {
	return r.GetInt(SettingKeyDefaultPushInterval, DefaultPushInterval)
}

// SetDefaultPushInterval 设置默认推送间隔
func (r *settingsRepository) SetDefaultPushInterval(intervalMs int) error {
	return r.SetInt(SettingKeyDefaultPushInterval, intervalMs, "默认推送间隔（毫秒）")
}

// GetDataRetentionDays 获取数据保留天数
func (r *settingsRepository) GetDataRetentionDays() int {
	return r.GetInt(SettingKeyDataRetentionDays, DefaultDataRetentionDays)
}

// SetDataRetentionDays 设置数据保留天数
func (r *settingsRepository) SetDataRetentionDays(days int) error {
	return r.SetInt(SettingKeyDataRetentionDays, days, "数据保留天数")
}

// GetFrontendRefreshInterval 获取前端刷新间隔
func (r *settingsRepository) GetFrontendRefreshInterval() int {
	return r.GetInt(SettingKeyFrontendRefreshInterval, DefaultFrontendRefresh)
}

// SetFrontendRefreshInterval 设置前端刷新间隔
func (r *settingsRepository) SetFrontendRefreshInterval(seconds int) error {
	return r.SetInt(SettingKeyFrontendRefreshInterval, seconds, "前端刷新间隔（秒）")
}

// GetDeviceOfflineTimeout 获取设备离线超时
func (r *settingsRepository) GetDeviceOfflineTimeout() int {
	return r.GetInt(SettingKeyDeviceOfflineTimeout, DefaultOfflineTimeout)
}

// SetDeviceOfflineTimeout 设置设备离线超时
func (r *settingsRepository) SetDeviceOfflineTimeout(seconds int) error {
	return r.SetInt(SettingKeyDeviceOfflineTimeout, seconds, "设备离线超时（秒）")
}

// GetFollowPushInterval 获取是否跟随推送间隔
func (r *settingsRepository) GetFollowPushInterval() bool {
	return r.GetBool(SettingKeyFollowPushInterval, DefaultFollowPush)
}

// SetFollowPushInterval 设置是否跟随推送间隔
func (r *settingsRepository) SetFollowPushInterval(follow bool) error {
	return r.SetBool(SettingKeyFollowPushInterval, follow, "前端刷新是否跟随推送间隔")
}

// InitDefaults 初始化默认设置
func (r *settingsRepository) InitDefaults() error {
	intDefaults := []struct {
		Key         string
		Value       int
		Description string
	}{
		{SettingKeyDefaultPushInterval, DefaultPushInterval, "默认推送间隔（毫秒）"},
		{SettingKeyDataRetentionDays, DefaultDataRetentionDays, "数据保留天数"},
		{SettingKeyFrontendRefreshInterval, DefaultFrontendRefresh, "前端刷新间隔（秒）"},
		{SettingKeyDeviceOfflineTimeout, DefaultOfflineTimeout, "设备离线超时（秒）"},
	}
	
	for _, d := range intDefaults {
		// 只在不存在时创建
		existing, _ := r.Get(d.Key)
		if existing == "" {
			if err := r.SetInt(d.Key, d.Value, d.Description); err != nil {
				return err
			}
		}
	}
	
	// 初始化布尔类型设置
	existing, _ := r.Get(SettingKeyFollowPushInterval)
	if existing == "" {
		if err := r.SetBool(SettingKeyFollowPushInterval, DefaultFollowPush, "前端刷新是否跟随推送间隔"); err != nil {
			return err
		}
	}
	
	return nil
}
