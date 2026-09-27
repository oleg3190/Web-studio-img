package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/oleg3190/Web-studio-img/backend/internal/config"
	"github.com/oleg3190/Web-studio-img/backend/internal/generation"
	"github.com/oleg3190/Web-studio-img/backend/internal/providers/yandexart"
	"github.com/oleg3190/Web-studio-img/backend/internal/provenance"
	"github.com/oleg3190/Web-studio-img/backend/internal/queue"
	"github.com/oleg3190/Web-studio-img/backend/internal/httpapi"
	"github.com/oleg3190/Web-studio-img/backend/internal/iterations"
	"github.com/oleg3190/Web-studio-img/backend/internal/projects"
	"github.com/oleg3190/Web-studio-img/backend/internal/security"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	limiter := security.NewRateLimiter(cfg.RateLimit, cfg.RateWindow)
	var projectDB *sql.DB
	var projectHandler *projects.Handler
	var iterationHandler *iterations.Handler
	var generationHandler *generation.Handler
	if cfg.DatabaseURL != "" {
		projectDB, err = sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			logger.Error("database open failed", "error", err)
			os.Exit(1)
		}
		defer projectDB.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = projectDB.PingContext(ctx)
		cancel()
		if err != nil {
			logger.Error("database ping failed", "error", err)
			os.Exit(1)
		}

		store, storeErr := projects.NewSQLStore(projectDB)
		if storeErr != nil {
			logger.Error("project store initialization failed", "error", storeErr)
			os.Exit(1)
		}
		projectHandler, err = projects.NewHandler(store)
		if err != nil {
			logger.Error("project handler initialization failed", "error", err)
			os.Exit(1)
		}
		iterationStore, iterationStoreErr := iterations.NewSQLStore(projectDB)
		if iterationStoreErr != nil {
			logger.Error("iteration store initialization failed", "error", iterationStoreErr)
			os.Exit(1)
		}
		iterationHandler, err = iterations.NewHandler(iterationStore)
		if err != nil {
			logger.Error("iteration handler initialization failed", "error", err)
			os.Exit(1)
		}

		if cfg.RedisURL != "" && cfg.YandexARTAPIKey != "" && cfg.YandexARTFolderID != "" {
			redisCfg, redisErr := queue.ParseRedisURL(cfg.RedisURL)
			if redisErr != nil { logger.Error("redis configuration failed", "error", redisErr); os.Exit(1) }
			queueClient, queueErr := queue.NewClient(redisCfg)
			if queueErr != nil { logger.Error("queue initialization failed", "error", queueErr); os.Exit(1) }
			defer queueClient.Close()
			provider, providerErr := yandexart.New(yandexart.Config{Endpoint: cfg.YandexARTEndpoint, OperationEndpoint: cfg.YandexARTOperationEndpoint, APIKey: cfg.YandexARTAPIKey, FolderID: cfg.YandexARTFolderID, Model: cfg.YandexARTModel})
			if providerErr != nil { logger.Error("YandexART initialization failed", "error", providerErr); os.Exit(1) }
			generationStore, generationErr := generation.NewSQLStore(projectDB)
			if generationErr != nil { logger.Error("generation store initialization failed", "error", generationErr); os.Exit(1) }
			provenanceStore, provenanceErr := provenance.NewStore(projectDB)
			if provenanceErr != nil { logger.Error("provenance store initialization failed", "error", provenanceErr); os.Exit(1) }
			generationHandler, err = generation.NewHandlerWithProvenance(generationStore, queueClient, provider, provenanceStore)
			if err != nil { logger.Error("generation handler initialization failed", "error", err); os.Exit(1) }
		}
	}

	api := httpapi.NewServerWithProjectsIterationsAndGeneration(logger, cfg.CORSOrigins, limiter, projectHandler, iterationHandler, generationHandler)
	srv := api.HTTPServer(":"+cfg.Port, cfg.ReadTimeout, cfg.WriteTimeout, cfg.IdleTimeout)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("api server starting", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sig, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sig.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("api server failed", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("api server stopped")
}
