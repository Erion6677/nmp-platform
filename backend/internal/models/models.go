package models

import (
	"gorm.io/gorm"
)

// AllModels 返回所有需要迁移的模型
func AllModels() []interface{} {
	return []interface{}{
		// 用户相关模型
		&User{},
		&Role{},
		&Permission{},
		&UserRole{},
		&RolePermission{},

		// 设备相关模型
		&Device{},
		&Interface{},
		&Tag{},
		&DeviceGroup{},
		&DeviceTag{},
		&DeviceGroupMember{},

		// 监控相关模型
		&Proxy{},
		&CollectorScript{},
		&PingTarget{},
		&SystemSetting{},
		&UserDevicePermission{},

		// 插件相关模型
		&Plugin{},
		&PluginRoute{},
		&PluginMenu{},
		&SystemConfig{},
	}
}

// AutoMigrate 执行数据库迁移
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(AllModels()...)
}

// CreateIndexes 创建额外的索引
func CreateIndexes(db *gorm.DB) error {
	// 用户相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)").Error; err != nil {
		return err
	}

	// 设备相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_host ON devices(host)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_type ON devices(type)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_os_type ON devices(os_type)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_devices_proxy_id ON devices(proxy_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_interfaces_device_id ON interfaces(device_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_interfaces_status ON interfaces(status)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_interfaces_monitored ON interfaces(monitored)").Error; err != nil {
		return err
	}

	// 代理相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_proxies_type ON proxies(type)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_proxies_enabled ON proxies(enabled)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status)").Error; err != nil {
		return err
	}

	// 采集器相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_collector_scripts_status ON collector_scripts(status)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_collector_scripts_enabled ON collector_scripts(enabled)").Error; err != nil {
		return err
	}

	// Ping 目标相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_ping_targets_device_id ON ping_targets(device_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_ping_targets_enabled ON ping_targets(enabled)").Error; err != nil {
		return err
	}

	// 用户设备权限索引
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_device_permissions_unique ON user_device_permissions(user_id, device_id)").Error; err != nil {
		return err
	}

	// 插件相关索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_plugins_name ON plugins(name)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_plugins_status ON plugins(status)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_plugin_routes_plugin_id ON plugin_routes(plugin_id)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_plugin_menus_plugin_id ON plugin_menus(plugin_id)").Error; err != nil {
		return err
	}

	// 系统配置索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_system_configs_category ON system_configs(category)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_system_configs_is_public ON system_configs(is_public)").Error; err != nil {
		return err
	}

	return nil
}