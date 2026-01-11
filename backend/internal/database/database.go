package database

import (
	"fmt"
	"log"
	"nmp-platform/internal/config"
	"nmp-platform/internal/models"
	"nmp-platform/internal/service"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB 全局数据库实例
var DB *gorm.DB

// Connect 连接到PostgreSQL数据库
func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)

	// 配置GORM日志（生产环境使用 Warn 级别，开发环境使用 Info 级别）
	logLevel := logger.Warn
	if os.Getenv("GIN_MODE") != "release" {
		logLevel = logger.Info
	}
	
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// 设置连接池参数（可根据环境调整）
	// 生产环境建议：MaxIdleConns=25, MaxOpenConns=100
	// 开发环境建议：MaxIdleConns=10, MaxOpenConns=50
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Println("Successfully connected to PostgreSQL database")
	return db, nil
}

// Migrate 执行数据库迁移
func Migrate(db *gorm.DB) error {
	log.Println("Starting database migration...")

	// 执行自动迁移
	if err := models.AutoMigrate(db); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	// 创建额外的索引
	if err := models.CreateIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// SeedData 初始化基础数据
func SeedData(db *gorm.DB) error {
	log.Println("Starting database seeding...")

	// 创建默认权限
	if err := seedPermissions(db); err != nil {
		return fmt.Errorf("failed to seed permissions: %w", err)
	}

	// 创建默认角色
	if err := seedRoles(db); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	// 创建默认管理员用户
	if err := seedAdminUser(db); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	// 创建系统配置
	if err := seedSystemConfigs(db); err != nil {
		return fmt.Errorf("failed to seed system configs: %w", err)
	}

	log.Println("Database seeding completed successfully")
	return nil
}

// seedPermissions 创建默认权限
func seedPermissions(db *gorm.DB) error {
	permissions := []models.Permission{
		// 用户管理权限
		{Resource: "user", Action: "create", Scope: "all", Description: "创建用户"},
		{Resource: "user", Action: "read", Scope: "all", Description: "查看用户"},
		{Resource: "user", Action: "update", Scope: "all", Description: "更新用户"},
		{Resource: "user", Action: "delete", Scope: "all", Description: "删除用户"},
		{Resource: "user", Action: "read", Scope: "own", Description: "查看自己的信息"},
		{Resource: "user", Action: "update", Scope: "own", Description: "更新自己的信息"},

		// 角色管理权限
		{Resource: "role", Action: "create", Scope: "all", Description: "创建角色"},
		{Resource: "role", Action: "read", Scope: "all", Description: "查看角色"},
		{Resource: "role", Action: "update", Scope: "all", Description: "更新角色"},
		{Resource: "role", Action: "delete", Scope: "all", Description: "删除角色"},

		// 设备管理权限
		{Resource: "device", Action: "create", Scope: "all", Description: "创建设备"},
		{Resource: "device", Action: "read", Scope: "all", Description: "查看设备"},
		{Resource: "device", Action: "update", Scope: "all", Description: "更新设备"},
		{Resource: "device", Action: "delete", Scope: "all", Description: "删除设备"},

		// 插件管理权限
		{Resource: "plugin", Action: "create", Scope: "all", Description: "安装插件"},
		{Resource: "plugin", Action: "read", Scope: "all", Description: "查看插件"},
		{Resource: "plugin", Action: "update", Scope: "all", Description: "更新插件"},
		{Resource: "plugin", Action: "delete", Scope: "all", Description: "卸载插件"},

		// 系统管理权限
		{Resource: "system", Action: "config", Scope: "all", Description: "系统配置"},
		{Resource: "system", Action: "monitor", Scope: "all", Description: "系统监控"},
	}

	for _, permission := range permissions {
		var existing models.Permission
		result := db.Where("resource = ? AND action = ? AND scope = ?", 
			permission.Resource, permission.Action, permission.Scope).First(&existing)
		
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&permission).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// seedRoles 创建默认角色
func seedRoles(db *gorm.DB) error {
	// 获取所有权限
	var allPermissions []models.Permission
	if err := db.Find(&allPermissions).Error; err != nil {
		return err
	}

	// 管理员权限（所有权限）
	adminRole := models.Role{
		Name:        "admin",
		DisplayName: "系统管理员",
		Description: "拥有系统所有权限的管理员角色",
		IsSystem:    true,
		Permissions: allPermissions,
	}

	var existingAdmin models.Role
	result := db.Where("name = ?", "admin").First(&existingAdmin)
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&adminRole).Error; err != nil {
			return err
		}
	}

	// 操作员权限（设备管理和查看）
	var operatorPermissions []models.Permission
	db.Where("resource IN ? AND action IN ?", 
		[]string{"device", "user"}, 
		[]string{"read", "update"}).Find(&operatorPermissions)

	operatorRole := models.Role{
		Name:        "operator",
		DisplayName: "操作员",
		Description: "设备管理和基本操作权限",
		IsSystem:    true,
		Permissions: operatorPermissions,
	}

	var existingOperator models.Role
	result = db.Where("name = ?", "operator").First(&existingOperator)
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&operatorRole).Error; err != nil {
			return err
		}
	}

	// 查看者权限（只读）
	var viewerPermissions []models.Permission
	db.Where("action = ?", "read").Find(&viewerPermissions)

	viewerRole := models.Role{
		Name:        "viewer",
		DisplayName: "查看者",
		Description: "只读权限，可以查看但不能修改",
		IsSystem:    true,
		Permissions: viewerPermissions,
	}

	var existingViewer models.Role
	result = db.Where("name = ?", "viewer").First(&existingViewer)
	if result.Error == gorm.ErrRecordNotFound {
		if err := db.Create(&viewerRole).Error; err != nil {
			return err
		}
	}

	return nil
}

// seedAdminUser 创建默认管理员用户
func seedAdminUser(db *gorm.DB) error {
	var existingUser models.User
	result := db.Where("username = ?", "admin").First(&existingUser)
	
	if result.Error == gorm.ErrRecordNotFound {
		// 获取管理员角色
		var adminRole models.Role
		if err := db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
			return err
		}

		// 使用 PasswordService 哈希密码
		passwordService := service.GetPasswordService()
		hashedPassword, err := passwordService.Hash("admin123")
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		// 创建管理员用户（密码已哈希）
		adminUser := models.User{
			Username: "admin",
			Password: hashedPassword, // bcrypt 哈希后的密码
			Email:    "admin@nmp.local",
			FullName: "系统管理员",
			Status:   models.UserStatusActive,
			Roles:    []models.Role{adminRole},
		}

		if err := db.Create(&adminUser).Error; err != nil {
			return err
		}
		
		log.Println("Created admin user with hashed password")
		log.Println("WARNING: Default admin password is 'admin123'. Please change it immediately in production!")
	}

	return nil
}

// seedSystemConfigs 创建系统配置
func seedSystemConfigs(db *gorm.DB) error {
	configs := []models.SystemConfig{
		{
			Key:         "system.name",
			Value:       "网络监控平台",
			Type:        "string",
			Category:    "system",
			Description: "系统名称",
			IsPublic:    true,
		},
		{
			Key:         "system.version",
			Value:       "1.0.0",
			Type:        "string",
			Category:    "system",
			Description: "系统版本",
			IsPublic:    true,
		},
		{
			Key:         "system.theme",
			Value:       "light",
			Type:        "string",
			Category:    "system",
			Description: "默认主题",
			IsPublic:    true,
		},
		{
			Key:         "monitoring.data_retention_days",
			Value:       "30",
			Type:        "number",
			Category:    "monitoring",
			Description: "监控数据保留天数",
			IsPublic:    false,
		},
		{
			Key:         "monitoring.collection_interval",
			Value:       "60",
			Type:        "number",
			Category:    "monitoring",
			Description: "数据采集间隔（秒）",
			IsPublic:    false,
		},
	}

	for _, config := range configs {
		var existing models.SystemConfig
		result := db.Where("key = ?", config.Key).First(&existing)
		
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&config).Error; err != nil {
				return err
			}
		}
	}

	// 创建监控系统设置（使用 system_settings 表）
	monitoringSettings := []models.SystemSetting{
		{
			Key:         "default_push_interval",
			Value:       "1000",
			Description: "默认推送间隔（毫秒）",
		},
		{
			Key:         "data_retention_days",
			Value:       "10",
			Description: "数据保留天数",
		},
		{
			Key:         "frontend_refresh_interval",
			Value:       "10",
			Description: "前端刷新间隔（秒）",
		},
		{
			Key:         "device_offline_timeout",
			Value:       "60",
			Description: "设备离线超时时间（秒）",
		},
	}

	for _, setting := range monitoringSettings {
		var existing models.SystemSetting
		result := db.Where("key = ?", setting.Key).First(&existing)
		
		if result.Error == gorm.ErrRecordNotFound {
			if err := db.Create(&setting).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}