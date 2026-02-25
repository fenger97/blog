# Fenger Journal

这是一个使用 Go 语言开发的极简博客系统，采用类似 Jason Fried 博客的简洁设计风格。除了博客功能外，该项目还包含一个隐藏的 VPN 节点订阅服务。

## 功能特性

- **极简设计风格**：采用 Inter 字体，黑白主色调，大量留白。
- **智能文章摘要**：首页显示文章摘要，自动去除 Markdown 标记。
- **文章详情页面**：支持完整内容展示和图片。
- **管理功能**：包含管理员登录、文章创建、**编辑**和**删除**功能。
- **Markdown 支持**：完整的解析和预览功能。
- **MongoDB 存储**：数据持久化，性能优异。
- **图片上传**：支持本地存储和展示。
- **隐藏功能**：内置 **VLESS/Clash 订阅转换接口**。

## 技术栈

### 后端
- **Go 1.23.0+**
- 标准库 `net/http` 用于 HTTP 服务
- **MongoDB** 用于数据持久化
- `github.com/gomarkdown/markdown` 用于 Markdown 解析
- `go.mongodb.org/mongo-driver` 用于 MongoDB 驱动

### 前端
- 原生 HTML5 / CSS3 / JavaScript (ES6+)
- **Inter 字体**：现代无衬线字体
- 响应式设计，适配桌面端和移动端

## 项目结构

```
.
├── cmd/                    # 主程序入口
│   └── server/            # 服务器入口
│       └── main.go        # 主逻辑与路由
├── internal/              # 内部包
│   ├── database/         # MongoDB 连接管理
│   ├── handlers/         # 处理器 (auth, post, upload, vpn)
│   ├── models/           # 数据模型 (Post, MongoStore)
├── vpn/                    # VPN 节点配置文件 [隐藏]
│   ├── vpn.txt           # VLESS 原始链接
│   └── clash.txt         # Clash 配置文件
├── static/               # 静态文件
│   ├── css/             # 极简样式
│   ├── js/              # 前端交互逻辑
│   └── uploads/         # 用户上传的图片
├── configs/              # 应用配置 (config.go)
├── docker-compose.yml   # Docker 部署配置
└── README.md            # 项目说明
```

## 快速开始

### 环境要求
- Go 1.23.0+
- MongoDB 4.4+ (或通过 Docker 运行)

### 本地运行
1. **启动 MongoDB**：使用 `docker compose up -d mongodb`。
2. **连接配置**：在 `configs/config.go` 或环境变量中检查配置。
3. **运行**：`go run cmd/server/main.go`。
4. **访问**：`http://localhost:1834`。

### 管理员登录
- **默认用户名**：`fenger`
- **默认密码**：`zjf1834.`
- **注意**：README 之前标注的 admin/123456 是不准确的。

## API 接口手册

### 博客功能
- `GET /posts`：获取所有文章摘要。
- `GET /post/{id}`：获取文章详情。
- `POST /posts`：创建新文章（需登录）。
- `PUT /post/{id}`：更新文章内容（需登录）。
- `DELETE /post/{id}`：删除文章（需登录）。
- `POST /upload/image`：上传图片。

### 隐藏订阅功能 (/sub)
该接口用于分发 VPN 节点信息，模拟常见的订阅格式。
- **接口地址**：`/sub?token=xxxx&format={base64|clash}`
- **参数说明**：
    - `token`：必须提供（目前代码中仅检查是否存在，未做值校验）。
    - `format`：默认为 `base64`。
        - `base64`：返回 `vpn/vpn.txt` 加密后的内容。
        - `clash`：返回 `vpn/clash.txt` 的原始 YAML 内容。

> [!CAUTION]
> **安全警告**：项目的 `/sub` 接口目前存在安全风险。虽然 URL 包含 token 参数，但服务器端并未进行有效性验证。任何知道该接口的人都可以轻易获取您的节点信息。

## 开发计划 (路线图)

- [x] 用户认证与权限控制
- [x] Markdown 格式渲染
- [x] MongoDB 数据持久化
- [x] **文章编辑与更新功能** (已完成)
- [x] **文章删除功能** (已完成)
- [x] 图片上传与展示
- [x] 跨平台 Docker 部署
- [ ] 文章分类与搜索
- [ ] 评论系统
- [ ] RSS/邮件订阅
- [ ] **增强 `/sub` 接口的 Token 校验**

## 许可证
MIT License