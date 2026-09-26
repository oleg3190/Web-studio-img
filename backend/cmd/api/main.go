package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/oleg3190/Web-studio-img/backend/internal/config"
	"github.com/oleg3190/Web-studio-img/backend/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	api := httpapi.NewServer(logger, cfg.CORSOrigins)
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
