#!/bin/bash

echo "=== AlertBot 开发环境启动脚本 ==="

# 检查Docker是否运行
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未运行，请先启动 Docker"
    exit 1
fi

echo "🐘 启动PostgreSQL数据库..."
docker-compose up -d postgres

echo "⏳ 等待数据库启动..."
sleep 5

# 检查数据库是否可用
until docker-compose exec postgres pg_isready -U alertbot > /dev/null 2>&1; do
    echo "等待数据库连接..."
    sleep 2
done

echo "✅ 数据库已就绪"

# 如果Go可用，运行迁移
if command -v go &> /dev/null; then
    echo "🗄️  运行数据库迁移..."
    go run cmd/migrate/main.go
    
    echo "🚀 启动后端服务..."
    go run cmd/server/main.go &
    BACKEND_PID=$!
    
    echo "后端服务已启动 (PID: $BACKEND_PID)"
else
    echo "⚠️  Go 未安装，跳过后端启动"
fi

# 如果Node.js可用，启动前端
if command -v node &> /dev/null; then
    echo "🌐 启动前端开发服务器..."
    cd web
    if [ ! -d "node_modules" ]; then
        echo "📦 安装前端依赖..."
        npm install
    fi
    npm run dev &
    FRONTEND_PID=$!
    cd ..
    
    echo "前端服务已启动 (PID: $FRONTEND_PID)"
else
    echo "⚠️  Node.js 未安装，跳过前端启动"
fi

echo ""
echo "🎉 开发环境启动完成！"
echo ""
echo "服务地址："
echo "  - 后端API: http://localhost:8080"
echo "  - 前端界面: http://localhost:3000"
echo "  - 健康检查: http://localhost:8080/health"
echo ""
echo "按 Ctrl+C 停止所有服务"

# 等待用户中断
wait