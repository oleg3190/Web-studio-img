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
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
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
		ReadTimeout:     durationEnv("HTTP_READ_TIMEOUT", 10*time.Second),
		WriteTimeout:    durationEnv("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     durationEnv("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: durationEnv("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	if cfg.Port == "" {
		return Config{}, errors.New("APP_PORT must not be empty")
	}
	if _, err := strconv.Atoi(cfg.Port); err != nil {
		return Config{}, fmt.Errorf("APP_PORT must be numeric: %w", err)
	}
	if cfg.Env == "production" && cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required in production")
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
