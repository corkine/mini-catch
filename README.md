# MiniCatch 剧集追踪系统

一个用于追踪 [Mini4k](https://www.mini4k.com) 剧集更新的 Web 应用程序，支持 Slack 通知和外部爬虫集成。

## 功能特性

- 🎬 剧集管理：添加、编辑、删除剧集
- 📊 状态追踪：追踪观看状态和更新状态
- 🔔 Slack 通知：新集数更新时自动发送通知
- 🕷️ 爬虫集成：提供标准化的爬虫接口

## 技术栈

- **后端**: Go + Chi Router + SQLite
- **前端**: Tailwind CSS + Alpine.js + Font Awesome
- **通知**: Slack Webhook
- **数据库**: SQLite

## 安装和运行

### 1. 克隆项目

```bash
git clone <repository-url>
cd mini-catch
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 配置

编辑 `config.json` 文件：

```json
{
    "port": "8080",
    "auth": {
        "username": "admin",
        "password": "admin123"
    }
}
```

**配置说明：**
- `port`: 服务器监听端口
- `auth`: 认证配置
  - `username`: 登录用户名
  - `password`: 登录密码

### 4. 运行应用

> 使用 go cli 本地运行

```bash
go run cmd/server/main.go
go run cmd/crawler/main.go
```

> 基于容器构建和运行

```bash
# 构建服务器镜像
podman build -t corkine/mini-catch:latest . --network=host

# 构建爬虫镜像
podman build -t corkine/mini-catch-crawler:latest -f Dockerfile.crawler . --network=host

# 运行服务器
podman run -d --name mini-catch-server \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data:Z \
  -v $(pwd)/config.json:/app/config.json:Z \
  --restart=always \
  --user=root \
  corkine/mini-catch:latest

# 运行爬虫
podman run --rm corkine/mini-catch-crawler:latest \
  --server http://localhost:8080 \
  --username admin \
  --password admin123
```

## 许可证

MIT License

## 贡献

欢迎提交 Issue 和 Pull Request！ 