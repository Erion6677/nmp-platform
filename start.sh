#!/bin/bash

# NMP Platform Docker 一键启动脚本
# ============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

echo -e "${CYAN}"
echo "  _   _ __  __ ____    ____  _       _    __                      "
echo " | \ | |  \/  |  _ \  |  _ \| | __ _| |_ / _| ___  _ __ _ __ ___  "
echo " |  \| | |\/| | |_) | | |_) | |/ _\` | __| |_ / _ \| '__| '\_ \` _ \ "
echo " | |\  | |  | |  __/  |  __/| | (_| | |_|  _| (_) | |  | | | | | |"
echo " |_| \_|_|  |_|_|     |_|   |_|\__,_|\__|_|  \___/|_|  |_| |_| |_|"
echo -e "${NC}"
echo "  Network Monitoring Platform - Docker 部署"
echo "  =================================================="
echo

# 检查 Docker
if ! command -v docker &>/dev/null; then
    log_error "Docker 未安装，请先安装 Docker"
    log_info "安装命令: curl -fsSL https://get.docker.com | sh"
    exit 1
fi

# 检查 Docker Compose
if ! docker compose version &>/dev/null; then
    log_error "Docker Compose 未安装"
    exit 1
fi

log_success "Docker 环境检查通过"

# 获取服务器 IP
SERVER_IP=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
[[ -z "$SERVER_IP" ]] && SERVER_IP=$(hostname -I | awk '{print $1}')
[[ -z "$SERVER_IP" ]] && SERVER_IP="localhost"

log_info "服务器 IP: $SERVER_IP"

# 更新前端 API 地址
log_info "配置前端 API 地址..."
cat > frontend/.env.production << EOF
NEXT_PUBLIC_API_URL=http://$SERVER_IP/api
EOF

# 构建并启动
log_info "构建 Docker 镜像（首次可能需要几分钟）..."
docker compose build --no-cache

log_info "启动服务..."
docker compose up -d

# 等待服务启动
log_info "等待服务启动..."
sleep 10

# 检查服务状态
log_info "检查服务状态..."
if docker compose ps | grep -q "Up"; then
    log_success "服务启动成功"
else
    log_error "部分服务启动失败"
    docker compose ps
    exit 1
fi

# 等待后端健康
log_info "等待后端服务就绪..."
for i in {1..30}; do
    if curl -s http://localhost/health > /dev/null 2>&1; then
        log_success "后端服务就绪"
        break
    fi
    sleep 2
done

# 创建默认管理员
log_info "创建默认管理员账户..."
sleep 3
curl -s -X POST "http://localhost/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123","email":"admin@nmp.local"}' > /dev/null 2>&1 || true

echo
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}    NMP Platform 启动成功！${NC}"
echo -e "${GREEN}============================================${NC}"
echo
echo -e "  访问地址: ${CYAN}http://$SERVER_IP${NC}"
echo -e "  默认账户: ${CYAN}admin${NC}"
echo -e "  默认密码: ${CYAN}admin123${NC}"
echo
echo "  管理命令:"
echo "    docker compose ps       # 查看服务状态"
echo "    docker compose logs -f  # 查看日志"
echo "    docker compose down     # 停止服务"
echo "    docker compose restart  # 重启服务"
echo
echo -e "${GREEN}============================================${NC}"
