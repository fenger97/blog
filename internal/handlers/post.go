package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"blog/internal/models"
)

// PostHandler 处理文章相关的请求
type PostHandler struct {
	store *models.MongoStore
}

// NewPostHandler 创建一个新的文章处理器
func NewPostHandler(store *models.MongoStore) *PostHandler {
	return &PostHandler{store: store}
}

// CreatePost 处理创建文章的请求
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var post models.Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.store.CreatePost(&post)
	if err != nil {
		http.Error(w, "Failed to create post", http.StatusInternalServerError)
		return
	}

	post.ID = id

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

// GetPost 获取单篇文章
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request, postID string) {
	post, err := h.store.GetPost(postID)
	if err != nil {
		log.Printf("Error getting post: %v", err)
		http.Error(w, "文章不存在", http.StatusNotFound)
		return
	}

	// 返回包含完整 HTML 内容的文章
	type PostWithHTML struct {
		*models.Post
		HTMLContent string `json:"html_content"`
	}

	postWithHTML := PostWithHTML{
		Post:        post,
		HTMLContent: post.GetHTMLContent(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(postWithHTML)
}

// GetAllPosts 获取所有文章
func (h *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	posts, err := h.store.GetAllPosts()
	if err != nil {
		log.Printf("Error finding posts: %v", err)
		http.Error(w, "获取文章失败", http.StatusInternalServerError)
		return
	}

	// 转换为包含摘要的响应格式
	type PostSummary struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Excerpt   string    `json:"excerpt"`
		CreatedAt time.Time `json:"created_at"`
	}

	postSummaries := make([]PostSummary, len(posts))
	for i, post := range posts {
		postSummaries[i] = PostSummary{
			ID:        post.ID.Hex(),
			Title:     post.Title,
			Excerpt:   post.GetExcerpt(),
			CreatedAt: post.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(postSummaries)
}
