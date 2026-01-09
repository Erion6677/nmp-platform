#!/bin/bash

# ============================================
# NMP Platform Docker 一键安装脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/Erion6677/nmp-platform/main/install-docker.sh | bash
# 或指定端口: curl -fsSL ... | bash -s -- --port 8080
# ============================================

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 默认配置
FRONTEND_PORT=80
INSTALL_DIR="/opt/nmp-platform"
GITHUB_REPO="Erion6677/nmp-platform"

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

print_banner() {
    echo -e "${CYAN}"
    echo "  _   _ __  __ ____    ____  _       _    __                      "
    echo " | \ | |  \/  |  _ \  |  _ \| | __ _| |_ / _| ___  _ __ _ __ ___  "
    echo " |  \| | |\/| | |_) | | |_) | |/ _\` | __| |_ / _ \| '__| '\_ \` _ \ "
    echo " | |\  | |  | |  __/  |  __/| | (_| | |_|  _| (_) | |  | | | | | |"
    echo " |_| \_|_|  |_|_|     |_|   |_|\__,_|\__|_|  \___/|_|  |_| |_| |_|"
    echo -e "${NC}"
    echo "  Network Monitoring Platform - Docker 一键安装"
    echo "  =================================================="
    echo
}

# 解析参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --port|-p)
                FRONTEND_PORT="$2"
                shift 2
                ;;
            --help|-h)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --port, -p <端口>  指定前端访问端口 (默认: 80)"
                echo "  --help, -h         显示帮助信息"
                exit 0
                ;;
            *)
                shift
                ;;
        esac
    done
}

# 检查 root 权限
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "请使用 root 用户运行此脚本"
        log_info "sudo bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/$GITHUB_REPO/main/install-docker.sh)\""
        exit 1
    fi
}

# 检测操作系统
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS_NAME=$ID
        OS_VERSION=$VERSION_ID
        log_info "检测到系统: $PRETTY_NAME"
    else
        log_error "无法检测操作系统"
        exit 1
    fi
}

# 获取服务器 IP
detect_ip() {
    SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
    [[ -z "$SERVER_IP" ]] && SERVER_IP=$(hostname -I | awk '{print $1}')
    [[ -z "$SERVER_IP" ]] && SERVER_IP="127.0.0.1"
    log_info "服务器 IP: $SERVER_IP"
}

# 检查端口是否被占用
check_port() {
    if ss -tlnp | grep -q ":$FRONTEND_PORT "; then
        log_error "端口 $FRONTEND_PORT 已被占用"
        log_info "请使用 --port 参数指定其他端口"
        exit 1
    fi
    log_success "端口 $FRONTEND_PORT 可用"
}

# 安装 Docker
install_docker() {
    if command -v docker &>/dev/null; then
        DOCKER_VERSION=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        log_success "Docker 已安装 (版本: $DOCKER_VERSION)"
    else
        log_info "安装 Docker..."
        curl -fsSL https://get.docker.com | sh
        systemctl enable docker
        systemctl start docker
        log_success "Docker 安装完成"
    fi
    
    # 检查 Docker Compose
    if ! docker compose version &>/dev/null; then
        log_error "Docker Compose 未安装或版本过低"
        log_info "请升级 Docker 到最新版本"
        exit 1
    fi
    log_success "Docker Compose 可用"
}

# 生成随机密钥
generate_secrets() {
    JWT_SECRET=$(openssl rand -base64 64 | tr -d '\n')
    DB_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
    REDIS_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
    INFLUXDB_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
    INFLUXDB_TOKEN=$(openssl rand -base64 48 | tr -d '/+=')
}

# 下载项目文件
download_project() {
    log_info "下载 NMP Platform..."
    
    rm -rf $INSTALL_DIR
    mkdir -p $INSTALL_DIR
    
    # 尝试从 GitHub Release 下载
    RELEASE_URL="https://github.com/$GITHUB_REPO/releases/latest/download/nmp-docker.tar.gz"
    if curl -fsSL --head "$RELEASE_URL" 2>/dev/null | grep -q "200"; then
        curl -fsSL "$RELEASE_URL" | tar -xz -C $INSTALL_DIR --strip-components=1
    else
        # 从 GitHub 克隆
        apt-get install -y -qq git >/dev/null 2>&1 || true
        git clone --depth 1 https://github.com/$GITHUB_REPO.git /tmp/nmp-platform
        cp -r /tmp/nmp-platform/* $INSTALL_DIR/
        rm -rf /tmp/nmp-platform
    fi
    
    log_success "下载完成"
}

# 创建 Dockerfile
create_dockerfiles() {
    log_info "创建 Docker 构建文件..."
    
    # 后端 Dockerfile
    cat > $INSTALL_DIR/backend/Dockerfile << 'DOCKERFILE'
FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl tzdata && rm -rf /var/lib/apt/lists/*
ENV TZ=Asia/Shanghai
COPY server.linux-amd64 ./server
RUN chmod +x ./server
COPY configs ./configs
RUN mkdir -p /app/logs /opt/nmp/plugins
EXPOSE 8080
CMD ["./server"]
DOCKERFILE

    # 前端 Dockerfile（使用 Debian-slim 避免 Alpine SIGSEGV 问题）
    cat > $INSTALL_DIR/frontend/Dockerfile << 'DOCKERFILE'
FROM node:22-slim AS base

FROM base AS deps
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 make g++ && rm -rf /var/lib/apt/lists/*
COPY package.json package-lock.json* ./
RUN npm ci --legacy-peer-deps

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production
ENV NODE_OPTIONS="--max-old-space-size=2048"
RUN echo "NEXT_PUBLIC_API_URL=" > .env.production
# 修改 next.config 跳过类型检查避免 SIGSEGV
RUN if [ -f next.config.ts ]; then \
    sed -i 's/const nextConfig: NextConfig = {/const nextConfig: NextConfig = {\n  typescript: { ignoreBuildErrors: true },\n  eslint: { ignoreDuringBuilds: true },/' next.config.ts; \
    fi
RUN npm run build

FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN groupadd --system --gid 1001 nodejs
RUN useradd --system --uid 1001 nextjs
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
USER nextjs
EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]
DOCKERFILE

    log_success "Docker 构建文件创建完成"
}

# 创建 Docker Compose 配置
create_docker_compose() {
    log_info "生成配置文件..."
    
    cat > $INSTALL_DIR/docker-compose.yml << EOF
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: nmp-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: nmp
      POSTGRES_USER: nmp
      POSTGRES_PASSWORD: $DB_PASSWORD
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - nmp-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U nmp -d nmp"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: nmp-redis
    restart: unless-stopped
    command: redis-server --requirepass $REDIS_PASSWORD
    volumes:
      - redis_data:/data
    networks:
      - nmp-network
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "$REDIS_PASSWORD", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  influxdb:
    image: influxdb:2.7-alpine
    container_name: nmp-influxdb
    restart: unless-stopped
    environment:
      DOCKER_INFLUXDB_INIT_MODE: setup
      DOCKER_INFLUXDB_INIT_USERNAME: admin
      DOCKER_INFLUXDB_INIT_PASSWORD: $INFLUXDB_PASSWORD
      DOCKER_INFLUXDB_INIT_ORG: nmp
      DOCKER_INFLUXDB_INIT_BUCKET: monitoring
      DOCKER_INFLUXDB_INIT_ADMIN_TOKEN: $INFLUXDB_TOKEN
    volumes:
      - influxdb_data:/var/lib/influxdb2
    networks:
      - nmp-network
    healthcheck:
      test: ["CMD", "influx", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: nmp-backend
    restart: unless-stopped
    environment:
      - GIN_MODE=release
    volumes:
      - ./backend/configs:/app/configs:ro
      - backend_logs:/app/logs
      - plugins_data:/opt/nmp/plugins
    networks:
      - nmp-network
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      influxdb:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: nmp-frontend
    restart: unless-stopped
    environment:
      - NODE_ENV=production
    networks:
      - nmp-network
    depends_on:
      backend:
        condition: service_healthy

  nginx:
    image: nginx:alpine
    container_name: nmp-nginx
    restart: unless-stopped
    ports:
      - "$FRONTEND_PORT:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    networks:
      - nmp-network
    depends_on:
      - frontend
      - backend

networks:
  nmp-network:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
  influxdb_data:
  backend_logs:
  plugins_data:
EOF
}

# 创建后端配置
create_backend_config() {
    mkdir -p $INSTALL_DIR/backend/configs
    
    cat > $INSTALL_DIR/backend/configs/config.yaml << EOF
server:
  host: "0.0.0.0"
  port: 8080
  mode: "release"
  read_timeout: "30s"
  write_timeout: "30s"
  public_url: "http://$SERVER_IP"

database:
  host: "postgres"
  port: 5432
  database: "nmp"
  username: "nmp"
  password: "$DB_PASSWORD"
  ssl_mode: "disable"

redis:
  host: "redis"
  port: 6379
  password: "$REDIS_PASSWORD"
  db: 0

influxdb:
  url: "http://influxdb:8086"
  token: "$INFLUXDB_TOKEN"
  org: "nmp"
  bucket: "monitoring"

auth:
  jwt_secret: "$JWT_SECRET"
  token_expiry: "24h"
  refresh_expiry: "168h"

plugins:
  directory: "/opt/nmp/plugins"
EOF
}

# 创建前端环境配置
create_frontend_config() {
    cat > $INSTALL_DIR/frontend/.env.production << EOF
NEXT_PUBLIC_API_URL=http://$SERVER_IP:$FRONTEND_PORT/api
EOF
}

# 创建 Nginx 配置
create_nginx_config() {
    mkdir -p $INSTALL_DIR/nginx
    
    cat > $INSTALL_DIR/nginx/nginx.conf << 'NGINXEOF'
user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    sendfile on;
    keepalive_timeout 65;
    gzip on;
    gzip_types text/plain text/css application/json application/javascript;

    upstream frontend { server frontend:3000; }
    upstream backend { server backend:8080; }

    server {
        listen 80;
        server_name _;

        location / {
            proxy_pass http://frontend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }

        location /api/ {
            proxy_pass http://backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }

        location /health {
            proxy_pass http://backend/health;
            access_log off;
        }

        location /ws {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_read_timeout 86400;
        }
    }
}
NGINXEOF
}

# 构建并启动服务
start_services() {
    log_info "构建 Docker 镜像（首次需要几分钟）..."
    cd $INSTALL_DIR
    docker compose build --quiet
    
    log_info "启动服务..."
    docker compose up -d
    
    log_info "等待服务启动..."
    sleep 15
    
    # 检查服务状态
    if ! docker compose ps | grep -q "Up"; then
        log_error "服务启动失败"
        docker compose logs --tail=20
        exit 1
    fi
}

# 创建管理员账户
create_admin() {
    log_info "创建管理员账户..."
    sleep 5
    
    for i in {1..10}; do
        if curl -s -X POST "http://localhost:$FRONTEND_PORT/api/v1/auth/register" \
            -H "Content-Type: application/json" \
            -d '{"username":"admin","password":"admin1234","email":"admin@nmp.local"}' | grep -q "success\|admin"; then
            log_success "管理员账户创建成功"
            return 0
        fi
        sleep 2
    done
    log_warn "管理员账户可能已存在"
}

# 保存凭据
save_credentials() {
    cat > $INSTALL_DIR/.credentials << EOF
# NMP Platform 安装凭据
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ⚠️ 请妥善保管此文件！

访问地址: http://$SERVER_IP:$FRONTEND_PORT
默认账户: admin
默认密码: admin1234

数据库密码: $DB_PASSWORD
Redis密码: $REDIS_PASSWORD
InfluxDB密码: $INFLUXDB_PASSWORD
EOF
    chmod 600 $INSTALL_DIR/.credentials
}

# 显示结果
show_result() {
    echo
    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}    NMP Platform 安装成功！${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo
    echo -e "  访问地址: ${CYAN}http://$SERVER_IP:$FRONTEND_PORT${NC}"
    echo -e "  默认账户: ${CYAN}admin${NC}"
    echo -e "  默认密码: ${CYAN}admin1234${NC}"
    echo
    echo -e "  ${YELLOW}⚠️ 请首次登录后修改密码！${NC}"
    echo
    echo "  管理命令:"
    echo "    cd $INSTALL_DIR"
    echo "    docker compose ps        # 查看状态"
    echo "    docker compose logs -f   # 查看日志"
    echo "    docker compose restart   # 重启服务"
    echo "    docker compose down      # 停止服务"
    echo
    echo "  凭据文件: $INSTALL_DIR/.credentials"
    echo
    echo -e "${GREEN}============================================${NC}"
}

# 主函数
main() {
    print_banner
    parse_args "$@"
    check_root
    detect_os
    detect_ip
    check_port
    
    log_info "前端访问端口: $FRONTEND_PORT"
    echo
    
    install_docker
    generate_secrets
    download_project
    create_dockerfiles
    create_docker_compose
    create_backend_config
    create_frontend_config
    create_nginx_config
    start_services
    create_admin
    save_credentials
    show_result
}

main "$@"
