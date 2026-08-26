#!/bin/bash
# build_benzhi_docker.sh - Docker 构建脚本
# 用法：./build_benzhi_docker.sh [镜像名] [标签] [平台]
# 默认参数：镜像名=logalert，标签=latest，平台=linux/amd64

set -e

# 配置变量
IMAGE_NAME=${1:-logalert}
TAG=${2:-latest}
PLATFORM=${3:-linux/amd64}
DOCKERFILE="benzhi.Dockerfile"
FULL_IMAGE="${IMAGE_NAME}:${TAG}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 打印配置
echo -e "${YELLOW}=== 构建配置 ===${NC}"
echo "镜像名: ${IMAGE_NAME}"
echo "标签: ${TAG}"
echo "平台: ${PLATFORM}"
echo "Dockerfile: ${DOCKERFILE}"
echo "完整镜像: ${FULL_IMAGE}"
echo ""

# 检查 Docker 是否可用
echo -e "${YELLOW}=== 环境检查 ===${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误：未检测到 docker 命令${NC}"
    echo "请先安装 Docker：https://docs.docker.com/get-docker/"
    exit 1
fi

# 检查 Docker 守护进程
if ! docker info &> /dev/null; then
    echo -e "${RED}错误：Docker 守护进程未运行${NC}"
    echo "请启动 Docker 服务：sudo systemctl start docker"
    exit 1
fi

echo -e "${GREEN}✓ Docker 可用${NC}"
echo "Docker 版本: $(docker --version)"

# 检查 Dockerfile 是否存在
if [ ! -f "$DOCKERFILE" ]; then
    echo -e "${RED}错误：${DOCKERFILE} 不存在${NC}"
    exit 1
fi

# 检查项目文件
if [ ! -f "go.mod" ]; then
    echo -e "${RED}错误：go.mod 不存在，请在项目根目录执行此脚本${NC}"
    exit 1
fi

echo -e "${GREEN}✓ 项目文件检查通过${NC}"
echo ""

# 执行构建
echo -e "${YELLOW}=== 开始构建 ===${NC}"
echo "docker build -f ${DOCKERFILE} -t ${FULL_IMAGE} --platform ${PLATFORM} ."
echo ""

BUILD_START=$(date +%s)

docker build \
    -f "${DOCKERFILE}" \
    -t "${FULL_IMAGE}" \
    --platform "${PLATFORM}" \
    .

BUILD_END=$(date +%s)
BUILD_TIME=$((BUILD_END - BUILD_START))

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=== 构建成功 ===${NC}"
    echo "镜像: ${FULL_IMAGE}"
    echo "平台: ${PLATFORM}"
    echo "耗时: ${BUILD_TIME}秒"
    echo ""
    echo -e "${YELLOW}=== 运行示例 ===${NC}"
    echo "docker run -d --name ${IMAGE_NAME} -p 8080:8080 ${FULL_IMAGE}"
    echo ""
    echo "查看日志: docker logs -f ${IMAGE_NAME}"
    echo "停止容器: docker stop ${IMAGE_NAME}"
    echo "删除容器: docker rm -f ${IMAGE_NAME}"
    echo ""
    echo "测试接口:"
    echo "  curl http://localhost:8080/health"
    echo "  curl http://localhost:8080/ready"
    echo "  curl -X POST http://localhost:8080/api/logs -H 'Content-Type: application/json' -d '{\"level\":\"INFO\",\"source\":\"test\",\"message\":\"hello\"}'"
else
    echo -e "${RED}=== 构建失败 ===${NC}"
    exit 1
fi
