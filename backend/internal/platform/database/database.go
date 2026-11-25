package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
)

const driverName = "mysql"

type DB struct {
	cfg config.MySQLConfig
	sql *sql.DB
}

func Open(ctx context.Context, cfg config.MySQLConfig) (*DB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate database configuration: %w", err)
	}

	dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, MapError(err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	db := &DB{cfg: cfg, sql: sqlDB}
	if err := db.Ping(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return db, nil
}

func OpenSQL(database *sql.DB) *DB {
	return &DB{sql: database}
}

func (db *DB) SQL() *sql.DB {
	return db.sql
}

func (db *DB) Ping(ctx context.Context) error {
	timeout := db.cfg.PingTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := db.sql.PingContext(pingCtx); err != nil {
		return MapError(err)
	}

	return nil
}

func (db *DB) Close() error {
	if db == nil || db.sql == nil {
		return nil
	}

	return db.sql.Close()
}

func (db *DB) WithTransaction(
	ctx context.Context,
	options *sql.TxOptions,
	fn func(context.Context, *sql.Tx) error,
) error {
	if fn == nil {
		return fmt.Errorf("transaction callback is required")
	}

	txOptions := sql.TxOptions{Isolation: sql.LevelReadCommitted}
	if options != nil {
		txOptions = *options
	}

	tx, err := db.sql.BeginTx(ctx, &txOptions)
	if err != nil {
		return MapError(err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(ctx, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return MapError(err)
	}

	return nil
}

func buildDSN(cfg config.MySQLConfig) (string, error) {
	if cfg.DSN != "" {
		parsed, err := mysqlDriver.ParseDSN(cfg.DSN)
		if err != nil {
			return "", fmt.Errorf("parse MYSQL_DSN: %w", err)
		}
		applyDriverDefaults(parsed)
		return parsed.FormatDSN(), nil
	}

	if !validTLSMode(cfg.TLSMode) {
		return "", fmt.Errorf("unsupported MYSQL_TLS_MODE %q", cfg.TLSMode)
	}

	parsed := mysqlDriver.NewConfig()
	parsed.User = cfg.User
	parsed.Passwd = cfg.Password
	parsed.Net = "tcp"
	parsed.Addr = net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	parsed.DBName = cfg.Database
	parsed.Collation = "utf8mb4_0900_ai_ci"
	parsed.TLSConfig = cfg.TLSMode
	applyDriverDefaults(parsed)

	return parsed.FormatDSN(), nil
}

func applyDriverDefaults(cfg *mysqlDriver.Config) {
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	if cfg.Params == nil {
		cfg.Params = make(map[string]string)
	}
	if _, exists := cfg.Params["time_zone"]; !exists {
		cfg.Params["time_zone"] = "'+00:00'"
	}
}

func validTLSMode(value string) bool {
	switch value {
	case "", "false", "true", "skip-verify", "preferred":
		return true
	default:
		return false
	}
}
