package handlers

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"blog/configs"
)

var (
	config     *configs.Config
	cookieName = "blog_session"
)

// 简单的内存 session 存储
var sessions = make(map[string]struct{})

// InitAuth 初始化认证配置
func InitAuth(cfg *configs.Config) {
	config = cfg
}

// LoginRequest 登录请求体
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginHandler 处理登录请求
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username != config.AdminUsername || req.Password != config.AdminPassword {
		http.Error(w, "用户名或密码错误", http.StatusUnauthorized)
		return
	}

	// 生成简单 session id
	sid := generateSessionID()
	sessions[sid] = struct{}{}

	// 设置 cookie
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"登录成功"}`))
}

// LogoutHandler 处理退出登录请求
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err == nil {
		delete(sessions, cookie.Value)
	}

	// 让 cookie 过期
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"已退出"}`))
}

// StatusResponse 定义登录状态的返回格式
type StatusResponse struct {
	LoggedIn bool `json:"logged_in"`
}

// StatusHandler 检查并返回当前登录状态
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	loggedIn := CheckLogin(r)
	resp := StatusResponse{LoggedIn: loggedIn}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// CheckLogin 检查请求是否已登录
func CheckLogin(r *http.Request) bool {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return false
	}
	_, ok := sessions[cookie.Value]
	return ok
}

// 生成简单 session id
func generateSessionID() string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 32)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
