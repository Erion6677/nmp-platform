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

func TestConfigManagerIntegration(t *testing.T) {
	logger := zap.NewNop()
	
	// 创建临时配置文件
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
    monitoring:
      enabled: true
      interval: 60
`
	
	err := os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)
	
	// 创建配置管理器
	cm, err := NewConfigManager(configPath, logger)
	require.NoError(t, err)
	defer cm.Stop()
	
	// 启动配置管理器
	err = cm.Start()
	require.NoError(t, err)
	
	// 验证初始配置
	config := cm.GetConfig()
	assert.Equal(t, "localhost", config.Server.Host)
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "debug", config.Server.Mode)
	
	// 获取配置摘要
	summary := cm.GetConfigSummary()
	assert.Equal(t, "localhost", summary["server"].(map[string]interface{})["host"])
	assert.Equal(t, 8080, summary["server"].(map[string]interface{})["port"])
	
	// 测试热重载支持检查
	assert.True(t, cm.IsHotReloadSupported("server.mode"))
	assert.False(t, cm.IsHotReloadSupported("server.host"))
	assert.False(t, cm.IsHotReloadSupported("server.port"))
	
	// 获取可热重载配置
	hotConfig := cm.GetHotReloadableConfig()
	assert.True(t, hotConfig.DebugMode)
	assert.Contains(t, hotConfig.PluginConfigs, "monitoring")
	
	// 创建自定义配置变更处理器
	customHandler := NewMockConfigChangeHandler("custom_handler")
	cm.Subscribe(customHandler)
	
	// 修改配置文件（热重载测试）
	updatedConfig := `
server:
  host: "localhost"
  port: 8080
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
  token_expiry: "48h"
  refresh_expiry: "336h"

plugins:
  directory: "./plugins"
  configs:
    monitoring:
      enabled: false
      interval: 120
    alerting:
      enabled: true
      channels: ["email"]
`
	
	err = os.WriteFile(configPath, []byte(updatedConfig), 0644)
	require.NoError(t, err)
	
	// 等待配置热重载
	time.Sleep(1 * time.Second)
	
	// 验证配置已更新
	newConfig := cm.GetConfig()
	assert.Equal(t, "release", newConfig.Server.Mode)
	assert.Equal(t, 48*time.Hour, newConfig.Auth.TokenExpiry)
	
	// 验证自定义处理器被调用
	assert.Equal(t, 1, customHandler.changeCount)
	
	// 测试手动重新加载
	err = cm.ReloadConfig()
	assert.NoError(t, err)
	
	// 测试配置验证
	err = cm.ValidateConfig()
	assert.NoError(t, err)
	
	// 取消订阅
	cm.Unsubscribe("custom_handler")
	
	// 再次修改配置
	err = os.WriteFile(configPath, []byte(initialConfig), 0644)
	require.NoError(t, err)
	
	// 等待配置变更
	time.Sleep(1 * time.Second)
	
	// 验证自定义处理器没有再次被调用
	assert.Equal(t, 1, customHandler.changeCount)
}

func TestConfigManagerErrorHandling(t *testing.T) {
	logger := zap.NewNop()
	
	// 创建有效的配置管理器
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	
	validConfig := `
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
  configs: {}
`
	
	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	require.NoError(t, err)
	
	cm, err := NewConfigManager(configPath, logger)
	require.NoError(t, err)
	defer cm.Stop()
	
	err = cm.Start()
	require.NoError(t, err)
	
	// 测试错误处理器
	errorHandler := NewMockConfigChangeHandler("error_handler")
	errorHandler.SetShouldError(true)
	cm.Subscribe(errorHandler)
	
	// 修改配置文件
	invalidConfig := `
server:
  host: "localhost"
  port: 70000  # 无效端口
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
  configs: {}
`
	
	err = os.WriteFile(configPath, []byte(invalidConfig), 0644)
	require.NoError(t, err)
	
	// 等待配置变更处理
	time.Sleep(1 * time.Second)
	
	// 配置应该没有更新（因为验证失败）
	config := cm.GetConfig()
	assert.Equal(t, 8080, config.Server.Port) // 仍然是原来的端口
}