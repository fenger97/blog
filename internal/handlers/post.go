package handlers

import (
	"encoding/json"
	"log"
	"net/http"

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

// GetPost 处理获取单篇文章的请求
func (h *PostHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	post, err := h.store.GetPost(id)
	if err != nil {
		http.Error(w, "Failed to get post", http.StatusInternalServerError)
		return
	}

	if post == nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}

// GetAllPosts 处理获取所有文章的请求
func (h *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	posts, err := h.store.GetAllPosts()
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		http.Error(w, "Failed to get posts", http.StatusInternalServerError)
		return
	}

	// 为每篇文章添加 HTML 内容
	type PostWithHTML struct {
		*models.Post
		HTMLContent string `json:"html_content"`
	}

	postsWithHTML := make([]PostWithHTML, len(posts))
	for i, post := range posts {
		postsWithHTML[i] = PostWithHTML{
			Post:        post,
			HTMLContent: post.GetHTMLContent(),
		}
	}

	log.Printf("Returning %d posts to client", len(postsWithHTML))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(postsWithHTML)
}
