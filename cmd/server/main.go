package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"blog/configs"
	"blog/internal/content"
	"blog/internal/handlers"
)

func main() {
	cfg := configs.LoadConfig()

	postStore := content.NewGitHubStore(content.Config{
		IndexURL:  cfg.ContentIndexURL,
		BaseURL:   cfg.ContentBaseURL,
		CacheTTL:  cfg.ContentCacheTTL,
		AuthToken: cfg.GitHubToken,
	})
	postHandler := handlers.NewPostHandler(postStore)
	router := newRouter(postHandler)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		os.Exit(0)
	}()

	log.Printf("Server starting on %s...", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, router); err != nil {
		log.Fatal(err)
	}
}

func newRouter(postHandler *handlers.PostHandler) http.Handler {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", requireReadMethod(http.StripPrefix("/static/", fs)))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if !isReadMethod(r.Method) {
			methodNotAllowed(w)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})

	mux.Handle("/posts", requireReadMethod(http.HandlerFunc(postHandler.GetAllPosts)))

	mux.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		if !isReadMethod(r.Method) {
			methodNotAllowed(w)
			return
		}
		id := strings.Trim(r.URL.Path[len("/post/"):], "/")
		if id == "" {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}
		http.ServeFile(w, r, "static/post.html")
	})

	mux.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		if !isReadMethod(r.Method) {
			methodNotAllowed(w)
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "Post ID is required", http.StatusBadRequest)
			return
		}
		postHandler.GetPost(w, id)
	})

	return mux
}

func requireReadMethod(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isReadMethod(r.Method) {
			methodNotAllowed(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
