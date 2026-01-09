package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockConfigChangeHandler 模拟配置变更处理器
type MockConfigChangeHandler struct {
	name         string
	changeCount  int
	lastOldConfig *Config
	lastNewConfig *Config
	shouldError  bool
}

func NewMockConfigChangeHandler(name string) *MockConfigChangeHandler {
	return &MockConfigChangeHandler{
		name: name,
	}
}

func (m *MockConfigChangeHandler) OnConfigChange(oldConfig, newConfig *Config) error {
	m.changeCount++
	m.lastOldConfig = oldConfig
	m.lastNewConfig = newConfig
	
	if m.shouldError {
		return assert.AnError
	}
	
	return nil
}

func (m *MockConfigChangeHandler) GetName() string {
	return m.name
}

func (m *MockConfigChangeHandler) SetShouldError(shouldError bool) {
	m.shouldError = shouldError
}

func TestConfigWatcher(t *testing.T) {
	logger := zap.NewNop()
	
	// 创建临时目录和配置文件
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	initialConfig := `
server:
  host: "localhost"
  port: 8080
  mode: "debug"
  read_timeout: "30s"
  write_timeout: "30s"

database:
  host: "localhost"
  port: 5432
  database: "test"
  username: "test"
  password: "test"
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

influxdb:
  url: "http://localhost:8086"
  token: ""
  org: "test"
  bucket: "test"

auth:
  jwt_secret: "this-is-a-very-long-secret-key-for-testing-purposes"
  token_expiry: "24h"
  refresh_expiry: "168h"

plugins:
  directory: "./plugins"
  configs:
    test-plugin:
      enabled: true
      interval: 60
`
	
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)
	
	// 加载初始配置
	config, err := LoadWithPath(tempDir)
	require.NoError(t, err)
	SetConfig(config)
	
	// 创建配置监听器
	watcher, err := NewConfigWatcher(configPath, logger)
	require.NoError(t, err)
	defer watcher.Stop()
	
	// 创建模拟处理器
	handler1 := NewMockConfigChangeHandler("handler1")
	handler2 := NewMockConfigChangeHandler("handler2")
	
	// 订阅配置变更
	watcher.Subscribe(handler1)
	watcher.Subscribe(handler2)
	
	// 启动监听器
	err = watcher.Start()
	require.NoError(t, err)
	
	// 等待监听器启动
	time.Sleep(100 * time.Millisecond)
	
	// 修改配置文件
	updatedConfig := `
server:
  host: "localhost"
  port: 9090
  mode: "release"
  read_timeout: "60s"
  write_timeout: "60s"

database:
  host: "localhost"
  port: 5432
  database: "test"
  username: "test"
  password: "test"
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0

influxdb:
  url: "http://localhost:8086"
  token: ""
  org: "test"
  bucket: "test"

auth:
  jwt_secret: "this-is-a-very-long-secret-key-for-testing-purposes"
  token_expiry: "24h"
  refresh_expiry: "168h"

plugins:
  directory: "./plugins"
  configs:
    test-plugin:
      enabled: false
      interval: 120
`
	
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)
	
	// 等待配置变更处理
	time.Sleep(1 * time.Second)
	
	// 验证处理器被调用
	assert.Equal(t, 1, handler1.changeCount)
	assert.Equal(t, 1, handler2.changeCount)
	
	// 验证新配置被加载
	newConfig := GetConfig()
	assert.Equal(t, 9090, newConfig.Server.Port)
	assert.Equal(t, "release", newConfig.Server.Mode)
	
	// 测试取消订阅
	watcher.Unsubscribe("handler1")
	
	// 再次修改配置
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)
	
	// 等待配置变更处理
	time.Sleep(1 * time.Second)
	
	// 验证只有 handler2 被调用
	assert.Equal(t, 1, handler1.changeCount) // 没有增加
	assert.Equal(t, 2, handler2.changeCount) // 增加了
}

func TestConfigChangeHandlers(t *testing.T) {
	logger := zap.NewNop()
	
	oldConfig := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			Mode: "debug",
			ReadTimeout: 30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Plugins: PluginConfigs{
			Configs: map[string]interface{}{
				"plugin1": map[string]interface{}{
					"enabled": true,
					"setting": "value1",
				},
			},
		},
	}
	
	newConfig := &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			Mode: "release",
			ReadTimeout: 60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		Plugins: PluginConfigs{
			Configs: map[string]interface{}{
				"plugin1": map[string]interface{}{
					"enabled": false,
					"setting": "value2",
				},
				"plugin2": map[string]interface{}{
					"enabled": true,
				},
			},
		},
	}
	
	t.Run("LogLevelHandler", func(t *testing.T) {
		handler := NewLogLevelHandler(logger)
		err := handler.OnConfigChange(oldConfig, newConfig)
		assert.NoError(t, err)
		assert.Equal(t, "log_level_handler", handler.GetName())
	})
	
	t.Run("ServerConfigHandler", func(t *testing.T) {
		handler := NewServerConfigHandler(logger)
		err := handler.OnConfigChange(oldConfig, newConfig)
		assert.NoError(t, err)
		
		// 测试需要重启的配置变更
		configWithPortChange := &Config{
			Server: ServerConfig{
				Host: "localhost",
				Port: 9090, // 端口变更
				Mode: "release",
			},
		}
		
		err = handler.OnConfigChange(oldConfig, configWithPortChange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires restart")
	})
	
	t.Run("DatabaseConfigHandler", func(t *testing.T) {
		handler := NewDatabaseConfigHandler(logger)
		
		// 相同的数据库配置
		err := handler.OnConfigChange(oldConfig, newConfig)
		assert.NoError(t, err)
		
		// 数据库配置变更
		configWithDBChange := &Config{
			Database: DatabaseConfig{
				Host: "new-host", // 主机变更
			},
		}
		
		err = handler.OnConfigChange(oldConfig, configWithDBChange)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requires restart")
	})
	
	t.Run("ConfigNotificationHandler", func(t *testing.T) {
		handler := NewConfigNotificationHandler(logger)
		
		var receivedEvent ConfigChangeEvent
		handler.Subscribe(func(event ConfigChangeEvent) {
			receivedEvent = event
		})
		
		err := handler.OnConfigChange(oldConfig, newConfig)
		assert.NoError(t, err)
		
		// 等待异步通知
		time.Sleep(100 * time.Millisecond)
		
		assert.Equal(t, "config_changed", receivedEvent.Type)
		assert.Equal(t, oldConfig, receivedEvent.OldConfig)
		assert.Equal(t, newConfig, receivedEvent.NewConfig)
	})
}

func TestCompareConfigs(t *testing.T) {
	oldConfig := &Config{
		Server: ServerConfig{
			Mode: "debug",
			Port: 8080,
		},
		Auth: AuthConfig{
			TokenExpiry: 24 * time.Hour,
		},
		Plugins: PluginConfigs{
			Configs: map[string]interface{}{
				"plugin1": "config1",
			},
		},
	}
	
	newConfig := &Config{
		Server: ServerConfig{
			Mode: "release",
			Port: 9090,
		},
		Auth: AuthConfig{
			TokenExpiry: 48 * time.Hour,
		},
		Plugins: PluginConfigs{
			Configs: map[string]interface{}{
				"plugin1": "config1",
				"plugin2": "config2",
			},
		},
	}
	
	changes := CompareConfigs(oldConfig, newConfig)
	
	assert.Contains(t, changes, "server.mode")
	assert.Contains(t, changes, "server.port")
	assert.Contains(t, changes, "auth.token_expiry")
	assert.Contains(t, changes, "plugins.configs.count")
}

func TestExtractHotReloadableConfig(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Mode: "debug",
		},
		Plugins: PluginConfigs{
			Configs: map[string]interface{}{
				"plugin1": "config1",
			},
		},
	}
	
	hotConfig := ExtractHotReloadableConfig(config)
	
	assert.True(t, hotConfig.DebugMode)
	assert.Equal(t, config.Plugins.Configs, hotConfig.PluginConfigs)
}