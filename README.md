# 极简博客系统

这是一个使用 Go 语言开发的极简博客系统，采用类似 Jason Fried 博客的简洁设计风格，具有优雅的界面和完整的功能。

## 功能特性

- **极简设计风格**：采用 Inter 字体，黑白主色调，大量留白
- **智能文章摘要**：首页只显示文章摘要，去除 Markdown 标记和图片
- **文章详情页面**：独立的文章详情页，显示完整内容和图片
- **响应式设计**：完美适配桌面端和移动端
- **管理员登录**与权限控制
- **Markdown 支持**：完整的 Markdown 编辑和预览功能
- **MongoDB 数据持久化存储**
- **图片上传和展示功能**
- **智能日期显示**：相对时间（昨天、3天前等）

## 技术栈

### 后端
- Go 1.16+
- 标准库 `net/http` 用于 HTTP 服务
- **MongoDB** 用于数据持久化存储
- `github.com/gomarkdown/markdown` 用于 Markdown 解析
- `golang.org/x/crypto/bcrypt` 用于密码加密
- `go.mongodb.org/mongo-driver` 用于 MongoDB 驱动

### 前端
- 原生 HTML5
- CSS3（使用现代特性如变量、Flexbox、Grid等）
- 原生 JavaScript（ES6+）
- **Inter 字体**：现代无衬线字体
- 响应式设计

## 项目结构

```
.
├── cmd/                    # 主程序入口
│   └── server/            # 服务器入口
│       └── main.go        # 主程序
├── internal/              # 内部包
│   ├── models/           # 数据模型
│   │   ├── post.go      # 文章模型（包含摘要功能）
│   │   └── mongo_store.go # MongoDB 存储实现
│   ├── handlers/         # HTTP 处理器
│   │   ├── auth.go      # 认证处理器
│   │   ├── post.go      # 文章处理器
│   │   └── upload.go    # 图片上传处理器
│   └── database/         # 数据库相关
│       └── mongo.go     # MongoDB 连接管理
├── static/               # 静态文件
│   ├── css/             # 样式文件
│   │   └── style.css    # 主样式文件（极简风格）
│   ├── js/              # JavaScript 文件
│   │   ├── login.js     # 登录页脚本
│   │   ├── main.js      # 主页脚本
│   │   └── post.js      # 文章详情页脚本
│   ├── uploads/         # 上传文件
│   │   └── images/      # 图片存储目录
│   ├── index.html       # 主页面（文章列表）
│   ├── login.html       # 登录页面
│   └── post.html        # 文章详情页面
├── docker-compose.yml   # Docker Compose 配置
├── mongo-init.js        # MongoDB 初始化脚本
└── README.md            # 项目说明
```

## 快速开始

### 环境要求

- Go 1.16 或更高版本
- Docker 和 Docker Compose（用于运行 MongoDB）
- 现代浏览器（支持 ES6+）

### 方式一：本地开发

1. 克隆项目
```bash
git clone <repository-url>
cd blog
```

2. 启动 MongoDB
```bash
docker compose up -d mongodb
```

3. 运行服务器
```bash
go run cmd/server/main.go
```

4. 访问网站
- **首页**：打开浏览器访问 `http://localhost:1834`
- **登录页**：管理员请访问 `http://localhost:1834/login`
  - 默认用户名：`admin`
  - 默认密码：`123456` (可在 `internal/handlers/auth.go` 中修改)

### 方式二：容器化部署（推荐）

1. 克隆项目
```bash
git clone <repository-url>
cd blog
```

2. 配置环境变量
```bash
cp env.example .env
# 编辑 .env 文件，设置你的管理员密码
```

3. 一键部署
```bash
./deploy.sh
```

4. 访问网站
- **首页**：`http://localhost:1834`
- **登录页**：`http://localhost:1834/login`

### 方式三：手动 Docker 部署

1. 构建镜像
```bash
docker compose build
```

2. 启动服务
```bash
docker compose up -d
```

3. 查看状态
```bash
docker compose ps
```

4. 查看日志
```bash
docker compose logs -f
```

## API 接口

### 文章相关接口

1. **获取所有文章（摘要）**
```
GET /posts
```
返回文章摘要列表，包含标题、时间、内容摘要

2. **获取单篇文章（完整内容）**
```
GET /post/{id}
Accept: application/json
```
返回完整的文章内容，包含 HTML 渲染

3. **创建新文章**
```
POST /posts
Content-Type: application/json

{
    "title": "文章标题",
    "content": "文章内容（支持 Markdown）"
}
```

### 图片上传接口

1. **上传图片**
```
POST /upload/image
Content-Type: multipart/form-data

Form Data:
- image: 图片文件 (支持 JPG, PNG, GIF, WebP，最大 10MB)
```

**响应示例**：
```json
{
    "success": true,
    "url": "/static/uploads/images/1234567890.jpg"
}
```

### 认证相关接口

1. **登录**
```
POST /login
Content-Type: application/json

{
    "username": "admin",
    "password": "yourpassword"
}
```

2. **退出登录**
```
POST /logout
```

3. **检查登录状态**
```
GET /status
```

## 数据库

### MongoDB 配置
- 数据库名：`blog`
- 集合名：`posts`
- 连接地址：`mongodb://admin:password@localhost:27018`
- 端口映射：27018 -> 27017 (Docker)

### 数据模型
```json
{
  "_id": "ObjectId",
  "title": "文章标题",
  "content": "Markdown 内容",
  "created_at": "创建时间",
  "updated_at": "更新时间"
}
```

### 数据库特性
- **文档型存储**：天然适合博客文章
- **自动索引**：按创建时间和全文搜索优化
- **灵活结构**：可以轻松添加新字段
- **高性能**：适合读写操作

## 前端开发

### 设计理念

- **极简主义**：去除不必要的装饰元素
- **内容优先**：让文章内容成为主角
- **阅读友好**：合适的行高和字体大小
- **现代感**：使用 Inter 字体和现代设计语言

### 页面结构

- **首页**：文章列表，显示摘要信息
- **文章详情页**：完整的文章内容展示
- **登录页**：管理员登录界面
- **写文章页**：Markdown 编辑器

### 样式特点

- **Inter 字体**：现代无衬线字体
- **黑白主色调**：减少视觉干扰
- **大量留白**：突出内容
- **单栏布局**：阅读体验更佳
- **响应式设计**：完美适配各种设备
- **智能摘要**：自动去除 Markdown 标记

### 交互功能

- **智能日期显示**：相对时间（昨天、3天前等）
- **文章摘要**：自动生成内容摘要
- **Markdown 编辑器**：支持编辑/预览切换
- **图片上传**：支持拖拽或点击上传图片
- **文章导航**：点击标题进入详情，返回首页
- **空状态处理**：友好的空状态提示

## 开发计划

- [x] 添加用户认证
- [x] 支持 Markdown 格式
- [x] MongoDB 数据持久化存储
- [x] 文章创建和查看
- [x] 图片上传和展示功能
- [x] **极简风格重新设计**
- [x] **文章摘要功能**
- [x] **文章详情页面**
- [ ] 添加文章编辑功能
- [ ] 添加文章删除功能
- [ ] 实现文章分类
- [ ] 添加评论功能
- [ ] 实现文章搜索
- [ ] 添加文章标签功能
- [ ] 邮件订阅功能
- [ ] RSS 订阅功能

## 部署说明

### 容器化部署

#### 开发环境
```bash
# 启动所有服务
docker compose up -d

# 查看日志
docker compose logs -f

# 停止服务
docker compose down
```

#### 生产环境
```bash
# 启动生产环境（包含 Nginx）
docker compose --profile production up -d

# 配置 SSL 证书
# 1. 将证书文件放在 ssl/ 目录
# 2. 修改 nginx.conf 中的域名
# 3. 重启 Nginx 服务
```

#### 环境变量配置
```bash
# 复制环境变量模板
cp env.example .env

# 编辑 .env 文件
ADMIN_USERNAME=your_admin_username
ADMIN_PASSWORD=your_secure_password
SERVER_PORT=1834
MONGO_URI=mongodb://admin:password@mongodb:27017
```

### 生产环境建议
- 使用 HTTPS 协议
- 配置 MongoDB 认证
- 设置防火墙规则
- 定期备份数据库
- 使用强密码
- 配置日志轮转

## 贡献指南

欢迎提交 Issue 和 Pull Request 来帮助改进项目。

## 许可证

MIT License 