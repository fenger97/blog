package configs

import (
	"os"
)

// Config 应用配置
type Config struct {
	AdminUsername string
	AdminPassword string
	ServerPort    string
	MongoURI      string
	SubToken      string
}

// LoadConfig 加载配置
func LoadConfig() *Config {
	return &Config{
		AdminUsername: getEnv("ADMIN_USERNAME", "fenger"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "zjf1834."),
		ServerPort:    getEnv("SERVER_PORT", "1834"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://admin:password@localhost:27018"),
		SubToken:      getEnv("SUB_TOKEN", "default_secret_token_change_me"),
	}
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
