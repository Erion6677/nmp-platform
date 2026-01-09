#!/bin/bash

# ============================================
# NMP Platform 卸载脚本
# ============================================

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "[INFO] $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }

echo
echo "============================================"
echo "    NMP Platform 卸载脚本"
echo "============================================"
echo

if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}请使用 root 用户运行${NC}"
    exit 1
fi

read -p "确定要卸载 NMP Platform 吗？(y/N): " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "取消卸载"
    exit 0
fi

echo
read -p "是否同时删除数据库数据？(y/N): " delete_data

# 停止服务
log_info "停止服务..."
systemctl stop nmp-backend 2>/dev/null || true
pm2 delete nmp-frontend 2>/dev/null || true

# 禁用服务
log_info "禁用服务..."
systemctl disable nmp-backend 2>/dev/null || true
rm -f /etc/systemd/system/nmp-backend.service
systemctl daemon-reload

# 删除 Nginx 配置
log_info "删除 Nginx 配置..."
rm -f /etc/nginx/sites-enabled/nmp
rm -f /etc/nginx/sites-available/nmp
systemctl reload nginx 2>/dev/null || true

# 删除安装目录
log_info "删除安装目录..."
rm -rf /opt/nmp-platform
rm -rf /opt/nmp
rm -rf /etc/nmp
rm -rf /var/log/nmp

# 删除数据库（可选）
if [[ "$delete_data" == "y" || "$delete_data" == "Y" ]]; then
    log_warn "删除数据库..."
    sudo -u postgres psql -c "DROP DATABASE IF EXISTS nmp;" 2>/dev/null || true
    sudo -u postgres psql -c "DROP USER IF EXISTS nmp;" 2>/dev/null || true
    
    # 删除 InfluxDB 数据
    influx bucket delete -n monitoring -o nmp 2>/dev/null || true
    influx org delete -n nmp 2>/dev/null || true
fi

log_success "NMP Platform 已卸载"
echo
echo "注意: PostgreSQL、Redis、InfluxDB、Node.js、Nginx 等依赖未删除"
echo "如需删除，请手动执行: apt-get remove --purge <package>"
echo
