package web

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"blog/internal/content"
	"blog/internal/models"
)

type fakeStore struct {
	posts []*models.Post
	err   error
	state content.Status
}

func (f fakeStore) GetAllPosts() ([]*models.Post, error) { return f.posts, f.err }
func (f fakeStore) GetPost(id string) (*models.Post, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, p := range f.posts {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, nil
}
func (f fakeStore) Status() content.Status { return f.state }
func fixture() fakeStore {
	return fakeStore{posts: []*models.Post{
		{ID: "old", Title: "Old", Excerpt: "old story", Content: "## Old section", CreatedAt: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), Tags: []string{"生活"}},
		{ID: "jeju", Title: "济州岛 <海风>", Excerpt: "落日与橘子", Content: "## 海边\n\n旅行正文。\n\n![海](assets/sea.jpg)\n\n[旧文](old.md#section-1)\n\n```go\nfmt.Println(\"Hello\")\n```\n\n<script>alert(1)</script><img src=x onerror=alert(2)>", CreatedAt: time.Date(2025, 10, 27, 0, 0, 0, 0, time.UTC), Tags: []string{"旅行", "济州岛"}, TravelStart: "2025-10-23", TravelEnd: "2025-10-27"},
	}}
}
func request(t *testing.T, router http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestPagesRenderContentWithoutJavaScript(t *testing.T) {
	router := New(fixture(), Config{SiteURL: "https://example.com", Version: "test"})
	index := request(t, router, "GET", "/")
	if index.Code != 200 {
		t.Fatalf("index %d: %s", index.Code, index.Body.String())
	}
	if !strings.Contains(index.Body.String(), "济州岛 &lt;海风&gt;") || !strings.Contains(index.Body.String(), "2025-10-27") {
		t.Fatalf("missing SSR article: %s", index.Body.String())
	}
	if strings.Index(index.Body.String(), "/post/jeju") > strings.Index(index.Body.String(), "/post/old") {
		t.Fatal("newest article must appear first")
	}
	post := request(t, router, "GET", "/post/jeju")
	if post.Code != 200 {
		t.Fatalf("post %d: %s", post.Code, post.Body.String())
	}
	body := post.Body.String()
	for _, want := range []string{"旅行正文。", "href=\"#section-1\"", "id=\"section-1\"", "/content/assets/sea.jpg", "/post/old#section-1", "loading=\"lazy\"", "https://example.com/post/jeju", "article:published_time", "class=\"chroma\"", "2025-10-23 — 2025-10-27"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, unsafe := range []string{"<script>alert(1)", "onerror=", "fonts.googleapis.com"} {
		if strings.Contains(body, unsafe) {
			t.Errorf("unsafe or external dependency %q", unsafe)
		}
	}
}

func TestFilterAndPaginationPages(t *testing.T) {
	source := fixture()
	for i := 0; i < 14; i++ {
		source.posts = append(source.posts, &models.Post{ID: strings.Repeat("a", i+1), Title: "More", Excerpt: "Extra", CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)})
	}
	router := New(source, Config{})
	cases := []struct {
		path, contains string
		status         int
	}{
		{"/?q=橘子", "共 1 篇", 200}, {"/?tag=旅行&year=2025", "共 1 篇", 200}, {"/?q=nothing", "没有找到匹配的文章", 200}, {"/?page=2", "第 2 / 2 页", 200}, {"/?page=3", "这一页没有文章", 404}, {"/?page=0", "page 参数无效", 400}, {"/?year=wrong", "年份格式不正确", 400},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			w := request(t, router, "GET", tc.path)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
				t.Fatalf("%d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestErrorStatusAndHEAD(t *testing.T) {
	router := New(fixture(), Config{})
	for _, path := range []string{"/post/missing", "/post/invalid/extra", "/login", "/status"} {
		w := request(t, router, "GET", path)
		if w.Code != 404 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
	broken := New(fakeStore{err: errors.New("secret-sensitive detail")}, Config{})
	for _, path := range []string{"/", "/post/jeju"} {
		w := request(t, broken, "GET", path)
		if w.Code != 502 || strings.Contains(w.Body.String(), "secret-sensitive") {
			t.Fatalf("bad public error: %d %s", w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/", "/post/jeju", "/posts", "/version", "/rss.xml"} {
		w := request(t, router, "HEAD", path)
		if w.Code != 200 || w.Body.Len() != 0 {
			t.Errorf("HEAD %s = %d, %d bytes", path, w.Code, w.Body.Len())
		}
	}
	for _, path := range []string{"/", "/post/jeju", "/healthz", "/version", "/rss.xml"} {
		w := request(t, router, "POST", path)
		if w.Code != 405 || w.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("POST %s = %d", path, w.Code)
		}
	}
}

func TestFeedsAndOperationalEndpoints(t *testing.T) {
	source := fixture()
	source.state = content.Status{Stale: true, LastError: "upstream temporarily unavailable", LastSuccess: time.Now()}
	router := New(source, Config{SiteURL: "https://example.com", Version: "v1", Commit: "abc123", BuildTime: "now", ContentMode: "github"})
	for _, path := range []string{"/rss.xml", "/sitemap.xml"} {
		w := request(t, router, "GET", path)
		if w.Code != 200 {
			t.Fatalf("%s = %d", path, w.Code)
		}
		decoder := xml.NewDecoder(strings.NewReader(w.Body.String()))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err != io.EOF {
					t.Fatal(err)
				}
				break
			}
		}
		if !strings.Contains(w.Body.String(), "https://example.com/post/jeju") {
			t.Errorf("missing absolute URL: %s", path)
		}
	}
	health := request(t, router, "GET", "/healthz")
	var state map[string]any
	if err := json.Unmarshal(health.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if health.Code != 200 || state["ok"] != true || state["degraded"] != true || state["article_count"] != float64(2) {
		t.Fatalf("health: %d %v", health.Code, state)
	}
	if strings.Contains(health.Body.String(), "upstream temporarily") {
		t.Fatal("raw upstream failure should not be exposed")
	}
	version := request(t, router, "GET", "/version")
	if !strings.Contains(version.Body.String(), "abc123") {
		t.Fatal(version.Body.String())
	}
	broken := request(t, New(fakeStore{err: errors.New("offline")}, Config{}), "GET", "/healthz")
	if broken.Code != 503 {
		t.Fatalf("unhealthy status %d", broken.Code)
	}
}

func TestLocalAssetsCannotEscapeContentDirectory(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "sea.jpg"), []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "private.jpg"), []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "private.jpg"), filepath.Join(dir, "assets", "escape.jpg")); err != nil {
		t.Fatal(err)
	}
	router := New(fixture(), Config{ContentMode: "local", ContentDir: dir})
	if w := request(t, router, "GET", "/content/assets/sea.jpg"); w.Code != 200 || w.Body.String() != "image" {
		t.Fatalf("asset %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/content/assets/escape.jpg", "/content/assets/file.html", "/content/index.json"} {
		if w := request(t, router, "GET", path); w.Code != 404 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}
