package repository

import (
	"testing"

	"nmp-platform/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Feature: nmp-bugfix-iteration, Property 4: 设置持久化一致性
// **Validates: Requirements 3.2**
// *For any* 通过 API 保存的采集设置，再次通过 API 获取时，返回的值必须与保存时的值完全一致。

// setupTestDB 创建测试用的内存数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// 自动迁移 SystemSetting 表
	err = db.AutoMigrate(&models.SystemSetting{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	return db
}

// CollectionSettings 采集设置结构体（用于测试）
type CollectionSettings struct {
	DefaultPushInterval     int
	DataRetentionDays       int
	FrontendRefreshInterval int
	DeviceOfflineTimeout    int
	FollowPushInterval      bool
}

func TestSettingsPersistenceProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性1: 整数设置的持久化一致性
	// 对于任意有效的整数值，保存后再读取应该得到相同的值
	properties.Property("integer settings should persist correctly",
		prop.ForAll(
			func(value int) bool {
				db := setupTestDB(t)
				repo := NewSettingsRepository(db)

				// 保存设置
				err := repo.SetInt("test_int_key", value, "test description")
				if err != nil {
					return false
				}

				// 读取设置
				retrieved := repo.GetInt("test_int_key", -1)

				// 验证一致性
				return retrieved == value
			},
			gen.IntRange(1, 1000000), // 生成 1 到 1000000 之间的整数
		))

	// 属性2: 布尔设置的持久化一致性
	// 对于任意布尔值，保存后再读取应该得到相同的值
	properties.Property("boolean settings should persist correctly",
		prop.ForAll(
			func(value bool) bool {
				db := setupTestDB(t)
				repo := NewSettingsRepository(db)

				// 保存设置
				err := repo.SetBool("test_bool_key", value, "test description")
				if err != nil {
					return false
				}

				// 读取设置
				retrieved := repo.GetBool("test_bool_key", !value) // 使用相反值作为默认值

				// 验证一致性
				return retrieved == value
			},
			gen.Bool(),
		))

	// 属性3: 字符串设置的持久化一致性
	// 对于任意非空字符串，保存后再读取应该得到相同的值
	properties.Property("string settings should persist correctly",
		prop.ForAll(
			func(value string) bool {
				if value == "" {
					return true // 跳过空字符串
				}

				db := setupTestDB(t)
				repo := NewSettingsRepository(db)

				// 保存设置
				err := repo.Set("test_string_key", value, "test description")
				if err != nil {
					return false
				}

				// 读取设置
				retrieved, err := repo.Get("test_string_key")
				if err != nil {
					return false
				}

				// 验证一致性
				return retrieved == value
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 1000 }),
		))

	// 属性4: 采集设置的完整持久化一致性（核心属性）
	// 对于任意有效的采集设置组合，保存后再读取应该得到完全相同的值
	properties.Property("collection settings should persist with full consistency",
		prop.ForAll(
			func(settings CollectionSettings) bool {
				db := setupTestDB(t)
				repo := NewSettingsRepository(db)

				// 保存所有采集设置
				if err := repo.SetDefaultPushInterval(settings.DefaultPushInterval); err != nil {
					return false
				}
				if err := repo.SetDataRetentionDays(settings.DataRetentionDays); err != nil {
					return false
				}
				if err := repo.SetFrontendRefreshInterval(settings.FrontendRefreshInterval); err != nil {
					return false
				}
				if err := repo.SetDeviceOfflineTimeout(settings.DeviceOfflineTimeout); err != nil {
					return false
				}
				if err := repo.SetFollowPushInterval(settings.FollowPushInterval); err != nil {
					return false
				}

				// 读取所有采集设置
				retrievedPushInterval := repo.GetDefaultPushInterval()
				retrievedRetentionDays := repo.GetDataRetentionDays()
				retrievedRefreshInterval := repo.GetFrontendRefreshInterval()
				retrievedOfflineTimeout := repo.GetDeviceOfflineTimeout()
				retrievedFollowPush := repo.GetFollowPushInterval()

				// 验证所有值的一致性
				return retrievedPushInterval == settings.DefaultPushInterval &&
					retrievedRetentionDays == settings.DataRetentionDays &&
					retrievedRefreshInterval == settings.FrontendRefreshInterval &&
					retrievedOfflineTimeout == settings.DeviceOfflineTimeout &&
					retrievedFollowPush == settings.FollowPushInterval
			},
			genCollectionSettings(),
		))

	// 属性5: 设置更新的持久化一致性
	// 对于任意两个不同的值，更新后应该得到新值而非旧值
	properties.Property("updated settings should persist the new value",
		prop.ForAll(
			func(oldValue, newValue int) bool {
				if oldValue == newValue {
					return true // 跳过相同值
				}

				db := setupTestDB(t)
				repo := NewSettingsRepository(db)

				// 保存旧值
				if err := repo.SetInt("test_update_key", oldValue, "test"); err != nil {
					return false
				}

				// 更新为新值
				if err := repo.SetInt("test_update_key", newValue, "test"); err != nil {
					return false
				}

				// 读取应该得到新值
				retrieved := repo.GetInt("test_update_key", -1)
				return retrieved == newValue
			},
			gen.IntRange(100, 10000),
			gen.IntRange(100, 10000),
		))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genCollectionSettings 生成有效的采集设置
func genCollectionSettings() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(100, 60000),   // DefaultPushInterval: 100ms - 60s
		gen.IntRange(1, 365),       // DataRetentionDays: 1 - 365 days
		gen.IntRange(3, 60),        // FrontendRefreshInterval: 3 - 60 seconds
		gen.IntRange(10, 3600),     // DeviceOfflineTimeout: 10s - 1 hour
		gen.Bool(),                 // FollowPushInterval
	).Map(func(values []interface{}) CollectionSettings {
		return CollectionSettings{
			DefaultPushInterval:     values[0].(int),
			DataRetentionDays:       values[1].(int),
			FrontendRefreshInterval: values[2].(int),
			DeviceOfflineTimeout:    values[3].(int),
			FollowPushInterval:      values[4].(bool),
		}
	})
}

// 单元测试：验证具体场景
func TestSettingsRepositoryUnit(t *testing.T) {
	t.Run("get non-existent key should return default", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSettingsRepository(db)

		value := repo.GetInt("non_existent_key", 42)
		assert.Equal(t, 42, value)
	})

	t.Run("get non-existent bool should return default", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSettingsRepository(db)

		value := repo.GetBool("non_existent_bool", true)
		assert.True(t, value)

		value = repo.GetBool("non_existent_bool", false)
		assert.False(t, value)
	})

	t.Run("init defaults should create all settings", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSettingsRepository(db)

		err := repo.InitDefaults()
		assert.NoError(t, err)

		// 验证默认值
		assert.Equal(t, DefaultPushInterval, repo.GetDefaultPushInterval())
		assert.Equal(t, DefaultDataRetentionDays, repo.GetDataRetentionDays())
		assert.Equal(t, DefaultFrontendRefresh, repo.GetFrontendRefreshInterval())
		assert.Equal(t, DefaultOfflineTimeout, repo.GetDeviceOfflineTimeout())
		assert.Equal(t, DefaultFollowPush, repo.GetFollowPushInterval())
	})

	t.Run("init defaults should not overwrite existing values", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSettingsRepository(db)

		// 先设置一个自定义值
		customValue := 5000
		err := repo.SetDefaultPushInterval(customValue)
		assert.NoError(t, err)

		// 初始化默认值
		err = repo.InitDefaults()
		assert.NoError(t, err)

		// 自定义值应该保留
		assert.Equal(t, customValue, repo.GetDefaultPushInterval())
	})

	t.Run("delete setting should remove it", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSettingsRepository(db)

		// 设置一个值
		err := repo.SetInt("to_delete", 123, "test")
		assert.NoError(t, err)

		// 验证存在
		assert.Equal(t, 123, repo.GetInt("to_delete", -1))

		// 删除
		err = repo.Delete("to_delete")
		assert.NoError(t, err)

		// 验证已删除（返回默认值）
		assert.Equal(t, -1, repo.GetInt("to_delete", -1))
	})

	t.Run("get all settings should return all", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSettingsRepository(db)

		// 设置多个值
		repo.SetInt("key1", 100, "desc1")
		repo.SetInt("key2", 200, "desc2")
		repo.SetBool("key3", true, "desc3")

		// 获取所有
		all, err := repo.GetAll()
		assert.NoError(t, err)
		assert.Len(t, all, 3)
		assert.Equal(t, "100", all["key1"])
		assert.Equal(t, "200", all["key2"])
		assert.Equal(t, "true", all["key3"])
	})
}
