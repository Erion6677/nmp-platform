# NMP Platform 部署指南

## 概述

本文档描述如何在生产环境中部署NMP Platform。系统采用原生部署方式，不依赖Docker容器。

## 系统要求

### 硬件要求
- CPU: 4核心以上
- 内存: 8GB以上
- 存储: 100GB以上SSD
- 网络: 1Gbps以上

### 软件要求
- 操作系统: Ubuntu 20.04+ 或 CentOS 8+
- Go: 1.21+
- Node.js: 18+
- PostgreSQL: 15+
- Redis: 7+
- InfluxDB: 2.7+
- Nginx: 1.18+

## 部署步骤

### 1. 环境准备

#### 1.1 运行安装脚本
```bash
# 下载并运行一键安装脚本
sudo bash install.sh
```

#### 1.2 手动安装（可选）
如果自动安装失败，可以手动安装各组件：

```bash
# 安装系统依赖
sudo apt update
sudo apt install -y curl wget git unzip systemd

# 安装PostgreSQL
sudo apt install -y postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 安装Redis
sudo apt install -y redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# 安装InfluxDB
wget -qO- https://repos.influxdata.com/influxdb.key | sudo apt-key add -
echo "deb https://repos.influxdata.com/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/influxdb.list
sudo apt update
sudo apt install -y influxdb2

# 安装Nginx
sudo apt install -y nginx
```

### 2. 构建应用

#### 2.1 构建后端
```bash
cd backend
go mod download
make build
```

#### 2.2 构建前端
```bash
cd frontend
npm install
npm run build
```

### 3. 部署应用

#### 3.1 使用部署脚本
```bash
cd deployments
sudo bash deploy.sh
```

#### 3.2 手动部署
```bash
# 复制后端文件
sudo cp ../backend/bin/nmp-backend /opt/nmp/bin/
sudo chmod +x /opt/nmp/bin/nmp-backend

# 复制前端文件
sudo cp -r ../frontend/dist/* /var/www/html/

# 复制配置文件
sudo cp config.prod.yaml /etc/nmp/config.yaml

# 安装systemd服务
sudo cp nmp-backend.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable nmp-backend

# 配置Nginx
sudo cp nginx.conf /etc/nginx/sites-available/nmp
sudo ln -sf /etc/nginx/sites-available/nmp /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl restart nginx
```

### 4. 启动服务

```bash
# 启动NMP后端服务
sudo systemctl start nmp-backend

# 检查服务状态
sudo systemctl status nmp-backend

# 查看日志
sudo journalctl -u nmp-backend -f
```

## 配置说明

### 主配置文件
配置文件位置: `/etc/nmp/config.yaml`

重要配置项：
- 数据库连接信息
- Redis连接信息
- InfluxDB连接信息
- JWT密钥
- 插件配置

### 环境变量
可以通过环境变量覆盖配置文件中的设置：
- `DB_HOST`: 数据库主机
- `DB_PASSWORD`: 数据库密码
- `REDIS_PASSWORD`: Redis密码
- `INFLUXDB_TOKEN`: InfluxDB访问令牌
- `JWT_SECRET`: JWT密钥

## 服务管理

### 常用命令
```bash
# 启动服务
sudo systemctl start nmp-backend

# 停止服务
sudo systemctl stop nmp-backend

# 重启服务
sudo systemctl restart nmp-backend

# 查看状态
sudo systemctl status nmp-backend

# 查看日志
sudo journalctl -u nmp-backend -f

# 重新加载配置
sudo systemctl reload nmp-backend
```

### 日志文件
- 应用日志: `/var/log/nmp/nmp.log`
- 系统日志: `journalctl -u nmp-backend`
- Nginx日志: `/var/log/nginx/access.log`, `/var/log/nginx/error.log`

## 更新部署

### 1. 准备新版本
```bash
# 构建新版本
cd backend && make build
cd frontend && npm run build
```

### 2. 部署更新
```bash
cd deployments
sudo bash deploy.sh
```

### 3. 回滚（如果需要）
```bash
sudo bash deploy.sh rollback
```

## 监控和维护

### 健康检查
```bash
# 检查后端API
curl http://localhost:8080/api/health

# 检查前端
curl http://localhost/

# 检查数据库连接
sudo -u postgres psql -c "SELECT 1;"

# 检查Redis
redis-cli ping

# 检查InfluxDB
curl http://localhost:8086/ping
```

### 备份
系统会自动备份：
- 备份目录: `/var/lib/nmp/backups`
- 备份频率: 每次部署前自动备份
- 保留策略: 保留最近7个备份

手动备份：
```bash
# 备份数据库
sudo -u postgres pg_dump nmp > nmp_backup_$(date +%Y%m%d).sql

# 备份配置文件
sudo cp /etc/nmp/config.yaml /var/lib/nmp/backups/config_$(date +%Y%m%d).yaml
```

### 性能优化
1. 调整PostgreSQL配置
2. 优化Redis内存设置
3. 配置InfluxDB数据保留策略
4. 调整Nginx缓存设置

## 故障排除

### 常见问题

#### 1. 服务启动失败
```bash
# 查看详细错误信息
sudo journalctl -u nmp-backend -n 50

# 检查配置文件语法
/opt/nmp/bin/nmp-backend --config=/etc/nmp/config.yaml --check-config
```

#### 2. 数据库连接失败
```bash
# 检查PostgreSQL状态
sudo systemctl status postgresql

# 测试数据库连接
sudo -u postgres psql -c "SELECT version();"
```

#### 3. 前端页面无法访问
```bash
# 检查Nginx状态
sudo systemctl status nginx

# 测试Nginx配置
sudo nginx -t

# 检查文件权限
ls -la /var/www/html/
```

#### 4. API请求失败
```bash
# 检查后端服务状态
sudo systemctl status nmp-backend

# 检查端口监听
sudo netstat -tlnp | grep 8080

# 测试API连接
curl -v http://localhost:8080/api/health
```

## 安全建议

1. 定期更新系统和依赖包
2. 使用强密码和复杂的JWT密钥
3. 启用防火墙，只开放必要端口
4. 配置SSL/TLS证书
5. 定期备份数据
6. 监控系统日志和异常

## 联系支持

如果遇到问题，请：
1. 查看日志文件获取详细错误信息
2. 检查系统资源使用情况
3. 参考故障排除章节
4. 联系技术支持团队