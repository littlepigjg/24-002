# benzhi.Dockerfile - 评测专用 Dockerfile
# 基于 golang:1.22 官方镜像，保留完整 Go 工具链

FROM golang:1.22

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和源代码
COPY go.mod ./
COPY *.go ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/
COPY static/ ./static/

# 预先下载依赖（编译在容器启动时通过 go run 完成）
RUN go mod download

# 暴露端口
EXPOSE 8080

# 启动服务
CMD ["go", "run", "./cmd/server"]
