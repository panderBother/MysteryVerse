package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestClientTemporaryOperations(t *testing.T) {
	server := miniredis.RunT(t)
	client := NewClient(goredis.NewClient(&goredis.Options{Addr: server.Addr()}))
	t.Cleanup(func() {
		_ = client.Close()
	})

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	type roomProjection struct {
		Status string `json:"status"`
	}
	if err := client.SetJSON(ctx, "room:123", roomProjection{Status: "waiting"}, time.Minute); err != nil {
		t.Fatalf("SetJSON() error = %v", err)
	}

	var projection roomProjection
	if err := client.GetJSON(ctx, "room:123", &projection); err != nil {
		t.Fatalf("GetJSON() error = %v", err)
	}
	if projection.Status != "waiting" {
		t.Fatalf("projection.Status = %q, want waiting", projection.Status)
	}

	if err := client.GetJSON(ctx, "room:missing", &projection); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetJSON() error = %v, want ErrNotFound", err)
	}
}

func TestClientLeaseAndPresence(t *testing.T) {
	server := miniredis.RunT(t)
	client := NewClient(goredis.NewClient(&goredis.Options{Addr: server.Addr()}))
	t.Cleanup(func() {
		_ = client.Close()
	})
	ctx := context.Background()

	acquired, err := client.AcquireLease(ctx, "game:1", "owner-a", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("AcquireLease() = %t, %v", acquired, err)
	}

	acquired, err = client.AcquireLease(ctx, "game:1", "owner-b", time.Minute)
	if err != nil {
		t.Fatalf("AcquireLease() second error = %v", err)
	}
	if acquired {
		t.Fatal("AcquireLease() allowed a second owner")
	}

	refreshed, err := client.RefreshLease(ctx, "game:1", "owner-b", time.Minute)
	if err != nil {
		t.Fatalf("RefreshLease() error = %v", err)
	}
	if refreshed {
		t.Fatal("RefreshLease() allowed a non-owner")
	}

	released, err := client.ReleaseLease(ctx, "game:1", "owner-a")
	if err != nil || !released {
		t.Fatalf("ReleaseLease() = %t, %v", released, err)
	}

	if err := client.SetPresence(ctx, "room:1", "user:1", time.Minute); err != nil {
		t.Fatalf("SetPresence() error = %v", err)
	}
	present, err := client.HasPresence(ctx, "room:1", "user:1")
	if err != nil || !present {
		t.Fatalf("HasPresence() = %t, %v", present, err)
	}
}

func TestClientRateLimit(t *testing.T) {
	server := miniredis.RunT(t)
	client := NewClient(goredis.NewClient(&goredis.Options{Addr: server.Addr()}))
	t.Cleanup(func() {
		_ = client.Close()
	})
	ctx := context.Background()

	for attempt := int64(1); attempt <= 3; attempt++ {
		allowed, count, err := client.AllowRateLimit(ctx, "chat:user:1", 2, time.Minute)
		if err != nil {
			t.Fatalf("AllowRateLimit() error = %v", err)
		}
		if count != attempt {
			t.Fatalf("count = %d, want %d", count, attempt)
		}
		if allowed != (attempt <= 2) {
			t.Fatalf("allowed = %t at attempt %d", allowed, attempt)
		}
	}
}

func TestClientMapsUnavailableRedis(t *testing.T) {
	server := miniredis.RunT(t)
	addr := server.Addr()
	server.Close()

	client := NewClient(goredis.NewClient(&goredis.Options{
		Addr:        addr,
		DialTimeout: 50 * time.Millisecond,
	}))
	t.Cleanup(func() {
		_ = client.Close()
	})

	err := client.Ping(context.Background())
	var redisErr *Error
	if !errors.As(err, &redisErr) {
		t.Fatalf("Ping() error = %T, want *Error", err)
	}
	if !redisErr.Retryable {
		t.Fatal("Ping() error should be retryable")
	}
}
