# NMP Platform - 网络监控平台

NMP (Network Monitoring Platform) 是一个现代化的网络设备监控平台，支持 MikroTik RouterOS、Linux 服务器等设备的实时监控。

## 🚀 一键安装

在全新的 Debian 11/12 或 Ubuntu 20.04+ 系统上，只需一条命令即可完成安装：

```bash
curl -fsSL https://raw.githubusercontent.com/Erion6677/nmp-platform/main/install.sh | bash
```

安装完成后：
- 访问地址: `http://服务器IP`
- 默认账户: `admin`
- 默认密码: `admin1234`

### 安装内容

脚本会自动安装和配置：
- PostgreSQL 数据库
- Redis 缓存
- InfluxDB 时序数据库
- Node.js 20.x
- Go 1.21
- Nginx 反向代理
- PM2 进程管理
- systemd 服务

## 版本

v3.0.1-beta1

## ✨ 功能特性

- 🖥️ **设备管理**：支持 MikroTik、Linux 等多种设备类型
- 📊 **实时监控**：CPU、内存、带宽、Ping 延迟等指标
- 📈 **数据可视化**：实时图表展示监控数据
- 🔐 **权限管理**：基于 RBAC 的用户权限控制
- 🌐 **代理支持**：SSH/SOCKS5 代理，支持链式代理
- 🎨 **主题切换**：支持亮色/暗色主题
- 🔌 **插件系统**：支持从 GitHub 安装扩展插件

## 🛠️ 技术栈

### 后端
- Go 1.21+
- Gin Web Framework
- GORM (PostgreSQL)
- InfluxDB (时序数据)
- Redis (缓存)

### 前端
- Next.js 16 + TypeScript
- Tailwind CSS
- Heroicons
- Zustand (状态管理)

## 📦 手动安装

如果需要手动安装，请按以下步骤操作：

### 环境要求

- Debian 11/12 或 Ubuntu 20.04+
- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- InfluxDB 2.x
- Redis 6+

### 安装步骤

1. 克隆项目
```bash
git clone https://github.com/Erion6677/nmp-platform.git
cd nmp-platform
```

2. 配置后端
```bash
cd backend
cp configs/config.example.yaml configs/config.yaml
# 编辑 config.yaml 配置数据库等信息
go build -o server ./cmd/server
./server
```

3. 配置前端
```bash
cd frontend
npm install
npm run build
npm run start
```

## 📁 目录结构

```
nmp-platform/
├── backend/           # 后端代码
│   ├── cmd/          # 入口程序
│   ├── configs/      # 配置文件
│   ├── internal/     # 内部包
│   ├── plugins/      # 插件目录
│   └── migrations/   # 数据库迁移
├── frontend/         # 前端代码
│   ├── src/
│   │   ├── app/      # 页面
│   │   ├── components/  # 组件
│   │   ├── lib/      # 工具库
│   │   └── stores/   # 状态管理
│   └── public/
├── deployments/      # 部署配置
├── docs/            # 文档
├── install.sh       # 一键安装脚本
└── uninstall.sh     # 卸载脚本
```

## 🔧 服务管理

```bash
# 后端服务
systemctl status nmp-backend    # 查看状态
systemctl restart nmp-backend   # 重启服务
journalctl -u nmp-backend -f    # 查看日志

# 前端服务
pm2 status                      # 查看状态
pm2 restart nmp-frontend        # 重启服务
pm2 logs nmp-frontend           # 查看日志
```

## 🗑️ 卸载

```bash
curl -fsSL https://raw.githubusercontent.com/Erion6677/nmp-platform/main/uninstall.sh | bash
```

或本地执行：
```bash
./uninstall.sh
```

## 📄 许可证

MIT License
