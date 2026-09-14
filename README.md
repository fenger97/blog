# Fenger's Page

一个只读的个人文章展示服务。它从 MongoDB 读取已有文章，在首页展示摘要，并在详情页将 Markdown 渲染为 HTML。

服务不提供登录、发布、编辑、删除、图片上传或订阅功能。文章需要通过站点外部的数据库维护流程更新。

## 技术栈

- Go 1.23+
- 标准库 `net/http`
- MongoDB
- `github.com/gomarkdown/markdown`
- 原生 HTML、CSS 和 JavaScript

## 项目结构

```
.
├── cmd/server/          # 服务入口与只读路由
├── configs/             # 运行配置
├── internal/
│   ├── database/        # MongoDB 连接
│   ├── handlers/        # 文章读取接口
│   └── models/          # 文章模型与只读存储
├── static/              # 页面、样式、脚本与图片
├── env.example          # 环境变量示例
└── README.md
```

## 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_ADDR` | `127.0.0.1:8080` | 服务监听地址 |
| `SERVER_PORT` | `8080` | 未设置 `SERVER_ADDR` 时使用的端口 |
| `MONGO_URI` | `mongodb://127.0.0.1:27017` | MongoDB 连接地址 |

生产环境应使用仅有 `read` 权限的 MongoDB 账户，并通过系统环境文件传入 `MONGO_URI`。不要提交 `.env` 或任何数据库凭据。

## 本地运行

1. 启动 MongoDB。
2. 在仓库根目录设置环境变量，例如 `MONGO_URI`。
3. 执行 `go run ./cmd/server`。
4. 访问 `http://127.0.0.1:8080`。

服务会按相对路径读取 `static/`，因此必须从仓库根目录启动。

## 生产部署

生产环境使用以下结构：

```
Nginx (443) -> systemd service (127.0.0.1:8080) -> MongoDB container (127.0.0.1:27017)
```

- 服务直接运行 Linux 二进制文件，不放入容器。
- MongoDB 使用 Docker 持久卷，并只映射到本机回环地址。
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
After=network-online.target docker.service
Requires=docker.service

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

`/etc/fenger-service/service.env` 至少应包含：

```ini
SERVER_ADDR=127.0.0.1:8080
MONGO_URI=mongodb://service_reader:replace_me@127.0.0.1:27017/blog?authSource=blog
```

## 接口

- `GET /`：文章列表页面
- `GET /post/{id}`：文章详情页面
- `GET /posts`：文章摘要 JSON
- `GET /api/post?id={id}`：渲染后的文章详情 JSON
- `GET /static/*`：静态资源

以上路由只接受 `GET` 和 `HEAD` 请求。其他方法返回 `405 Method Not Allowed`；旧的 `/login`、`/logout`、`/status`、`/upload/image` 和 `/sub` 返回 `404 Not Found`。

## 验证

```bash
go test ./...
go vet ./...
```

## 许可证

MIT License
