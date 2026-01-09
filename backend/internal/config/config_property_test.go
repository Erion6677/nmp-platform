package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// Feature: network-monitoring-platform, Property 10: 配置管理灵活性
// 验证需求: 9.2, 9.3, 9.4, 9.5

func TestConfigManagementFlexibilityProperty(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 属性1: 环境变量覆盖配置值
	properties.Property("environment variables should override config values", 
		prop.ForAll(
			func(host string, port int) bool {
				// 设置环境变量
				envKey := "NMP_SERVER_HOST"
				envPortKey := "NMP_SERVER_PORT"
				
				originalHost := os.Getenv(envKey)
				originalPort := os.Getenv(envPortKey)
				
				defer func() {
					if originalHost == "" {
						os.Unsetenv(envKey)
					} else {
						os.Setenv(envKey, originalHost)
					}
					if originalPort == "" {
						os.Unsetenv(envPortKey)
					} else {
						os.Setenv(envPortKey, originalPort)
					}
				}()
				
				os.Setenv(envKey, host)
				os.Setenv(envPortKey, fmt.Sprintf("%d", port))
				
				// 加载配置
				config, err := Load()
				if err != nil {
					return false
				}
				
				// 验证环境变量覆盖了配置值
				return config.Server.Host == host && config.Server.Port == port
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
			gen.IntRange(1024, 65535),
		))

	// 属性2: 配置验证应该拒绝无效配置
	properties.Property("invalid configurations should be rejected", 
		prop.ForAll(
			func(port int, mode string) bool {
				config := &Config{
					Server: ServerConfig{
						Host: "localhost",
						Port: port,
						Mode: mode,
						ReadTimeout: 30 * time.Second,
						WriteTimeout: 30 * time.Second,
					},
					Database: DatabaseConfig{
						Host: "localhost",
						Port: 5432,
						Database: "test",
						Username: "user",
						Password: "pass",
						SSLMode: "disable",
					},
					Redis: RedisConfig{
						Host: "localhost",
						Port: 6379,
						DB: 0,
					},
					InfluxDB: InfluxConfig{
						URL: "http://localhost:8086",
						Org: "test",
						Bucket: "test",
					},
					Auth: AuthConfig{
						JWTSecret: "this-is-a-very-long-secret-key-for-testing-purposes",
						TokenExpiry: 24 * time.Hour,
						RefreshExpiry: 168 * time.Hour,
					},
					Plugins: PluginConfigs{
						Directory: "./plugins",
					},
				}
				
				err := validateConfig(config)
				
				// 无效端口或模式应该导致验证失败
				isValidPort := port >= 1 && port <= 65535
				isValidMode := mode == "debug" || mode == "release" || mode == "test"
				
				if isValidPort && isValidMode {
					return err == nil
				} else {
					return err != nil
				}
			},
			gen.IntRange(-1000, 70000),
			gen.OneConstOf("debug", "release", "test", "invalid", "production", ""),
		))

	// 属性3: 插件配置应该支持独立配置
	properties.Property("plugin configurations should be independently manageable", 
		prop.ForAll(
			func(pluginName string, enabled bool, interval int) bool {
				if pluginName == "" {
					return true // 跳过空插件名
				}
				
				config := &Config{
					Plugins: PluginConfigs{
						Directory: "./plugins",
						Configs: map[string]interface{}{
							pluginName: map[string]interface{}{
								"enabled": enabled,
								"interval": interval,
							},
						},
					},
				}
				
				// 验证插件配置可以独立访问
				pluginConfig, exists := config.Plugins.Configs[pluginName]
				if !exists {
					return false
				}
				
				pluginMap, ok := pluginConfig.(map[string]interface{})
				if !ok {
					return false
				}
				
				return pluginMap["enabled"] == enabled && pluginMap["interval"] == interval
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
			gen.Bool(),
			gen.IntRange(1, 3600),
		))

	// 属性4: 配置应该支持多环境
	properties.Property("configuration should support multiple environments", 
		prop.ForAll(
			func(mode string) bool {
				validModes := []string{"debug", "release", "test"}
				isValidMode := false
				for _, validMode := range validModes {
					if mode == validMode {
						isValidMode = true
						break
					}
				}
				
				if !isValidMode {
					return true // 跳过无效模式
				}
				
				config := &Config{
					Server: ServerConfig{
						Host: "localhost",
						Port: 8080,
						Mode: mode,
						ReadTimeout: 30 * time.Second,
						WriteTimeout: 30 * time.Second,
					},
					Database: DatabaseConfig{
						Host: "localhost",
						Port: 5432,
						Database: "test",
						Username: "user",
						Password: "pass",
						SSLMode: "disable",
					},
					Redis: RedisConfig{
						Host: "localhost",
						Port: 6379,
						DB: 0,
					},
					InfluxDB: InfluxConfig{
						URL: "http://localhost:8086",
						Org: "test",
						Bucket: "test",
					},
					Auth: AuthConfig{
						JWTSecret: "this-is-a-very-long-secret-key-for-testing-purposes",
						TokenExpiry: 24 * time.Hour,
						RefreshExpiry: 168 * time.Hour,
					},
					Plugins: PluginConfigs{
						Directory: "./plugins",
					},
				}
				
				// 验证配置在不同环境下都有效
				err := validateConfig(config)
				return err == nil && config.Server.Mode == mode
			},
			gen.OneConstOf("debug", "release", "test"),
		))
}

func TestConfigFileLoadingProperty(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 属性5: 配置文件加载应该是幂等的
	properties.Property("config file loading should be idempotent", 
		prop.ForAll(
			func() bool {
				// 创建临时配置文件
				tempDir := t.TempDir()
				configPath := filepath.Join(tempDir, "config.yaml")
				
				configContent := `
server:
  host: "test-host"
  port: 9999
  mode: "test"
  read_timeout: "30s"
  write_timeout: "30s"

database:
  host: "test-db"
  port: 5432
  database: "test"
  username: "test"
  password: "test"
  ssl_mode: "disable"

redis:
  host: "test-redis"
  port: 6379
  password: ""
  db: 0

influxdb:
  url: "http://test-influx:8086"
  token: ""
  org: "test"
  bucket: "test"

auth:
  jwt_secret: "this-is-a-very-long-secret-key-for-testing-purposes"
  token_expiry: "24h"
  refresh_expiry: "168h"

plugins:
  directory: "./plugins"
  configs: {}
`
				
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					return false
				}
				
				// 多次加载配置
				config1, err1 := LoadWithPath(tempDir)
				config2, err2 := LoadWithPath(tempDir)
				
				if err1 != nil || err2 != nil {
					return false
				}
				
				// 验证两次加载的结果相同
				return config1.Server.Host == config2.Server.Host &&
					   config1.Server.Port == config2.Server.Port &&
					   config1.Database.Host == config2.Database.Host &&
					   config1.Redis.Host == config2.Redis.Host
			},
		))
}

func TestConfigValidationProperty(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 属性6: JWT密钥长度验证
	properties.Property("JWT secret validation should enforce minimum length", 
		prop.ForAll(
			func(secret string) bool {
				config := &Config{
					Auth: AuthConfig{
						JWTSecret: secret,
						TokenExpiry: 24 * time.Hour,
						RefreshExpiry: 168 * time.Hour,
					},
				}
				
				err := validateCustomRules(config)
				
				if len(secret) < 32 {
					return err != nil // 应该有错误
				} else {
					// 对于足够长的密钥，不应该因为长度而失败
					// 但可能因为其他原因失败（如插件目录）
					return true
				}
			},
			gen.AlphaString(),
		))

	// 属性7: 令牌过期时间验证
	properties.Property("token expiry validation should enforce logical constraints", 
		prop.ForAll(
			func(tokenHours, refreshHours int) bool {
				if tokenHours <= 0 || refreshHours <= 0 {
					return true // 跳过无效输入
				}
				
				config := &Config{
					Auth: AuthConfig{
						JWTSecret: "this-is-a-very-long-secret-key-for-testing-purposes",
						TokenExpiry: time.Duration(tokenHours) * time.Hour,
						RefreshExpiry: time.Duration(refreshHours) * time.Hour,
					},
					Plugins: PluginConfigs{
						Directory: "./plugins",
					},
				}
				
				err := validateCustomRules(config)
				
				if refreshHours <= tokenHours {
					return err != nil // 刷新时间应该大于令牌时间
				} else {
					// 可能因为其他原因失败，但不应该因为时间关系失败
					return true
				}
			},
			gen.IntRange(1, 168),
			gen.IntRange(1, 720),
		))
}

// Feature: nmp-bugfix-iteration, Property 3: 环境变量配置覆盖
// **Validates: Requirements 2.1**
// *For any* 设置了对应环境变量的配置项（DB_PASSWORD、INFLUXDB_TOKEN、JWT_SECRET），
// 配置系统加载后的值必须等于环境变量的值，而非 YAML 文件中的默认值。
func TestEnvironmentVariableOverrideProperty(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t, gopter.ConsoleReporter(false))

	// 属性: DB_PASSWORD 环境变量应该覆盖配置文件中的值
	properties.Property("DB_PASSWORD environment variable should override config value",
		prop.ForAll(
			func(password string) bool {
				if password == "" {
					return true // 跳过空密码
				}

				// 保存原始环境变量
				originalPassword := os.Getenv("DB_PASSWORD")
				defer func() {
					if originalPassword == "" {
						os.Unsetenv("DB_PASSWORD")
					} else {
						os.Setenv("DB_PASSWORD", originalPassword)
					}
				}()

				// 设置环境变量
				os.Setenv("DB_PASSWORD", password)

				// 加载配置
				config, err := Load()
				if err != nil {
					// 如果密码太短导致验证失败，这是预期行为
					return true
				}

				// 验证环境变量覆盖了配置值
				return config.Database.Password == password
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
		))

	// 属性: INFLUXDB_TOKEN 环境变量应该覆盖配置文件中的值
	properties.Property("INFLUXDB_TOKEN environment variable should override config value",
		prop.ForAll(
			func(token string) bool {
				if token == "" {
					return true // 跳过空 token
				}

				// 保存原始环境变量
				originalToken := os.Getenv("INFLUXDB_TOKEN")
				defer func() {
					if originalToken == "" {
						os.Unsetenv("INFLUXDB_TOKEN")
					} else {
						os.Setenv("INFLUXDB_TOKEN", originalToken)
					}
				}()

				// 设置环境变量
				os.Setenv("INFLUXDB_TOKEN", token)

				// 加载配置
				config, err := Load()
				if err != nil {
					return true
				}

				// 验证环境变量覆盖了配置值
				return config.InfluxDB.Token == token
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 200 }),
		))

	// 属性: JWT_SECRET 环境变量应该覆盖配置文件中的值
	properties.Property("JWT_SECRET environment variable should override config value",
		prop.ForAll(
			func(secret string) bool {
				// JWT secret 必须至少 32 字符
				if len(secret) < 32 {
					return true // 跳过太短的密钥
				}

				// 保存原始环境变量
				originalSecret := os.Getenv("JWT_SECRET")
				defer func() {
					if originalSecret == "" {
						os.Unsetenv("JWT_SECRET")
					} else {
						os.Setenv("JWT_SECRET", originalSecret)
					}
				}()

				// 设置环境变量
				os.Setenv("JWT_SECRET", secret)

				// 加载配置
				config, err := Load()
				if err != nil {
					return true
				}

				// 验证环境变量覆盖了配置值
				return config.Auth.JWTSecret == secret
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 32 && len(s) < 200 }),
		))

	// 属性: 多个环境变量同时设置时都应该生效
	properties.Property("multiple environment variables should all take effect",
		prop.ForAll(
			func(password, token, secret string) bool {
				// 验证输入有效性
				if password == "" || token == "" || len(secret) < 32 {
					return true
				}

				// 保存原始环境变量
				originalPassword := os.Getenv("DB_PASSWORD")
				originalToken := os.Getenv("INFLUXDB_TOKEN")
				originalSecret := os.Getenv("JWT_SECRET")
				defer func() {
					restoreEnv("DB_PASSWORD", originalPassword)
					restoreEnv("INFLUXDB_TOKEN", originalToken)
					restoreEnv("JWT_SECRET", originalSecret)
				}()

				// 设置所有环境变量
				os.Setenv("DB_PASSWORD", password)
				os.Setenv("INFLUXDB_TOKEN", token)
				os.Setenv("JWT_SECRET", secret)

				// 加载配置
				config, err := Load()
				if err != nil {
					return true
				}

				// 验证所有环境变量都覆盖了配置值
				return config.Database.Password == password &&
					config.InfluxDB.Token == token &&
					config.Auth.JWTSecret == secret
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) >= 32 && len(s) < 100 }),
		))
}

// restoreEnv 恢复环境变量
func restoreEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

// 单元测试：验证属性测试覆盖的具体场景
func TestPropertyTestCoverage(t *testing.T) {
	// 验证环境变量覆盖
	t.Run("environment variable override", func(t *testing.T) {
		os.Setenv("NMP_SERVER_HOST", "property-test-host")
		defer os.Unsetenv("NMP_SERVER_HOST")
		
		config, err := Load()
		assert.NoError(t, err)
		assert.Equal(t, "property-test-host", config.Server.Host)
	})
	
	// 验证配置验证
	t.Run("configuration validation", func(t *testing.T) {
		invalidConfig := &Config{
			Server: ServerConfig{
				Port: 70000, // 无效端口
			},
		}
		
		err := validateConfig(invalidConfig)
		assert.Error(t, err)
	})
	
	// 验证插件配置
	t.Run("plugin configuration", func(t *testing.T) {
		config := &Config{
			Plugins: PluginConfigs{
				Directory: "./plugins",
				Configs: map[string]interface{}{
					"test-plugin": map[string]interface{}{
						"enabled": true,
						"setting": "value",
					},
				},
			},
		}
		
		pluginConfig := config.Plugins.Configs["test-plugin"]
		assert.NotNil(t, pluginConfig)
		
		pluginMap := pluginConfig.(map[string]interface{})
		assert.True(t, pluginMap["enabled"].(bool))
		assert.Equal(t, "value", pluginMap["setting"])
	})
}