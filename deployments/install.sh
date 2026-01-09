#!/bin/bash

# NMP Platform 一键安装脚本
# 支持 Ubuntu 20.04+ 和 CentOS 8+

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
NMP_USER="nmp"
NMP_HOME="/opt/nmp"
NMP_CONFIG_DIR="/etc/nmp"
NMP_LOG_DIR="/var/log/nmp"
NMP_DATA_DIR="/var/lib/nmp"

# 数据库配置（将自动生成安全密码）
DB_NAME="nmp"
DB_USER="nmp"
DB_PASSWORD=""
REDIS_PASSWORD=""
INFLUXDB_TOKEN=""
INFLUXDB_PASSWORD=""
JWT_SECRET=""

# 服务端口
BACKEND_PORT=8080
FRONTEND_PORT=3000
POSTGRES_PORT=5432
REDIS_PORT=6379
INFLUXDB_PORT=8086

# 服务器IP（自动检测）
SERVER_IP=""

# 打印函数
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查是否为root用户
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "此脚本需要root权限运行"
        exit 1
    fi
}

# 检测操作系统
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS=$NAME
        VER=$VERSION_ID
    else
        print_error "无法检测操作系统"
        exit 1
    fi
    
    print_info "检测到操作系统: $OS $VER"
}

# 检测服务器IP地址
detect_server_ip() {
    print_info "检测服务器IP地址..."
    
    # 方法1: 获取默认路由的网卡IP
    SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
    
    # 方法2: 如果方法1失败，尝试hostname -I
    if [[ -z "$SERVER_IP" ]]; then
        SERVER_IP=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    
    # 方法3: 如果还是失败，尝试获取第一个非回环网卡的IP
    if [[ -z "$SERVER_IP" ]]; then
        SERVER_IP=$(ip addr show | grep 'inet ' | grep -v '127.0.0.1' | head -1 | awk '{print $2}' | cut -d/ -f1)
    fi
    
    # 方法4: 尝试通过外部服务获取公网IP（可选）
    if [[ -z "$SERVER_IP" ]]; then
        SERVER_IP=$(curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || echo "")
    fi
    
    if [[ -z "$SERVER_IP" ]]; then
        print_warning "无法自动检测服务器IP，请手动设置"
        read -p "请输入服务器IP地址: " SERVER_IP
    fi
    
    print_success "服务器IP: $SERVER_IP"
}

# 生成安全的随机密码
generate_secure_password() {
    local length=${1:-32}
    # 使用 openssl 生成安全的随机密码，只包含字母数字
    openssl rand -base64 $length | tr -dc 'a-zA-Z0-9' | head -c $length
}

# 生成所有安全凭据
generate_credentials() {
    print_info "生成安全凭据..."
    
    # 数据库密码 (24字符)
    if [[ -z "$DB_PASSWORD" ]]; then
        DB_PASSWORD=$(generate_secure_password 24)
    fi
    
    # Redis密码 (24字符)
    if [[ -z "$REDIS_PASSWORD" ]]; then
        REDIS_PASSWORD=$(generate_secure_password 24)
    fi
    
    # InfluxDB Token (48字符)
    if [[ -z "$INFLUXDB_TOKEN" ]]; then
        INFLUXDB_TOKEN=$(generate_secure_password 48)
    fi
    
    # InfluxDB 管理员密码 (24字符)
    if [[ -z "$INFLUXDB_PASSWORD" ]]; then
        INFLUXDB_PASSWORD=$(generate_secure_password 24)
    fi
    
    # JWT Secret (64字符，用于签名)
    if [[ -z "$JWT_SECRET" ]]; then
        JWT_SECRET=$(generate_secure_password 64)
    fi
    
    print_success "安全凭据生成完成"
}

# 安装依赖包
install_dependencies() {
    print_info "安装系统依赖..."
    
    if [[ $OS == *"Ubuntu"* ]] || [[ $OS == *"Debian"* ]]; then
        apt-get update
        apt-get install -y curl wget git unzip systemd postgresql postgresql-contrib redis-server nginx
    elif [[ $OS == *"CentOS"* ]] || [[ $OS == *"Red Hat"* ]]; then
        yum update -y
        yum install -y curl wget git unzip systemd postgresql postgresql-server postgresql-contrib redis nginx
        postgresql-setup initdb
    else
        print_error "不支持的操作系统: $OS"
        exit 1
    fi
    
    print_success "系统依赖安装完成"
}

# 创建用户和目录
create_user_and_dirs() {
    print_info "创建NMP用户和目录..."
    
    # 创建用户
    if ! id "$NMP_USER" &>/dev/null; then
        useradd -r -s /bin/false -d $NMP_HOME $NMP_USER
        print_success "创建用户: $NMP_USER"
    else
        print_info "用户 $NMP_USER 已存在"
    fi
    
    # 创建目录
    mkdir -p $NMP_HOME/{bin,config,plugins,logs,data}
    mkdir -p $NMP_CONFIG_DIR
    mkdir -p $NMP_LOG_DIR
    mkdir -p $NMP_DATA_DIR/{postgres,redis,influxdb}
    
    # 设置权限
    chown -R $NMP_USER:$NMP_USER $NMP_HOME
    chown -R $NMP_USER:$NMP_USER $NMP_LOG_DIR
    chown -R $NMP_USER:$NMP_USER $NMP_DATA_DIR
    
    print_success "目录创建完成"
}

# 安装Go
install_go() {
    print_info "安装Go语言环境..."
    
    GO_VERSION="1.21.5"
    GO_ARCH="linux-amd64"
    
    if command -v go &> /dev/null; then
        CURRENT_GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [[ "$CURRENT_GO_VERSION" == "$GO_VERSION" ]]; then
            print_info "Go $GO_VERSION 已安装"
            return
        fi
    fi
    
    cd /tmp
    wget https://golang.org/dl/go${GO_VERSION}.${GO_ARCH}.tar.gz
    tar -C /usr/local -xzf go${GO_VERSION}.${GO_ARCH}.tar.gz
    
    # 设置环境变量
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    
    print_success "Go安装完成"
}

# 安装Node.js
install_nodejs() {
    print_info "安装Node.js环境..."
    
    NODE_VERSION="18"
    
    if command -v node &> /dev/null; then
        CURRENT_NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
        if [[ "$CURRENT_NODE_VERSION" == "$NODE_VERSION" ]]; then
            print_info "Node.js $NODE_VERSION 已安装"
            return
        fi
    fi
    
    curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash -
    apt-get install -y nodejs
    
    print_success "Node.js安装完成"
}

# 安装InfluxDB
install_influxdb() {
    print_info "安装InfluxDB..."
    
    if command -v influxd &> /dev/null; then
        print_info "InfluxDB已安装"
        return
    fi
    
    if [[ $OS == *"Ubuntu"* ]] || [[ $OS == *"Debian"* ]]; then
        wget -qO- https://repos.influxdata.com/influxdb.key | apt-key add -
        echo "deb https://repos.influxdata.com/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/influxdb.list
        apt-get update
        apt-get install -y influxdb2
    elif [[ $OS == *"CentOS"* ]] || [[ $OS == *"Red Hat"* ]]; then
        cat <<EOF | tee /etc/yum.repos.d/influxdb.repo
[influxdb]
name = InfluxDB Repository - RHEL \$releasever
baseurl = https://repos.influxdata.com/rhel/\$releasever/\$basearch/stable
enabled = 1
gpgcheck = 1
gpgkey = https://repos.influxdata.com/influxdb.key
EOF
        yum install -y influxdb2
    fi
    
    print_success "InfluxDB安装完成"
}

# 配置数据库
configure_databases() {
    print_info "配置数据库..."
    
    # 配置PostgreSQL
    systemctl start postgresql
    systemctl enable postgresql
    
    # 检查用户是否已存在
    if sudo -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER'" | grep -q 1; then
        print_info "PostgreSQL用户 $DB_USER 已存在，更新密码..."
        sudo -u postgres psql -c "ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"
    else
        sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';"
    fi
    
    # 检查数据库是否已存在
    if sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME'" | grep -q 1; then
        print_info "PostgreSQL数据库 $DB_NAME 已存在"
    else
        sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;"
    fi
    sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;"
    
    # 配置Redis
    if [[ -f /etc/redis/redis.conf ]]; then
        sed -i "s/^# requirepass .*/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
        sed -i "s/^requirepass .*/requirepass $REDIS_PASSWORD/" /etc/redis/redis.conf
    elif [[ -f /etc/redis.conf ]]; then
        sed -i "s/^# requirepass .*/requirepass $REDIS_PASSWORD/" /etc/redis.conf
        sed -i "s/^requirepass .*/requirepass $REDIS_PASSWORD/" /etc/redis.conf
    fi
    systemctl start redis-server || systemctl start redis
    systemctl enable redis-server || systemctl enable redis
    
    # 配置InfluxDB
    systemctl start influxdb
    systemctl enable influxdb
    
    # 等待InfluxDB启动
    sleep 3
    
    # 检查InfluxDB是否已初始化
    if influx org list 2>/dev/null | grep -q "nmp"; then
        print_info "InfluxDB已初始化，跳过设置"
    else
        # 初始化InfluxDB（使用安全生成的密码）
        influx setup \
            --username admin \
            --password "$INFLUXDB_PASSWORD" \
            --org nmp \
            --bucket monitoring \
            --token "$INFLUXDB_TOKEN" \
            --force
    fi
    
    print_success "数据库配置完成"
}

# 下载和编译NMP
build_nmp() {
    print_info "构建NMP应用..."
    
    # 检查源代码是否存在
    if [[ ! -d "/tmp/nmp-platform" ]]; then
        print_info "下载NMP Platform源代码..."
        cd /tmp
        # git clone https://github.com/your-org/nmp-platform.git
        print_warning "请手动下载源代码到 /tmp/nmp-platform 目录"
        print_warning "或修改此脚本中的git仓库地址"
        return
    fi
    
    cd /tmp/nmp-platform
    
    # 构建后端
    print_info "构建后端服务..."
    cd backend
    go mod download
    go build -o $NMP_HOME/bin/nmp-backend ./cmd/server
    
    # 构建前端
    print_info "构建前端应用..."
    cd ../frontend
    npm install
    npm run build
    
    # 创建前端目录
    mkdir -p /var/www/html
    cp -r dist/* /var/www/html/
    chown -R www-data:www-data /var/www/html/
    
    # 复制插件文件
    if [[ -d "../backend/plugins" ]]; then
        cp -r ../backend/plugins/* $NMP_HOME/plugins/
    fi
    
    print_success "NMP构建完成"
}

# 创建配置文件
create_config() {
    print_info "创建配置文件..."
    
    cat > $NMP_CONFIG_DIR/config.yaml << EOF
# NMP Platform 生产环境配置
# 自动生成于: $(date '+%Y-%m-%d %H:%M:%S')
# 服务器IP: $SERVER_IP

server:
  host: "0.0.0.0"
  port: $BACKEND_PORT
  mode: "release"
  read_timeout: "30s"
  write_timeout: "30s"
  public_url: "http://$SERVER_IP:$BACKEND_PORT"

database:
  host: "localhost"
  port: $POSTGRES_PORT
  database: "$DB_NAME"
  username: "$DB_USER"
  password: "$DB_PASSWORD"
  ssl_mode: "disable"

redis:
  host: "localhost"
  port: $REDIS_PORT
  password: "$REDIS_PASSWORD"
  db: 0

influxdb:
  url: "http://localhost:$INFLUXDB_PORT"
  token: "$INFLUXDB_TOKEN"
  org: "nmp"
  bucket: "monitoring"

auth:
  jwt_secret: "$JWT_SECRET"
  token_expiry: "24h"
  refresh_expiry: "168h"

plugins:
  directory: "$NMP_HOME/plugins"
  configs:
    monitoring:
      enabled: true
      collection_interval: 60
    alerting:
      enabled: true
      notification_channels: ["email", "webhook"]
    system:
      enabled: true
EOF
    
    chown $NMP_USER:$NMP_USER $NMP_CONFIG_DIR/config.yaml
    chmod 600 $NMP_CONFIG_DIR/config.yaml
    
    # 保存凭据到安全文件（仅root可读）
    cat > $NMP_CONFIG_DIR/.credentials << EOF
# NMP Platform 凭据文件
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ⚠️ 警告: 此文件包含敏感信息，请妥善保管！

SERVER_IP=$SERVER_IP
DB_PASSWORD=$DB_PASSWORD
REDIS_PASSWORD=$REDIS_PASSWORD
INFLUXDB_TOKEN=$INFLUXDB_TOKEN
INFLUXDB_PASSWORD=$INFLUXDB_PASSWORD
JWT_SECRET=$JWT_SECRET
EOF
    
    chmod 600 $NMP_CONFIG_DIR/.credentials
    
    print_success "配置文件创建完成"
}

# 配置Nginx
configure_nginx() {
    print_info "配置Nginx..."
    
    cat > /etc/nginx/sites-available/nmp << EOF
# NMP Platform Nginx 配置
# 服务器IP: $SERVER_IP

server {
    listen 80;
    server_name $SERVER_IP _;
    
    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # 前端静态文件
    location / {
        root /var/www/html;
        index index.html;
        try_files \$uri \$uri/ /index.html;
        
        # 缓存静态资源
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
            expires 30d;
            add_header Cache-Control "public, immutable";
        }
    }
    
    # API代理
    location /api/ {
        proxy_pass http://127.0.0.1:$BACKEND_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
    
    # WebSocket支持
    location /ws/ {
        proxy_pass http://127.0.0.1:$BACKEND_PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # WebSocket超时
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
    
    # 健康检查端点
    location /health {
        proxy_pass http://127.0.0.1:$BACKEND_PORT/api/health;
        access_log off;
    }
}
EOF
    
    # 创建sites-enabled目录（如果不存在）
    mkdir -p /etc/nginx/sites-enabled
    
    ln -sf /etc/nginx/sites-available/nmp /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
    
    # 测试配置
    if nginx -t; then
        systemctl restart nginx
        systemctl enable nginx
        print_success "Nginx配置完成"
    else
        print_error "Nginx配置测试失败"
        exit 1
    fi
}

# 创建systemd服务
create_systemd_service() {
    print_info "创建systemd服务..."
    
    cat > /etc/systemd/system/nmp-backend.service << EOF
[Unit]
Description=NMP Platform Backend Service
After=network.target postgresql.service redis-server.service influxdb.service
Wants=postgresql.service redis-server.service influxdb.service

[Service]
Type=simple
User=$NMP_USER
Group=$NMP_USER
WorkingDirectory=$NMP_HOME
ExecStart=$NMP_HOME/bin/nmp-backend --config=$NMP_CONFIG_DIR/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nmp-backend

# 环境变量
Environment=GIN_MODE=release

# 安全设置
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$NMP_LOG_DIR $NMP_DATA_DIR

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    systemctl enable nmp-backend
    
    print_success "systemd服务创建完成"
}

# 设置防火墙
configure_firewall() {
    print_info "配置防火墙..."
    
    if command -v ufw &> /dev/null; then
        ufw allow 80/tcp
        ufw allow 443/tcp
        ufw allow ssh
        ufw --force enable
    elif command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https
        firewall-cmd --permanent --add-service=ssh
        firewall-cmd --reload
    fi
    
    print_success "防火墙配置完成"
}

# 启动服务
start_services() {
    print_info "启动NMP服务..."
    
    systemctl start nmp-backend
    
    # 等待服务启动
    sleep 5
    
    if systemctl is-active --quiet nmp-backend; then
        print_success "NMP后端服务启动成功"
    else
        print_error "NMP后端服务启动失败"
        systemctl status nmp-backend
        exit 1
    fi
    
    print_success "所有服务启动完成"
}

# 显示安装信息
show_install_info() {
    echo
    echo "=============================================="
    echo -e "${GREEN}  NMP Platform 安装完成！${NC}"
    echo "=============================================="
    echo
    echo "=== 访问地址 ==="
    echo "  Web界面: http://$SERVER_IP"
    echo "  后端API: http://$SERVER_IP/api"
    echo
    echo "=== 数据库信息 ==="
    echo "  PostgreSQL:"
    echo "    数据库: $DB_NAME"
    echo "    用户名: $DB_USER"
    echo "    密码: $DB_PASSWORD"
    echo
    echo "  Redis密码: $REDIS_PASSWORD"
    echo
    echo "  InfluxDB:"
    echo "    用户名: admin"
    echo "    密码: $INFLUXDB_PASSWORD"
    echo "    Token: $INFLUXDB_TOKEN"
    echo
    echo "=== JWT密钥 ==="
    echo "  $JWT_SECRET"
    echo
    echo "=== 服务管理 ==="
    echo "  启动服务: systemctl start nmp-backend"
    echo "  停止服务: systemctl stop nmp-backend"
    echo "  重启服务: systemctl restart nmp-backend"
    echo "  查看状态: systemctl status nmp-backend"
    echo "  查看日志: journalctl -u nmp-backend -f"
    echo
    echo "=== 配置文件 ==="
    echo "  主配置: $NMP_CONFIG_DIR/config.yaml"
    echo "  凭据备份: $NMP_CONFIG_DIR/.credentials"
    echo "  日志目录: $NMP_LOG_DIR"
    echo "  数据目录: $NMP_DATA_DIR"
    echo
    echo "=============================================="
    print_warning "⚠️  请妥善保存以上凭据信息！"
    print_warning "⚠️  凭据已保存到: $NMP_CONFIG_DIR/.credentials"
    echo "=============================================="
}

# 主函数
main() {
    echo
    echo "=============================================="
    echo "       NMP Platform 一键安装脚本"
    echo "=============================================="
    echo
    
    check_root
    detect_os
    detect_server_ip
    generate_credentials
    install_dependencies
    create_user_and_dirs
    install_go
    install_nodejs
    install_influxdb
    configure_databases
    build_nmp
    create_config
    configure_nginx
    create_systemd_service
    configure_firewall
    start_services
    show_install_info
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi