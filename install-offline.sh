#!/bin/bash
set -e

RED="\033[0;31m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
NC="\033[0m"

INSTALL_DIR="/opt/nmp-platform"
NODE_DIR="/opt/node"
PLUGINS_DIR="/opt/nmp/plugins"

DB_NAME="nmp"
DB_USER="nmp"
DB_PASSWORD="NmpSecure2024!"
REDIS_PASSWORD="NmpRedis2024!"
INFLUXDB_PASSWORD="NmpInflux2024!"
INFLUXDB_TOKEN="NmpInfluxToken2024SecureRandomString1234567890"

BACKEND_PORT=8080
FRONTEND_PORT=3000
SERVER_IP=""
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_ok() { echo -e "${GREEN}[OK]${NC} $1"; }
log_err() { echo -e "${RED}[ERROR]${NC} $1"; }

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_err "请使用 root 用户运行"
        exit 1
    fi
}

detect_ip() {
    SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
    [[ -z "$SERVER_IP" ]] && SERVER_IP=$(hostname -I | awk '{print $1}')
    log_info "服务器IP: $SERVER_IP"
}

install_packages() {
    log_info "安装系统依赖..."
    apt-get update -qq
    apt-get install -y -qq curl wget openssl postgresql postgresql-contrib redis-server nginx
    log_ok "系统依赖安装完成"
}

install_nodejs() {
    log_info "安装 Node.js..."
    mkdir -p $NODE_DIR
    tar -xJf "$SCRIPT_DIR/node-v22.21.1-linux-x64.tar.xz" -C $NODE_DIR --strip-components=1
    ln -sf $NODE_DIR/bin/node /usr/local/bin/node
    ln -sf $NODE_DIR/bin/npm /usr/local/bin/npm
    ln -sf $NODE_DIR/bin/npx /usr/local/bin/npx
    log_ok "Node.js $(node -v) 安装完成"
}

install_influxdb() {
    log_info "安装 InfluxDB..."
    
    # 临时禁用服务自动启动
    export DEBIAN_FRONTEND=noninteractive
    
    # 创建 policy-rc.d 文件阻止服务自动启动
    cat > /usr/sbin/policy-rc.d << 'EOF'
#!/bin/sh
exit 101
EOF
    chmod +x /usr/sbin/policy-rc.d
    
    # 安装 InfluxDB
    dpkg -i "$SCRIPT_DIR/influxdb2-2.7.1-amd64.deb" 2>/dev/null || {
        log_info "dpkg 安装有警告，继续..."
    }
    
    # 删除 policy-rc.d
    rm -f /usr/sbin/policy-rc.d
    
    # 安装 CLI
    tar -xzf "$SCRIPT_DIR/influxdb2-client-2.7.3-linux-amd64.tar.gz" -C /tmp
    mv /tmp/influx /usr/local/bin/
    
    # 手动启动 InfluxDB
    log_info "启动 InfluxDB 服务..."
    if systemctl is-system-running >/dev/null 2>&1 || [[ $? -eq 1 ]]; then
        systemctl enable influxdb 2>/dev/null || true
        systemctl start influxdb 2>/dev/null || {
            log_info "systemctl 启动失败，尝试直接启动..."
            nohup /usr/bin/influxd --config /etc/influxdb/config.toml > /var/log/influxdb.log 2>&1 &
        }
    else
        log_info "直接启动 InfluxDB..."
        nohup /usr/bin/influxd --config /etc/influxdb/config.toml > /var/log/influxdb.log 2>&1 &
    fi
    
    sleep 5
    
    # 检查 InfluxDB 是否启动成功
    if curl -s http://localhost:8086/ping >/dev/null 2>&1; then
        log_ok "InfluxDB 启动成功"
    else
        log_info "InfluxDB 可能还在启动中..."
    fi
    
    log_ok "InfluxDB 安装完成"
}

config_postgresql() {
    log_info "配置 PostgreSQL..."
    
    # 直接使用 service 命令启动 PostgreSQL
    service postgresql start 2>/dev/null || /etc/init.d/postgresql start 2>/dev/null || {
        log_info "尝试手动启动 PostgreSQL..."
        su - postgres -c "/usr/lib/postgresql/*/bin/pg_ctl -D /var/lib/postgresql/*/main -l /var/log/postgresql/postgresql.log start" 2>/dev/null || true
    }
    
    # 等待 PostgreSQL 启动
    sleep 5
    
    # 检查 PostgreSQL 是否启动
    if ! su - postgres -c "psql -c 'SELECT 1;'" >/dev/null 2>&1; then
        log_err "PostgreSQL 启动失败，请检查系统环境"
        exit 1
    fi
    
    su - postgres -c "psql -c \"DROP DATABASE IF EXISTS $DB_NAME;\"" 2>/dev/null || true
    su - postgres -c "psql -c \"DROP USER IF EXISTS $DB_USER;\"" 2>/dev/null || true
    su - postgres -c "psql -c \"CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';\""
    su - postgres -c "psql -c \"CREATE DATABASE $DB_NAME OWNER $DB_USER;\""
    log_ok "PostgreSQL 配置完成"
}

config_redis() {
    log_info "配置 Redis..."
    sed -i '/^requirepass/d' /etc/redis/redis.conf
    echo "requirepass $REDIS_PASSWORD" >> /etc/redis/redis.conf
    
    # 直接使用 service 命令
    service redis-server restart 2>/dev/null || /etc/init.d/redis-server restart 2>/dev/null || {
        log_info "尝试手动启动 Redis..."
        redis-server /etc/redis/redis.conf --daemonize yes
    }
    
    sleep 2
    log_ok "Redis 配置完成"
}

config_influxdb() {
    log_info "配置 InfluxDB..."
    influx setup --username admin --password "$INFLUXDB_PASSWORD" --org nmp --bucket monitoring --token "$INFLUXDB_TOKEN" --force 2>/dev/null || true
    log_ok "InfluxDB 配置完成"
}

deploy_backend() {
    log_info "部署后端..."
    JWT_SECRET=$(openssl rand -base64 64 | tr -d '\n')
    mkdir -p $INSTALL_DIR /etc/nmp /var/log/nmp $PLUGINS_DIR
    cp -r "$SCRIPT_DIR/nmp-platform/backend" $INSTALL_DIR/
    cp "$INSTALL_DIR/backend/server.linux-amd64" "$INSTALL_DIR/backend/server"
    chmod +x "$INSTALL_DIR/backend/server"
    
    cat > $INSTALL_DIR/backend/configs/config.yaml << EOF
server:
  host: "0.0.0.0"
  port: $BACKEND_PORT
  mode: "release"
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
    log_ok "后端部署完成"
}

deploy_frontend() {
    log_info "部署前端..."
    cp -r "$SCRIPT_DIR/nmp-platform/frontend" $INSTALL_DIR/
    
    # 配置 API 地址（使用当前服务器IP）
    echo "NEXT_PUBLIC_API_URL=http://$SERVER_IP" > $INSTALL_DIR/frontend/.env.production
    
    cd $INSTALL_DIR/frontend
    log_info "安装前端依赖..."
    npm install --legacy-peer-deps --silent 2>/dev/null || true
    
    # 重新构建前端（必须，因为 API URL 是构建时写入的）
    log_info "构建前端（这可能需要几分钟）..."
    npm run build
    
    log_ok "前端部署完成"
}

config_nginx() {
    log_info "配置 Nginx..."
    cat > /etc/nginx/sites-available/nmp << EOF
server {
    listen 80;
    server_name $SERVER_IP _;
    location / {
        proxy_pass http://127.0.0.1:$FRONTEND_PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
    location /api/ {
        proxy_pass http://127.0.0.1:$BACKEND_PORT;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
    location /health {
        proxy_pass http://127.0.0.1:$BACKEND_PORT/health;
    }
}
EOF
    ln -sf /etc/nginx/sites-available/nmp /etc/nginx/sites-enabled/
    rm -f /etc/nginx/sites-enabled/default
    
    # 测试配置并重启
    nginx -t && {
        service nginx restart 2>/dev/null || /etc/init.d/nginx restart 2>/dev/null || nginx -s reload
    }
    
    log_ok "Nginx 配置完成"
}

create_services() {
    log_info "创建启动脚本..."
    
    # 创建后端启动脚本
    cat > /usr/local/bin/nmp-backend-start << 'EOF'
#!/bin/bash
cd /opt/nmp-platform/backend
export GIN_MODE=release
exec ./server
EOF
    chmod +x /usr/local/bin/nmp-backend-start
    
    # 创建前端启动脚本
    cat > /usr/local/bin/nmp-frontend-start << 'EOF'
#!/bin/bash
cd /opt/nmp-platform/frontend
export NODE_ENV=production
export PATH=/opt/node/bin:$PATH
exec npx next start -p 3000 -H 0.0.0.0
EOF
    chmod +x /usr/local/bin/nmp-frontend-start
    
    # 尝试创建 systemd 服务（如果可用）
    if [[ -d /etc/systemd/system ]]; then
        cat > /etc/systemd/system/nmp-backend.service << EOF
[Unit]
Description=NMP Backend
After=network.target
[Service]
Type=simple
ExecStart=/usr/local/bin/nmp-backend-start
Restart=always
RestartSec=5
[Install]
WantedBy=multi-user.target
EOF

        cat > /etc/systemd/system/nmp-frontend.service << EOF
[Unit]
Description=NMP Frontend
After=network.target
[Service]
Type=simple
ExecStart=/usr/local/bin/nmp-frontend-start
Restart=on-failure
RestartSec=10
[Install]
WantedBy=multi-user.target
EOF
        systemctl daemon-reload 2>/dev/null || true
        systemctl enable nmp-backend nmp-frontend 2>/dev/null || true
    fi
    
    log_ok "启动脚本创建完成"
}

start_services() {
    log_info "启动服务..."
    
    # 启动后端
    nohup /usr/local/bin/nmp-backend-start > /var/log/nmp-backend.log 2>&1 &
    sleep 3
    
    # 检查后端是否启动成功
    if curl -s http://localhost:8080/health > /dev/null; then
        log_ok "后端启动成功"
    else
        log_err "后端启动失败，查看日志: tail -f /var/log/nmp-backend.log"
        exit 1
    fi
    
    # 启动前端
    nohup /usr/local/bin/nmp-frontend-start > /var/log/nmp-frontend.log 2>&1 &
    sleep 5
    
    log_ok "前端启动成功"
}

create_admin() {
    log_info "创建管理员..."
    sleep 2
    curl -s -X POST "http://localhost:8080/api/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin1234","email":"admin@nmp.local"}' > /dev/null 2>&1 || true
    log_ok "管理员创建完成"
}

show_result() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}    NMP Platform 安装成功!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "  访问地址: ${BLUE}http://$SERVER_IP${NC}"
    echo -e "  账户: ${BLUE}admin${NC}"
    echo -e "  密码: ${BLUE}admin1234${NC}"
    echo ""
}

main() {
    echo "NMP Platform 离线安装"
    echo "====================="
    check_root
    detect_ip
    install_packages
    install_nodejs
    install_influxdb
    config_postgresql
    config_redis
    config_influxdb
    deploy_backend
    deploy_frontend
    config_nginx
    create_services
    start_services
    create_admin
    show_result
}

main "$@"
