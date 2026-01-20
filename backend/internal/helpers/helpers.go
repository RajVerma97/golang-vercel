package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/joho/godotenv"
)

func GetEnv[T any](key string, defaultValue T) T {
	if value, exists := os.LookupEnv(key); exists {
		var result T
		switch any(defaultValue).(type) {
		case string:
			result = any(value).(T)
		case int:
			if v, err := strconv.Atoi(value); err == nil {
				result = any(v).(T)
			} else {
				result = defaultValue
			}
		case bool:
			if v, err := strconv.ParseBool(value); err == nil {
				result = any(v).(T)
			} else {
				result = defaultValue
			}
		default:
			result = defaultValue
		}
		return result
	}
	return defaultValue
}

func LoadEnv() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Try multiple paths for flexibility
	paths := []string{
		".env",
		"/app/.env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	var loaded bool
	for _, path := range paths {
		if err := godotenv.Load(path); err == nil {
			logger.Info(fmt.Sprintf("Loaded .env from: %s", path))
			loaded = true
			break
		}
	}

	if !loaded {
		logger.Info("No .env file found, using system environment variables")
	}
}

func InitLogger() {
	err := logger.Init(os.Getenv("APP_ENV"))
	if err != nil {
		logger.Error("failed to init logger", err)
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
}
