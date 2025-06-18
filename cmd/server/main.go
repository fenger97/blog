package main

import (
	"log"
	"net/http"

	"blog/internal/handlers"
	"blog/internal/models"
)

func main() {
	// 创建存储实例
	store := models.NewStore()

	// 创建处理器
	postHandler := handlers.NewPostHandler(store)

	// 设置静态文件服务
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 设置主页
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	})

	// 设置API路由
	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			postHandler.CreatePost(w, r)
		case http.MethodGet:
			if r.URL.Query().Get("id") != "" {
				postHandler.GetPost(w, r)
			} else {
				postHandler.GetAllPosts(w, r)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 启动服务器
	log.Println("Server starting on :1834...")
	if err := http.ListenAndServe(":1834", nil); err != nil {
		log.Fatal(err)
	}
}
