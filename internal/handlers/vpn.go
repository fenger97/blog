package handlers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
)

// SubscriptionHandler 公共订阅接口：读取文件并根据Format参数返回不同格式
// 访问格式示例：/sub?token=xxxx&format=base64 或 /sub?token=xxxx&format=clash
// 支持的Format值：base64（默认）、clash
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

	// 获取Format参数，默认为base64
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "base64"
	}

	// 根据Format参数返回不同格式
	switch format {
	case "base64":
		data, err := os.ReadFile("vpn/vpn.txt")
		if err != nil {
			http.Error(w, "failed to read config file", http.StatusInternalServerError)
			return
		}
		// fmt.Println(string(data))
		encoded := base64.StdEncoding.EncodeToString(data)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(encoded))
		// fmt.Println(encoded)
		fmt.Println("Returned base64 format")
	case "clash":
		data, err := os.ReadFile("vpn/clash.txt")
		if err != nil {
			http.Error(w, "failed to read config file", http.StatusInternalServerError)
			return
		}
		// fmt.Println(string(data))
		// 直接返回原始文件内容，适用于Clash格式
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		fmt.Println("Returned clash format")
	default:
		http.Error(w, "unsupported format. supported formats: base64, clash", http.StatusBadRequest)
		return
	}
}
