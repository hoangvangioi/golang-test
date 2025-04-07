package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Configuration struct định nghĩa các biến cấu hình
type Configuration struct {
	DatabaseURL string
	GroqAPIKey  string
	GroqAPIURL  string
}

// LoadConfig đọc cấu hình từ file .env hoặc biến môi trường
func LoadConfig() (*Configuration, error) {
	err := godotenv.Load()
	if err != nil {
		// Nếu không tìm thấy file .env, thử đọc từ biến môi trường
	}

	return &Configuration{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		GroqAPIKey:  getEnv("GROQ_API_KEY", ""),
		GroqAPIURL:  getEnv("GROQ_API_URL", ""),
	}, nil
}

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
