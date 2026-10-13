package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"blog/internal/models"
)

func TestListFiltersSortsAndPaginates(t *testing.T) {
	posts := []*models.Post{
		{ID: "old", Title: "Older", Excerpt: "story", CreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Tags: []string{"生活"}},
		{ID: "new", Title: "Jeju", Excerpt: "海边的落日", CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), Tags: []string{"旅行"}},
	}
	h := NewPostHandler(&fakePostReader{posts: posts})
	for _, tc := range []struct {
		path  string
		count int
		first string
		total string
	}{{"/posts", 2, "new", "2"}, {"/posts?q=落日&tag=旅行&year=2025", 1, "new", "1"}, {"/posts?page=2&limit=1", 1, "old", "2"}, {"/posts?q=absent", 0, "", "0"}} {
		w := httptest.NewRecorder()
		h.GetAllPosts(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		var got []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || len(got) != tc.count || w.Header().Get("X-Total-Count") != tc.total {
			t.Fatalf("%s: %d %v headers %v", tc.path, w.Code, got, w.Header())
		}
		if tc.first != "" && got[0]["id"] != tc.first {
			t.Fatalf("%s first = %v", tc.path, got)
		}
	}
	for _, path := range []string{"/posts?page=0", "/posts?limit=101", "/posts?year=abc"} {
		w := httptest.NewRecorder()
		h.GetAllPosts(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 400 {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
}
