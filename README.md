# 极简博客系统

这是一个使用 Go 语言开发的极简博客系统，具有简洁的界面和基本的功能。

## 功能特性

- 文章的创建和查看
- 响应式设计，支持移动端访问
- 简洁美观的用户界面
- 实时文章列表更新
- **管理员登录**与权限控制
- **Markdown 支持**
- **MongoDB 数据持久化存储**

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
- CSS3（使用现代特性如变量、Flexbox等）
- 原生 JavaScript（ES6+）
- 响应式设计

## 项目结构

```
.
├── cmd/                    # 主程序入口
│   └── server/            # 服务器入口
│       └── main.go        # 主程序
├── internal/              # 内部包
│   ├── models/           # 数据模型
│   │   ├── post.go      # 文章模型
│   │   └── mongo_store.go # MongoDB 存储实现
│   ├── handlers/         # HTTP 处理器
│   │   ├── auth.go      # 认证处理器
│   │   └── post.go      # 文章处理器
│   └── database/         # 数据库相关
│       └── mongo.go     # MongoDB 连接管理
├── static/               # 静态文件
│   ├── css/             # 样式文件
│   │   └── style.css    # 主样式文件
│   ├── js/              # JavaScript 文件
│   │   ├── login.js     # 登录页脚本
│   │   └── main.js      # 主页脚本
│   ├── index.html       # 主页面
│   └── login.html       # 登录页面
├── docker-compose.yml   # Docker Compose 配置
├── mongo-init.js        # MongoDB 初始化脚本
└── README.md            # 项目说明
```

## 快速开始

### 环境要求

- Go 1.16 或更高版本
- Docker 和 Docker Compose（用于运行 MongoDB）
- 现代浏览器（支持 ES6+）

### 安装和运行

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

## API 接口

### 文章相关接口

1. 获取所有文章
```
GET /posts
```

2. 获取单篇文章
```
GET /posts?id=<post_id>
```

3. 创建新文章
```
POST /posts
Content-Type: application/json

{
    "title": "文章标题",
    "content": "文章内容（支持 Markdown）"
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

### 页面结构

- 顶部导航栏：显示博客标题和新建文章按钮
- 文章表单：用于创建新文章（支持 Markdown 编辑和预览）
- 文章列表：展示所有文章

### 样式特点

- 使用 CSS 变量管理主题颜色
- 响应式布局，适配不同屏幕尺寸
- 卡片式设计展示文章
- 简洁现代的视觉风格
- **Markdown 编辑器**：支持编辑/预览切换

### 交互功能

- 点击"写新文章"显示表单
- **Markdown 编辑器**：支持实时预览
- 表单提交后自动刷新文章列表
- 实时显示文章发布时间
- 错误处理和用户提示

## 开发计划

- [x] 添加用户认证
- [x] 支持 Markdown 格式
- [x] MongoDB 数据持久化存储
- [x] 文章创建和查看
- [ ] 添加文章编辑功能
- [ ] 添加文章删除功能
- [ ] 实现文章分类
- [ ] 添加评论功能
- [ ] 实现文章搜索
- [ ] 添加文章标签功能

## 部署说明

### 生产环境建议
- 使用 HTTPS 协议
- 配置 MongoDB 认证
- 设置防火墙规则
- 定期备份数据库

### Docker 部署
```bash
# 启动 MongoDB
docker compose up -d mongodb

# 构建并运行应用
docker build -t blog .
docker run -p 1834:1834 blog
```

## 贡献指南

欢迎提交 Issue 和 Pull Request 来帮助改进项目。

## 许可证

MIT License 