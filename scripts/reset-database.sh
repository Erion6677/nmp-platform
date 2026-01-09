#!/bin/bash

# NMP Platform 数据库完整重置脚本
# 清理所有数据并重置为全新状态，所有 ID 从 1 开始

set -e

echo "=========================================="
echo "NMP Platform 数据库完整重置脚本"
echo "=========================================="

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 数据库配置
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-nmp}"
DB_USER="${DB_USER:-nmp}"
DB_PASSWORD="${DB_PASSWORD:-nmp123}"

# InfluxDB 配置
INFLUX_URL="${INFLUX_URL:-http://localhost:8086}"
INFLUX_ORG="${INFLUX_ORG:-nmp}"
INFLUX_BUCKET="${INFLUX_BUCKET:-monitoring}"
INFLUX_TOKEN="${INFLUX_TOKEN:-EBJhKx75kMy1l62_L-qf3-V1g18tKANniQ7c2jrYzy2U7Zhr1gko9jWnPzK3PbOr5UYq_NzxNt5qiHZlzH2tmw==}"
 
# Redis 配置
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

echo ""
echo "警告：此操作将删除所有数据！"
echo "-------------------------------------------"
read -p "确认继续吗？(y/N): " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "操作已取消"
    exit 0
fi

echo ""
echo "1. 清理 PostgreSQL 数据库..."
echo "-------------------------------------------"

# 删除并重建数据库
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres << EOF
-- 断开所有连接
SELECT pg_terminate_backend(pg_stat_activity.pid)
FROM pg_stat_activity
WHERE pg_stat_activity.datname = '$DB_NAME'
  AND pid <> pg_backend_pid();

-- 删除数据库
DROP DATABASE IF EXISTS $DB_NAME;

-- 创建新数据库
CREATE DATABASE $DB_NAME OWNER $DB_USER;
EOF

echo "PostgreSQL 数据库已重置"

echo ""
echo "2. 清理 InfluxDB 数据..."
echo "-------------------------------------------"

# 获取 bucket ID
BUCKET_ID=$(curl -s -H "Authorization: Token $INFLUX_TOKEN" \
    "$INFLUX_URL/api/v2/buckets?org=$INFLUX_ORG&name=$INFLUX_BUCKET" | \
    grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$BUCKET_ID" ]; then
    # 删除 bucket
    curl -s -X DELETE "$INFLUX_URL/api/v2/buckets/$BUCKET_ID" \
        -H "Authorization: Token $INFLUX_TOKEN" 2>/dev/null || true
    echo "已删除 bucket: $INFLUX_BUCKET"
fi

# 获取 org ID
ORG_ID=$(curl -s -H "Authorization: Token $INFLUX_TOKEN" \
    "$INFLUX_URL/api/v2/orgs?org=$INFLUX_ORG" | \
    grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$ORG_ID" ]; then
    # 创建新 bucket（保留 10 天数据）
    curl -s -X POST "$INFLUX_URL/api/v2/buckets" \
        -H "Authorization: Token $INFLUX_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"$INFLUX_BUCKET\",\"orgID\":\"$ORG_ID\",\"retentionRules\":[{\"type\":\"expire\",\"everySeconds\":864000}]}" 2>/dev/null || true
    echo "已创建新 bucket: $INFLUX_BUCKET"
fi

echo "InfluxDB 数据已清理"

echo ""
echo "3. 清理 Redis 数据..."
echo "-------------------------------------------"

redis-cli -h $REDIS_HOST -p $REDIS_PORT FLUSHALL 2>/dev/null || true

echo "Redis 数据已清理"

echo ""
echo "4. 初始化数据库表和默认数据..."
echo "-------------------------------------------"

# 检查是否有编译好的 migrate 工具
MIGRATE_BIN="$PROJECT_DIR/backend/cmd/migrate/migrate"
if [ -f "$MIGRATE_BIN" ]; then
    cd "$PROJECT_DIR/backend"
    ./cmd/migrate/migrate -seed
    echo "数据库表和默认数据已初始化"
else
    echo "提示：migrate 工具未编译，请手动执行以下命令："
    echo "  cd $PROJECT_DIR/backend"
    echo "  go run cmd/migrate/main.go -seed"
    echo ""
    echo "或者重启后端服务，服务会自动创建表结构"
fi

echo ""
echo "=========================================="
echo "数据库重置完成！"
echo "=========================================="
echo ""
echo "所有表的 ID 序列已重置，新数据将从 ID=1 开始"
echo ""
echo "下一步："
echo "1. 如果还没有初始化默认数据，运行："
echo "   cd $PROJECT_DIR/backend && go run cmd/migrate/main.go -seed"
echo ""
echo "2. 重启后端服务"
echo ""
echo "3. 使用默认管理员账号登录: admin / admin123"
echo ""
