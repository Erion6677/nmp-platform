# 后端 Dockerfile - 使用预编译的二进制文件
FROM debian:bookworm-slim

WORKDIR /app

# 安装运行时依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl tzdata \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai

# 复制预编译的二进制文件
COPY server.linux-amd64 ./server
RUN chmod +x ./server

# 复制配置目录
COPY configs ./configs

# 创建日志目录
RUN mkdir -p /app/logs /opt/nmp/plugins

EXPOSE 8080

CMD ["./server"]
