# Fenger's Page

一个以 Markdown 和 GitHub 为内容源的个人博客。服务端直接输出文章列表和正文，页面提供目录、阅读时长、标签、搜索、年份归档、分页、代码复制和相邻文章导航。无需登录、数据库或 Docker，文章通过 Git 提交发布。

## 本地运行与写作

使用仓库固定的 Go 1.23.10，从仓库根目录运行：

```bash
go run ./cmd/content check
CONTENT_MODE=local CONTENT_DIR=content go run ./cmd/server
```

打开 `http://127.0.0.1:8080`。本地模式直接读取 `content/`，修改正文或已有文章的 frontmatter 后刷新页面即可预览；发布前或新增文章后需要生成索引。公开页面的静态资源位于 `static/`，必须从仓库根目录启动。`.env` 不会自动载入，环境变量需要由 shell 或 systemd 注入。

新建文章：

```bash
go run ./cmd/content new my-new-post -title "我的新文章" -date 2026-10-13 -tags "生活,随笔"
```

`new` 生成带 YAML frontmatter 的 `content/my-new-post.md`，不覆盖已有文件。`-title` 必填；`-date` 可使用 `YYYY-MM-DD` 或 RFC3339，省略时使用北京时间当天日期；`-tags` 为逗号分隔的标签。每个子命令都支持 `--dir` 指定其他内容目录。

写好正文后生成索引并检查：

```bash
go run ./cmd/content index
go run ./cmd/content check
```

`index` 根据目录中的 Markdown 重建 `index.json`，生成缺失的摘要和阅读时长，按发布日期倒序、同日期按 slug 排序。frontmatter 中提供的字段优先；未填写摘要的 frontmatter 文章会在每次生成索引时从正文更新摘要，阅读时长始终从正文重新计算。没有 frontmatter 的旧文章仍可通过已有索引提供标题、日期和人工摘要。索引仅在整批内容和链接检查通过后原子替换。

`check` 检查 JSON 字段、slug 唯一性、标题、发布日期和更新时间、旅行日期、正文是否存在、frontmatter 和阅读时长是否与索引一致，以及本地图片和站内文章链接。遗漏索引的 Markdown 也会报错。失败时返回退出码 `1`，适合在 CI 或提交前运行。仓库的 GitHub Actions 会在推送和拉取请求时运行内容检查、测试、静态检查及编译。外部链接只解析格式，不发出联网请求。

## 内容格式

文章正文存放在 `content/{slug}.md`。slug 只允许小写字母、数字、短横线和下划线；frontmatter 中的 slug 如果填写，必须与文件名一致。

```markdown
---
slug: my-new-post
title: 我的新文章
excerpt: 可选的人工摘要；不填写时由 index 命令生成。
created_at: 2026-10-13T00:00:00+08:00
updated_at: 2026-10-13T00:00:00+08:00
tags:
  - 生活
  - 随笔
---

## 第一节

从这里开始写正文。
```

日期可写 `YYYY-MM-DD` 或 RFC3339。只有日期没有时区时，frontmatter 解析为 UTC；需要北京时间请使用示例中的 `+08:00`。`updated_at` 不得早于 `created_at`；新文章在生成索引时，未填写的更新时间默认与发布日期相同。

旅行文章可以额外填写 `travel_start: 2025-10-23` 和 `travel_end: 2025-10-27`，两者需要同时提供，且结束日期不得早于开始日期。旅行日期不改变文章的发布日期。

`content/index.json` 是发布到 GitHub 的读取入口，例如：

```json
[
  {
    "slug": "my-new-post",
    "title": "我的新文章",
    "excerpt": "文章摘要",
    "created_at": "2026-10-13T00:00:00+08:00",
    "updated_at": "2026-10-13T00:00:00+08:00",
    "tags": ["生活", "随笔"]
  }
]
```

图片建议放在 `content/assets/`，正文使用相对路径：

```markdown
![海边落日](assets/jeju-sunset.jpg)
[另一篇文章](my-new-post.md)
```

相对图片会转换为 `/content/assets/...`：本地模式提供目录中的文件，GitHub 模式跳转到配置的公共内容源。本地图片仅支持 `assets/`、`/content/assets/` 或 `/static/` 路径，格式为 jpg、jpeg、png、gif、webp、avif；SVG 和内容目录根部的图片不会通过检查。也可使用外部 HTTPS 图片或现有的 `/static/images/...`。相对 Markdown 文章链接会转换为 `/post/{slug}`，也可以直接使用该站内路径。提交文章时一并提交索引和引用的图片。

发布流程：

```bash
go run ./cmd/content index
go run ./cmd/content check
git add content/
git commit -m "Publish article"
git push origin master
```

GitHub 模式的默认缓存时间为 5 分钟。纯内容更新在缓存刷新后生效，无需重新编译；页面功能或服务代码更新需要部署新二进制及静态资源。

## 配置

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `SERVER_ADDR` | `127.0.0.1:8080` | 服务监听地址 |
| `SERVER_PORT` | `8080` | 未设置 `SERVER_ADDR` 时使用的端口 |
| `CONTENT_MODE` | `github` | `github` 从公共内容源读取；`local` 读取本地目录 |
| `CONTENT_DIR` | `content` | 本地模式内容目录 |
| `CONTENT_INDEX_URL` | 本仓库 `content/index.json` 的 Raw 地址 | GitHub 模式索引地址 |
| `CONTENT_BASE_URL` | 本仓库 `content/` 的 Raw 地址 | GitHub 模式正文和图片目录 |
| `CONTENT_CACHE_TTL` | `5m` | GitHub 内存缓存有效时间；`0` 表示每次刷新 |
| `CONTENT_CACHE_DIR` | 空 | 可选的磁盘快照目录，需可写；仅用于 GitHub 模式 |
| `SITE_URL` | `https://fenger.xin` | 文章规范链接、分享卡片、RSS 和站点地图使用的公开站点地址 |
| `GITHUB_TOKEN` | 空 | 私有内容源的读取 token；公开仓库无需填写 |

可参考 `env.example`。不要把 token 或其他凭据提交到仓库。GitHub 模式的图片跳转适用于公共内容源；私有仓库的认证不会传给浏览器，图片请使用浏览器可访问的地址。

## 缓存与可用性

GitHub 暂时不可用时，服务继续提供最近一次成功读取的索引和正文。配置 `CONTENT_CACHE_DIR` 后，这些成功读取的内容还会保存为磁盘快照，进程重启后也能恢复；首次成功读取前，没有快照的文章仍依赖内容源。快照用于故障回退，不会延长正常缓存刷新时间。

例如生产环境可以设置 `CONTENT_CACHE_DIR=/var/lib/fenger-service/content-cache`，由服务用户持有并写入。私有内容的快照保存在该目录中，应限制目录权限；快照不保存认证 token。

## 页面、搜索与接口

首页提供标题、摘要和标签搜索、标签筛选、年份归档与分页。文章页直接输出正文、文章标题、日期和阅读时长，提供标题目录和代码复制，并包含文章专属的 SEO 和分享元数据。不存在的文章页面返回 HTTP 404；暂时无法读取内容时显示可重试的错误页。

| 路由 | 用途 |
| --- | --- |
| `GET /` | 文章列表，支持 `q`、`tag`、`year`、`page`、`limit` |
| `GET /post/{slug}` | 服务端渲染的文章页面 |
| `GET /posts` | 文章摘要 JSON 数组 |
| `GET /api/post?id={slug}` | 渲染后的文章详情 JSON |
| `GET /healthz` | 内容源与服务可用状态 |
| `GET /version` | 构建版本、提交号和构建时间 |
| `GET /sitemap.xml` | 文章站点地图 |
| `GET /robots.txt` | 搜索引擎抓取配置 |
| `GET /rss.xml` | RSS 订阅源 |
| `GET /content/assets/*` | 文章图片资源 |
| `GET /static/*` | 页面样式、脚本与公共图片 |

`/posts` 保持默认返回完整 JSON 数组的行为，同时支持过滤：

```bash
curl 'http://127.0.0.1:8080/posts?q=海风&tag=旅行&year=2025'
curl -i 'http://127.0.0.1:8080/posts?page=1&limit=12'
```

`q` 匹配标题、摘要和标签；`tag` 为精确标签；`year` 是四位年份。传入 `page` 或 `limit` 后启用分页，默认每页 `12` 条，`limit` 范围为 `1..100`，`page` 范围为 `1..1000000`。分页仍返回数组，并在 `X-Total-Count`、`X-Page`、`X-Page-Size` 响应头提供数量和页码。无效参数返回 `400`。

列表和详情的 JSON 包含 `reading_minutes`（预计阅读分钟数）。该值由写作工具写入索引，列表无需为此下载每篇正文；frontmatter 无需填写。修改正文后重新运行 `index`，保持摘要和阅读时长最新。

`/healthz` 返回 `ok`、`degraded`、`content_mode`、`article_count`、`last_success` 和 `version`。可读取内容或故障时可使用缓存，返回 `200`；内容完全不可用时返回 `503`。`degraded: true` 表示内容源或可选磁盘快照存在故障；服务可能仍在提供有效内容。`/version` 返回 `version`、`commit` 和 `build_time`，不展示配置地址或凭据。

以上路由支持 `GET` 和 `HEAD`；其他方法返回 `405`。旧的 `/login`、`/logout`、`/status`、`/upload/image` 和 `/sub` 返回 `404`。

## 构建与生产部署

推荐结构：

```
Nginx (443) -> systemd service (127.0.0.1:8080) -> GitHub Raw
```

从仓库根目录构建 Linux amd64 二进制，写入可供 `/version` 确认的构建信息：

```bash
BLOG_VERSION=1.1.0
BLOG_COMMIT=$(git rev-parse HEAD)
BLOG_BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags "-X main.version=$BLOG_VERSION -X main.commit=$BLOG_COMMIT -X main.buildTime=$BLOG_BUILD_TIME" \
  -o /tmp/fenger-service ./cmd/server
```

将二进制和当前 `static/` 一起部署。页面模板嵌入二进制；GitHub 内容模式不需要把 Markdown 复制到主机，本地模式还需部署完整 `content/`。替换前保留旧二进制和静态目录用于回滚，替换后重启服务，核对 `/version`、`/healthz`、首页和文章页。运行进程中直接覆盖二进制可能失败，先停止服务或上传为临时文件后原子替换。

systemd 示例：

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

`service.env` 示例：

```ini
SERVER_ADDR=127.0.0.1:8080
CONTENT_MODE=github
CONTENT_INDEX_URL=https://raw.githubusercontent.com/fenger97/blog/master/content/index.json
CONTENT_BASE_URL=https://raw.githubusercontent.com/fenger97/blog/master/content
CONTENT_CACHE_TTL=5m
CONTENT_CACHE_DIR=/var/lib/fenger-service/content-cache
SITE_URL=https://fenger.xin
```

部署前创建快照目录并让 `fenger` 用户可写。Nginx 将应用请求转发至 `127.0.0.1:8080`，保留上述 API、RSS、地图和图片路径。当前生产约定是 `fenger.xin` 为规范域名、`www.fenger.xin` 重定向；TLS 使用 acme.sh 的 TLS-ALPN 校验，因此无需开放 `80`。本地测试站点应设置自己的 `SITE_URL`，避免生成生产域名的分享链接。

## 验证与技术栈

```bash
go run ./cmd/content check
go test ./...
go vet ./...
```

项目使用 Go 1.23.10、标准库 `net/http` 和嵌入模板、`gomarkdown/markdown`、YAML frontmatter，以及原生 HTML/CSS/JavaScript。服务端模板位于 `internal/web/templates/`，内容读取与快照位于 `internal/content/`，写作工具位于 `cmd/content/`。

## 许可证

MIT License
