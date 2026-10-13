package content

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestInvalidIndexRefreshFallsBackToLastValidSnapshot(t *testing.T) {
	var invalid atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".md") {
			fmt.Fprint(w, "Body")
			return
		}
		if invalid.Load() {
			fmt.Fprint(w, `[{"slug":"../escape","title":"Bad"}]`)
			return
		}
		fmt.Fprint(w, `[{"slug":"ok","title":"Valid","excerpt":"Summary"}]`)
	}))
	defer server.Close()
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheTTL: 0})
	if _, err := store.GetAllPosts(); err != nil {
		t.Fatal(err)
	}
	invalid.Store(true)
	posts, err := store.GetAllPosts()
	if err != nil || len(posts) != 1 || posts[0].Title != "Valid" || !store.Status().Stale || !strings.Contains(store.Status().LastError, "invalid article slug") {
		t.Fatalf("fallback: %#v %#v %v", posts, store.Status(), err)
	}
}

func TestMissingArticleDoesNotPreventListAndHasExplicitFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/index.json":
			fmt.Fprint(w, `[{"slug":"good","title":"Good","created_at":"2026-10-10T00:00:00Z"},{"slug":"missing","title":"Missing","created_at":"2026-10-11T00:00:00Z"}]`)
		case "/good.md":
			fmt.Fprint(w, "# Good\n\nBody")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheTTL: time.Hour})
	posts, err := store.GetAllPosts()
	if err != nil || len(posts) != 2 || posts[0].ID != "missing" || posts[0].GetExcerpt() != unavailableExcerpt || posts[1].Content == "" {
		t.Fatalf("posts: %#v %v", posts, err)
	}
	if status := store.Status(); !status.Stale || status.ArticleCount != 2 || status.LastSuccess.IsZero() {
		t.Fatalf("status: %#v", status)
	}
}

func TestConcurrentRequestsShareIndexAndArticleRefresh(t *testing.T) {
	var indexCalls, bodyCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(15 * time.Millisecond)
		if r.URL.Path == "/index.json" {
			indexCalls.Add(1)
			fmt.Fprint(w, `[{"slug":"one","title":"One"}]`)
			return
		}
		bodyCalls.Add(1)
		fmt.Fprint(w, "Body")
	}))
	defer server.Close()
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheTTL: time.Hour})
	var readers sync.WaitGroup
	for n := 0; n < 30; n++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			if _, err := store.GetPost("one"); err != nil {
				t.Errorf("GetPost: %v", err)
			}
		}()
	}
	readers.Wait()
	if indexCalls.Load() != 1 || bodyCalls.Load() != 1 {
		t.Fatalf("requests index=%d body=%d", indexCalls.Load(), bodyCalls.Load())
	}
}

func TestDiskSnapshotSurvivesRestartAndSourceOutage(t *testing.T) {
	var unavailable atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if unavailable.Load() {
			http.Error(w, "offline", http.StatusBadGateway)
			return
		}
		if r.URL.Path == "/index.json" {
			fmt.Fprint(w, `[{"slug":"cached","title":"Cached"}]`)
			return
		}
		fmt.Fprint(w, "---\nslug: cached\ntags: [旅行]\n---\nBody saved on disk")
	}))
	defer server.Close()
	cfg := Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheDir: t.TempDir(), CacheTTL: 0}
	first := NewGitHubStore(cfg)
	if _, err := first.GetPost("cached"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cfg.CacheDir, "content-snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Body saved on disk") {
		t.Fatal("body was not persisted")
	}
	unavailable.Store(true)
	restarted := NewGitHubStore(cfg)
	post, err := restarted.GetPost("cached")
	if err != nil || post == nil || post.Content != "Body saved on disk" || len(post.Tags) != 1 {
		t.Fatalf("restored: %#v %v", post, err)
	}
	if !restarted.Status().Stale {
		t.Fatalf("expected stale status: %#v", restarted.Status())
	}
	other := cfg
	other.IndexURL = server.URL + "/different.json"
	if _, err := NewGitHubStore(other).GetAllPosts(); err == nil {
		t.Fatal("snapshot used for different source")
	}
}

func TestMalformedDiskSnapshotIsNotUsed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "content-snapshot.json"), []byte(`{"version":1,"entries":[{"slug":"../secret","title":"Bad"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewGitHubStore(Config{IndexURL: "http://127.0.0.1:1/index.json", CacheDir: dir})
	if _, err := store.GetAllPosts(); err == nil {
		t.Fatal("malformed snapshot accepted")
	}
}

func TestSnapshotDoesNotPersistSourceCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			fmt.Fprint(w, `[{"slug":"one","title":"One"}]`)
			return
		}
		fmt.Fprint(w, "Body")
	}))
	defer server.Close()
	cfg := Config{IndexURL: server.URL + "/index.json?access_token=private-query-value", BaseURL: server.URL, AuthToken: "private-auth-value", CacheDir: t.TempDir()}
	if _, err := NewGitHubStore(cfg).GetPost("one"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(cfg.CacheDir, "content-snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-query-value", "private-auth-value", server.URL} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("snapshot contains source credential/configuration %q", secret)
		}
	}
}

func TestSnapshotWriteFailureIsVisibleWithoutHidingContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			fmt.Fprint(w, `[{"slug":"one","title":"One"}]`)
			return
		}
		fmt.Fprint(w, "Body")
	}))
	defer server.Close()
	cachePath := filepath.Join(t.TempDir(), "file-not-directory")
	if err := os.WriteFile(cachePath, []byte("existing file"), 0600); err != nil {
		t.Fatal(err)
	}
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheDir: cachePath})
	post, err := store.GetPost("one")
	if err != nil || post == nil || post.Content != "Body" {
		t.Fatalf("valid content hidden: %#v %v", post, err)
	}
	if status := store.Status(); status.Stale || !strings.Contains(status.LastError, "snapshot:") {
		t.Fatalf("disk failure status: %#v", status)
	}
}

func TestMetadataListsDoNotFetchBodiesForReadingTime(t *testing.T) {
	var bodyCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			fmt.Fprint(w, `[{"slug":"one","title":"One","excerpt":"Summary","reading_minutes":7},{"slug":"legacy","title":"Legacy","excerpt":"Existing summary"}]`)
			return
		}
		bodyCalls.Add(1)
		http.Error(w, "Body unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheTTL: time.Hour})
	posts, err := store.GetAllPosts()
	if err != nil || len(posts) != 2 || bodyCalls.Load() != 0 {
		t.Fatalf("metadata listing fetched bodies=%d, posts=%#v, err=%v", bodyCalls.Load(), posts, err)
	}
	for _, post := range posts {
		if post.ID == "one" && post.ReadingMinutes() != 7 {
			t.Fatalf("reading estimate lost: %#v", post)
		}
	}
}

func TestFailedRefreshBackoffServesCacheWithoutRepeatedRequests(t *testing.T) {
	var unavailable atomic.Bool
	var indexCalls, bodyCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			indexCalls.Add(1)
		} else {
			bodyCalls.Add(1)
		}
		if unavailable.Load() {
			http.Error(w, "offline", http.StatusBadGateway)
			return
		}
		if r.URL.Path == "/index.json" {
			fmt.Fprint(w, `[{"slug":"one","title":"One","excerpt":"Summary"}]`)
			return
		}
		fmt.Fprint(w, "Cached body")
	}))
	defer server.Close()
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheTTL: 0})
	if _, err := store.GetPost("one"); err != nil {
		t.Fatal(err)
	}
	unavailable.Store(true)
	for n := 0; n < 3; n++ {
		post, err := store.GetPost("one")
		if err != nil || post == nil || post.Content != "Cached body" {
			t.Fatalf("stale post: %#v %v", post, err)
		}
		if _, err := store.GetAllPosts(); err != nil {
			t.Fatal(err)
		}
	}
	if indexCalls.Load() != 2 || bodyCalls.Load() != 2 {
		t.Fatalf("backoff failed: index calls=%d, body calls=%d", indexCalls.Load(), bodyCalls.Load())
	}
	unavailable.Store(false)
	store.mu.Lock()
	store.indexFailure.retryAfter = time.Now().Add(-time.Second)
	failure := store.articleFailures["one"]
	failure.retryAfter = time.Now().Add(-time.Second)
	store.articleFailures["one"] = failure
	store.mu.Unlock()
	if _, err := store.GetPost("one"); err != nil {
		t.Fatal(err)
	}
	if store.Status().Stale {
		t.Fatalf("recovery did not clear status: %#v", store.Status())
	}
}

func TestLegacyExcerptFetchingIsBoundedPerListRequest(t *testing.T) {
	var bodyCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.json" {
			fmt.Fprint(w, "[")
			for i := 0; i < 13; i++ {
				if i > 0 {
					fmt.Fprint(w, ",")
				}
				fmt.Fprintf(w, `{"slug":"post-%d","title":"Post %d"}`, i, i)
			}
			fmt.Fprint(w, "]")
			return
		}
		bodyCalls.Add(1)
		http.Error(w, "unavailable", http.StatusBadGateway)
	}))
	defer server.Close()
	store := NewGitHubStore(Config{IndexURL: server.URL + "/index.json", BaseURL: server.URL, CacheTTL: time.Hour})
	posts, err := store.GetAllPosts()
	if err != nil || len(posts) != 13 || bodyCalls.Load() != 4 {
		t.Fatalf("fetch budget exceeded: calls=%d len=%d err=%v", bodyCalls.Load(), len(posts), err)
	}
}
