package content

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"blog/internal/models"
)

// Status describes the content currently served, including degraded responses.
type Status struct {
	Mode         string    `json:"mode"`
	LastSuccess  time.Time `json:"last_success"`
	Stale        bool      `json:"stale"`
	LastError    string    `json:"last_error,omitempty"`
	ArticleCount int       `json:"article_count"`
}

const unavailableExcerpt = "正文暂时无法加载，请稍后重试。"

// LocalStore reads the current files on every request so writing needs no restart.
type LocalStore struct {
	dir    string
	mu     sync.RWMutex
	status Status
}

func NewLocalStore(dir string) *LocalStore {
	return &LocalStore{dir: dir, status: Status{Mode: "local"}}
}

func (s *LocalStore) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *LocalStore) record(count int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.ArticleCount = count
	s.status.Stale = err != nil
	s.status.LastError = ""
	if err == nil {
		s.status.LastSuccess = time.Now()
	} else {
		s.status.LastError = err.Error()
	}
}

func (s *LocalStore) GetAllPosts() ([]*models.Post, error) {
	entries, err := LoadIndex(filepath.Join(s.dir, "index.json"))
	if err != nil {
		s.record(0, err)
		return nil, err
	}
	posts := make([]*models.Post, 0, len(entries))
	var firstError error
	for _, entry := range entries {
		meta, body, err := ReadDocument(filepath.Join(s.dir, entry.Slug+".md"))
		post := entry.post()
		if err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("article %s: %w", entry.Slug, err)
			}
			if post.Excerpt == "" {
				post.Excerpt = unavailableExcerpt
			}
		} else {
			post = previewMetadata(entry, meta).post()
			post.Content = body
		}
		posts = append(posts, post)
	}
	sortPosts(posts)
	s.record(len(posts), firstError)
	return posts, nil
}

func (s *LocalStore) GetPost(slug string) (*models.Post, error) {
	if !models.IsValidSlug(slug) {
		return nil, nil
	}
	entries, err := LoadIndex(filepath.Join(s.dir, "index.json"))
	if err != nil {
		s.record(0, err)
		return nil, err
	}
	for _, entry := range entries {
		if entry.Slug != slug {
			continue
		}
		meta, body, err := ReadDocument(filepath.Join(s.dir, slug+".md"))
		if err != nil {
			s.record(len(entries), err)
			if errors.Is(err, os.ErrNotExist) {
				return nil, nil
			}
			return nil, err
		}
		post := previewMetadata(entry, meta).post()
		post.Content = body
		s.record(len(entries), nil)
		return post, nil
	}
	s.record(len(entries), nil)
	return nil, nil
}

func sortPosts(posts []*models.Post) {
	sort.SliceStable(posts, func(i, j int) bool {
		if !posts[i].CreatedAt.Equal(posts[j].CreatedAt) {
			return posts[i].CreatedAt.After(posts[j].CreatedAt)
		}
		return posts[i].ID < posts[j].ID
	})
}
