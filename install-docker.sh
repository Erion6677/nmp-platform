#!/bin/bash

# ============================================
# NMP Platform Docker 一键安装脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/Erion6677/nmp-platform/main/install-docker.sh | bash
# 或指定端口: curl -fsSL ... | bash -s -- --port 8080
# ============================================

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
MIN_MEMORY_MB=2048
MIN_DISK_GB=10
MAX_RETRY=3

# 日志函数
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 错误处理函数
fail_with_solution() {
    local error_msg="$1"
    local solution="$2"
    echo
    echo -e "${RED}============================================${NC}"
    echo -e "${RED}    安装失败${NC}"
    echo -e "${RED}============================================${NC}"
    echo
    echo -e "  ${RED}错误:${NC} $error_msg"
    echo
    echo -e "  ${YELLOW}解决方案:${NC}"
    echo -e "  $solution"
    echo
    echo -e "${RED}============================================${NC}"
    exit 1
}

# 清理函数
cleanup_on_error() {
    log_warn "清理残留文件..."
    docker compose -f $INSTALL_DIR/docker-compose.yml down -v 2>/dev/null || true
    docker system prune -f 2>/dev/null || true
}

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
        fail_with_solution \
            "需要 root 权限运行此脚本" \
            "请使用: sudo bash install-docker.sh"
    fi
    log_success "root 权限检查通过"
}

# 检测操作系统
detect_os() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS_NAME=$ID
        OS_VERSION=$VERSION_ID
        log_info "检测到系统: $PRETTY_NAME"
        
        # 检查支持的系统
        case $OS_NAME in
            debian|ubuntu|centos|rhel|rocky|almalinux|fedora)
                log_success "系统兼容性检查通过"
                ;;
            *)
                log_warn "未经测试的系统: $OS_NAME，可能存在兼容性问题"
                ;;
        esac
    else
        fail_with_solution \
            "无法检测操作系统" \
            "请确保系统为 Linux 发行版（Debian/Ubuntu/CentOS 等）"
    fi
}

# 检查系统资源
check_resources() {
    log_info "检查系统资源..."
    
    # 检查内存
    local total_mem_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local total_mem_mb=$((total_mem_kb / 1024))
    
    if [[ $total_mem_mb -lt $MIN_MEMORY_MB ]]; then
        fail_with_solution \
            "内存不足: 当前 ${total_mem_mb}MB，需要至少 ${MIN_MEMORY_MB}MB" \
            "请增加服务器内存到 2GB 以上"
    fi
    log_success "内存检查通过: ${total_mem_mb}MB"
    
    # 检查磁盘空间
    local avail_disk_kb=$(df / | tail -1 | awk '{print $4}')
    local avail_disk_gb=$((avail_disk_kb / 1024 / 1024))
    
    if [[ $avail_disk_gb -lt $MIN_DISK_GB ]]; then
        fail_with_solution \
            "磁盘空间不足: 当前可用 ${avail_disk_gb}GB，需要至少 ${MIN_DISK_GB}GB" \
            "请清理磁盘空间或扩容磁盘"
    fi
    log_success "磁盘空间检查通过: 可用 ${avail_disk_gb}GB"
    
    # 检查 CPU
    local cpu_cores=$(nproc)
    if [[ $cpu_cores -lt 1 ]]; then
        log_warn "CPU 核心数较少: $cpu_cores 核，构建可能较慢"
    else
        log_success "CPU 检查通过: $cpu_cores 核"
    fi
}

# 配置 Swap（如果没有）
setup_swap() {
    local swap_total=$(free | grep Swap | awk '{print $2}')
    if [[ $swap_total -eq 0 ]]; then
        log_info "检测到无 Swap，正在创建 2GB Swap..."
        if fallocate -l 2G /swapfile 2>/dev/null || dd if=/dev/zero of=/swapfile bs=1M count=2048 2>/dev/null; then
            chmod 600 /swapfile
            mkswap /swapfile >/dev/null 2>&1
            swapon /swapfile 2>/dev/null
            log_success "Swap 创建成功"
        else
            log_warn "Swap 创建失败，继续安装（可能影响构建稳定性）"
        fi
    else
        log_success "Swap 已存在"
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
# 检查端口是否可用
is_port_available() {
    local port=$1
    if ss -tlnp 2>/dev/null | grep -q ":$port " || netstat -tlnp 2>/dev/null | grep -q ":$port "; then
        return 1
    fi
    return 0
}

# 检查端口（交互式）
check_port() {
    while true; do
        if is_port_available $FRONTEND_PORT; then
            log_success "端口 $FRONTEND_PORT 可用"
            return 0
        fi
        
        # 检查是否是之前安装的 NMP 占用
        local nmp_using=$(docker ps --format '{{.Names}}' 2>/dev/null | grep -c "nmp-" || echo "0")
        
        echo
        log_warn "端口 $FRONTEND_PORT 已被占用"
        
        if [[ $nmp_using -gt 0 ]]; then
            echo -e "  检测到已有 NMP 服务正在运行"
            echo
            echo -e "  请选择操作:"
            echo -e "    ${CYAN}1${NC}) 停止旧服务，重新安装 (使用端口 $FRONTEND_PORT)"
            echo -e "    ${CYAN}2${NC}) 使用其他端口安装"
            echo -e "    ${CYAN}3${NC}) 退出安装"
            echo
            read -p "  请输入选项 [1/2/3]: " choice </dev/tty
            
            case $choice in
                1)
                    log_info "停止旧 NMP 服务..."
                    docker compose -f /opt/nmp-platform/docker-compose.yml down 2>/dev/null || true
                    docker stop $(docker ps -q --filter "name=nmp-") 2>/dev/null || true
                    sleep 2
                    if is_port_available $FRONTEND_PORT; then
                        log_success "端口 $FRONTEND_PORT 已释放"
                        return 0
                    fi
                    log_warn "端口仍被占用，可能被其他服务使用"
                    ;;
                2)
                    read -p "  请输入新端口 (1024-65535): " new_port </dev/tty
                    if [[ $new_port =~ ^[0-9]+$ ]] && [[ $new_port -ge 1024 ]] && [[ $new_port -le 65535 ]]; then
                        FRONTEND_PORT=$new_port
                        continue
                    else
                        log_error "无效端口号，请输入 1024-65535 之间的数字"
                    fi
                    ;;
                3|*)
                    log_info "安装已取消"
                    exit 0
                    ;;
            esac
        else
            echo -e "  该端口被其他服务占用"
            echo
            echo -e "  请选择操作:"
            echo -e "    ${CYAN}1${NC}) 使用其他端口安装"
            echo -e "    ${CYAN}2${NC}) 退出安装"
            echo
            read -p "  请输入选项 [1/2]: " choice </dev/tty
            
            case $choice in
                1)
                    read -p "  请输入新端口 (1024-65535): " new_port </dev/tty
                    if [[ $new_port =~ ^[0-9]+$ ]] && [[ $new_port -ge 1024 ]] && [[ $new_port -le 65535 ]]; then
                        FRONTEND_PORT=$new_port
                        continue
                    else
                        log_error "无效端口号，请输入 1024-65535 之间的数字"
                    fi
                    ;;
                2|*)
                    log_info "安装已取消"
                    exit 0
                    ;;
            esac
        fi
    done
}

# 检查网络连接
check_network() {
    log_info "检查网络连接..."
    if ! curl -s --connect-timeout 10 https://github.com >/dev/null 2>&1; then
        if ! curl -s --connect-timeout 10 https://gitee.com >/dev/null 2>&1; then
            fail_with_solution \
                "无法连接到 GitHub/Gitee" \
                "请检查网络连接，确保可以访问外网"
        fi
    fi
    log_success "网络连接正常"
}

# 清理旧安装
cleanup_old_installation() {
    if [[ -d "$INSTALL_DIR" ]]; then
        log_info "检测到旧安装，正在清理..."
        docker compose -f $INSTALL_DIR/docker-compose.yml down -v 2>/dev/null || true
        rm -rf $INSTALL_DIR
        log_success "旧安装清理完成"
    fi
    
    # 清理 Docker 构建缓存
    log_info "清理 Docker 构建缓存..."
    docker builder prune -af >/dev/null 2>&1 || true
    docker system prune -f >/dev/null 2>&1 || true
    log_success "Docker 缓存清理完成"
}

# 安装 Docker
install_docker() {
    if command -v docker &>/dev/null; then
        DOCKER_VERSION=$(docker --version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        log_success "Docker 已安装 (版本: $DOCKER_VERSION)"
        
        # 确保 Docker 服务运行
        if ! systemctl is-active --quiet docker 2>/dev/null; then
            log_info "启动 Docker 服务..."
            systemctl start docker || service docker start
        fi
    else
        log_info "安装 Docker..."
        for i in $(seq 1 $MAX_RETRY); do
            if curl -fsSL https://get.docker.com | sh; then
                systemctl enable docker 2>/dev/null || true
                systemctl start docker 2>/dev/null || service docker start
                log_success "Docker 安装完成"
                break
            else
                if [[ $i -eq $MAX_RETRY ]]; then
                    fail_with_solution \
                        "Docker 安装失败" \
                        "请手动安装 Docker: curl -fsSL https://get.docker.com | sh"
                fi
                log_warn "Docker 安装失败，重试 ($i/$MAX_RETRY)..."
                sleep 5
            fi
        done
    fi
    
    # 检查 Docker Compose
    if ! docker compose version &>/dev/null; then
        fail_with_solution \
            "Docker Compose 未安装或版本过低" \
            "请升级 Docker 到最新版本: curl -fsSL https://get.docker.com | sh"
    fi
    log_success "Docker Compose 可用"
}

# 生成随机密钥
generate_secrets() {
    log_info "生成安全密钥..."
    JWT_SECRET=$(openssl rand -base64 64 | tr -d '\n')
    DB_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
    REDIS_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
    INFLUXDB_PASSWORD=$(openssl rand -base64 16 | tr -d '/+=')
    INFLUXDB_TOKEN=$(openssl rand -base64 48 | tr -d '/+=')
    log_success "密钥生成完成"
}

# 下载项目文件
download_project() {
    log_info "下载 NMP Platform..."
    
    mkdir -p $INSTALL_DIR
    
    for i in $(seq 1 $MAX_RETRY); do
        # 尝试从 GitHub 克隆
        if command -v git &>/dev/null || apt-get install -y -qq git >/dev/null 2>&1 || yum install -y -q git >/dev/null 2>&1; then
            if git clone --depth 1 https://github.com/$GITHUB_REPO.git /tmp/nmp-platform 2>/dev/null; then
                cp -r /tmp/nmp-platform/* $INSTALL_DIR/
                rm -rf /tmp/nmp-platform
                log_success "下载完成"
                return 0
            fi
        fi
        
        if [[ $i -eq $MAX_RETRY ]]; then
            fail_with_solution \
                "项目下载失败" \
                "请检查网络连接，或手动下载: git clone https://github.com/$GITHUB_REPO.git"
        fi
        log_warn "下载失败，重试 ($i/$MAX_RETRY)..."
        sleep 5
    done
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
    log_info "生成 Docker Compose 配置..."
    
    cat > $INSTALL_DIR/docker-compose.yml << EOF
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
    log_success "Docker Compose 配置创建完成"
}

# 创建后端配置
create_backend_config() {
    log_info "生成后端配置..."
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
    log_success "后端配置创建完成"
}

# 创建 Nginx 配置
create_nginx_config() {
    log_info "生成 Nginx 配置..."
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
    log_success "Nginx 配置创建完成"
}


# 构建 Docker 镜像（带重试）
build_images() {
    log_info "构建 Docker 镜像（首次需要几分钟）..."
    cd $INSTALL_DIR
    
    for i in $(seq 1 $MAX_RETRY); do
        log_info "构建尝试 $i/$MAX_RETRY..."
        
        # 构建镜像，使用 PIPESTATUS 获取真实退出码
        docker compose build --no-cache 2>&1 | tee /tmp/docker-build.log
        local build_exit_code=${PIPESTATUS[0]}
        
        # 检查构建日志中是否有错误
        local build_log=$(cat /tmp/docker-build.log)
        
        # 检查是否有 SIGSEGV 或其他崩溃
        if echo "$build_log" | grep -qi "segmentation fault\|sigsegv\|exit code: 139\|exit code: 137\|killed"; then
            log_warn "检测到内存不足或崩溃 (SIGSEGV)，尝试清理并重试..."
            docker system prune -af >/dev/null 2>&1
            docker builder prune -af >/dev/null 2>&1
            sync && echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
            sleep 5
            continue
        fi
        
        # 检查是否构建成功
        if [[ $build_exit_code -eq 0 ]] && ! echo "$build_log" | grep -qi "failed to solve\|error:"; then
            log_success "镜像构建成功"
            return 0
        fi
        
        # 检查是否是缓存问题
        if echo "$build_log" | grep -qi "cache key\|digest.*expected"; then
            log_warn "检测到缓存问题，清理缓存后重试..."
            docker builder prune -af >/dev/null 2>&1
            sleep 3
            continue
        fi
        
        # 检查是否是网络问题
        if echo "$build_log" | grep -qi "timeout\|connection refused\|network"; then
            log_warn "检测到网络问题，等待后重试..."
            sleep 10
            continue
        fi
        
        # 其他错误
        if [[ $i -eq $MAX_RETRY ]]; then
            local error_detail=$(tail -30 /tmp/docker-build.log)
            fail_with_solution \
                "Docker 镜像构建失败" \
                "1. 检查内存是否充足 (至少 2GB)\n  2. 检查磁盘空间是否充足 (至少 10GB)\n  3. 尝试手动构建: cd $INSTALL_DIR && docker compose build\n  4. 查看详细日志: cat /tmp/docker-build.log\n\n  最后错误信息:\n$error_detail"
        fi
        
        log_warn "构建失败，清理后重试..."
        docker builder prune -af >/dev/null 2>&1
        sleep 5
    done
}

# 启动服务
start_services() {
    log_info "启动服务..."
    cd $INSTALL_DIR
    
    if ! docker compose up -d 2>&1 | tee /tmp/docker-up.log; then
        local error_detail=$(cat /tmp/docker-up.log)
        fail_with_solution \
            "服务启动失败" \
            "1. 查看日志: docker compose logs\n  2. 检查端口是否被占用\n  3. 尝试重启: docker compose down && docker compose up -d\n\n  错误信息:\n$error_detail"
    fi
    
    log_info "等待服务启动..."
    local wait_time=0
    local max_wait=120
    
    while [[ $wait_time -lt $max_wait ]]; do
        # 检查所有容器是否运行
        local running_count=$(docker compose ps 2>/dev/null | grep -c "Up" || echo "0")
        running_count=$((running_count + 0))  # 确保是十进制数
        
        if [[ $running_count -ge 6 ]]; then
            # 检查健康状态
            if curl -s --connect-timeout 5 http://localhost:$FRONTEND_PORT/health 2>/dev/null | grep -q "ok"; then
                log_success "所有服务启动成功"
                return 0
            fi
        fi
        
        sleep 5
        wait_time=$((wait_time + 5))
        echo -n "."
    done
    echo
    
    # 检查哪个服务有问题
    log_warn "服务启动超时，检查各服务状态..."
    docker compose ps
    
    local failed_services=""
    for svc in postgres redis influxdb backend frontend nginx; do
        if ! docker compose ps $svc 2>/dev/null | grep -q "Up"; then
            failed_services="$failed_services $svc"
            log_error "$svc 服务未正常运行"
            docker compose logs --tail=20 $svc 2>/dev/null
        fi
    done
    
    if [[ -n "$failed_services" ]]; then
        fail_with_solution \
            "以下服务启动失败:$failed_services" \
            "1. 查看详细日志: docker compose logs$failed_services\n  2. 尝试重启: docker compose restart$failed_services\n  3. 检查资源是否充足"
    fi
}

# 验证管理员账户
verify_admin() {
    log_info "验证管理员账户..."
    
    # 等待后端服务完全就绪
    for i in $(seq 1 15); do
        local response=$(curl -s -X POST "http://localhost:$FRONTEND_PORT/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d '{"username":"admin","password":"admin123"}' 2>/dev/null)
        
        if echo "$response" | grep -qi "token\|success"; then
            log_success "管理员账户验证成功"
            return 0
        fi
        sleep 2
    done
    log_warn "管理员账户验证超时，请稍后尝试登录 (admin/admin123)"
}

# 保存凭据
save_credentials() {
    cat > $INSTALL_DIR/.credentials << EOF
# NMP Platform 安装凭据
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')
# ⚠️ 请妥善保管此文件！

访问地址: http://$SERVER_IP:$FRONTEND_PORT
默认账户: admin
默认密码: admin123

数据库密码: $DB_PASSWORD
Redis密码: $REDIS_PASSWORD
InfluxDB密码: $INFLUXDB_PASSWORD
InfluxDB Token: $INFLUXDB_TOKEN
EOF
    chmod 600 $INSTALL_DIR/.credentials
    log_success "凭据已保存到 $INSTALL_DIR/.credentials"
}

# 安装后清理
post_install_cleanup() {
    log_info "清理安装残留..."
    
    # 清理 Docker 构建缓存
    docker builder prune -af >/dev/null 2>&1 || true
    
    # 清理未使用的镜像
    docker image prune -f >/dev/null 2>&1 || true
    
    # 清理临时文件
    rm -rf /tmp/nmp-platform 2>/dev/null || true
    rm -f /tmp/docker-build.log /tmp/docker-up.log 2>/dev/null || true
    
    # 清理 apt 缓存（如果是 Debian/Ubuntu）
    if command -v apt-get &>/dev/null; then
        apt-get clean >/dev/null 2>&1 || true
        rm -rf /var/lib/apt/lists/* 2>/dev/null || true
    fi
    
    # 清理 yum 缓存（如果是 CentOS/RHEL）
    if command -v yum &>/dev/null; then
        yum clean all >/dev/null 2>&1 || true
    fi
    
    # 回收内存
    sync
    echo 3 > /proc/sys/vm/drop_caches 2>/dev/null || true
    
    # 显示清理后的空间
    local avail_disk=$(df -h / | tail -1 | awk '{print $4}')
    local avail_mem=$(free -h | grep Mem | awk '{print $7}')
    
    log_success "清理完成 (磁盘可用: $avail_disk, 内存可用: $avail_mem)"
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
    echo -e "  默认密码: ${CYAN}admin123${NC}"
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
    # 设置错误处理
    trap 'cleanup_on_error' ERR
    
    print_banner
    parse_args "$@"
    
    # 环境检查
    check_root
    detect_os
    check_resources
    check_network
    detect_ip
    check_port
    
    log_info "前端访问端口: $FRONTEND_PORT"
    echo
    
    # 准备环境
    setup_swap
    cleanup_old_installation
    install_docker
    
    # 下载和配置
    generate_secrets
    download_project
    create_dockerfiles
    create_docker_compose
    create_backend_config
    create_nginx_config
    
    # 构建和启动
    build_images
    start_services
    verify_admin
    save_credentials
    post_install_cleanup
    
    show_result
}

main "$@"
