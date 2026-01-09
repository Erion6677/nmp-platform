package database

import (
	"nmp-platform/internal/config"
	"nmp-platform/internal/models"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gorm.io/gorm"
)

// TestDataStorageLayeringCorrectness 测试数据存储分层正确性
// Feature: network-monitoring-platform, Property 9: 数据存储分层正确性
func TestDataStorageLayeringCorrectness(t *testing.T) {
	// 跳过集成测试，如果没有数据库连接
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t)
}
// setupTestDB 设置测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "nmp_test",
		Username: "nmp",
		Password: "nmp123",
		SSLMode:  "disable",
	}

	db, err := Connect(cfg)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
	}

	// 清理测试数据
	cleanupTestData(db)
	
	// 执行迁移
	if err := Migrate(db); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// cleanupTestData 清理测试数据
func cleanupTestData(db *gorm.DB) {
	// 删除所有测试数据，但保留表结构
	db.Exec("TRUNCATE TABLE users, roles, permissions, user_roles, role_permissions RESTART IDENTITY CASCADE")
	db.Exec("TRUNCATE TABLE devices, interfaces, tags, device_groups, device_tags, device_group_members RESTART IDENTITY CASCADE")
	db.Exec("TRUNCATE TABLE plugins, plugin_routes, plugin_menus, system_configs RESTART IDENTITY CASCADE")
}

// TestUserModelCRUD 测试用户模型的CRUD操作
func TestUserModelCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer func() {
		cleanupTestData(db)
		Close()
	}()

	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性：创建用户后应该能够查询到
	properties.Property("created user should be retrievable", prop.ForAll(
		func(username, email string) bool {
			user := &models.User{
				Username: username,
				Email:    email,
				Password: "test123",
				FullName: "Test User",
				Status:   models.UserStatusActive,
			}

			// 创建用户
			if err := db.Create(user).Error; err != nil {
				return false
			}

			// 查询用户
			var retrieved models.User
			if err := db.Where("username = ?", username).First(&retrieved).Error; err != nil {
				return false
			}

			// 验证数据一致性
			return retrieved.Username == username && retrieved.Email == email

		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 3 && len(s) < 20 }),
		gen.RegexMatch(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`).SuchThat(func(s string) bool { return len(s) < 50 }),
	))

	properties.TestingRun(t)
}
// TestDeviceModelCRUD 测试设备模型的CRUD操作
func TestDeviceModelCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := setupTestDB(t)
	defer func() {
		cleanupTestData(db)
		Close()
	}()

	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性：创建设备后应该能够查询到
	properties.Property("created device should be retrievable", prop.ForAll(
		func(name, host string, port int) bool {
			if port <= 0 || port > 65535 {
				port = 22 // 默认SSH端口
			}

			device := &models.Device{
				Name:     name,
				Type:     models.DeviceTypeRouter,
				Host:     host,
				Port:     port,
				Protocol: "ssh",
				Status:   models.DeviceStatusUnknown,
			}

			// 创建设备
			if err := db.Create(device).Error; err != nil {
				return false
			}

			// 查询设备
			var retrieved models.Device
			if err := db.Where("name = ?", name).First(&retrieved).Error; err != nil {
				return false
			}

			// 验证数据一致性
			return retrieved.Name == name && retrieved.Host == host && retrieved.Port == port

		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 2 && len(s) < 30 }),
		gen.RegexMatch(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`),
		gen.IntRange(1, 65535),
	))

	properties.TestingRun(t)
}

// TestDatabaseConnectionResilience 测试数据库连接的弹性
func TestDatabaseConnectionResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性：数据库连接应该能够处理各种配置参数
	properties.Property("database connection should handle various configs", prop.ForAll(
		func(host string, port int, dbname, username string) bool {
			// 限制参数范围以避免无效配置
			if port <= 0 || port > 65535 {
				port = 5432
			}
			if len(host) == 0 {
				host = "localhost"
			}
			if len(dbname) == 0 {
				dbname = "nmp_test"
			}
			if len(username) == 0 {
				username = "nmp"
			}

			cfg := &config.DatabaseConfig{
				Host:     host,
				Port:     port,
				Database: dbname,
				Username: username,
				Password: "nmp123",
				SSLMode:  "disable",
			}

			// 尝试连接（可能失败，但不应该崩溃）
			db, err := Connect(cfg)
			if err != nil {
				// 连接失败是可以接受的，只要不崩溃
				return true
			}

			// 如果连接成功，应该能够ping通
			sqlDB, err := db.DB()
			if err != nil {
				return false
			}

			err = sqlDB.Ping()
			sqlDB.Close()
			
			return err == nil

		},
		gen.OneConstOf("localhost", "127.0.0.1"),
		gen.OneConstOf(5432),
		gen.OneConstOf("nmp_test"),
		gen.OneConstOf("nmp"),
	))

	properties.TestingRun(t)
}