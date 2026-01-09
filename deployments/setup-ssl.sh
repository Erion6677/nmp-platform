#!/bin/bash
# NMP Platform SSL 证书配置脚本
# 支持 Let's Encrypt 自动证书和自签名证书

set -e

SSL_DIR="/etc/nginx/ssl"
CERTBOT_DIR="/var/www/certbot"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

usage() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --domain DOMAIN     域名 (Let's Encrypt 必需)"
    echo "  --email EMAIL       邮箱 (Let's Encrypt 必需)"
    echo "  --self-signed       生成自签名证书 (开发/测试环境)"
    echo "  --help              显示帮助信息"
    echo ""
    echo "示例:"
    echo "  # Let's Encrypt 证书"
    echo "  $0 --domain example.com --email admin@example.com"
    echo ""
    echo "  # 自签名证书 (开发环境)"
    echo "  $0 --self-signed"
}

generate_self_signed() {
    log_info "生成自签名 SSL 证书..."
    
    mkdir -p "$SSL_DIR"
    
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout "$SSL_DIR/privkey.pem" \
        -out "$SSL_DIR/fullchain.pem" \
        -subj "/C=CN/ST=Beijing/L=Beijing/O=NMP/OU=Dev/CN=localhost"
    
    chmod 600 "$SSL_DIR/privkey.pem"
    chmod 644 "$SSL_DIR/fullchain.pem"
    
    log_info "自签名证书已生成:"
    log_info "  证书: $SSL_DIR/fullchain.pem"
    log_info "  私钥: $SSL_DIR/privkey.pem"
    log_warn "注意: 自签名证书仅用于开发/测试环境，浏览器会显示安全警告"
}

generate_letsencrypt() {
    local domain="$1"
    local email="$2"
    
    if [ -z "$domain" ] || [ -z "$email" ]; then
        log_error "Let's Encrypt 需要 --domain 和 --email 参数"
        exit 1
    fi
    
    log_info "使用 Let's Encrypt 生成 SSL 证书..."
    log_info "域名: $domain"
    log_info "邮箱: $email"
    
    # 检查 certbot 是否安装
    if ! command -v certbot &> /dev/null; then
        log_info "安装 certbot..."
        apt-get update
        apt-get install -y certbot python3-certbot-nginx
    fi
    
    mkdir -p "$CERTBOT_DIR"
    mkdir -p "$SSL_DIR"
    
    # 获取证书
    certbot certonly --webroot \
        -w "$CERTBOT_DIR" \
        -d "$domain" \
        --email "$email" \
        --agree-tos \
        --non-interactive
    
    # 创建符号链接
    ln -sf "/etc/letsencrypt/live/$domain/fullchain.pem" "$SSL_DIR/fullchain.pem"
    ln -sf "/etc/letsencrypt/live/$domain/privkey.pem" "$SSL_DIR/privkey.pem"
    
    log_info "Let's Encrypt 证书已生成并链接到 $SSL_DIR"
    
    # 设置自动续期
    setup_auto_renewal "$domain"
}

setup_auto_renewal() {
    local domain="$1"
    
    log_info "配置证书自动续期..."
    
    # 创建续期脚本
    cat > /etc/cron.d/certbot-renewal << EOF
# 每天凌晨 2:30 检查证书续期
30 2 * * * root certbot renew --quiet --post-hook "nginx -s reload"
EOF
    
    log_info "自动续期已配置 (每天 02:30 检查)"
}

# 解析参数
DOMAIN=""
EMAIL=""
SELF_SIGNED=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --domain)
            DOMAIN="$2"
            shift 2
            ;;
        --email)
            EMAIL="$2"
            shift 2
            ;;
        --self-signed)
            SELF_SIGNED=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            usage
            exit 1
            ;;
    esac
done

# 执行
if [ "$SELF_SIGNED" = true ]; then
    generate_self_signed
elif [ -n "$DOMAIN" ]; then
    generate_letsencrypt "$DOMAIN" "$EMAIL"
else
    log_error "请指定 --self-signed 或 --domain"
    usage
    exit 1
fi

log_info "SSL 配置完成!"
log_info "请使用 nginx-ssl.conf 配置文件启动 nginx:"
log_info "  cp nginx-ssl.conf /etc/nginx/nginx.conf"
log_info "  nginx -t && nginx -s reload"
