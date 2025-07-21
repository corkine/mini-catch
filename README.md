# Mini-Catch 剧集追踪系统

一个用于追踪剧集更新的 Web 应用程序，支持 Slack 通知和外部爬虫集成。

## 功能特性

- 🎬 剧集管理：添加、编辑、删除剧集
- 📊 状态追踪：追踪观看状态和更新状态
- 🔔 Slack 通知：新集数更新时自动发送通知
- 🕷️ 爬虫集成：提供标准化的爬虫接口
- 🔒 安全认证：用户名密码登录保护
- 📱 响应式界面：使用 Tailwind CSS 和 Alpine.js 构建的现代化界面
- 💾 SQLite 数据库：轻量级数据存储

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
    "database_path": "./data/mini-catch.db",
    "slack_webhook_url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "auth": {
        "username": "admin",
        "password": "admin123"
    }
}
```

**配置说明：**
- `port`: 服务器监听端口
- `database_path`: SQLite 数据库文件路径
- `slack_webhook_url`: Slack Webhook URL（可选，留空则不发送通知）
- `auth`: 认证配置
  - `username`: 登录用户名
  - `password`: 登录密码

### 4. 创建数据目录

```bash
mkdir -p data
```

### 5. 运行应用

#### 直接运行
```bash
# 运行服务器
go run ./cmd/server

# 运行爬虫模拟器
go run ./cmd/crawler
```

应用将在 `http://localhost:8080` 启动。

## 使用说明

### Web 界面

访问 `http://localhost:8080` 即可使用 Web 界面：

1. **添加剧集**: 点击"添加新剧集"按钮，输入剧集名称和 URL
2. **管理剧集**: 可以编辑、删除、标记观看状态、切换追踪状态
3. **查看统计**: 界面顶部显示总剧集数、已观看数、追踪中数量等统计信息

### API 接口

#### 剧集管理 API

- `GET /api/series` - 获取所有剧集
- `POST /api/series` - 创建新剧集
- `PUT /api/series/{id}` - 更新剧集
- `DELETE /api/series/{id}` - 删除剧集
- `PUT /api/series/{id}/watch` - 标记为已观看
- `PUT /api/series/{id}/unwatch` - 标记为未观看
- `PUT /api/series/{id}/toggle` - 切换追踪状态

#### 爬虫接口

- `GET /fetch` - 获取爬虫任务（需要认证）
- `POST /fetch` - 爬虫回调上报结果（需要认证）

### 爬虫集成

系统提供了标准化的爬虫接口，支持外部爬虫程序集成：

#### 获取任务

```bash
# 首先登录获取 token
curl -X POST "http://localhost:8080/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'

# 使用 token 获取任务
curl "http://localhost:8080/fetch" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

返回格式：
```json
{
    "success": true,
    "data": {
        "tasks": ["url1", "url2", "url3"],
        "callback_url": "https://your-domain.com/fetch"
    }
}
```

#### 上报结果

```bash
curl -X POST "http://localhost:8080/fetch" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -d '{
    "tasks": ["url1", "url2"],
    "results": [
      {
        "name": "剧集名称",
        "update": "S03E02",
        "url": "https://example.com/series",
        "series": ["S01E01", "S01E02", "S02E01", "S03E01", "S03E02"]
      }
    ],
    "status": 0,
    "message": "success"
  }'
```

## 数据结构

### 剧集信息

```json
{
    "id": 1,
    "name": "剧集名称",
    "url": "https://example.com/series",
    "history": ["S01E01", "S01E02", "S02E01"],
    "current": "S02E01",
    "is_watched": false,
    "is_tracking": true,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
}
```

## Slack 通知

当检测到新集数时，系统会自动发送 Slack 通知。通知包含：

- 剧集名称
- 新增集数列表
- 剧集链接
- 时间戳

## 开发

### 项目结构

```
mini-catch/
├── main.go          # 主程序入口
├── database.go      # 数据库模型和操作
├── handlers.go      # HTTP 处理器
├── slack.go         # Slack 通知功能
├── static/          # 静态文件
│   └── index.html   # 前端界面
├── data/            # 数据目录
├── config.json      # 配置文件
├── go.mod           # Go 模块文件
└── README.md        # 项目说明
```

### 构建和运行

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