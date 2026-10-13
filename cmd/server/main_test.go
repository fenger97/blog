package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"blog/internal/models"
)

type emptyPostReader struct{}

func (emptyPostReader) GetPost(string) (*models.Post, error) {
	return nil, nil
}

func (emptyPostReader) GetAllPosts() ([]*models.Post, error) {
	return nil, nil
}

func TestRouterRejectsWriteMethods(t *testing.T) {
	router := newRouter(emptyPostReader{})

	for _, path := range []string{"/", "/posts", "/post/hello-world", "/api/post?id=hello-world", "/static/css/style.css"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, nil))

		if recorder.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s: status = %d, want %d", path, recorder.Code, http.StatusMethodNotAllowed)
		}
		if allow := recorder.Header().Get("Allow"); allow != "GET, HEAD" {
			t.Errorf("POST %s: Allow = %q, want %q", path, allow, "GET, HEAD")
		}
	}
}

func TestRouterDoesNotRegisterRemovedRoutes(t *testing.T) {
	router := newRouter(emptyPostReader{})

	for _, path := range []string{"/login", "/logout", "/status", "/upload/image", "/sub", "/static/login.html"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

		if recorder.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want %d", path, recorder.Code, http.StatusNotFound)
		}
	}
}
