package database

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
)

func TestOpenMapsUnavailableDatabaseWithoutLeakingDSN(t *testing.T) {
	cfg := config.MySQLConfig{
		DSN:             "mv_user:super-secret@tcp(127.0.0.1:1)/mysteryverse",
		MaxOpenConns:    1,
		MaxIdleConns:    0,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
		PingTimeout:     200 * time.Millisecond,
	}

	_, err := Open(context.Background(), cfg)
	if err == nil {
		t.Fatal("Open() error = nil, want dependency error")
	}

	var storageErr *Error
	if !errors.As(err, &storageErr) {
		t.Fatalf("Open() error = %T, want *Error", err)
	}
	if storageErr.Code != CodeDependencyUnavailable {
		t.Fatalf("Code = %q, want %q", storageErr.Code, CodeDependencyUnavailable)
	}
	if !storageErr.Retryable {
		t.Fatal("unavailable database error should be retryable")
	}
	if strings.Contains(storageErr.Message, "super-secret") {
		t.Fatal("safe database error leaked a password")
	}
}
