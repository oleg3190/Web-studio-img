package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/hibiken/asynq"

	"github.com/oleg3190/Web-studio-img/backend/internal/assets"
	"github.com/oleg3190/Web-studio-img/backend/internal/config"
	"github.com/oleg3190/Web-studio-img/backend/internal/generation"
	"github.com/oleg3190/Web-studio-img/backend/internal/providers/yandexart"
	"github.com/oleg3190/Web-studio-img/backend/internal/provenance"
	"github.com/oleg3190/Web-studio-img/backend/internal/queue"
	"github.com/oleg3190/Web-studio-img/backend/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" || cfg.YandexARTAPIKey == "" || cfg.YandexARTFolderID == "" {
		logger.Error("worker requires DATABASE_URL, REDIS_URL, YANDEXART_API_KEY and YANDEXART_FOLDER_ID")
		os.Exit(1)
	}
	if cfg.S3Bucket == "" || cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		logger.Error("worker requires S3_BUCKET, S3_ACCESS_KEY and S3_SECRET_KEY")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		logger.Error("database open failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.PingContext(ctx); err != nil {
		cancel()
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	cancel()

	redisCfg, err := queue.ParseRedisURL(cfg.RedisURL)
	if err != nil {
		logger.Error("redis configuration failed", "error", err)
		os.Exit(1)
	}
	provider, err := yandexart.New(yandexart.Config{Endpoint: cfg.YandexARTEndpoint, OperationEndpoint: cfg.YandexARTOperationEndpoint, APIKey: cfg.YandexARTAPIKey, FolderID: cfg.YandexARTFolderID, Model: cfg.YandexARTModel})
	if err != nil {
		logger.Error("YandexART initialization failed", "error", err)
		os.Exit(1)
	}
	store, err := generation.NewSQLStore(db)
	if err != nil {
		logger.Error("generation store initialization failed", "error", err)
		os.Exit(1)
	}
	assetStore, err := assets.NewStore(db)
	if err != nil {
		logger.Error("asset store initialization failed", "error", err)
		os.Exit(1)
	}
	objectStorage, err := storage.NewS3Storage(context.Background(), storage.S3Config{
		Endpoint: cfg.S3Endpoint, Region: cfg.S3Region, Bucket: cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey, UsePathStyle: cfg.S3UsePathStyle,
	})
	if err != nil {
		logger.Error("object storage initialization failed", "error", err)
		os.Exit(1)
	}

	provenanceStore, err := provenance.NewStore(db)
	if err != nil { logger.Error("provenance store initialization failed", "error", err); os.Exit(1) }
	worker := &generation.Worker{
		Store: store, Provider: provider,
		Assets: &assets.Processor{Storage: objectStorage, Store: assetStore},
		Provenance: provenanceStore,
	}
	mux := asynq.NewServeMux()
	mux.HandleFunc(generation.TaskType, worker.Handle)
	server, err := queue.NewServer(queue.ServerConfig{
		Redis: redisCfg, Concurrency: 4, Queues: map[string]int{"default": 1}, ShutdownTimeout: cfg.ShutdownTimeout,
	}, mux)
	if err != nil {
		logger.Error("queue server initialization failed", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("generation worker starting")
		if err := server.Run(mux); err != nil {
			errCh <- err
		}
	}()

	sig, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-sig.Done():
		logger.Info("worker shutdown signal received")
	case err := <-errCh:
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}
	server.Shutdown()
	logger.Info("generation worker stopped")
}
