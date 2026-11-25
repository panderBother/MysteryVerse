package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Decision struct {
	Allowed           bool
	Remaining         int64
	RetryAfterSeconds int64
}

type Limiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (Decision, error)
}

type InMemory struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]bucket
}

type bucket struct {
	count   int64
	resetAt time.Time
}

func NewInMemory() *InMemory {
	return &InMemory{
		now:     time.Now,
		buckets: make(map[string]bucket),
	}
}

func (l *InMemory) Allow(
	ctx context.Context,
	key string,
	limit int64,
	window time.Duration,
) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return Decision{}, err
	}
	if key == "" {
		return Decision{}, fmt.Errorf("rate limit key is required")
	}
	if limit <= 0 || window <= 0 {
		return Decision{}, fmt.Errorf("rate limit and window must be positive")
	}

	now := l.now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	current, exists := l.buckets[key]
	if !exists || !now.Before(current.resetAt) {
		current = bucket{resetAt: now.Add(window)}
	}

	allowed := current.count < limit
	if allowed {
		current.count++
	}
	l.buckets[key] = current

	remaining := max(limit-current.count, 0)
	retryAfter := int64(0)
	if !allowed {
		remaining := current.resetAt.Sub(now)
		retryAfter = max(int64((remaining+time.Second-1)/time.Second), 1)
	}

	return Decision{
		Allowed:           allowed,
		Remaining:         remaining,
		RetryAfterSeconds: retryAfter,
	}, nil
}
