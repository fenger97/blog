package content

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"blog/internal/models"
)

var ErrNotFound = errors.New("content not found")

const (
	maxIndexBytes     = 1 << 20
	maxArticleBytes   = 10 << 20
	maxSnapshotBytes  = 128 << 20
	refreshRetryDelay = 30 * time.Second
)

// Config describes the remote GitHub content source. CacheDir is opt-in.
type Config struct {
	IndexURL  string
	BaseURL   string
	CacheTTL  time.Duration
	AuthToken string
	CacheDir  string
}

type cachedIndex struct {
	entries   []Metadata
	fetchedAt time.Time
}

type cachedArticle struct {
	Content   string    `json:"content"`
	Metadata  Metadata  `json:"metadata"`
	FetchedAt time.Time `json:"fetched_at"`
}

type indexFlight struct {
	done    chan struct{}
	entries []Metadata
	err     error
}

type articleFlight struct {
	done    chan struct{}
	article cachedArticle
	err     error
}

type refreshFailure struct {
	err        error
	retryAfter time.Time
}

// GitHubStore serves verified content and keeps its last usable response through
// source failures. In-flight requests are shared by concurrent readers.
type GitHubStore struct {
	indexURL  string
	baseURL   string
	cacheTTL  time.Duration
	authToken string
	cacheDir  string
	client    *http.Client

	mu              sync.RWMutex
	index           cachedIndex
	articles        map[string]cachedArticle
	indexPending    *indexFlight
	articlePending  map[string]*articleFlight
	indexError      string
	articleErrors   map[string]string
	indexFailure    refreshFailure
	articleFailures map[string]refreshFailure
	lastSuccess     time.Time
	snapshotError   string
	diskMu          sync.Mutex
}

func NewGitHubStore(cfg Config) *GitHubStore {
	if cfg.CacheTTL < 0 {
		cfg.CacheTTL = 0
	}
	s := &GitHubStore{
		indexURL: cfg.IndexURL,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
		cacheTTL: cfg.CacheTTL, authToken: cfg.AuthToken, cacheDir: cfg.CacheDir,
		client:          &http.Client{Timeout: 15 * time.Second},
		articles:        make(map[string]cachedArticle),
		articlePending:  make(map[string]*articleFlight),
		articleErrors:   make(map[string]string),
		articleFailures: make(map[string]refreshFailure),
	}
	s.restoreSnapshot()
	return s
}

func (s *GitHubStore) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := Status{Mode: "github", LastSuccess: s.lastSuccess, ArticleCount: len(s.index.entries)}
	if s.indexError != "" {
		status.LastError = "index: " + s.indexError
	}
	if len(s.articleErrors) > 0 {
		keys := make([]string, 0, len(s.articleErrors))
		for slug := range s.articleErrors {
			keys = append(keys, slug)
		}
		sort.Strings(keys)
		if status.LastError != "" {
			status.LastError += "; "
		}
		status.LastError += "article " + keys[0] + ": " + s.articleErrors[keys[0]]
	}
	status.Stale = status.LastError != ""
	if s.snapshotError != "" {
		if status.LastError != "" {
			status.LastError += "; "
		}
		status.LastError += "snapshot: " + s.snapshotError
	}
	return status
}

// GetAllPosts uses index metadata and already cached bodies. Legacy entries
// without excerpts may fetch at most four bodies per list request, so slow
// content origins cannot multiply the request timeout by the article count.
func (s *GitHubStore) GetAllPosts() ([]*models.Post, error) {
	entries, err := s.loadIndex()
	if err != nil {
		return nil, err
	}
	posts := make([]*models.Post, len(entries))
	tasks := make([]int, 0, 4)
	s.mu.RLock()
	originUnavailable := s.indexFailure.err != nil && time.Now().Before(s.indexFailure.retryAfter)
	s.mu.RUnlock()
	for i, entry := range entries {
		post := entry.post()
		s.mu.RLock()
		article, cached := s.articles[entry.Slug]
		s.mu.RUnlock()
		if cached {
			post = mergeMetadata(entry, article.Metadata).post()
			post.Content = article.Content
		} else if post.Excerpt == "" {
			post.Excerpt = "点击阅读文章全文。"
			if len(tasks) < 4 && !originUnavailable {
				tasks = append(tasks, i)
			}
		}
		posts[i] = post
	}
	jobs := make(chan int)
	var workers sync.WaitGroup
	for n := 0; n < len(tasks); n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for i := range jobs {
				entry := entries[i]
				post := entry.post()
				article, err := s.loadArticle(entry.Slug)
				if err == nil {
					post = mergeMetadata(entry, article.Metadata).post()
					post.Content = article.Content
				} else if post.Excerpt == "" {
					post.Excerpt = unavailableExcerpt
				}
				posts[i] = post
			}
		}()
	}
	for _, i := range tasks {
		jobs <- i
	}
	close(jobs)
	workers.Wait()
	sortPosts(posts)
	return posts, nil
}

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
		article, err := s.loadArticle(slug)
		if errors.Is(err, ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		post := mergeMetadata(entry, article.Metadata).post()
		post.Content = article.Content
		return post, nil
	}
	return nil, nil
}

func (s *GitHubStore) loadIndex() ([]Metadata, error) {
	s.mu.Lock()
	if s.index.entries != nil && time.Since(s.index.fetchedAt) < s.cacheTTL {
		entries := cloneEntries(s.index.entries)
		s.mu.Unlock()
		return entries, nil
	}
	if s.indexFailure.err != nil && time.Now().Before(s.indexFailure.retryAfter) {
		entries := cloneEntries(s.index.entries)
		err := s.indexFailure.err
		if entries != nil {
			err = nil
		}
		s.mu.Unlock()
		return entries, err
	}
	if pending := s.indexPending; pending != nil {
		s.mu.Unlock()
		<-pending.done
		return cloneEntries(pending.entries), pending.err
	}
	pending := &indexFlight{done: make(chan struct{})}
	s.indexPending = pending
	s.mu.Unlock()

	data, err := s.fetch(s.indexURL, maxIndexBytes)
	var entries []Metadata
	if err == nil {
		entries, err = decodeIndex(data)
	}
	s.mu.Lock()
	if err != nil {
		s.indexError = err.Error()
		s.indexFailure = refreshFailure{err: err, retryAfter: time.Now().Add(refreshRetryDelay)}
		if s.index.entries != nil {
			entries, err = cloneEntries(s.index.entries), nil
		}
	} else {
		now := time.Now()
		s.index = cachedIndex{entries: cloneEntries(entries), fetchedAt: now}
		s.lastSuccess = now
		s.indexError = ""
		s.indexFailure = refreshFailure{}
		valid := make(map[string]bool, len(entries))
		for _, entry := range entries {
			valid[entry.Slug] = true
		}
		for slug := range s.articles {
			if !valid[slug] {
				delete(s.articles, slug)
			}
		}
		for slug := range s.articleErrors {
			if !valid[slug] {
				delete(s.articleErrors, slug)
				delete(s.articleFailures, slug)
			}
		}
	}
	pending.entries, pending.err = entries, err
	s.indexPending = nil
	close(pending.done)
	s.mu.Unlock()
	if err == nil {
		s.persistSnapshot()
	}
	return cloneEntries(entries), err
}

func (s *GitHubStore) loadArticle(slug string) (cachedArticle, error) {
	s.mu.Lock()
	cached, ok := s.articles[slug]
	if ok && time.Since(cached.FetchedAt) < s.cacheTTL {
		s.mu.Unlock()
		return cached, nil
	}
	if failure := s.articleFailures[slug]; failure.err != nil && time.Now().Before(failure.retryAfter) {
		err := failure.err
		if ok {
			err = nil
		}
		s.mu.Unlock()
		return cached, err
	}
	if pending := s.articlePending[slug]; pending != nil {
		s.mu.Unlock()
		<-pending.done
		return pending.article, pending.err
	}
	pending := &articleFlight{done: make(chan struct{})}
	s.articlePending[slug] = pending
	s.mu.Unlock()

	data, err := s.fetch(s.baseURL+"/"+url.PathEscape(slug+".md"), maxArticleBytes)
	var meta Metadata
	var body string
	if err == nil {
		meta, body, err = parseDocument(data)
	}
	if err == nil && meta.Slug != "" && meta.Slug != slug {
		err = fmt.Errorf("frontmatter slug does not match filename")
	}
	article := cachedArticle{}
	s.mu.Lock()
	if err != nil {
		s.articleErrors[slug] = err.Error()
		s.articleFailures[slug] = refreshFailure{err: err, retryAfter: time.Now().Add(refreshRetryDelay)}
		if ok {
			article, err = cached, nil
		}
	} else {
		now := time.Now()
		article = cachedArticle{Content: body, Metadata: meta, FetchedAt: now}
		s.articles[slug] = article
		s.lastSuccess = now
		delete(s.articleErrors, slug)
		delete(s.articleFailures, slug)
	}
	pending.article, pending.err = article, err
	delete(s.articlePending, slug)
	close(pending.done)
	s.mu.Unlock()
	if err == nil {
		s.persistSnapshot()
	}
	return article, err
}

func (s *GitHubStore) fetch(rawURL string, maxBytes int64) ([]byte, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid content source URL")
	}
	request.Header.Set("User-Agent", "fenger-service/1.0")
	if s.authToken != "" {
		request.Header.Set("Authorization", "Bearer "+s.authToken)
	}
	response, err := s.client.Do(request)
	if err != nil {
		// URL/query credentials must not appear in public health responses.
		var urlError *url.Error
		if errors.As(err, &urlError) {
			err = urlError.Err
		}
		return nil, fmt.Errorf("content source request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("content source returned HTTP %d", response.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("content response exceeds %d bytes", maxBytes)
	}
	return data, nil
}

// A snapshot records only content that passed index/frontmatter/UTF-8 checks.
type diskSnapshot struct {
	Version     int                      `json:"version"`
	SourceHash  string                   `json:"source_hash"`
	Entries     []Metadata               `json:"entries"`
	FetchedAt   time.Time                `json:"fetched_at"`
	Articles    map[string]cachedArticle `json:"articles"`
	LastSuccess time.Time                `json:"last_success"`
}

func (s *GitHubStore) persistSnapshot() {
	if s.cacheDir == "" {
		return
	}
	s.diskMu.Lock()
	defer s.diskMu.Unlock()
	s.mu.RLock()
	if s.index.entries == nil {
		s.mu.RUnlock()
		return
	}
	snapshot := diskSnapshot{Version: 2, SourceHash: s.sourceHash(), Entries: cloneEntries(s.index.entries), FetchedAt: s.index.fetchedAt, LastSuccess: s.lastSuccess, Articles: make(map[string]cachedArticle, len(s.articles))}
	for _, entry := range s.index.entries {
		if article, ok := s.articles[entry.Slug]; ok {
			snapshot.Articles[entry.Slug] = article
		}
	}
	s.mu.RUnlock()
	data, err := json.Marshal(snapshot)
	if err == nil && len(data) > maxSnapshotBytes {
		err = fmt.Errorf("snapshot exceeds %d bytes", maxSnapshotBytes)
	}
	if err == nil {
		err = os.MkdirAll(s.cacheDir, 0700)
	}
	if err == nil {
		err = atomicWrite(filepath.Join(s.cacheDir, "content-snapshot.json"), data, 0600)
	}
	// A disk failure does not hide freshly loaded content or invalidate the
	// in-memory cache, but remains visible in diagnostics.
	if err != nil {
		s.reportSnapshotError("cannot persist last valid content")
	} else {
		s.reportSnapshotError("")
	}
}

func (s *GitHubStore) sourceHash() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s.indexURL+"\x00"+s.baseURL)))
}

func (s *GitHubStore) reportSnapshotError(message string) {
	s.mu.Lock()
	changed := message != s.snapshotError
	s.snapshotError = message
	s.mu.Unlock()
	if changed && message != "" {
		log.Printf("content snapshot: %s", message)
	}
}

func (s *GitHubStore) restoreSnapshot() {
	if s.cacheDir == "" {
		return
	}
	data, err := readBoundedFile(filepath.Join(s.cacheDir, "content-snapshot.json"), maxSnapshotBytes)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.reportSnapshotError("cannot read last valid content")
		}
		return
	}
	var snapshot diskSnapshot
	if json.Unmarshal(data, &snapshot) != nil || snapshot.Version != 2 || snapshot.Entries == nil || snapshot.FetchedAt.IsZero() || validateEntries(snapshot.Entries) != nil {
		s.reportSnapshotError("invalid last valid content snapshot")
		return
	}
	if snapshot.SourceHash != s.sourceHash() {
		return
	}
	valid := make(map[string]bool, len(snapshot.Entries))
	for _, entry := range snapshot.Entries {
		valid[entry.Slug] = true
	}
	for slug, article := range snapshot.Articles {
		if !valid[slug] || article.FetchedAt.IsZero() || len(article.Content) > maxArticleBytes {
			s.reportSnapshotError("invalid last valid content snapshot")
			return
		}
		if err := validateArticleText([]byte(article.Content)); err != nil {
			s.reportSnapshotError("invalid last valid content snapshot")
			return
		}
		if validateOptionalMetadata(article.Metadata) != nil || (article.Metadata.Slug != "" && article.Metadata.Slug != slug) {
			s.reportSnapshotError("invalid last valid content snapshot")
			return
		}
	}
	sortEntries(snapshot.Entries)
	s.index = cachedIndex{entries: snapshot.Entries, fetchedAt: snapshot.FetchedAt}
	s.articles = snapshot.Articles
	if s.articles == nil {
		s.articles = make(map[string]cachedArticle)
	}
	s.lastSuccess = snapshot.LastSuccess
}
