# benzhi.Dockerfile - 评测专用 Dockerfile
# 多阶段构建，支持 amd64 和 arm64 跨架构构建
# 保留完整 Go 工具链用于容器内 go build / go vet 验证
# 服务器二进制文件放在 /usr/local/bin/ 避免被卷挂载覆盖

# Stage 1: 构建阶段（使用原生架构进行交叉编译，加速构建）
FROM --platform=$BUILDPLATFORM golang:1.22 AS builder

WORKDIR /app

# 复制 go.mod 并下载依赖
COPY go.mod ./
RUN go mod download

# 复制源代码
COPY *.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY static/ ./static/

# 交叉编译目标架构二进制
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -o /server ./cmd/server

# Stage 2: 运行阶段（使用目标架构基础镜像，保留 Go 工具链）
FROM --platform=$TARGETARCH golang:1.22

WORKDIR /app

# 将服务器二进制文件放到 /usr/local/bin/ （不受卷挂载影响）
COPY --from=builder /server /usr/local/bin/server
RUN chmod +x /usr/local/bin/server

# 复制源代码到 /app（用于容器内 go build / go vet 验证）
COPY go.mod ./
COPY *.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY static/ ./static/

# 预先下载依赖（加速容器内 go build / go vet）
RUN go mod download

# 暴露端口
EXPOSE 8080

# 启动服务（从 /usr/local/bin/ 运行，不受 /app 卷挂载影响）
CMD ["/usr/local/bin/server"]
