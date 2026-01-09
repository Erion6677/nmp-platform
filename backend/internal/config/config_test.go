package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// 测试加载默认配置
	config, err := Load()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// 验证默认值
	assert.Equal(t, "0.0.0.0", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "debug", config.Server.Mode)
}

func TestEnvironmentVariableOverride(t *testing.T) {
	// 设置环境变量
	os.Setenv("NMP_SERVER_HOST", "127.0.0.1")
	os.Setenv("NMP_SERVER_PORT", "9090")
	defer func() {
		os.Unsetenv("NMP_SERVER_HOST")
		os.Unsetenv("NMP_SERVER_PORT")
	}()

	config, err := Load()
	require.NoError(t, err)

	// 验证环境变量覆盖
	assert.Equal(t, "127.0.0.1", config.Server.Host)
	assert.Equal(t, 9090, config.Server.Port)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				Server: ServerConfig{
					Host:         "localhost",
					Port:         8080,
					Mode:         "debug",
					ReadTimeout:  30 * time.Second,
					WriteTimeout: 30 * time.Second,
				},
				Database: DatabaseConfig{
					Host:     "localhost",
					Port:     5432,
					Database: "test",
					Username: "user",
					Password: "pass",
					SSLMode:  "disable",
				},
				Redis: RedisConfig{
					Host: "localhost",
					Port: 6379,
					DB:   0,
				},
				InfluxDB: InfluxConfig{
					URL:    "http://localhost:8086",
					Org:    "test",
					Bucket: "test",
				},
				Auth: AuthConfig{
					JWTSecret:     "this-is-a-very-long-secret-key-for-testing-purposes",
					TokenExpiry:   24 * time.Hour,
					RefreshExpiry: 168 * time.Hour,
				},
				Plugins: PluginConfigs{
					Directory: "./plugins",
				},
			},
			expectError: false,
		},
		{
			name: "invalid port",
			config: Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 70000, // 无效端口
					Mode: "debug",
				},
			},
			expectError: true,
		},
		{
			name: "invalid mode",
			config: Config{
				Server: ServerConfig{
					Host: "localhost",
					Port: 8080,
					Mode: "invalid", // 无效模式
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCustomValidationRules(t *testing.T) {
	config := &Config{
		Auth: AuthConfig{
			JWTSecret:     "short", // 太短的密钥
			TokenExpiry:   24 * time.Hour,
			RefreshExpiry: 12 * time.Hour, // 刷新时间小于令牌时间
		},
		Plugins: PluginConfigs{
			Directory: "/nonexistent/directory",
		},
	}

	err := validateCustomRules(config)
	assert.Error(t, err)
}

func TestHelperFunctions(t *testing.T) {
	// 测试设置和获取配置
	testConfig := &Config{
		Server: ServerConfig{
			Host: "test-host",
			Port: 9999,
		},
	}

	SetConfig(testConfig)
	retrievedConfig := GetConfig()
	assert.Equal(t, testConfig, retrievedConfig)
}