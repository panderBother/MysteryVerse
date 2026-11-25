package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/health"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/httpserver"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/logging"
	"github.com/mysteryverse/mysteryverse/backend/pkg/version"
)

func main() {
	cfg, err := config.Load(config.ServiceRealtimeRuntime)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel, os.Stdout)
	logger.Info("service configured", "service", cfg.ServiceName, "environment", cfg.Environment)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := httpserver.New(
		cfg.HTTPAddr,
		logger,
		cfg.ShutdownTimeout,
		health.New(cfg.ServiceName, version.ServiceVersion),
	)

	if err := server.Run(ctx); err != nil {
		logger.Error("run realtime runtime service", "error", err)
		os.Exit(1)
	}
}
