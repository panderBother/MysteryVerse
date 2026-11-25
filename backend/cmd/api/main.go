package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	gameruntime "github.com/mysteryverse/mysteryverse/backend/internal/game/runtime"
	"github.com/mysteryverse/mysteryverse/backend/internal/identity"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/health"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/httpserver"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/logging"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/secret"
	"github.com/mysteryverse/mysteryverse/backend/internal/room"
	"github.com/mysteryverse/mysteryverse/backend/internal/script"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
	"github.com/mysteryverse/mysteryverse/backend/pkg/version"
)

func main() {
	cfg, err := config.Load(config.ServiceAPI)
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel, os.Stdout)
	logger.Info("service configured", "service", cfg.ServiceName, "environment", cfg.Environment)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.MySQL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	repository := storage.NewMySQLRepository(db)
	limiter := ratelimit.NewInMemory()
	identityService, err := identity.NewService(repository, cfg.Auth, limiter)
	if err != nil {
		logger.Error("initialize identity service", "error", err)
		os.Exit(1)
	}
	identityHandler := identity.NewHandler(identityService)
	roomService := room.NewService(repository, cfg.Auth, limiter)
	roomHandler := room.NewHandler(roomService)
	scriptService := script.NewService(repository)
	scriptHandler := script.NewHandler(scriptService)
	cipher, err := secret.NewCipher(cfg.Secrets.Key, cfg.Secrets.KeyID)
	if err != nil {
		logger.Error("initialize secret service", "error", err)
		os.Exit(1)
	}
	gameRuntime := gameruntime.NewService(
		repository,
		room.NewStartCoordinator(),
		cipher,
	)
	gameHandler := gameruntime.NewHandler(gameRuntime)

	root := httpserver.NewRootHandler(
		logger,
		health.New(cfg.ServiceName, version.ServiceVersion),
	)
	identityHandler.RegisterRoutes(root)
	roomHandler.RegisterRoutes(root, identityHandler)
	gameHandler.RegisterRoutes(root, identityHandler)
	scriptHandler.RegisterRoutes(root, identityHandler)

	server := httpserver.NewWithHandler(cfg.HTTPAddr, logger, cfg.ShutdownTimeout, root)

	if err := server.Run(ctx); err != nil {
		logger.Error("run API service", "error", err)
		os.Exit(1)
	}
}
