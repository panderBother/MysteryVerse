package migrations

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pressly/goose/v3"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/database"
)

func TestMigrationsUpStatusAndDown(t *testing.T) {
	dsn := os.Getenv("MYSQL_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_MIGRATION_TEST_DSN is not set")
	}

	ctx := context.Background()
	db, err := database.Open(ctx, config.MySQLConfig{
		DSN:             dsn,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
		PingTimeout:     2 * time.Second,
	})
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.SQL().ExecContext(ctx, "DROP DATABASE IF EXISTS mv_ai"); err != nil {
		t.Fatalf("drop stale mv_ai schema: %v", err)
	}
	if _, err := db.SQL().ExecContext(ctx, `
CREATE DATABASE mv_ai CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci
`); err != nil {
		t.Fatalf("create mv_ai schema: %v", err)
	}

	var existingTables int
	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name <> 'goose_db_version'
`).Scan(&existingTables); err != nil {
		t.Fatalf("count existing tables: %v", err)
	}
	if existingTables != 0 {
		t.Fatalf("migration test database is not empty: %d tables", existingTables)
	}

	goose.SetBaseFS(FS)
	if err := goose.SetDialect("mysql"); err != nil {
		t.Fatalf("SetDialect() error = %v", err)
	}
	if err := goose.Up(db.SQL(), "."); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if err := goose.Up(db.SQL(), "."); err != nil {
		t.Fatalf("repeated Up() error = %v", err)
	}

	var tableCount int
	var triggerCount int
	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name <> 'goose_db_version'
`).Scan(&tableCount); err != nil {
		t.Fatalf("count migrated tables: %v", err)
	}
	if tableCount < 40 {
		t.Fatalf("migrated table count = %d, want at least 40", tableCount)
	}
	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.triggers
WHERE trigger_schema = DATABASE()
`).Scan(&triggerCount); err != nil {
		t.Fatalf("count audit triggers: %v", err)
	}
	if triggerCount != 2 {
		t.Fatalf("audit trigger count = %d, want 2", triggerCount)
	}
	var aiTableCount int
	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'mv_ai'
`).Scan(&aiTableCount); err != nil {
		t.Fatalf("count mv_ai tables: %v", err)
	}
	if aiTableCount != 8 {
		t.Fatalf("mv_ai table count = %d, want 8", aiTableCount)
	}

	if err := goose.Down(db.SQL(), "."); err != nil {
		t.Fatalf("Down() trigger-only migration error = %v", err)
	}

	for {
		version, err := goose.GetDBVersion(db.SQL())
		if err != nil {
			t.Fatalf("GetDBVersion() error = %v", err)
		}
		if version == 0 {
			break
		}
		if err := goose.Down(db.SQL(), "."); err != nil {
			t.Fatalf("Down() from version %d error = %v", version, err)
		}
	}

	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name <> 'goose_db_version'
`).Scan(&tableCount); err != nil {
		t.Fatalf("count tables after rollback: %v", err)
	}
	if tableCount != 0 {
		t.Fatalf("table count after rollback = %d, want 0", tableCount)
	}
	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'mv_ai'
`).Scan(&aiTableCount); err != nil {
		t.Fatalf("count mv_ai tables after rollback: %v", err)
	}
	if aiTableCount != 0 {
		t.Fatalf("mv_ai table count after rollback = %d, want 0", aiTableCount)
	}

	if err := goose.Up(db.SQL(), "."); err != nil {
		t.Fatalf("second Up() after full rollback error = %v", err)
	}
	if err := db.SQL().QueryRowContext(ctx, `
SELECT COUNT(*)
FROM information_schema.tables
WHERE table_schema = 'mv_ai'
`).Scan(&aiTableCount); err != nil {
		t.Fatalf("count mv_ai tables after re-execution: %v", err)
	}
	if aiTableCount != 8 {
		t.Fatalf("mv_ai table count after re-execution = %d, want 8", aiTableCount)
	}
	for {
		version, err := goose.GetDBVersion(db.SQL())
		if err != nil {
			t.Fatalf("GetDBVersion() after re-execution error: %v", err)
		}
		if version == 0 {
			break
		}
		if err := goose.Down(db.SQL(), "."); err != nil {
			t.Fatalf("final Down() from version %d error: %v", version, err)
		}
	}
}
