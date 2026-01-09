# NMP Platform 安装和配置指南

## 目录
- [系统要求](#系统要求)
- [快速安装](#快速安装)
- [手动安装](#手动安装)
- [配置说明](#配置说明)
- [首次设置](#首次设置)
- [常见问题](#常见问题)

## 系统要求

### 最低配置
- **操作系统**: Ubuntu 20.04+ / CentOS 8+ / Debian 11+
- **CPU**: 2核心
- **内存**: 4GB RAM
- **存储**: 50GB 可用空间
- **网络**: 100Mbps

### 推荐配置
- **操作系统**: Ubuntu 22.04 LTS
- **CPU**: 4核心以上
- **内存**: 8GB RAM以上
- **存储**: 100GB SSD
- **网络**: 1Gbps

### 支持的设备规模
- **小型环境**: 50台设备以下
- **中型环境**: 50-200台设备
- **大型环境**: 200-1000台设备

## 快速安装

### 一键安装脚本

```bash
# 下载安装脚本
wget https://github.com/your-org/nmp-platform/releases/latest/download/install.sh

# 或者从项目目录复制
cp deployments/install.sh /tmp/

# 运行安装脚本
sudo bash install.sh
```

安装脚本会自动完成：
1. 检测操作系统版本
2. 安装系统依赖包
3. 创建用户和目录
4. 安装Go和Node.js环境
5. 安装数据库（PostgreSQL、Redis、InfluxDB）
6. 配置数据库和用户权限
7. 构建和部署应用
8. 配置系统服务
9. 启动所有服务

### 安装完成后

安装完成后，您将看到类似以下的信息：

```
=== 安装信息 ===
Web界面: http://192.168.1.100
后端API: http://192.168.1.100/api

=== 数据库信息 ===
PostgreSQL:
  数据库: nmp
  用户名: nmp
  密码: [随机生成的密码]

Redis密码: [随机生成的密码]
InfluxDB Token: [随机生成的Token]
```

**重要**: 请妥善保存这些密码和Token信息！

## 手动安装

如果自动安装失败，可以按照以下步骤手动安装。

### 1. 安装系统依赖

#### Ubuntu/Debian
```bash
sudo apt update
sudo apt install -y curl wget git unzip build-essential
```

#### CentOS/RHEL
```bash
sudo yum update -y
sudo yum install -y curl wget git unzip gcc gcc-c++ make
```

### 2. 安装Go语言环境

```bash
# 下载Go 1.21
cd /tmp
wget https://golang.org/dl/go1.21.5.linux-amd64.tar.gz

# 安装Go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# 设置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
source /etc/profile

# 验证安装
go version
```

### 3. 安装Node.js

```bash
# 安装Node.js 18
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# 验证安装
node --version
npm --version
```

### 4. 安装数据库

#### PostgreSQL
```bash
# Ubuntu/Debian
sudo apt install -y postgresql postgresql-contrib

# CentOS/RHEL
sudo yum install -y postgresql postgresql-server postgresql-contrib
sudo postgresql-setup initdb

# 启动服务
sudo systemctl start postgresql
sudo systemctl enable postgresql

# 创建数据库和用户
sudo -u postgres psql << EOF
CREATE USER nmp WITH PASSWORD 'your_password_here';
CREATE DATABASE nmp OWNER nmp;
GRANT ALL PRIVILEGES ON DATABASE nmp TO nmp;
\q
EOF
```

#### Redis
```bash
# Ubuntu/Debian
sudo apt install -y redis-server

# CentOS/RHEL
sudo yum install -y redis

# 配置密码
sudo sed -i 's/# requirepass foobared/requirepass your_redis_password/' /etc/redis/redis.conf

# 启动服务
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

#### InfluxDB
```bash
# Ubuntu/Debian
wget -qO- https://repos.influxdata.com/influxdb.key | sudo apt-key add -
echo "deb https://repos.influxdata.com/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/influxdb.list
sudo apt update
sudo apt install -y influxdb2

# 启动服务
sudo systemctl start influxdb
sudo systemctl enable influxdb

# 初始化设置
influx setup \
  --username admin \
  --password your_influxdb_password \
  --org nmp \
  --bucket monitoring \
  --token your_influxdb_token \
  --force
```

### 5. 创建系统用户和目录

```bash
# 创建nmp用户
sudo useradd -r -s /bin/false -d /opt/nmp nmp

# 创建目录结构
sudo mkdir -p /opt/nmp/{bin,config,plugins,logs,data}
sudo mkdir -p /etc/nmp
sudo mkdir -p /var/log/nmp
sudo mkdir -p /var/lib/nmp/backups

# 设置权限
sudo chown -R nmp:nmp /opt/nmp
sudo chown -R nmp:nmp /var/log/nmp
sudo chown -R nmp:nmp /var/lib/nmp
```

### 6. 构建应用

```bash
# 下载源代码
git clone https://github.com/your-org/nmp-platform.git
cd nmp-platform

# 构建后端
cd backend
go mod download
go build -o /opt/nmp/bin/nmp-backend ./cmd/server

# 构建前端
cd ../frontend
npm install
npm run build

# 部署前端文件
sudo mkdir -p /var/www/html
sudo cp -r dist/* /var/www/html/
sudo chown -R www-data:www-data /var/www/html/
```

### 7. 安装Nginx

```bash
# Ubuntu/Debian
sudo apt install -y nginx

# CentOS/RHEL
sudo yum install -y nginx

# 复制配置文件
sudo cp deployments/nginx.conf /etc/nginx/sites-available/nmp
sudo ln -sf /etc/nginx/sites-available/nmp /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default

# 测试配置
sudo nginx -t

# 启动服务
sudo systemctl start nginx
sudo systemctl enable nginx
```

### 8. 配置系统服务

```bash
# 复制systemd服务文件
sudo cp deployments/nmp-backend.service /etc/systemd/system/

# 重新加载systemd
sudo systemctl daemon-reload

# 启用服务
sudo systemctl enable nmp-backend
```

## 配置说明

### 主配置文件

配置文件位置：`/etc/nmp/config.yaml`

```yaml
# 基本配置示例
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"

database:
  host: "localhost"
  port: 5432
  database: "nmp"
  username: "nmp"
  password: "your_db_password"

redis:
  host: "localhost"
  port: 6379
  password: "your_redis_password"

influxdb:
  url: "http://localhost:8086"
  token: "your_influxdb_token"
  org: "nmp"
  bucket: "monitoring"

auth:
  jwt_secret: "your_jwt_secret_key"
  token_expiry: "24h"
```

### 环境变量配置

可以通过环境变量覆盖配置文件设置：

```bash
# 数据库配置
export DB_HOST=localhost
export DB_PASSWORD=your_password

# Redis配置
export REDIS_PASSWORD=your_redis_password

# InfluxDB配置
export INFLUXDB_TOKEN=your_token

# JWT配置
export JWT_SECRET=your_secret_key
```

### 插件配置

插件配置在主配置文件的 `plugins` 部分：

```yaml
plugins:
  directory: "/opt/nmp/plugins"
  configs:
    monitoring:
      enabled: true
      collection_interval: 60
    alerting:
      enabled: true
      notification_channels: ["email", "webhook"]
    system:
      enabled: true
```

## 首次设置

### 1. 启动服务

```bash
# 启动NMP后端服务
sudo systemctl start nmp-backend

# 检查服务状态
sudo systemctl status nmp-backend

# 查看日志
sudo journalctl -u nmp-backend -f
```

### 2. 访问Web界面

打开浏览器访问：`http://your_server_ip`

### 3. 初始化管理员账户

首次访问时，系统会提示创建管理员账户：

1. 用户名：admin（建议修改）
2. 密码：设置强密码
3. 邮箱：管理员邮箱地址

### 4. 基本配置

登录后进行基本配置：

1. **系统设置**
   - 修改系统名称
   - 设置时区
   - 配置邮件服务器

2. **用户管理**
   - 创建普通用户账户
   - 分配角色和权限

3. **设备管理**
   - 添加第一台监控设备
   - 配置设备分组

### 5. 测试功能

1. **添加测试设备**
   ```bash
   # 测试设备连接
   curl -X POST http://localhost:8080/api/devices/test \
     -H "Content-Type: application/json" \
     -d '{"host":"192.168.1.1","username":"admin","password":"password"}'
   ```

2. **推送测试数据**
   ```bash
   # 推送监控数据
   curl -X POST http://localhost:8080/api/data/push \
     -H "Content-Type: application/json" \
     -d '{"device_id":"test","metrics":{"cpu":50,"memory":60}}'
   ```

3. **查询数据**
   ```bash
   # 查询实时数据
   curl http://localhost:8080/api/data/realtime?device_id=test
   ```

## 常见问题

### Q1: 服务启动失败

**症状**: `systemctl start nmp-backend` 失败

**解决方法**:
```bash
# 查看详细错误信息
sudo journalctl -u nmp-backend -n 50

# 检查配置文件语法
/opt/nmp/bin/nmp-backend --config=/etc/nmp/config.yaml --check-config

# 检查文件权限
ls -la /opt/nmp/bin/nmp-backend
```

### Q2: 数据库连接失败

**症状**: 日志显示数据库连接错误

**解决方法**:
```bash
# 检查PostgreSQL状态
sudo systemctl status postgresql

# 测试数据库连接
sudo -u postgres psql -c "SELECT version();"

# 检查用户权限
sudo -u postgres psql -c "\du"
```

### Q3: 前端页面无法访问

**症状**: 浏览器显示404或502错误

**解决方法**:
```bash
# 检查Nginx状态
sudo systemctl status nginx

# 测试Nginx配置
sudo nginx -t

# 检查文件权限
ls -la /var/www/html/

# 重启Nginx
sudo systemctl restart nginx
```

### Q4: API请求失败

**症状**: 前端无法连接后端API

**解决方法**:
```bash
# 检查后端服务状态
sudo systemctl status nmp-backend

# 检查端口监听
sudo netstat -tlnp | grep 8080

# 测试API连接
curl -v http://localhost:8080/api/health
```

### Q5: 内存使用过高

**症状**: 系统内存不足

**解决方法**:
1. 调整InfluxDB内存设置
2. 优化Redis配置
3. 增加系统内存
4. 调整数据保留策略

### Q6: 磁盘空间不足

**症状**: 磁盘使用率过高

**解决方法**:
```bash
# 清理日志文件
sudo journalctl --vacuum-time=7d

# 清理旧备份
find /var/lib/nmp/backups -name "*.tar.gz" -mtime +30 -delete

# 配置数据保留策略
# 编辑 /etc/nmp/config.yaml 中的数据保留设置
```

## 获取帮助

如果遇到其他问题：

1. 查看系统日志：`sudo journalctl -u nmp-backend -f`
2. 检查配置文件语法
3. 参考开发者文档
4. 联系技术支持

## 下一步

安装完成后，建议阅读：
- [用户操作手册](user-manual.md)
- [开发者文档](developer-guide.md)
- [API文档](api-reference.md)