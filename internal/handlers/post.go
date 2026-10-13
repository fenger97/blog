package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"blog/internal/models"
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
	if !models.IsValidSlug(postID) {
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
		ID             string    `json:"id"`
		Title          string    `json:"title"`
		HTMLContent    string    `json:"html_content"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
		Tags           []string  `json:"tags"`
		ReadingMinutes int       `json:"reading_minutes"`
		TravelStart    string    `json:"travel_start,omitempty"`
		TravelEnd      string    `json:"travel_end,omitempty"`
	}

	detail := postDetail{
		ID:             post.ID,
		Title:          post.Title,
		HTMLContent:    models.NormalizeArticleHTML(post.GetHTMLContent()),
		CreatedAt:      post.CreatedAt,
		UpdatedAt:      post.UpdatedAt,
		Tags:           post.Tags,
		ReadingMinutes: post.ReadingMinutes(),
		TravelStart:    post.TravelStart,
		TravelEnd:      post.TravelEnd,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

// GetAllPosts returns article summaries in reverse creation order.
func (h *PostHandler) GetAllPosts(w http.ResponseWriter, r *http.Request) {
	query, err := ParseListQuery(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	posts, err := h.store.GetAllPosts()
	if err != nil {
		log.Printf("Error finding posts: %v", err)
		http.Error(w, "获取文章失败", http.StatusInternalServerError)
		return
	}

	posts = FilterPosts(posts, query)
	w.Header().Set("X-Total-Count", strconv.Itoa(len(posts)))
	if query.Paginate {
		w.Header().Set("X-Page", strconv.Itoa(query.Page))
		w.Header().Set("X-Page-Size", strconv.Itoa(query.Limit))
		posts = PagePosts(posts, query.Page, query.Limit)
	}
	type postSummary struct {
		ID             string    `json:"id"`
		Title          string    `json:"title"`
		Excerpt        string    `json:"excerpt"`
		CreatedAt      time.Time `json:"created_at"`
		UpdatedAt      time.Time `json:"updated_at"`
		Tags           []string  `json:"tags"`
		ReadingMinutes int       `json:"reading_minutes"`
	}

	summaries := make([]postSummary, len(posts))
	for i, post := range posts {
		summaries[i] = postSummary{
			ID:             post.ID,
			Title:          post.Title,
			Excerpt:        post.GetExcerpt(),
			CreatedAt:      post.CreatedAt,
			UpdatedAt:      post.UpdatedAt,
			Tags:           post.Tags,
			ReadingMinutes: post.ReadingMinutes(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}
