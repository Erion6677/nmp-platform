# NMP Platform 开发指南

## 项目概述

NMP Platform（网络监控平台）是一个基于插件化架构的现代化网络设备监控系统，支持大规模设备监控（500-1000台设备）。

## 技术架构

### 后端技术栈
- **语言**: Go 1.21+
- **Web框架**: Gin
- **ORM**: GORM
- **权限管理**: Casbin (RBAC)
- **配置管理**: Viper
- **日志**: Zap
- **数据库**: PostgreSQL + InfluxDB + Redis

### 前端技术栈
- **框架**: Vue 3 + TypeScript
- **UI库**: NaiveUI
- **构建工具**: Vite
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **图表**: ECharts

## 开发环境搭建

### 环境要求
- Go 1.21+
- Node.js 18+
- PostgreSQL 13+
- Redis 7+
- InfluxDB 2.0+
- Redis 6.0+

### 快速开始

1. **克隆项目**
```bash
git clone <repository-url>
cd nmp-platform
```

2. **运行设置脚本**
```bash
./scripts/dev-setup.sh
```

3. **启动开发环境**
```bash
./start-dev.sh
```

### 手动设置

如果自动设置脚本失败，可以手动设置：

1. **启动数据库服务**
```bash
# 启动PostgreSQL
sudo systemctl start postgresql

# 启动Redis
sudo systemctl start redis-server

# 启动InfluxDB
sudo systemctl start influxdb
```

2. **设置后端**
```bash
cd backend
go mod tidy
go install github.com/cosmtrek/air@latest
air  # 热重载开发
```

3. **设置前端**
```bash
cd frontend
npm install
npm run dev
```

## 项目结构

```
nmp-platform/
├── backend/                 # Go后端项目
│   ├── cmd/                # 应用入口
│   │   └── server/         # 服务器主程序
│   ├── internal/           # 内部包
│   │   ├── config/         # 配置管理
│   │   ├── server/         # HTTP服务器
│   │   ├── auth/           # 认证服务
│   │   ├── plugin/         # 插件系统
│   │   └── models/         # 数据模型
│   ├── pkg/                # 公共包
│   ├── plugins/            # 插件目录
│   ├── configs/            # 配置文件
│   └── scripts/            # 脚本文件
├── frontend/               # Vue3前端项目
│   ├── src/                # 源代码
│   │   ├── components/     # 组件
│   │   ├── views/          # 页面
│   │   ├── stores/         # 状态管理
│   │   ├── router/         # 路由配置
│   │   └── styles/         # 样式文件
│   ├── public/             # 静态资源
│   └── plugins/            # 前端插件
├── docs/                   # 文档
├── scripts/                # 项目脚本
└── deployments/            # 部署配置
```

## 开发规范

### 代码规范

**Go代码规范**:
- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查
- 遵循 Go 官方编码规范
- 包名使用小写，不使用下划线
- 接口名以 `er` 结尾（如 `Handler`, `Manager`）

**TypeScript代码规范**:
- 使用 ESLint + Prettier 格式化代码
- 使用 TypeScript 严格模式
- 组件名使用 PascalCase
- 文件名使用 kebab-case

### Git 提交规范

使用 Conventional Commits 规范：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

类型说明：
- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

示例：
```
feat(auth): add JWT token validation
fix(device): resolve device connection timeout issue
docs: update API documentation
```

## 插件开发

### 插件架构

NMP Platform 采用插件化架构，支持动态加载和管理功能模块。

### 插件接口

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Dependencies() []string
    
    Initialize(ctx context.Context, config interface{}) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() error
    
    GetRoutes() []Route
    GetMenus() []MenuItem
    GetPermissions() []Permission
    GetConfigSchema() interface{}
}
```

### 创建插件

1. **创建插件目录**
```bash
mkdir -p backend/plugins/my-plugin
```

2. **实现插件接口**
```go
package main

import (
    "context"
    "nmp-platform/pkg/plugin"
)

type MyPlugin struct {
    config *Config
}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

// 实现其他接口方法...

func NewPlugin() plugin.Plugin {
    return &MyPlugin{}
}
```

3. **注册插件**
```go
func init() {
    plugin.Register("my-plugin", NewPlugin)
}
```

## API 开发

### RESTful API 规范

- 使用标准HTTP方法（GET, POST, PUT, DELETE）
- URL使用名词，避免动词
- 使用复数形式（如 `/api/v1/devices`）
- 使用HTTP状态码表示结果

### API响应格式

```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "timestamp": "2024-01-01T00:00:00Z"
}
```

错误响应：
```json
{
  "code": 400,
  "message": "validation failed",
  "errors": [
    {
      "field": "username",
      "message": "username is required"
    }
  ],
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## 测试

### 后端测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/auth

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 前端测试

```bash
# 运行单元测试
npm run test

# 运行E2E测试
npm run test:e2e

# 生成测试覆盖率报告
npm run test:coverage
```

## 部署

### 开发环境
```bash
./start-dev.sh
```

### 生产环境
```bash
# 构建应用
# 原生部署
make build

# 使用部署脚本
./deployments/deploy.sh
```

## 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查数据库服务是否启动
   - 验证连接配置
   - 检查防火墙设置

2. **前端编译失败**
   - 清除node_modules重新安装
   - 检查Node.js版本
   - 更新依赖包

3. **后端启动失败**
   - 检查Go版本
   - 验证配置文件
   - 查看日志文件

### 日志查看

```bash
# 查看后端日志
# 查看应用日志
tail -f backend/logs/app.log

# 查看系统服务日志
sudo journalctl -u nmp-backend -f
```

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

MIT License