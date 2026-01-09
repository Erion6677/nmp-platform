#!/bin/bash

# NMP Platform 开发环境设置脚本（原生部署）

set -e

echo "🚀 设置 NMP Platform 开发环境..."

# 检查必要工具
check_tool() {
    if ! command -v $1 &> /dev/null; then
        echo "❌ $1 未安装，请先安装 $1"
        exit 1
    else
        echo "✅ $1 已安装"
    fi
}

echo "📋 检查必要工具..."
check_tool "go"
check_tool "node"

# 检查Go版本
GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+' | sed 's/go//')
if [[ $(echo "$GO_VERSION 1.21" | awk '{print ($1 >= $2)}') == 1 ]]; then
    echo "✅ Go 版本: $GO_VERSION (满足要求 >= 1.21)"
else
    echo "❌ Go 版本过低: $GO_VERSION，需要 >= 1.21"
    exit 1
fi

# 检查Node版本
NODE_VERSION=$(node --version | sed 's/v//')
if [[ $(echo "$NODE_VERSION 18.0.0" | awk '{print ($1 >= $2)}') == 1 ]]; then
    echo "✅ Node.js 版本: $NODE_VERSION (满足要求 >= 18.0.0)"
else
    echo "❌ Node.js 版本过低: $NODE_VERSION，需要 >= 18.0.0"
    exit 1
fi

# 检查数据库服务
echo "�️ 建检查数据库服务..."

# 检查PostgreSQL
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL 客户端已安装"
    if systemctl is-active --quiet postgresql 2>/dev/null; then
        echo "✅ PostgreSQL 服务正在运行"
    else
        echo "⚠️  PostgreSQL 服务未运行，请手动启动："
        echo "   sudo systemctl start postgresql"
    fi
else
    echo "⚠️  PostgreSQL 未安装，请先安装数据库"
fi

# 检查Redis
if command -v redis-cli &> /dev/null; then
    echo "✅ Redis 客户端已安装"
    if systemctl is-active --quiet redis-server 2>/dev/null || systemctl is-active --quiet redis 2>/dev/null; then
        echo "✅ Redis 服务正在运行"
    else
        echo "⚠️  Redis 服务未运行，请手动启动："
        echo "   sudo systemctl start redis-server"
    fi
else
    echo "⚠️  Redis 未安装，请先安装Redis"
fi

# 检查InfluxDB
if command -v influx &> /dev/null; then
    echo "✅ InfluxDB 客户端已安装"
    if systemctl is-active --quiet influxdb 2>/dev/null; then
        echo "✅ InfluxDB 服务正在运行"
    else
        echo "⚠️  InfluxDB 服务未运行，请手动启动："
        echo "   sudo systemctl start influxdb"
    fi
else
    echo "⚠️  InfluxDB 未安装，请先安装InfluxDB"
fi

# 创建必要目录
echo "📁 创建项目目录..."
mkdir -p backend/{bin,tmp,logs}
mkdir -p frontend/dist
mkdir -p docs
mkdir -p deployments

# 设置后端
echo "🔧 设置后端环境..."
cd backend

# 下载Go依赖
echo "📦 下载Go依赖..."
go mod tidy
go mod download

# 安装开发工具
echo "🛠️ 安装开发工具..."
go install github.com/cosmtrek/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

cd ..

# 设置前端
echo "🎨 设置前端环境..."
cd frontend

# 安装Node依赖
echo "📦 安装Node.js依赖..."
npm install

cd ..

# 创建开发配置文件
echo "⚙️ 创建开发配置..."
if [ ! -f backend/configs/config.dev.yaml ]; then
    cp backend/configs/config.yaml backend/configs/config.dev.yaml
    echo "✅ 创建开发配置文件"
fi

# 创建启动脚本
cat > start-dev.sh << 'EOF'
#!/bin/bash

echo "🚀 启动 NMP Platform 开发环境..."

# 检查数据库服务
echo "📊 检查数据库服务..."
if ! systemctl is-active --quiet postgresql 2>/dev/null; then
    echo "⚠️  PostgreSQL 服务未运行，尝试启动..."
    sudo systemctl start postgresql || echo "❌ 无法启动PostgreSQL，请手动启动"
fi

if ! systemctl is-active --quiet redis-server 2>/dev/null && ! systemctl is-active --quiet redis 2>/dev/null; then
    echo "⚠️  Redis 服务未运行，尝试启动..."
    sudo systemctl start redis-server 2>/dev/null || sudo systemctl start redis 2>/dev/null || echo "❌ 无法启动Redis，请手动启动"
fi

if ! systemctl is-active --quiet influxdb 2>/dev/null; then
    echo "⚠️  InfluxDB 服务未运行，尝试启动..."
    sudo systemctl start influxdb 2>/dev/null || echo "❌ 无法启动InfluxDB，请手动启动"
fi

# 等待服务启动
sleep 3

# 启动后端 (后台)
echo "🔧 启动后端服务..."
cd backend && air &
BACKEND_PID=$!

# 启动前端 (后台)
echo "🎨 启动前端服务..."
cd ../frontend && npm run dev &
FRONTEND_PID=$!

echo "✅ 开发环境启动完成!"
echo "� 前端地址址: http://localhost:3000"
echo "🔧 后端地址: http://localhost:8080"
echo "📊 InfluxDB: http://localhost:8086"
echo ""
echo "按 Ctrl+C 停止所有服务"

# 等待中断信号
trap "echo '� 停止"服务...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit" INT
wait
EOF

chmod +x start-dev.sh

# 创建停止脚本
cat > stop-dev.sh << 'EOF'
#!/bin/bash

echo "🛑 停止 NMP Platform 开发环境..."

# 停止所有相关进程
pkill -f "air" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true

echo "✅ 开发环境已停止"
EOF

chmod +x stop-dev.sh

echo ""
echo "🎉 开发环境设置完成!"
echo ""
echo "📋 使用说明:"
echo "  启动开发环境: ./start-dev.sh"
echo "  停止开发环境: ./stop-dev.sh"
echo ""
echo "🔗 服务地址:"
echo "  前端: http://localhost:3000"
echo "  后端: http://localhost:8080"
echo "  InfluxDB: http://localhost:8086"
echo "  PostgreSQL: localhost:5432"
echo "  Redis: localhost:6379"
echo ""
echo "👤 默认管理员账户:"
echo "  用户名: admin"
echo "  密码: admin123"
echo ""
echo "⚠️  注意事项:"
echo "  - 请确保已安装并启动PostgreSQL、Redis、InfluxDB服务"
echo "  - 如需自动安装数据库，请运行: ./deployments/install.sh"