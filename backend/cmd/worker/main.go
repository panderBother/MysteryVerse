package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	gameruntime "github.com/mysteryverse/mysteryverse/backend/internal/game/runtime"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/lifecycle"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/logging"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/ratelimit"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/secret"
	"github.com/mysteryverse/mysteryverse/backend/internal/room"
	"github.com/mysteryverse/mysteryverse/backend/internal/storage"
)

func main() {
	cfg, err := config.Load(config.ServiceWorker)
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

	roomService := room.NewService(
		storage.NewMySQLRepository(db),
		cfg.Auth,
		ratelimit.NewInMemory(),
	)
	cipher, err := secret.NewCipher(cfg.Secrets.Key, cfg.Secrets.KeyID)
	if err != nil {
		logger.Error("initialize secret service", "error", err)
		os.Exit(1)
	}
	gameService := gameruntime.NewService(
		storage.NewMySQLRepository(db),
		room.NewStartCoordinator(),
		cipher,
	)
	worker := lifecycle.Worker{
		Name:     cfg.ServiceName,
		Interval: cfg.WorkerInterval,
		Logger:   logger,
		OnTick: func(ctx context.Context) error {
			if _, err := roomService.ExpireWaitingRooms(ctx); err != nil {
				return err
			}
			_, err := gameService.ExpireRunningGames(ctx)
			return err
		},
	}

	if err := worker.Run(ctx); err != nil {
		logger.Error("run worker", "error", err)
		os.Exit(1)
	}
}
