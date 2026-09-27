package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env             string
	Port            string
	DatabaseURL     string
	RedisURL        string
	S3Endpoint      string
	S3Region        string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3UsePathStyle  bool
	JWTSecret       string
	CORSOrigins     []string
	RateLimit       int
	RateWindow      time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	YandexARTEndpoint string
	YandexARTOperationEndpoint string
	YandexARTAPIKey string
	YandexARTFolderID string
	YandexARTModel string
}

func Load() (Config, error) {
	cfg := Config{
		Env:             getenv("APP_ENV", "development"),
		Port:            getenv("APP_PORT", "8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisURL:        os.Getenv("REDIS_URL"),
		S3Endpoint:      os.Getenv("S3_ENDPOINT"),
		S3Region:        getenv("S3_REGION", "ru-central1"),
		S3Bucket:        os.Getenv("S3_BUCKET"),
		S3AccessKey:     os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:     os.Getenv("S3_SECRET_KEY"),
		S3UsePathStyle:  boolEnv("S3_PATH_STYLE", false),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		CORSOrigins:     splitCSV(getenv("CORS_ORIGINS", "http://localhost:5173")),
		RateLimit:       intEnv("RATE_LIMIT_REQUESTS", 120),
		RateWindow:      durationEnv("RATE_LIMIT_WINDOW", time.Minute),
		ReadTimeout:     durationEnv("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    durationEnv("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     durationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: durationEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		YandexARTEndpoint: getenv("YANDEXART_ENDPOINT", "https://llm.api.cloud.yandex.net"),
		YandexARTOperationEndpoint: getenv("YANDEXART_OPERATION_ENDPOINT", "https://operation.api.cloud.yandex.net"),
		YandexARTAPIKey: os.Getenv("YANDEXART_API_KEY"),
		YandexARTFolderID: os.Getenv("YANDEXART_FOLDER_ID"),
		YandexARTModel: getenv("YANDEXART_MODEL", "yandex-art/latest"),
	}
	if cfg.Port == "" {
		return Config{}, errors.New("APP_PORT must not be empty")
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("APP_PORT must be numeric: %w", err)
	}
	if cfg.RateLimit <= 0 {
		return Config{}, errors.New("RATE_LIMIT_REQUESTS must be positive")
	}
	if cfg.Env == "production" {
		if len(cfg.JWTSecret) < 32 {
			return Config{}, errors.New("JWT_SECRET must be at least 32 characters in production")
		}
		if len(cfg.CORSOrigins) == 0 {
			return Config{}, errors.New("CORS_ORIGINS must not be empty in production")
		}
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return fallback
	}
	return duration
}

func intEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
