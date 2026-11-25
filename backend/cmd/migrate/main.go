package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/logging"
	"github.com/mysteryverse/mysteryverse/backend/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	command := flags.String("command", "status", "migration command: up, down, status")
	targetVersion := flags.Int64("version", 0, "target version for down")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}
	versionSpecified := false
	flags.Visit(func(current *flag.Flag) {
		if current.Name == "version" {
			versionSpecified = true
		}
	})

	cfg, err := config.Load(config.ServiceMigrator)
	if err != nil {
		return fmt.Errorf("load migration configuration: %w", err)
	}

	logger := logging.New(cfg.LogLevel, os.Stdout)
	ctx := context.Background()

	db, err := database.Open(ctx, cfg.MySQL)
	if err != nil {
		return fmt.Errorf("connect migration database: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	logger.Info("running database migration", "command", *command, "database", cfg.MySQL.Database)

	switch *command {
	case "up":
		if err := goose.Up(db.SQL(), "."); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
	case "down":
		if versionSpecified && *targetVersion == 0 {
			for {
				currentVersion, err := goose.GetDBVersion(db.SQL())
				if err != nil {
					return fmt.Errorf("read database schema version: %w", err)
				}
				if currentVersion == 0 {
					break
				}
				if err := goose.Down(db.SQL(), "."); err != nil {
					return fmt.Errorf("rollback migrations to zero: %w", err)
				}
			}
		} else if versionSpecified {
			if err := goose.DownTo(db.SQL(), ".", *targetVersion); err != nil {
				return fmt.Errorf("rollback migrations to %d: %w", *targetVersion, err)
			}
		} else if err := goose.Down(db.SQL(), "."); err != nil {
			return fmt.Errorf("rollback migration: %w", err)
		}
	case "status":
		if err := goose.Status(db.SQL(), "."); err != nil {
			return fmt.Errorf("read migration status: %w", err)
		}
	default:
		return fmt.Errorf("unsupported migration command %q", *command)
	}

	version, err := goose.GetDBVersion(db.SQL())
	if err != nil {
		return fmt.Errorf("read database schema version: %w", err)
	}
	logger.Info("database migration command completed", "schema_version", version)

	return nil
}
