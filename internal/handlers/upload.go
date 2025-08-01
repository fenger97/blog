package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	MaxFileSize = 10 << 20 // 10MB
	UploadDir   = "static/uploads/images"
)

// UploadResponse 上传响应结构
type UploadResponse struct {
	Success bool   `json:"success"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
}

// UploadImage 处理图片上传
func UploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 检查登录状态
	if !CheckLogin(r) {
		http.Error(w, "未登录或无权限", http.StatusUnauthorized)
		return
	}

	// 解析多部分表单
	err := r.ParseMultipartForm(MaxFileSize)
	if err != nil {
		log.Printf("Parse multipart form error: %v", err)
		http.Error(w, "文件解析失败", http.StatusBadRequest)
		return
	}

	// 获取上传的文件
	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("Get form file error: %v", err)
		http.Error(w, "获取文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "只支持图片文件", http.StatusBadRequest)
		return
	}

	// 检查文件大小
	if header.Size > MaxFileSize {
		http.Error(w, "文件大小超过限制", http.StatusBadRequest)
		return
	}

	// 生成文件名
	ext := filepath.Ext(header.Filename)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d%s", timestamp, ext)
	filepath := filepath.Join(UploadDir, filename)

	// 创建目标文件
	dst, err := os.Create(filepath)
	if err != nil {
		log.Printf("Create file error: %v", err)
		http.Error(w, "创建文件失败", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 复制文件内容
	_, err = io.Copy(dst, file)
	if err != nil {
		log.Printf("Copy file error: %v", err)
		http.Error(w, "保存文件失败", http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	imageURL := fmt.Sprintf("/static/uploads/images/%s", filename)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success":true,"url":"%s"}`, imageURL)

	log.Printf("Image uploaded successfully: %s", imageURL)
}
