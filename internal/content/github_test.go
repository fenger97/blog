package content

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubStoreReadsIndexAndMarkdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/content/index.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[{"slug":"hello-world","title":"Hello","excerpt":"A summary","created_at":"2026-09-10T00:00:00Z"}]`)
		case "/content/hello-world.md":
			fmt.Fprint(w, "# Hello\n\nMarkdown body")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := NewGitHubStore(Config{
		IndexURL: server.URL + "/content/index.json",
		BaseURL:  server.URL + "/content",
		CacheTTL: time.Hour,
	})

	posts, err := store.GetAllPosts()
	if err != nil {
		t.Fatalf("GetAllPosts() error = %v", err)
	}
	if len(posts) != 1 || posts[0].ID != "hello-world" || posts[0].GetExcerpt() != "A summary" {
		t.Fatalf("unexpected posts: %#v", posts)
	}

	post, err := store.GetPost("hello-world")
	if err != nil {
		t.Fatalf("GetPost() error = %v", err)
	}
	if post == nil || post.Content != "# Hello\n\nMarkdown body" {
		t.Fatalf("unexpected post: %#v", post)
	}
}

func TestGitHubStoreUsesStaleCacheWhenSourceFails(t *testing.T) {
	available := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !available {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, `[{"slug":"cached","title":"Cached","excerpt":"old"}]`)
	}))
	defer server.Close()

	store := NewGitHubStore(Config{
		IndexURL: server.URL,
		BaseURL:  server.URL,
		CacheTTL: 0,
	})

	if _, err := store.GetAllPosts(); err != nil {
		t.Fatalf("initial GetAllPosts() error = %v", err)
	}
	available = false
	posts, err := store.GetAllPosts()
	if err != nil {
		t.Fatalf("stale GetAllPosts() error = %v", err)
	}
	if len(posts) != 1 || posts[0].ID != "cached" {
		t.Fatalf("unexpected stale posts: %#v", posts)
	}
}

func TestGitHubStoreKeepsEmptyIndexCache(t *testing.T) {
	available := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !available {
			http.Error(w, "unavailable", http.StatusBadGateway)
			return
		}
		fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	store := NewGitHubStore(Config{IndexURL: server.URL, BaseURL: server.URL, CacheTTL: 0})
	if posts, err := store.GetAllPosts(); err != nil || len(posts) != 0 {
		t.Fatalf("initial empty index = %#v, error = %v", posts, err)
	}
	available = false
	if posts, err := store.GetAllPosts(); err != nil || len(posts) != 0 {
		t.Fatalf("stale empty index = %#v, error = %v", posts, err)
	}
}

func TestGitHubStoreRejectsInvalidIndexSlug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"slug":"../secret","title":"Invalid"}]`)
	}))
	defer server.Close()

	store := NewGitHubStore(Config{IndexURL: server.URL, BaseURL: server.URL, CacheTTL: time.Hour})
	if _, err := store.GetAllPosts(); err == nil {
		t.Fatal("GetAllPosts() error = nil, want invalid slug error")
	}
}
