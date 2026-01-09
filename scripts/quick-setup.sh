#!/bin/bash

# NMP Platform 快速设置脚本

set -e

echo "🚀 快速设置 NMP Platform 开发环境..."

# 检查必要工具
echo "📋 检查必要工具..."
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装"
    exit 1
fi
echo "✅ Go 已安装: $(go version)"

if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装"
    exit 1
fi
echo "✅ Node.js 已安装: $(node --version)"

# 创建必要目录
echo "📁 创建项目目录..."
mkdir -p backend/{bin,tmp,logs}
mkdir -p frontend/dist

# 设置后端
echo "🔧 设置后端..."
cd backend
echo "📦 下载Go依赖..."
go mod tidy

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
echo "🎨 设置前端..."
cd frontend
echo "📦 安装Node.js依赖..."
npm install --no-audit

cd ..

# 创建启动脚本
cat > start-backend.sh << 'EOF'
#!/bin/bash
echo "🔧 启动后端服务..."
cd backend
go run cmd/server/main.go
EOF

cat > start-frontend.sh << 'EOF'
#!/bin/bash
echo "🎨 启动前端服务..."
cd frontend
npm run dev
EOF

chmod +x start-backend.sh start-frontend.sh

echo ""
echo "🎉 快速设置完成!"
echo ""
echo "📋 启动说明:"
echo "  启动后端: ./start-backend.sh"
echo "  启动前端: ./start-frontend.sh"
echo ""
echo "🔗 服务地址:"
echo "  前端: http://localhost:3000"
echo "  后端: http://localhost:8080"
echo "  健康检查: http://localhost:8080/health"