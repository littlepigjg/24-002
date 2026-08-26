# benzhi.Dockerfile - 评测专用 Dockerfile
# 基于 golang:1.22 官方镜像，保留完整 Go 工具链
# 支持 linux/amd64 和 linux/arm64 两种架构

FROM golang:1.22

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum（go.sum 可能不存在，用 glob 匹配）
COPY go.mod ./
COPY go.sum* ./

# 复制源代码 - 根目录 .go 文件（如 red_green_test.go）
COPY *.go ./

# 复制子目录源码
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY static/ ./static/

# 预先下载依赖（不执行编译，编译将通过 docker exec 在容器内完成）
RUN go mod download

# 暴露端口
EXPOSE 8080

# 启动服务
CMD ["go", "run", "./cmd/server"]
