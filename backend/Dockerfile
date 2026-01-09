# 后端 Dockerfile - 多阶段构建
# 阶段1: 编译
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o server ./cmd/server

# 阶段2: 运行
FROM debian:bookworm-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

# 从构建阶段复制二进制文件
COPY --from=builder /app/server ./server
RUN chmod +x ./server

# 复制配置目录
COPY configs ./configs

# 创建日志目录
RUN mkdir -p /app/logs /opt/nmp/plugins

EXPOSE 8080

CMD ["./server"]
