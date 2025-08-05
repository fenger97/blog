package models

import (
	"regexp"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Post 表示一篇博客文章
type Post struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title     string             `json:"title" bson:"title"`
	Content   string             `json:"content" bson:"content"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// NewPost 创建一个新的文章
func NewPost(title, content string) *Post {
	now := time.Now()
	return &Post{
		Title:     title,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetHTMLContent 将 Markdown 内容转换为 HTML
func (p *Post) GetHTMLContent() string {
	// 创建 Markdown 解析器
	extensions := parser.CommonExtensions
	parser := parser.NewWithExtensions(extensions)

	// 解析 Markdown
	doc := parser.Parse([]byte(p.Content))

	// 创建 HTML 渲染器
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// 渲染为 HTML
	html := markdown.Render(doc, renderer)
	return string(html)
}

// GetExcerpt 获取内容摘要，去除 Markdown 标记并截取前 200 个字符
func (p *Post) GetExcerpt() string {
	// 去除 Markdown 标记
	excerpt := p.Content
	
	// 去除标题标记
	excerpt = regexp.MustCompile(`^#{1,6}\s+`).ReplaceAllString(excerpt, "")
	
	// 去除粗体和斜体标记
	excerpt = regexp.MustCompile(`\*\*(.*?)\*\*`).ReplaceAllString(excerpt, "$1")
	excerpt = regexp.MustCompile(`\*(.*?)\*`).ReplaceAllString(excerpt, "$1")
	
	// 去除链接标记，保留链接文本
	excerpt = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(excerpt, "$1")
	
	// 去除图片标记
	excerpt = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`).ReplaceAllString(excerpt, "")
	
	// 去除代码标记
	excerpt = regexp.MustCompile("`([^`]+)`").ReplaceAllString(excerpt, "$1")
	
	// 去除列表标记
	excerpt = regexp.MustCompile(`^[\s]*[-*+]\s+`).ReplaceAllString(excerpt, "")
	excerpt = regexp.MustCompile(`^[\s]*\d+\.\s+`).ReplaceAllString(excerpt, "")
	
	// 去除引用标记
	excerpt = regexp.MustCompile(`^>\s+`).ReplaceAllString(excerpt, "")
	
	// 去除多余的空行和空格
	excerpt = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(excerpt, "\n")
	excerpt = strings.TrimSpace(excerpt)
	
	// 截取前 200 个字符
	if len(excerpt) > 200 {
		// 尝试在句号处截断
		lastPeriod := strings.LastIndex(excerpt[:200], "。")
		if lastPeriod > 150 { // 如果句号位置合理，在句号处截断
			excerpt = excerpt[:lastPeriod+1]
		} else {
			excerpt = excerpt[:200] + "..."
		}
	}
	
	return excerpt
}

// ToMap 转换为 map 用于 MongoDB 插入
func (p *Post) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"title":      p.Title,
		"content":    p.Content,
		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}
