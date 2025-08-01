package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
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
			if r.URL.Query().Get("id") != "" {
				postHandler.GetPost(w, r)
			} else {
				postHandler.GetAllPosts(w, r)
			}
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
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
	if err := http.ListenAndServe(":"+cfg.ServerPort, nil); err != nil {
		log.Fatal(err)
	}
}
