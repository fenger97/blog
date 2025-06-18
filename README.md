# 极简博客系统

这是一个使用 Go 语言开发的极简博客系统，具有简洁的界面和基本的功能。

## 功能特性

- 文章的创建和查看
- 响应式设计，支持移动端访问
- 简洁美观的用户界面
- 实时文章列表更新

## 技术栈

### 后端
- Go 1.16+
- 标准库 `net/http` 用于 HTTP 服务
- 内存存储（可扩展为数据库存储）

### 前端
- 原生 HTML5
- CSS3（使用现代特性如变量、Flexbox等）
- 原生 JavaScript（ES6+）
- 响应式设计

## 项目结构

```
.
├── cmd/                    # 主程序入口
│   └── server/             # 服务器入口
│       └── main.go         # 主程序
├── internal/               # 内部包
│   ├── models/             # 数据模型
│   │   ├── post.go         # 文章模型
│   │   └── store.go        # 存储实现
│   └── handlers/           # HTTP 处理器
│       └── post.go         # 文章处理器
├── static/                 # 静态文件
│   ├── css/                # 样式文件
│   │   └── style.css       # 主样式文件
│   ├── js/                 # JavaScript 文件
│   │   └── main.js         # 主脚本文件
│   └── index.html          # 主页面
└── README.md               # 项目说明
```

## 快速开始

### 环境要求

- Go 1.16 或更高版本
- 现代浏览器（支持 ES6+）

### 安装和运行

1. 克隆项目
```bash
git clone <repository-url>
cd blog
```

2. 运行服务器
```bash
go run cmd/server/main.go
```

3. 访问网站
打开浏览器访问 http://localhost:8080

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
    "content": "文章内容"
}
```

## 前端开发

### 页面结构

- 顶部导航栏：显示博客标题和新建文章按钮
- 文章表单：用于创建新文章
- 文章列表：展示所有文章

### 样式特点

- 使用 CSS 变量管理主题颜色
- 响应式布局，适配不同屏幕尺寸
- 卡片式设计展示文章
- 简洁现代的视觉风格

### 交互功能

- 点击"写新文章"显示表单
- 表单提交后自动刷新文章列表
- 实时显示文章发布时间
- 错误处理和用户提示

## 开发计划

- [ ] 添加文章编辑功能
- [ ] 添加文章删除功能
- [ ] 实现文章分类
- [ ] 添加用户认证
- [ ] 支持 Markdown 格式
- [ ] 添加评论功能

## 贡献指南

欢迎提交 Issue 和 Pull Request 来帮助改进项目。

## 许可证

MIT License 