package models

import "time"

// Post 表示一篇博客文章
type Post struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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