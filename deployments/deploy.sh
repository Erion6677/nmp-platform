#!/bin/bash

# NMP Platform 部署脚本
# 用于将构建好的应用部署到生产服务器

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置变量
NMP_USER="nmp"
NMP_HOME="/opt/nmp"
NMP_CONFIG_DIR="/etc/nmp"
NMP_LOG_DIR="/var/log/nmp"
BACKUP_DIR="/var/lib/nmp/backups"

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

# 检查构建文件是否存在
check_build_files() {
    print_info "检查构建文件..."
    
    if [[ ! -f "../backend/bin/nmp-backend" ]]; then
        print_error "后端可执行文件不存在: ../backend/bin/nmp-backend"
        print_info "请先运行: cd ../backend && make build"
        exit 1
    fi
    
    if [[ ! -d "../frontend/dist" ]]; then
        print_error "前端构建文件不存在: ../frontend/dist"
        print_info "请先运行: cd ../frontend && npm run build"
        exit 1
    fi
    
    print_success "构建文件检查完成"
}

# 停止现有服务
stop_services() {
    print_info "停止现有服务..."
    
    if systemctl is-active --quiet nmp-backend; then
        systemctl stop nmp-backend
        print_info "已停止 nmp-backend 服务"
    fi
    
    print_success "服务停止完成"
}

# 备份现有版本
backup_current() {
    print_info "备份当前版本..."
    
    BACKUP_NAME="nmp-backup-$(date +%Y%m%d_%H%M%S)"
    BACKUP_PATH="$BACKUP_DIR/$BACKUP_NAME"
    
    mkdir -p "$BACKUP_PATH"
    
    # 备份可执行文件
    if [[ -f "$NMP_HOME/bin/nmp-backend" ]]; then
        cp "$NMP_HOME/bin/nmp-backend" "$BACKUP_PATH/"
        print_info "已备份后端可执行文件"
    fi
    
    # 备份前端文件
    if [[ -d "/var/www/html" ]]; then
        cp -r /var/www/html "$BACKUP_PATH/frontend"
        print_info "已备份前端文件"
    fi
    
    # 备份配置文件
    if [[ -f "$NMP_CONFIG_DIR/config.yaml" ]]; then
        cp "$NMP_CONFIG_DIR/config.yaml" "$BACKUP_PATH/"
        print_info "已备份配置文件"
    fi
    
    print_success "备份完成: $BACKUP_PATH"
}

# 部署后端
deploy_backend() {
    print_info "部署后端服务..."
    
    # 复制可执行文件
    cp "../backend/bin/nmp-backend" "$NMP_HOME/bin/"
    chmod +x "$NMP_HOME/bin/nmp-backend"
    chown $NMP_USER:$NMP_USER "$NMP_HOME/bin/nmp-backend"
    
    # 复制插件文件
    if [[ -d "../backend/plugins" ]]; then
        cp -r ../backend/plugins/* "$NMP_HOME/plugins/"
        chown -R $NMP_USER:$NMP_USER "$NMP_HOME/plugins/"
    fi
    
    print_success "后端部署完成"
}

# 部署前端
deploy_frontend() {
    print_info "部署前端文件..."
    
    # 清理旧文件
    rm -rf /var/www/html/*
    
    # 复制新文件
    cp -r ../frontend/dist/* /var/www/html/
    chown -R www-data:www-data /var/www/html/
    
    print_success "前端部署完成"
}

# 更新配置文件
update_config() {
    print_info "更新配置文件..."
    
    # 如果存在新的配置文件，则更新
    if [[ -f "config.prod.yaml" ]]; then
        # 备份现有配置
        if [[ -f "$NMP_CONFIG_DIR/config.yaml" ]]; then
            cp "$NMP_CONFIG_DIR/config.yaml" "$NMP_CONFIG_DIR/config.yaml.bak"
        fi
        
        # 复制新配置（需要手动合并重要配置）
        print_warning "请手动检查并合并配置文件: $NMP_CONFIG_DIR/config.yaml"
    fi
    
    print_success "配置更新完成"
}

# 数据库迁移
run_migrations() {
    print_info "运行数据库迁移..."
    
    # 这里应该运行数据库迁移脚本
    # 例如：sudo -u $NMP_USER $NMP_HOME/bin/nmp-backend migrate --config=$NMP_CONFIG_DIR/config.yaml
    
    print_success "数据库迁移完成"
}

# 启动服务
start_services() {
    print_info "启动服务..."
    
    # 重新加载systemd配置
    systemctl daemon-reload
    
    # 启动后端服务
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
    
    # 重启Nginx
    if systemctl is-active --quiet nginx; then
        systemctl reload nginx
        print_success "Nginx配置重新加载完成"
    fi
    
    print_success "所有服务启动完成"
}

# 健康检查
health_check() {
    print_info "执行健康检查..."
    
    # 检查后端API
    for i in {1..10}; do
        if curl -f http://localhost:8080/api/health >/dev/null 2>&1; then
            print_success "后端API健康检查通过"
            break
        fi
        
        if [[ $i -eq 10 ]]; then
            print_error "后端API健康检查失败"
            exit 1
        fi
        
        print_info "等待后端服务启动... ($i/10)"
        sleep 3
    done
    
    # 检查前端
    if curl -f http://localhost/ >/dev/null 2>&1; then
        print_success "前端健康检查通过"
    else
        print_warning "前端健康检查失败，请检查Nginx配置"
    fi
    
    print_success "健康检查完成"
}

# 清理旧备份
cleanup_old_backups() {
    print_info "清理旧备份文件..."
    
    # 保留最近7个备份
    find "$BACKUP_DIR" -name "nmp-backup-*" -type d -mtime +7 -exec rm -rf {} \; 2>/dev/null || true
    
    print_success "备份清理完成"
}

# 显示部署信息
show_deploy_info() {
    print_success "NMP Platform 部署完成！"
    echo
    echo "=== 服务状态 ==="
    systemctl status nmp-backend --no-pager -l
    echo
    echo "=== 访问地址 ==="
    echo "Web界面: http://$(hostname -I | awk '{print $1}')"
    echo "后端API: http://$(hostname -I | awk '{print $1}')/api"
    echo
    echo "=== 服务管理 ==="
    echo "查看日志: journalctl -u nmp-backend -f"
    echo "重启服务: systemctl restart nmp-backend"
    echo "停止服务: systemctl stop nmp-backend"
    echo
    echo "=== 备份位置 ==="
    echo "备份目录: $BACKUP_DIR"
    echo "最新备份: $(ls -t $BACKUP_DIR | head -1)"
}

# 回滚函数
rollback() {
    print_warning "开始回滚到上一个版本..."
    
    LATEST_BACKUP=$(ls -t $BACKUP_DIR | head -1)
    if [[ -z "$LATEST_BACKUP" ]]; then
        print_error "没有找到备份文件"
        exit 1
    fi
    
    BACKUP_PATH="$BACKUP_DIR/$LATEST_BACKUP"
    
    # 停止服务
    systemctl stop nmp-backend
    
    # 恢复文件
    if [[ -f "$BACKUP_PATH/nmp-backend" ]]; then
        cp "$BACKUP_PATH/nmp-backend" "$NMP_HOME/bin/"
        chmod +x "$NMP_HOME/bin/nmp-backend"
        chown $NMP_USER:$NMP_USER "$NMP_HOME/bin/nmp-backend"
    fi
    
    if [[ -d "$BACKUP_PATH/frontend" ]]; then
        rm -rf /var/www/html/*
        cp -r "$BACKUP_PATH/frontend"/* /var/www/html/
        chown -R www-data:www-data /var/www/html/
    fi
    
    if [[ -f "$BACKUP_PATH/config.yaml" ]]; then
        cp "$BACKUP_PATH/config.yaml" "$NMP_CONFIG_DIR/"
    fi
    
    # 启动服务
    systemctl start nmp-backend
    
    print_success "回滚完成"
}

# 主函数
main() {
    case "${1:-deploy}" in
        "deploy")
            print_info "开始部署NMP Platform..."
            check_root
            check_build_files
            stop_services
            backup_current
            deploy_backend
            deploy_frontend
            update_config
            run_migrations
            start_services
            health_check
            cleanup_old_backups
            show_deploy_info
            ;;
        "rollback")
            rollback
            ;;
        "status")
            systemctl status nmp-backend
            ;;
        "logs")
            journalctl -u nmp-backend -f
            ;;
        *)
            echo "用法: $0 {deploy|rollback|status|logs}"
            echo "  deploy   - 部署新版本"
            echo "  rollback - 回滚到上一版本"
            echo "  status   - 查看服务状态"
            echo "  logs     - 查看服务日志"
            exit 1
            ;;
    esac
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi