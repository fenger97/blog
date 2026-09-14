package configs

import (
	"os"
)

// Config 应用配置
type Config struct {
	ServerAddr string
	MongoURI   string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	serverPort := getEnv("SERVER_PORT", "8080")

	return &Config{
		ServerAddr: getEnv("SERVER_ADDR", "127.0.0.1:"+serverPort),
		MongoURI:   getEnv("MONGO_URI", "mongodb://127.0.0.1:27017"),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
