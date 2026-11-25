package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryLimiter(t *testing.T) {
	limiter := NewInMemory()
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }

	for attempt := int64(1); attempt <= 3; attempt++ {
		decision, err := limiter.Allow(context.Background(), "login:alice", 2, time.Minute)
		if err != nil {
			t.Fatalf("Allow() error = %v", err)
		}
		if decision.Allowed != (attempt <= 2) {
			t.Fatalf("Allowed = %t at attempt %d", decision.Allowed, attempt)
		}
	}

	now = now.Add(time.Minute)
	decision, err := limiter.Allow(context.Background(), "login:alice", 2, time.Minute)
	if err != nil || !decision.Allowed {
		t.Fatalf("Allow() after reset = %t, %v", decision.Allowed, err)
	}
}
