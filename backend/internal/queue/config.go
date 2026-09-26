package queue

import (
	"os"
	"strconv"
)

const (
	envRedisAddr     = "REDIS_ADDR"
	envRedisPassword = "REDIS_PASSWORD"
	envRedisDB       = "REDIS_DB"
	envRedisPrefix   = "REDIS_PREFIX"
)

func ConfigFromEnv() Config {
	cfg := Config{
		Address:  os.Getenv(envRedisAddr),
		Password: os.Getenv(envRedisPassword),
		Prefix:   os.Getenv(envRedisPrefix),
	}
	if cfg.Address == "" {
		cfg.Address = "localhost:6379"
	}
	if raw := os.Getenv(envRedisDB); raw != "" {
		if db, err := strconv.Atoi(raw); err == nil && db >= 0 {
			cfg.DB = db
		}
	}
	return cfg
}
