# Fenger's Page

一个只读的个人文章展示服务。文章以 Markdown 文件存放在 GitHub 仓库中，服务从 GitHub Raw 读取文章元数据和正文，在详情页渲染为 HTML。

服务不提供登录、发布、编辑、删除、图片上传或订阅功能。更新文章只需要提交 GitHub 内容变更。

## 内容格式

文章元数据集中放在 `content/index.json`：

```json
[
  {
    "slug": "hello-world",
    "title": "Hello World",
    "excerpt": "文章摘要",
    "created_at": "2026-09-29T00:00:00Z",
    "updated_at": "2026-09-29T00:00:00Z"
  }
]
```

正文放在同目录的 `content/{slug}.md`，例如 `content/hello-world.md`。`slug` 只允许小写字母、数字、短横线和下划线。

文章中的图片直接使用完整 URL，例如：

```markdown
![示例图片](https://example.com/image.jpg)
```

服务会缓存 GitHub 内容 5 分钟；GitHub 暂时不可用时，会继续使用最近一次成功读取的缓存。

## 技术栈

- Go 1.23+
- 标准库 `net/http`
- `github.com/gomarkdown/markdown`
- 原生 HTML、CSS 和 JavaScript

## 项目结构

```
.
├── cmd/server/          # 服务入口与只读路由
├── configs/             # 运行配置
├── content/             # GitHub 上的文章元数据与 Markdown 正文
├── internal/
│   ├── content/         # GitHub Raw 内容读取与缓存
│   ├── handlers/        # 文章读取接口
│   └── models/          # 文章模型与 Markdown 渲染
├── static/              # 页面、样式、脚本与图片
├── env.example          # 环境变量示例
└── README.md
```

## 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_ADDR` | `127.0.0.1:8080` | 服务监听地址 |
| `SERVER_PORT` | `8080` | 未设置 `SERVER_ADDR` 时使用的端口 |
| `CONTENT_INDEX_URL` | 当前仓库 `content/index.json` 的 Raw 地址 | 文章索引地址 |
| `CONTENT_BASE_URL` | 当前仓库 `content/` 的 Raw 地址 | Markdown 正文目录 |
| `CONTENT_CACHE_TTL` | `5m` | 内容缓存时间 |
| `GITHUB_TOKEN` | 空 | 私有仓库读取 token，公开仓库无需填写 |

## 本地运行

1. 确保 `content/index.json` 和文章 Markdown 已提交到 GitHub。
2. 按需设置环境变量，或直接使用默认内容地址。
3. 从仓库根目录执行 `go run ./cmd/server`。
4. 访问 `http://127.0.0.1:8080`。

服务按相对路径读取 `static/`，因此必须从仓库根目录启动。

## 生产部署

生产环境使用以下结构：

```
Nginx (443) -> systemd service (127.0.0.1:8080) -> GitHub Raw
```

- 服务直接运行 Linux 二进制文件，不需要 MongoDB 或 Docker。
- Nginx 只监听 `443`；`fenger.xin` 为规范域名，`www.fenger.xin` 重定向到主域名。
- TLS 证书使用 acme.sh 的 TLS-ALPN 校验签发和续期，因此无需开放 `80`。

编译 Linux 二进制：

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o fenger-service ./cmd/server
```

建议使用类似以下的 systemd 单元：

```ini
[Unit]
Description=Fenger Service
After=network-online.target
Wants=network-online.target

[Service]
User=fenger
Group=fenger
WorkingDirectory=/opt/fenger-service
EnvironmentFile=/etc/fenger-service/service.env
ExecStart=/opt/fenger-service/fenger-service
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

`/etc/fenger-service/service.env` 示例：

```ini
SERVER_ADDR=127.0.0.1:8080
CONTENT_INDEX_URL=https://raw.githubusercontent.com/fenger97/blog/master/content/index.json
CONTENT_BASE_URL=https://raw.githubusercontent.com/fenger97/blog/master/content
CONTENT_CACHE_TTL=5m
```

## 接口

- `GET /`：文章列表页面
- `GET /post/{slug}`：文章详情页面
- `GET /posts`：文章摘要 JSON
- `GET /api/post?id={slug}`：渲染后的文章详情 JSON
- `GET /static/*`：静态资源

以上路由只接受 `GET` 和 `HEAD` 请求。其他方法返回 `405 Method Not Allowed`；旧的 `/login`、`/logout`、`/status`、`/upload/image` 和 `/sub` 返回 `404 Not Found`。

## 验证

```bash
go test ./...
go vet ./...
```

## 许可证

MIT License
