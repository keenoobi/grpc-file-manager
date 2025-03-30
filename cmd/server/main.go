package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/keenoobi/grpc-file-manager/config"
	"github.com/keenoobi/grpc-file-manager/internal/app"
)

const shutdownTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		return
	}

	application := app.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запуск приложения в отдельной горутине
	errCh := make(chan error, 1)
	go func() {
		if err := application.Run(ctx); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		slog.Error("Application failed", "error", err)
	case <-ctx.Done():
		slog.Info("Shutting down gracefully...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		select {
		case <-shutdownCtx.Done():
			slog.Warn("Graceful shutdown timed out")
		case err := <-errCh:
			if err != nil {
				slog.Error("Error during shutdown", "error", err)
			}
		}
	}
}
