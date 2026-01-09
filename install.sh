#!/bin/bash

# ============================================
# NMP Platform 一键安装脚本
# 支持: Debian 11/12/13, Ubuntu 20.04+
# 用法: curl -fsSL https://raw.githubusercontent.com/Erion6677/nmp-platform/main/install.sh | bash
# ============================================

set -e

# 版本
NMP_VERSION="1.0.0"
GITHUB_REPO="Erion6677/nmp-platform"

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 安装目录
INSTALL_DIR="/opt/nmp-platform"
CONFIG_DIR="/etc/nmp"
LOG_DIR="/var/log/nmp"
PLUGINS_DIR="/opt/nmp/plugins"

# 数据库配置（固定密码，简化安装）
DB_NAME="nmp"
DB_USER="nmp"
DB_PASSWORD="NmpSecure2024!"
REDIS_PASSWORD="NmpRedis2024!"
INFLUXDB_PASSWORD="NmpInflux2024!"
INFLUXDB_TOKEN="NmpInfluxToken2024SecureRandomString1234567890"
JWT_SECRET=""  # 将在安装时生成

# 端口
BACKEND_PORT=8080
FRONTEND_PORT=3000

# 服务器IP
SERVER_IP=""

# ============================================
# 工具函数
# ============================================

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 修复 /tmp 目录权限
fix_tmp_permissions() {
    log_info "检查 /tmp 目录权限..."
    local tmp_perms=$(stat -c %a /tmp)
    if [[ "$tmp_perms" != "1777" ]]; then
        log_warn "/tmp 权限不正确 ($tmp_perms)，正在修复..."
        chmod 1777 /tmp
        log_success "/tmp 权限已修复"
    fi
}

print_banner() {
    echo -e "${CYAN}"
    echo "  _   _ __  __ ____    ____  _       _    __                      "
    echo " | \ | |  \/  |  _ \  |  _ \| | __ _| |_ / _| ___  _ __ _ __ ___  "
    echo " |  \| | |\/| | |_) | | |_) | |/ _\` | __| |_ / _ \| '__| '_ \` _ \ "
    echo " | |\  | |  | |  __/  |  __/| | (_| | |_|  _| (_) | |  | | | | | |"
    echo " |_| \_|_|  |_|_|     |_|   |_|\__,_|\__|_|  \___/|_|  |_| |_| |_|"
    echo -e "${NC}"
    echo "  Network Monitoring Platform - 一键安装脚本 v${NMP_VERSION}"
    echo "  =================================================="
    echo
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "请使用 root 用户运行此脚本"
        exit 1
    fi
}

detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS_NAME=$ID
        OS_VERSION=$VERSION_ID
    else
        log_error "无法检测操作系统"
        exit 1
    fi
    
    case $OS_NAME in
        debian|ubuntu)
            log_info "检测到: $PRETTY_NAME"
            ;;
        *)
            log_error "不支持的操作系统: $OS_NAME"
            log_info "支持的系统: Debian 11/12/13, Ubuntu 20.04+"
            exit 1
            ;;
    esac
}

detect_ip() {
    SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
    [[ -z "$SERVER_IP" ]] && SERVER_IP=$(hostname -I | awk '{print $1}')
    [[ -z "$SERVER_IP" ]] && SERVER_IP="127.0.0.1"
    log_info "服务器IP: $SERVER_IP"
}

generate_secrets() {
    log_info "生成安全密钥..."
    # 生成随机JWT密钥（64字节base64编码）
    JWT_SECRET=$(openssl rand -base64 64 | tr -d '\n')
    log_success "安全密钥生成完成"
}

# ============================================
# 安装依赖
# ============================================

install_base_packages() {
    log_info "更新系统并安装基础包..."
    
    # 清理 APT 缓存
    apt-get clean
    rm -rf /var/lib/apt/lists/partial/*
    
    # 更新包列表（带重试）
    local retry=0
    while [[ $retry -lt 3 ]]; do
        if apt-get update -qq 2>&1 | tee /tmp/apt-update.log; then
            break
        fi
        retry=$((retry + 1))
        log_warn "APT 更新失败，重试 $retry/3..."
        sleep 2
    done
    
    # 安装基础包和编译工具（原生模块需要）
    apt-get install -y -qq curl wget git unzip gnupg2 lsb-release ca-certificates apt-transport-https openssl \
        build-essential python3 make g++
    log_success "基础包安装完成"
}

install_postgresql() {
    log_info "安装 PostgreSQL..."
    if ! command -v psql &>/dev/null; then
        apt-get install -y -qq postgresql postgresql-contrib
    fi
    systemctl enable postgresql
    systemctl start postgresql
    log_success "PostgreSQL 安装完成"
}

install_redis() {
    log_info "安装 Redis..."
    if ! command -v redis-server &>/dev/null; then
        apt-get install -y -qq redis-server
    fi
    systemctl enable redis-server
    systemctl start redis-server
    log_success "Redis 安装完成"
}

install_influxdb() {
    log_info "安装 InfluxDB 2.x..."
    
    if ! command -v influxd &>/dev/null; then
        wget -q https://dl.influxdata.com/influxdb/releases/influxdb2-2.7.1-amd64.deb -O /tmp/influxdb2.deb
        dpkg -i /tmp/influxdb2.deb
        rm -f /tmp/influxdb2.deb
    fi
    
    # 安装 InfluxDB CLI
    if ! command -v influx &>/dev/null; then
        wget -q https://dl.influxdata.com/influxdb/releases/influxdb2-client-2.7.3-linux-amd64.tar.gz -O /tmp/influx-cli.tar.gz
        tar -xzf /tmp/influx-cli.tar.gz -C /tmp
        mv /tmp/influx /usr/local/bin/
        rm -f /tmp/influx-cli.tar.gz
    fi
    
    systemctl enable influxdb
    systemctl start influxdb
    sleep 3
    log_success "InfluxDB 安装完成"
}

install_nodejs() {
    log_info "安装 Node.js 22.x LTS..."
    
    # 关键：确保 /tmp 权限正确（apt/gpg 需要写入临时文件）
    log_info "修复 /tmp 目录权限..."
    chmod 1777 /tmp
    chown root:root /tmp
    # 清理可能存在的临时文件
    rm -rf /tmp/apt.conf.* 2>/dev/null || true
    
    # 首先彻底移除任何现有的 Node.js（无论版本）
    log_info "清理现有 Node.js 安装..."
    apt-get remove -y --purge nodejs npm libnode* node-* 2>/dev/null || true
    apt-get autoremove -y 2>/dev/null || true
    rm -f /etc/apt/sources.list.d/nodesource.list 2>/dev/null || true
    rm -f /etc/apt/keyrings/nodesource.gpg 2>/dev/null || true
    
    # 关键：彻底清理 apt 缓存
    apt-get clean
    rm -rf /var/lib/apt/lists/*
    
    # 再次确保 /tmp 权限（某些包安装可能会改变它）
    chmod 1777 /tmp
    
    # 使用 Node.js 22.x LTS（最新稳定版）
    log_info "添加 NodeSource 仓库..."
    # 设置 TMPDIR 确保临时文件写入正确位置
    export TMPDIR=/tmp
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    
    # 检查 NodeSource 源是否添加成功
    if [[ ! -f /etc/apt/sources.list.d/nodesource.list ]]; then
        log_error "NodeSource 仓库添加失败"
        log_info "尝试手动添加 NodeSource 仓库..."
        
        # 手动添加 NodeSource 仓库
        mkdir -p /etc/apt/keyrings
        curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg
        echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_22.x nodistro main" > /etc/apt/sources.list.d/nodesource.list
    fi
    
    # 再次更新以确保使用 NodeSource 源
    apt-get update -qq
    
    log_info "安装 Node.js..."
    # 明确指定从 nodesource 安装
    apt-get install -y nodejs
    
    # 验证安装
    if ! command -v node &>/dev/null; then
        log_error "Node.js 安装失败"
        exit 1
    fi
    
    # NodeSource 的 nodejs 包已包含 npm，验证
    if ! command -v npm &>/dev/null; then
        log_error "npm 未安装，NodeSource 安装可能失败"
        exit 1
    fi
    
    # 验证版本
    NODE_VER=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
    if [[ $NODE_VER -lt 22 ]]; then
        log_error "Node.js 版本不正确: $(node -v)，需要 22.x"
        exit 1
    fi
    
    log_success "Node.js $(node -v) 和 npm $(npm -v) 安装完成"
}

install_nginx() {
    log_info "安装 Nginx..."
    if ! command -v nginx &>/dev/null; then
        apt-get install -y -qq nginx
    fi
    systemctl enable nginx
    log_success "Nginx 安装完成"
}

# ============================================
# 配置数据库
# ============================================

configure_postgresql() {
    log_info "配置 PostgreSQL..."
    
    su - postgres -c "psql -c \"DROP DATABASE IF EXISTS $DB_NAME;\"" 2>/dev/null || true
    su - postgres -c "psql -c \"DROP USER IF EXISTS $DB_USER;\"" 2>/dev/null || true
    su - postgres -c "psql -c \"CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';\""
    su - postgres -c "psql -c \"CREATE DATABASE $DB_NAME OWNER $DB_USER;\""
    su - postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;\""
    
    log_success "PostgreSQL 配置完成"
}

configure_redis() {
    log_info "配置 Redis..."
    
    REDIS_CONF="/etc/redis/redis.conf"
    if [[ -f $REDIS_CONF ]]; then
        # 移除旧的 requirepass 配置
        sed -i '/^requirepass/d' $REDIS_CONF
        # 添加新密码
        echo "requirepass $REDIS_PASSWORD" >> $REDIS_CONF
        systemctl restart redis-server
    fi
    
    log_success "Redis 配置完成"
}

configure_influxdb() {
    log_info "配置 InfluxDB..."
    
    # 等待 InfluxDB 启动
    sleep 2
    
    # 检查是否已初始化
    if ! influx org list 2>/dev/null | grep -q "nmp"; then
        influx setup \
            --username admin \
            --password "$INFLUXDB_PASSWORD" \
            --org nmp \
            --bucket monitoring \
            --token "$INFLUXDB_TOKEN" \
            --force 2>/dev/null || true
    fi
    
    log_success "InfluxDB 配置完成"
}

# ============================================
# 下载 NMP
# ============================================

download_nmp() {
    log_info "下载 NMP Platform..."
    
    rm -rf $INSTALL_DIR
    mkdir -p $INSTALL_DIR
    
    cd /tmp
    rm -rf nmp-platform
    git clone --depth 1 https://github.com/$GITHUB_REPO.git
    cp -r nmp-platform/* $INSTALL_DIR/
    rm -rf nmp-platform
    
    log_success "下载完成"
}

# ============================================
# 构建
# ============================================

build_backend() {
    log_info "准备后端..."
    
    cd $INSTALL_DIR/backend
    
    # 优先使用预编译的二进制文件
    if [[ -f "server.linux-amd64" ]]; then
        log_info "使用预编译的后端二进制文件"
        cp server.linux-amd64 server
        chmod +x server
    elif [[ -f "server" ]]; then
        log_info "使用已有的后端二进制文件"
        chmod +x server
    else
        log_error "未找到后端二进制文件，请确保 server.linux-amd64 存在"
        exit 1
    fi
    
    log_success "后端准备完成"
}

build_frontend() {
    log_info "准备前端..."
    
    cd $INSTALL_DIR/frontend
    
    # 确保 npm 在 PATH 中
    export PATH="/usr/bin:/usr/local/bin:$PATH"
    
    # 检查 npm 是否可用
    if ! command -v npm &>/dev/null; then
        log_error "npm 未找到，请确保 Node.js 已正确安装"
        exit 1
    fi
    
    # 创建生产环境配置
    cat > .env.production << EOF
NEXT_PUBLIC_API_URL=http://$SERVER_IP:$BACKEND_PORT
EOF
    
    # 检查是否已有预构建的 .next 文件夹
    if [[ -d ".next" ]] && [[ -f ".next/BUILD_ID" ]]; then
        log_info "检测到预构建的前端，跳过构建步骤"
        log_success "前端准备完成（使用预构建版本）"
        return 0
    fi
    
    # 首先尝试从 GitHub Release 下载预构建的前端（推荐方式，避免构建问题）
    log_info "尝试下载预构建的前端..."
    PREBUILT_URL="https://github.com/$GITHUB_REPO/releases/latest/download/frontend-prebuilt.tar.gz"
    
    if wget -q --timeout=30 --spider "$PREBUILT_URL" 2>/dev/null; then
        log_info "正在下载预构建前端..."
        if wget -q --timeout=120 "$PREBUILT_URL" -O /tmp/frontend-prebuilt.tar.gz; then
            tar -xzf /tmp/frontend-prebuilt.tar.gz -C $INSTALL_DIR/frontend/
            rm -f /tmp/frontend-prebuilt.tar.gz
            if [[ -d ".next" ]] && [[ -f ".next/BUILD_ID" ]]; then
                log_success "预构建前端下载完成"
                return 0
            fi
        fi
    fi
    
    log_warn "预构建前端不可用，尝试本地构建..."
    
    # 清理缓存
    npm cache clean --force 2>/dev/null || true
    
    # 安装依赖
    log_info "安装前端依赖..."
    npm install --legacy-peer-deps 2>&1 | grep -v "^npm WARN" || true
    
    # 确保原生模块正确安装
    log_info "安装原生依赖模块..."
    npm install lightningcss --force 2>/dev/null || true
    npm rebuild 2>/dev/null || true
    
    # 创建简化的 next.config.js
    cat > next.config.js << 'NEXTCONFIG'
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  images: {
    unoptimized: true,
  },
};

module.exports = nextConfig;
NEXTCONFIG
    
    # 删除可能冲突的 TypeScript 配置
    rm -f next.config.ts next.config.mjs 2>/dev/null || true
    
    # 检查 Next.js 版本，如果是 16.x 则降级到 15.x（Turbopack 在某些环境下不稳定）
    NEXT_VER=$(npm list next 2>/dev/null | grep next@ | head -1 | grep -oE '[0-9]+' | head -1)
    if [[ "$NEXT_VER" == "16" ]]; then
        log_warn "检测到 Next.js 16.x，降级到 15.x 以避免 Turbopack 兼容性问题..."
        npm install next@15 --legacy-peer-deps 2>&1 | tail -5
    fi
    
    # 尝试构建
    log_info "构建前端（这可能需要几分钟）..."
    set +e  # 临时允许命令失败
    NEXT_TELEMETRY_DISABLED=1 timeout 600 npm run build 2>&1 | tee /tmp/frontend-build.log
    BUILD_EXIT_CODE=${PIPESTATUS[0]}
    set -e  # 恢复错误退出
    
    # 检查构建是否成功
    if [[ $BUILD_EXIT_CODE -eq 0 ]] && [[ -d ".next" ]] && [[ -f ".next/BUILD_ID" ]]; then
        log_success "前端构建完成"
    else
        log_error "前端构建失败（退出码: $BUILD_EXIT_CODE）"
        log_error "这通常是由于内存不足或系统兼容性问题导致的"
        log_error "解决方案："
        log_error "  1. 增加虚拟机内存到 4GB 以上"
        log_error "  2. 使用离线安装包（包含预构建前端）"
        log_error "  3. 在 GitHub Release 页面下载 frontend-prebuilt.tar.gz"
        log_error "构建日志: /tmp/frontend-build.log"
        exit 1
    fi
}

# ============================================
# 创建配置文件
# ============================================

create_backend_config() {
    log_info "创建后端配置..."
    
    mkdir -p $CONFIG_DIR $LOG_DIR $PLUGINS_DIR
    
    cat > $INSTALL_DIR/backend/configs/config.yaml << EOF
# NMP Platform 配置文件
# 自动生成于: $(date '+%Y-%m-%d %H:%M:%S')

server:
  host: "0.0.0.0"
  port: $BACKEND_PORT
  mode: "release"
  read_timeout: "30s"
  write_timeout: "30s"
  public_url: "http://$SERVER_IP"

database:
  host: "localhost"
  port: 5432
  database: "$DB_NAME"
  username: "$DB_USER"
  password: "$DB_PASSWORD"
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: 6379
  password: "$REDIS_PASSWORD"
  db: 0

influxdb:
  url: "http://localhost:8086"
  token: "$INFLUXDB_TOKEN"
  org: "nmp"
  bucket: "monitoring"

auth:
  jwt_secret: "$JWT_SECRET"
  token_expiry: "24h"
  refresh_expiry: "168h"

plugins:
  directory: "$PLUGINS_DIR"
EOF
    
    log_success "后端配置完成"
}

create_nginx_config() {
    log_info "配置 Nginx..."
    
    cat > /etc/nginx/sites-available/nmp << EOF
server {
    listen 80;
    server_name $SERVER_IP _;

    location / {
        proxy_pass http://127.0.0.1:$FRONTEND_PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_cache_bypass \$http_upgrade;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:$BACKEND_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    location /health {
        proxy_pass http://127.0.0.1:$BACKEND_PORT/health;
        access_log off;
    }
}
EOF
    
    ln -sf /etc/nginx/sites-available/nmp /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
    
    nginx -t && systemctl restart nginx
    
    log_success "Nginx 配置完成"
}

# ============================================
# 创建 systemd 服务
# ============================================

create_systemd_services() {
    log_info "创建 systemd 服务..."
    
    # 后端服务
    cat > /etc/systemd/system/nmp-backend.service << EOF
[Unit]
Description=NMP Platform Backend
After=network.target postgresql.service redis-server.service influxdb.service

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR/backend
ExecStart=$INSTALL_DIR/backend/server
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

    # 前端服务（始终使用生产模式）
    cat > /etc/systemd/system/nmp-frontend.service << EOF
[Unit]
Description=NMP Platform Frontend
After=network.target

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR/frontend
ExecStart=/usr/bin/npx next start -p $FRONTEND_PORT -H 0.0.0.0
Restart=on-failure
RestartSec=10
KillMode=mixed
KillSignal=SIGTERM
Environment=NEXT_TELEMETRY_DISABLED=1

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable nmp-backend nmp-frontend
    
    log_success "systemd 服务创建完成"
}

# ============================================
# 启动服务
# ============================================

start_services() {
    log_info "启动服务..."
    
    # 启动后端
    systemctl start nmp-backend
    sleep 3
    
    # 检查后端是否启动成功
    if ! curl -s http://localhost:$BACKEND_PORT/health > /dev/null; then
        log_error "后端启动失败，查看日志: journalctl -u nmp-backend"
        exit 1
    fi
    log_success "后端启动成功"
    
    # 启动前端
    systemctl start nmp-frontend
    sleep 5
    
    # 检查前端是否启动成功
    if ! curl -s http://localhost:$FRONTEND_PORT > /dev/null; then
        log_warn "前端可能还在启动中，请稍等..."
    fi
    log_success "前端启动成功"
}

# ============================================
# 创建默认管理员
# ============================================

create_admin_user() {
    log_info "创建默认管理员账户..."
    
    sleep 2
    
    # 通过 API 创建管理员
    curl -s -X POST "http://localhost:$BACKEND_PORT/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin1234","email":"admin@nmp.local"}' > /dev/null 2>&1 || true
    
    log_success "管理员账户创建完成"
}

# ============================================
# 保存凭据
# ============================================

save_credentials() {
    cat > $CONFIG_DIR/credentials << EOF
# ============================================
# NMP Platform 安装凭据
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ⚠️ 请妥善保管此文件！
# ============================================

访问地址: http://$SERVER_IP
默认账户: admin
默认密码: admin1234

PostgreSQL:
  数据库: $DB_NAME
  用户名: $DB_USER
  密码: $DB_PASSWORD

Redis:
  密码: $REDIS_PASSWORD

InfluxDB:
  用户名: admin
  密码: $INFLUXDB_PASSWORD
  Token: $INFLUXDB_TOKEN

JWT Secret: $JWT_SECRET
EOF
    
    chmod 600 $CONFIG_DIR/credentials
}

# ============================================
# 显示安装结果
# ============================================

show_result() {
    echo
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}    NMP Platform 安装成功！${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo
    echo -e "  访问地址: ${CYAN}http://$SERVER_IP${NC}"
    echo -e "  默认账户: ${CYAN}admin${NC}"
    echo -e "  默认密码: ${CYAN}admin1234${NC}"
    echo
    echo -e "  ${YELLOW}⚠️ 请首次登录后修改密码！${NC}"
    echo
    echo "  服务管理:"
    echo "    systemctl status nmp-backend    # 后端状态"
    echo "    systemctl status nmp-frontend   # 前端状态"
    echo "    journalctl -u nmp-backend -f    # 后端日志"
    echo "    journalctl -u nmp-frontend -f   # 前端日志"
    echo
    echo "  凭据文件: $CONFIG_DIR/credentials"
    echo
    echo -e "${GREEN}============================================${NC}"
}

# ============================================
# 主函数
# ============================================

main() {
    print_banner
    check_root
    detect_os
    detect_ip
    fix_tmp_permissions  # 修复 /tmp 权限
    generate_secrets
    
    echo
    log_info "开始安装 NMP Platform..."
    echo
    
    # 安装依赖
    install_base_packages
    install_postgresql
    install_redis
    install_influxdb
    install_nodejs
    install_nginx
    
    # 配置数据库
    configure_postgresql
    configure_redis
    configure_influxdb
    
    # 下载和构建
    download_nmp
    build_backend
    build_frontend
    
    # 配置
    create_backend_config
    create_nginx_config
    create_systemd_services
    
    # 启动
    start_services
    create_admin_user
    save_credentials
    
    show_result
}

# 运行
main "$@"
