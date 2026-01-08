#!/bin/bash

# --- 1. 配置你的 Docker Hub 用户名 ---
DOCKER_USER="villanel"  # <--- 请修改这里
TAG="latest"

# 定义服务及其对应的 Dockerfile 路径
declare -A SERVICES
SERVICES=(
    ["productpage"]="productpage/Dockerfile.productpage"
    ["reviews"]="reviews/Dockerfile.reviews"
    ["ratings"]="ratings/Dockerfile.ratings"
    ["details"]="details/Dockerfile.details"
)

# --- 2. 检查是否登录 ---
echo "检查 Docker Hub 登录状态..."
if ! docker info | grep -q "Username"; then
    echo "未检测到登录，请先执行 docker login"
    exit 1
fi

echo "开始构建并推送到 Docker Hub: $DOCKER_USER/..."

# --- 3. 循环构建并推送 ---
for SERVICE in "${!SERVICES[@]}"; do
    DOCKERFILE=${SERVICES[$SERVICE]}
    FULL_IMAGE_NAME="$DOCKER_USER/bookinfo-$SERVICE:$TAG"

    echo "=========================================="
    echo "正在构建: $SERVICE"
    
    # 关键：必须在根目录构建，以便 COPY pkg/ 能工作
    docker build -t "$FULL_IMAGE_NAME" -f "$DOCKERFILE" .

    if [ $? -eq 0 ]; then
        echo "构建成功，开始推送到 Docker Hub..."
        docker push "$FULL_IMAGE_NAME"
    else
        echo "❌ 错误: $SERVICE 构建失败！"
        exit 1
    fi
done

echo "=========================================="
echo "✅ 所有镜像已成功推送到 Docker Hub!"
echo "镜像列表："
for SERVICE in "${!SERVICES[@]}"; do
    echo "- $DOCKER_USER/bookinfo-$SERVICE:$TAG"
done
