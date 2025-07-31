package models

import (
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

// ToMap 转换为 map 用于 MongoDB 插入
func (p *Post) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"title":      p.Title,
		"content":    p.Content,
		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}
