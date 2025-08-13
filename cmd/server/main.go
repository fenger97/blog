package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"blog/configs"
	"blog/internal/database"
	"blog/internal/handlers"
	"blog/internal/models"
)

func main() {
	// 加载配置
	cfg := configs.LoadConfig()

	// 初始化认证模块
	handlers.InitAuth(cfg)

	// 连接到 MongoDB
	if err := database.ConnectMongoDB(); err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer database.CloseMongoDB()

	// 创建 MongoDB 存储实例
	store := models.NewMongoStore()

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

	// 注册登录接口
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.ServeFile(w, r, "static/login.html")
		} else if r.Method == http.MethodPost {
			handlers.LoginHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/logout", handlers.LogoutHandler)
	http.HandleFunc("/status", handlers.StatusHandler)

	// 注册图片上传接口
	http.HandleFunc("/upload/image", handlers.UploadImage)

	// 设置API路由
	http.HandleFunc("/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if !handlers.CheckLogin(r) {
				http.Error(w, "未登录或无权限", http.StatusUnauthorized)
				return
			}
			postHandler.CreatePost(w, r)
		case http.MethodGet:
			postHandler.GetAllPosts(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// 统一的文章详情/更新/删除路由
	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		// 稳健提取文章 ID：去掉前缀、清理尾部斜杠
		raw := strings.Trim(r.URL.Path[len("/post/"):], "/")
		accept := r.Header.Get("Accept")
		id := raw
		isJSONPath := false
		if strings.HasSuffix(raw, ".json") {
			id = strings.TrimSuffix(raw, ".json")
			isJSONPath = true
		}
		if id == "" {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			log.Printf("/post handler(GET): path=%s raw=%s id=%s accept=%s format=%s jsonPath=%v", r.URL.Path, raw, id, accept, r.URL.Query().Get("format"), isJSONPath)
			// 1) 明确的 .json 路径优先返回 JSON
			if isJSONPath {
				postHandler.GetPost(w, r, id)
				return
			}
			// 2) 其余情况按 format 或 Accept 处理
			if r.URL.Query().Get("format") == "json" || strings.Contains(accept, "application/json") {
				postHandler.GetPost(w, r, id)
				return
			}
			// 返回文章详情页面
			http.ServeFile(w, r, "static/post.html")
			return

		case http.MethodPut:
			if !handlers.CheckLogin(r) {
				http.Error(w, "未登录或无权限", http.StatusUnauthorized)
				return
			}
			postHandler.UpdatePost(w, r, id)
			return

		case http.MethodDelete:
			if !handlers.CheckLogin(r) {
				http.Error(w, "未登录或无权限", http.StatusUnauthorized)
				return
			}
			postHandler.DeletePost(w, r, id)
			return

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	})

	// JSON API：按 id 返回文章详情
	http.HandleFunc("/api/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			http.Error(w, "Post ID is required", http.StatusBadRequest)
			return
		}
		log.Printf("/api/post: id=%s", id)
		postHandler.GetPost(w, r, id)
	})

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		database.CloseMongoDB()
		os.Exit(0)
	}()

	// 启动服务器
	log.Printf("Server starting on :%s...", cfg.ServerPort)
	if err := http.ListenAndServe("127.0.0.1:"+cfg.ServerPort, nil); err != nil {
		log.Fatal(err)
	}
}
