# 使用官方 Go 镜像作为构建环境
FROM golang:1.24-alpine AS builder

# 安装构建依赖（包括 C 编译器）
RUN apk add --no-cache gcc musl-dev

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
ARG APP_VERSION=dev
RUN CGO_ENABLED=1 go build -ldflags "-X 'main.Version=${APP_VERSION}'" -o app .

# 使用轻量级的 alpine 镜像作为运行环境
FROM alpine:latest

# 安装 ca-certificates、sqlite 和时区数据
RUN apk --no-cache add ca-certificates sqlite tzdata

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/mini-catch .

# 复制配置文件
COPY config.json /app/config.json

# 复制静态文件
COPY static/ /app/static/

# 设置时区为东八区
ENV TZ=Asia/Shanghai

# 创建数据目录并设置权限
RUN mkdir -p /app/data && \
    chmod 755 /app/data && \
    chmod 644 /app/config.json

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/series || exit 1

# 启动应用
CMD ["./mini-catch"] 