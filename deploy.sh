#!/bin/bash

# 博客系统部署脚本

set -e

echo "🚀 开始部署博客系统..."

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "❌ Docker Compose 未安装，请先安装 Docker Compose"
    exit 1
fi

# 创建环境变量文件（如果不存在）
if [ ! -f .env ]; then
    echo "📝 创建环境变量文件..."
    cp env.example .env
    echo "⚠️  请编辑 .env 文件，设置你的管理员密码"
    echo "   默认用户名: admin"
    echo "   默认密码: your_secure_password_here"
fi

# 构建和启动服务
echo "🔨 构建 Docker 镜像..."
docker compose build

echo "🚀 启动服务..."
docker compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo "🔍 检查服务状态..."
docker compose ps

# 检查健康状态
echo "🏥 检查健康状态..."
if docker compose exec blog-app wget --no-verbose --tries=1 --spider http://localhost:1834/; then
    echo "✅ 博客应用运行正常"
else
    echo "❌ 博客应用启动失败"
    docker-compose logs blog-app
    exit 1
fi

echo "🎉 部署完成！"
echo "📱 访问地址: http://localhost:1834"
echo "🔐 登录地址: http://localhost:1834/login"
echo ""
echo "📋 常用命令:"
echo "  查看日志: docker-compose logs -f"
echo "  停止服务: docker-compose down"
echo "  重启服务: docker-compose restart"
echo "  更新部署: docker-compose up -d --build" 