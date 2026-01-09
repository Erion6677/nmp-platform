# 测试指南

## 概述

本目录包含网络监控平台的各种测试，包括集成测试、性能测试和备份功能测试。

## 测试类型

### 1. 集成测试 (Integration Tests)

位置: `tests/integration/`

集成测试验证系统各组件之间的交互，包括：
- 用户认证流程测试
- 插件加载和集成测试
- 端到端数据流测试
- 错误处理测试
- 并发请求测试
- 数据库连接测试

**运行集成测试:**
```bash
# 运行所有集成测试（需要数据库连接）
go test -v ./tests/integration/...

# 跳过集成测试（快速模式）
go test -v ./tests/integration/... -short
```

### 2. 性能测试 (Performance Tests)

位置: `tests/performance/`

性能测试评估系统在负载下的表现，包括：
- 数据推送吞吐量测试
- 数据查询性能测试
- 并发用户访问测试
- 内存使用测试

**运行性能测试:**
```bash
# 运行所有性能测试（需要数据库连接，耗时较长）
go test -v ./tests/performance/...

# 跳过性能测试（快速模式）
go test -v ./tests/performance/... -short
```

### 3. 备份功能测试 (Backup Tests)

位置: `internal/backup/`

备份功能测试验证数据备份和恢复功能，包括：
- 备份文件列表功能
- 旧备份清理功能
- 备份文件验证功能
- 调度器启停功能
- 备份状态查询功能

**运行备份测试:**
```bash
go test -v ./internal/backup/...
```

## 测试环境要求

### 集成测试和性能测试

这些测试需要完整的数据库环境：

1. **PostgreSQL** (端口 5432)
   - 数据库: `nmp_test` (集成测试), `nmp_perf_test` (性能测试)
   - 用户: `test`
   - 密码: `test`

2. **Redis** (端口 6379)
   - 数据库: 1 (集成测试), 2 (性能测试)

3. **InfluxDB** (端口 8086)
   - 组织: `test-org` (集成测试), `perf-test-org` (性能测试)
   - 存储桶: `test-bucket` (集成测试), `perf-test-bucket` (性能测试)

### 备份功能测试

备份测试需要系统安装以下工具：
- `pg_dump` (PostgreSQL客户端工具)
- `psql` (PostgreSQL客户端工具)

## 运行所有测试

```bash
# 运行所有单元测试和备份测试
go test -v ./internal/... -short

# 运行所有测试（包括集成和性能测试，需要完整环境）
go test -v ./...

# 运行特定包的测试
go test -v ./internal/auth/...
go test -v ./internal/backup/...
```

## 测试配置

测试使用独立的配置文件，避免影响开发环境：
- 集成测试: 临时配置文件 `test_config.yaml`
- 性能测试: 临时配置文件 `perf_config.yaml`

## 性能基准

性能测试包含以下基准要求：

### 数据推送性能
- 吞吐量: > 50 请求/秒
- 成功率: > 95%
- 平均延迟: < 100ms

### 数据查询性能
- 平均延迟: < 50ms
- 最大延迟: < 200ms

### 并发访问性能
- 并发用户: 50
- 成功率: > 98%
- 吞吐量: > 100 请求/秒
- 平均延迟: < 50ms

## 故障排除

### 常见问题

1. **数据库连接失败**
   - 确保PostgreSQL、Redis、InfluxDB服务正在运行
   - 检查连接参数是否正确
   - 确保测试数据库已创建

2. **权限问题**
   - 确保数据库用户有足够权限
   - 检查文件系统权限

3. **端口冲突**
   - 确保测试端口未被占用
   - 修改测试配置中的端口设置

### 调试模式

```bash
# 启用详细日志
go test -v ./tests/integration/... -args -log-level=debug

# 运行单个测试
go test -v ./tests/integration/... -run TestSystemIntegration_UserAuthenticationFlow
```