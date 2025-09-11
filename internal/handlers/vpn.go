package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
)

// SubscriptionHandler 公共订阅接口：读取文件并以 base64 返回
// 访问格式示例：/sub?token=xxxx
// 注意：不做登录校验、不校验 token，仅要求提供 token 参数（用于与常见客户端格式对齐）
func SubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 仅检查是否传了 token 参数以匹配常见订阅 URL 格式，不做实际校验
	if r.URL.Query().Get("token") == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile("vpn/vpn.txt")
	if err != nil {
		http.Error(w, "failed to read config file", http.StatusInternalServerError)
		return
	}
	fmt.Println(string(data))
	encoded := base64.StdEncoding.EncodeToString(data)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(encoded))
	fmt.Println(encoded)
}
