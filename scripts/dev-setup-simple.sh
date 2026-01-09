#!/bin/bash

# NMP Platform 开发环境设置脚本

set -e

echo "🚀 设置 NMP Platform 开发环境（简化版）..."

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
go install github.com/air-verse/air@latest

# 测试编译
echo "🔨 测试后端编译..."
go build -o bin/nmp-server cmd/server/main.go
if [ $? -eq 0 ]; then
    echo "✅ 后端编译成功"
else
    echo "❌ 后端编译失败"
    exit 1
fi

cd ..

# 设置前端
echo "🎨 设置前端环境..."
cd frontend

# 安装Node依赖
echo "📦 安装Node.js依赖..."
npm install

# 测试前端构建
echo "🔨 测试前端类型检查..."
npm run type-check
if [ $? -eq 0 ]; then
    echo "✅ 前端类型检查通过"
else
    echo "⚠️ 前端类型检查有警告，但可以继续"
fi

cd ..

# 创建开发配置文件
echo "⚙️ 创建开发配置..."
if [ ! -f backend/configs/config.dev.yaml ]; then
    cp backend/configs/config.yaml backend/configs/config.dev.yaml
    echo "✅ 创建开发配置文件"
fi

# 创建启动脚本
cat > start-dev-simple.sh << 'EOF'
#!/bin/bash

echo "🚀 启动 NMP Platform 开发环境（简化版）..."

# 启动后端 (后台)
echo "🔧 启动后端服务..."
cd backend && air &
BACKEND_PID=$!

# 等待后端启动
sleep 3

# 启动前端 (后台)
echo "🎨 启动前端服务..."
cd ../frontend && npm run dev &
FRONTEND_PID=$!

echo "✅ 开发环境启动完成!"
echo "📱 前端地址: http://localhost:3000"
echo "🔧 后端地址: http://localhost:8080"
echo "🔍 后端健康检查: http://localhost:8080/health"
echo ""
echo "⚠️  注意：此版本不包含数据库，某些功能可能无法正常工作"
echo "按 Ctrl+C 停止所有服务"

# 等待中断信号
trap "echo '🛑 停止服务...'; kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; exit" INT
wait
EOF

chmod +x start-dev-simple.sh

# 创建停止脚本
cat > stop-dev-simple.sh << 'EOF'
#!/bin/bash

echo "🛑 停止 NMP Platform 开发环境..."

# 停止所有相关进程
pkill -f "air" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true

echo "✅ 开发环境已停止"
EOF

chmod +x stop-dev-simple.sh

echo ""
echo "🎉 简化开发环境设置完成!"
echo ""
echo "📋 使用说明:"
echo "  启动开发环境: ./start-dev-simple.sh"
echo "  停止开发环境: ./stop-dev-simple.sh"
echo ""
echo "🔗 服务地址:"
echo "  前端: http://localhost:3000"
echo "  后端: http://localhost:8080"
echo "  健康检查: http://localhost:8080/health"
echo ""
echo "⚠️  注意事项:"
echo "  - 此版本使用原生数据库服务"
echo "  - 需要手动安装PostgreSQL、Redis、InfluxDB"
echo "  - 如需自动安装，请使用 ./deployments/install.sh"
echo ""
echo "🚀 现在可以运行 ./start-dev-simple.sh 启动开发环境"