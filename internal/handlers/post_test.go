package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"blog/internal/models"
)

type fakePostReader struct {
	post     *models.Post
	posts    []*models.Post
	getErr   error
	listErr  error
	getCalls int
}

func (f *fakePostReader) GetPost(string) (*models.Post, error) {
	f.getCalls++
	return f.post, f.getErr
}

func (f *fakePostReader) GetAllPosts() ([]*models.Post, error) {
	return f.posts, f.listErr
}

func TestGetPostReturnsRenderedDetail(t *testing.T) {
	post := &models.Post{
		ID:        "hello-world",
		Title:     "Hello",
		Content:   "# Heading\n\n**content**",
		CreatedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	}
	handler := NewPostHandler(&fakePostReader{post: post})
	recorder := httptest.NewRecorder()

	handler.GetPost(recorder, post.ID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["id"] != post.ID || response["title"] != post.Title {
		t.Fatalf("unexpected response: %#v", response)
	}
	if _, ok := response["content"]; ok {
		t.Fatalf("raw Markdown content must not be returned: %#v", response)
	}
	if htmlContent, ok := response["html_content"].(string); !ok || !strings.Contains(htmlContent, ">Heading</h1>") {
		t.Fatalf("html_content = %#v", response["html_content"])
	}
}

func TestGetPostValidatesAndHandlesErrors(t *testing.T) {
	validSlug := "hello-world"
	tests := []struct {
		name      string
		id        string
		store     *fakePostReader
		wantCode  int
		wantCalls int
	}{
		{name: "invalid slug", id: "../secret", store: &fakePostReader{}, wantCode: http.StatusBadRequest},
		{name: "missing post", id: validSlug, store: &fakePostReader{}, wantCode: http.StatusNotFound, wantCalls: 1},
		{name: "store error", id: validSlug, store: &fakePostReader{getErr: errors.New("content source unavailable")}, wantCode: http.StatusInternalServerError, wantCalls: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			NewPostHandler(test.store).GetPost(recorder, test.id)

			if recorder.Code != test.wantCode {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantCode)
			}
			if test.store.getCalls != test.wantCalls {
				t.Fatalf("store calls = %d, want %d", test.store.getCalls, test.wantCalls)
			}
		})
	}
}

func TestGetAllPostsReturnsSummaries(t *testing.T) {
	post := &models.Post{
		ID:        "hello-world",
		Title:     "Summary",
		Excerpt:   "A short article",
		CreatedAt: time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC),
	}
	handler := NewPostHandler(&fakePostReader{posts: []*models.Post{post}})
	recorder := httptest.NewRecorder()

	handler.GetAllPosts(recorder, httptest.NewRequest(http.MethodGet, "/posts", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response []map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 || response[0]["title"] != post.Title || response[0]["excerpt"] != post.Excerpt {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestGetAllPostsReturnsServerError(t *testing.T) {
	handler := NewPostHandler(&fakePostReader{listErr: errors.New("content source unavailable")})
	recorder := httptest.NewRecorder()

	handler.GetAllPosts(recorder, httptest.NewRequest(http.MethodGet, "/posts", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestDetailResolvesContentLinks(t *testing.T) {
	h := NewPostHandler(&fakePostReader{post: &models.Post{ID: "article", Content: "![photo](assets/photo.png)\n\n[related](related.md#section-2)"}})
	w := httptest.NewRecorder()
	h.GetPost(w, "article")
	var detail map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	body := detail["html_content"].(string)
	for _, expected := range []string{"/content/assets/photo.png", "/post/related#section-2"} {
		if !strings.Contains(body, expected) {
			t.Errorf("missing %s in %s", expected, body)
		}
	}
}
