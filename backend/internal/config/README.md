# 配置管理系统

网络监控平台的配置管理系统提供了强大的配置加载、验证和热重载功能。

## 功能特性

### 1. 配置文件加载
- 支持 YAML 格式配置文件
- 自动设置默认值
- 支持环境变量覆盖
- 配置验证和错误处理

### 2. 环境变量覆盖
系统支持通过环境变量覆盖配置文件中的值：

```bash
# 设置服务器配置
export NMP_SERVER_HOST=0.0.0.0
export NMP_SERVER_PORT=9090
export NMP_SERVER_MODE=release

# 设置数据库配置
export NMP_DATABASE_HOST=db.example.com
export NMP_DATABASE_PASSWORD=secret123

# 设置认证配置
export NMP_AUTH_JWT_SECRET=your-super-secret-key-here
```

### 3. 配置验证
- 结构体标签验证（使用 validator 库）
- 自定义验证规则
- 端口范围验证
- JWT 密钥长度验证
- 令牌过期时间逻辑验证

### 4. 配置热重载
- 文件系统监听
- 防抖动处理
- 配置变更通知
- 支持部分配置热重载

## 使用方法

### 基本使用

```go
package main

import (
    "nmp-platform/internal/config"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()
    
    // 加载配置
    cfg, err := config.Load()
    if err != nil {
        logger.Fatal("Failed to load config", zap.Error(err))
    }
    
    // 设置全局配置
    config.SetConfig(cfg)
    
    // 使用配置
    logger.Info("Server starting", 
        zap.String("host", cfg.Server.Host),
        zap.Int("port", cfg.Server.Port))
}
```

### 配置管理器使用

```go
// 创建配置管理器
configManager, err := config.NewConfigManager("./configs/config.yaml", logger)
if err != nil {
    logger.Fatal("Failed to create config manager", zap.Error(err))
}

// 启动配置管理器（包含热重载）
if err := configManager.Start(); err != nil {
    logger.Fatal("Failed to start config manager", zap.Error(err))
}
defer configManager.Stop()

// 订阅配置变更
customHandler := &MyConfigHandler{}
configManager.Subscribe(customHandler)
```

### 自定义配置变更处理器

```go
type MyConfigHandler struct {
    logger *zap.Logger
}

func (h *MyConfigHandler) OnConfigChange(oldConfig, newConfig *config.Config) error {
    // 处理配置变更
    if oldConfig.Server.Mode != newConfig.Server.Mode {
        h.logger.Info("Server mode changed", 
            zap.String("old", oldConfig.Server.Mode),
            zap.String("new", newConfig.Server.Mode))
    }
    return nil
}

func (h *MyConfigHandler) GetName() string {
    return "my_config_handler"
}
```

## 配置结构

### 服务器配置
```yaml
server:
  host: "0.0.0.0"          # 服务器监听地址
  port: 8080               # 服务器端口
  mode: "debug"            # 运行模式: debug, release, test
  read_timeout: "30s"      # 读取超时
  write_timeout: "30s"     # 写入超时
```

### 数据库配置
```yaml
database:
  host: "localhost"        # 数据库主机
  port: 5432              # 数据库端口
  database: "nmp"         # 数据库名称
  username: "nmp"         # 用户名
  password: "nmp123"      # 密码
  ssl_mode: "disable"     # SSL 模式
```

### Redis 配置
```yaml
redis:
  host: "localhost"       # Redis 主机
  port: 6379             # Redis 端口
  password: ""           # Redis 密码
  db: 0                  # Redis 数据库编号
```

### InfluxDB 配置
```yaml
influxdb:
  url: "http://localhost:8086"  # InfluxDB URL
  token: ""                     # 访问令牌
  org: "nmp"                   # 组织名称
  bucket: "monitoring"         # 存储桶名称
```

### 认证配置
```yaml
auth:
  jwt_secret: "your-secret-key"  # JWT 密钥（至少32字符）
  token_expiry: "24h"           # 令牌过期时间
  refresh_expiry: "168h"        # 刷新令牌过期时间
```

### 插件配置
```yaml
plugins:
  directory: "./plugins"        # 插件目录
  configs:                     # 插件配置
    monitoring:
      enabled: true
      interval: 60
    alerting:
      enabled: true
      channels: ["email", "webhook"]
```

## 热重载支持

### 支持热重载的配置项
- `server.mode` - 服务器运行模式
- `server.read_timeout` - 读取超时
- `server.write_timeout` - 写入超时
- `auth.token_expiry` - 令牌过期时间
- `auth.refresh_expiry` - 刷新令牌过期时间
- `plugins.configs` - 插件配置

### 不支持热重载的配置项（需要重启）
- `server.host` - 服务器地址
- `server.port` - 服务器端口
- `database.*` - 所有数据库配置
- `redis.*` - 所有 Redis 配置
- `influxdb.*` - 所有 InfluxDB 配置

## 环境变量映射

| 配置项 | 环境变量 |
|--------|----------|
| `server.host` | `NMP_SERVER_HOST` |
| `server.port` | `NMP_SERVER_PORT` |
| `server.mode` | `NMP_SERVER_MODE` |
| `database.host` | `NMP_DATABASE_HOST` |
| `database.port` | `NMP_DATABASE_PORT` |
| `database.database` | `NMP_DATABASE_NAME` |
| `database.username` | `NMP_DATABASE_USERNAME` |
| `database.password` | `NMP_DATABASE_PASSWORD` |
| `redis.host` | `NMP_REDIS_HOST` |
| `redis.port` | `NMP_REDIS_PORT` |
| `redis.password` | `NMP_REDIS_PASSWORD` |
| `auth.jwt_secret` | `NMP_AUTH_JWT_SECRET` |

## 验证规则

### 端口验证
- 端口范围：1-65535

### 服务器模式验证
- 允许的值：`debug`, `release`, `test`

### JWT 密钥验证
- 最小长度：32 字符

### 时间配置验证
- 刷新令牌过期时间必须大于访问令牌过期时间

## 错误处理

配置系统提供详细的错误信息：

```go
// 配置验证错误
config validation failed: validation errors: Field 'Port' failed validation: min; Field 'Mode' failed validation: oneof

// 配置文件读取错误
error reading config file: open config.yaml: no such file or directory

// 自定义验证错误
JWT secret must be at least 32 characters long
```

## 测试

运行配置系统测试：

```bash
# 运行所有配置测试
go test ./internal/config -v

# 运行属性测试
go test ./internal/config -run Property -v

# 运行集成测试
go test ./internal/config -run Integration -v
```

## 示例

查看 `examples/config_example.go` 获取完整的使用示例。