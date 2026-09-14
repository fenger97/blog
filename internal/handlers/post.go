package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"blog/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PostReader provides the data required by the public article endpoints.
type PostReader interface {
	GetPost(id string) (*models.Post, error)
	GetAllPosts() ([]*models.Post, error)
}

// PostHandler handles read-only article requests.
type PostHandler struct {
	store PostReader
}

// NewPostHandler creates a public article handler.
func NewPostHandler(store PostReader) *PostHandler {
	return &PostHandler{store: store}
}

// GetPost returns a rendered article.
func (h *PostHandler) GetPost(w http.ResponseWriter, postID string) {
	if !primitive.IsValidObjectID(postID) {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	post, err := h.store.GetPost(postID)
	if err != nil {
		log.Printf("Error getting post: %v", err)
		http.Error(w, "获取文章失败", http.StatusInternalServerError)
		return
	}
	if post == nil {
		http.Error(w, "文章不存在", http.StatusNotFound)
		return
	}

	type postDetail struct {
		ID          string    `json:"id"`
		Title       string    `json:"title"`
		HTMLContent string    `json:"html_content"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	detail := postDetail{
		ID:          post.ID.Hex(),
		Title:       post.Title,
		HTMLContent: post.GetHTMLContent(),
		CreatedAt:   post.CreatedAt,
		UpdatedAt:   post.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// GetAllPosts returns article summaries in reverse creation order.
func (h *PostHandler) GetAllPosts(w http.ResponseWriter, _ *http.Request) {
	posts, err := h.store.GetAllPosts()
	if err != nil {
		log.Printf("Error finding posts: %v", err)
		http.Error(w, "获取文章失败", http.StatusInternalServerError)
		return
	}

	type postSummary struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Excerpt   string    `json:"excerpt"`
		CreatedAt time.Time `json:"created_at"`
	}

	summaries := make([]postSummary, len(posts))
	for i, post := range posts {
		summaries[i] = postSummary{
			ID:        post.ID.Hex(),
			Title:     post.Title,
			Excerpt:   post.GetExcerpt(),
			CreatedAt: post.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}
