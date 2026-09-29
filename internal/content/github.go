package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"blog/internal/models"
)

var ErrNotFound = errors.New("content not found")

const (
	maxIndexBytes   = 1 << 20
	maxArticleBytes = 10 << 20
)

// Config describes the remote GitHub content source.
type Config struct {
	IndexURL  string
	BaseURL   string
	CacheTTL  time.Duration
	AuthToken string
}

type indexEntry struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Excerpt   string    `json:"excerpt"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type cachedIndex struct {
	entries   []indexEntry
	fetchedAt time.Time
}

type cachedArticle struct {
	content   string
	fetchedAt time.Time
}

// GitHubStore reads article metadata and Markdown from GitHub Raw URLs.
type GitHubStore struct {
	indexURL  string
	baseURL   string
	cacheTTL  time.Duration
	authToken string
	client    *http.Client

	mu       sync.RWMutex
	index    cachedIndex
	articles map[string]cachedArticle
}

func NewGitHubStore(cfg Config) *GitHubStore {
	if cfg.CacheTTL < 0 {
		cfg.CacheTTL = 0
	}
	return &GitHubStore{
		indexURL:  cfg.IndexURL,
		baseURL:   strings.TrimRight(cfg.BaseURL, "/"),
		cacheTTL:  cfg.CacheTTL,
		authToken: cfg.AuthToken,
		client:    &http.Client{Timeout: 15 * time.Second},
		articles:  make(map[string]cachedArticle),
	}
}

// GetAllPosts returns metadata from index.json. If an entry has no excerpt,
// its Markdown file is fetched and summarized on demand.
func (s *GitHubStore) GetAllPosts() ([]*models.Post, error) {
	entries, err := s.loadIndex()
	if err != nil {
		return nil, err
	}

	posts := make([]*models.Post, 0, len(entries))
	for _, entry := range entries {
		post := entry.post()
		if post.Excerpt == "" {
			post.Content, err = s.loadArticle(entry.Slug)
			if err != nil {
				return nil, err
			}
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// GetPost returns one article by slug.
func (s *GitHubStore) GetPost(slug string) (*models.Post, error) {
	if !models.IsValidSlug(slug) {
		return nil, nil
	}

	entries, err := s.loadIndex()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Slug != slug {
			continue
		}
		content, err := s.loadArticle(slug)
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		post := entry.post()
		post.Content = content
		return post, nil
	}

	return nil, nil
}

func (s *GitHubStore) loadIndex() ([]indexEntry, error) {
	now := time.Now()
	s.mu.RLock()
	if s.index.entries != nil && now.Sub(s.index.fetchedAt) < s.cacheTTL {
		entries := cloneEntries(s.index.entries)
		s.mu.RUnlock()
		return entries, nil
	}
	stale := cloneEntries(s.index.entries)
	s.mu.RUnlock()

	data, err := s.fetch(s.indexURL, maxIndexBytes)
	if err != nil {
		if stale != nil {
			return stale, nil
		}
		return nil, err
	}

	var entries []indexEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		if stale != nil {
			return stale, nil
		}
		return nil, fmt.Errorf("decode content index: %w", err)
	}
	for _, entry := range entries {
		if !models.IsValidSlug(entry.Slug) || strings.TrimSpace(entry.Title) == "" {
			return nil, fmt.Errorf("invalid content index entry %q", entry.Slug)
		}
	}
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if _, exists := seen[entry.Slug]; exists {
			return nil, fmt.Errorf("duplicate content index entry %q", entry.Slug)
		}
		seen[entry.Slug] = struct{}{}
	}

	s.mu.Lock()
	s.index = cachedIndex{entries: cloneEntries(entries), fetchedAt: now}
	s.mu.Unlock()
	return entries, nil
}

func (s *GitHubStore) loadArticle(slug string) (string, error) {
	now := time.Now()
	s.mu.RLock()
	cached, ok := s.articles[slug]
	s.mu.RUnlock()
	if ok && now.Sub(cached.fetchedAt) < s.cacheTTL {
		return cached.content, nil
	}

	articleURL := s.baseURL + "/" + url.PathEscape(slug+".md")
	data, err := s.fetch(articleURL, maxArticleBytes)
	if err != nil {
		if ok && !errors.Is(err, ErrNotFound) {
			return cached.content, nil
		}
		return "", err
	}

	content := string(data)
	s.mu.Lock()
	s.articles[slug] = cachedArticle{content: content, fetchedAt: now}
	s.mu.Unlock()
	return content, nil
}

func (s *GitHubStore) fetch(rawURL string, maxBytes int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "fenger-service/1.0")
	if s.authToken != "" {
		request.Header.Set("Authorization", "Bearer "+s.authToken)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("content source returned %s", response.Status)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("content source response exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func (entry indexEntry) post() *models.Post {
	return &models.Post{
		ID:        entry.Slug,
		Title:     entry.Title,
		Excerpt:   entry.Excerpt,
		CreatedAt: entry.CreatedAt,
		UpdatedAt: entry.UpdatedAt,
	}
}

func cloneEntries(entries []indexEntry) []indexEntry {
	if entries == nil {
		return nil
	}
	cloned := make([]indexEntry, len(entries))
	copy(cloned, entries)
	return cloned
}
