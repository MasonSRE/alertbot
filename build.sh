#!/bin/bash

echo "=== AlertBot 构建脚本 ==="

# 检查Go环境
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.21+"
    exit 1
fi

# 检查Node.js环境
if ! command -v node &> /dev/null; then
    echo "❌ Node.js 未安装，请先安装 Node.js 18+"
    exit 1
fi

echo "✅ 环境检查通过"

# 构建后端
echo "🔨 构建后端..."
go mod tidy
if ! go build -o bin/alertbot cmd/server/main.go; then
    echo "❌ 后端构建失败"
    exit 1
fi

if ! go build -o bin/migrate cmd/migrate/main.go; then
    echo "❌ 迁移工具构建失败"
    exit 1
fi

echo "✅ 后端构建成功"

# 构建前端
echo "🔨 构建前端..."
cd web
if ! npm install; then
    echo "❌ 前端依赖安装失败"
    exit 1
fi

if ! npm run build; then
    echo "❌ 前端构建失败"
    exit 1
fi

cd ..
echo "✅ 前端构建成功"

echo "🎉 AlertBot 构建完成！"
echo ""
echo "启动方式："
echo "1. 启动数据库: docker-compose up -d postgres"
echo "2. 运行迁移: ./bin/migrate"
echo "3. 启动服务: ./bin/alertbot"
echo "4. 前端开发: cd web && npm run dev"