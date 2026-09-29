package configs

import (
	"os"
	"time"
)

// Config 应用配置
type Config struct {
	ServerAddr      string
	ContentIndexURL string
	ContentBaseURL  string
	ContentCacheTTL time.Duration
	GitHubToken     string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	serverPort := getEnv("SERVER_PORT", "8080")

	return &Config{
		ServerAddr:      getEnv("SERVER_ADDR", "127.0.0.1:"+serverPort),
		ContentIndexURL: getEnv("CONTENT_INDEX_URL", "https://raw.githubusercontent.com/fenger97/blog/master/content/index.json"),
		ContentBaseURL:  getEnv("CONTENT_BASE_URL", "https://raw.githubusercontent.com/fenger97/blog/master/content"),
		ContentCacheTTL: getDuration("CONTENT_CACHE_TTL", 5*time.Minute),
		GitHubToken:     os.Getenv("GITHUB_TOKEN"),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration < 0 {
		return defaultValue
	}
	return duration
}
