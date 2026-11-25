package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
)

const (
	defaultKeyPrefix = "mv"
	defaultTTL       = 5 * time.Minute
)

var ErrNotFound = errors.New("redis key not found")

type Client struct {
	client      *goredis.Client
	prefix      string
	pingTimeout time.Duration
}

func Open(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate redis configuration: %w", err)
	}

	options, err := buildOptions(cfg)
	if err != nil {
		return nil, err
	}

	client := &Client{
		client:      goredis.NewClient(options),
		prefix:      defaultKeyPrefix,
		pingTimeout: cfg.PingTimeout,
	}
	if err := client.Ping(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func NewClient(client *goredis.Client) *Client {
	return &Client{
		client:      client,
		prefix:      defaultKeyPrefix,
		pingTimeout: 2 * time.Second,
	}
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	timeout := c.pingTimeout
	if timeout <= 0 {
		timeout = time.Second
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := c.client.Ping(pingCtx).Err(); err != nil {
		return mapError(err)
	}
	return nil
}

func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("temporary redis values require a positive ttl")
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal redis value: %w", err)
	}

	redisKey, err := c.key(key)
	if err != nil {
		return err
	}
	if err := c.client.Set(ctx, redisKey, payload, ttl).Err(); err != nil {
		return mapError(err)
	}
	return nil
}

func (c *Client) GetJSON(ctx context.Context, key string, destination any) error {
	redisKey, err := c.key(key)
	if err != nil {
		return err
	}
	payload, err := c.client.Get(ctx, redisKey).Bytes()
	if errors.Is(err, goredis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return mapError(err)
	}
	if err := json.Unmarshal(payload, destination); err != nil {
		return fmt.Errorf("unmarshal redis value: %w", err)
	}
	return nil
}

func (c *Client) SetPresence(ctx context.Context, scope string, actorID string, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("presence requires a positive ttl")
	}
	key, err := joinKey("presence", scope, actorID)
	if err != nil {
		return err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return err
	}
	return mapError(c.client.Set(ctx, redisKey, "1", ttl).Err())
}

func (c *Client) ClearPresence(ctx context.Context, scope string, actorID string) error {
	key, err := joinKey("presence", scope, actorID)
	if err != nil {
		return err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return err
	}
	return mapError(c.client.Del(ctx, redisKey).Err())
}

func (c *Client) HasPresence(ctx context.Context, scope string, actorID string) (bool, error) {
	key, err := joinKey("presence", scope, actorID)
	if err != nil {
		return false, err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return false, err
	}
	count, err := c.client.Exists(ctx, redisKey).Result()
	if err != nil {
		return false, mapError(err)
	}
	return count == 1, nil
}

func (c *Client) AcquireLease(
	ctx context.Context,
	name string,
	token string,
	ttl time.Duration,
) (bool, error) {
	if ttl <= 0 {
		return false, fmt.Errorf("lease requires a positive ttl")
	}
	key, err := joinKey("lease", name)
	if err != nil {
		return false, err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return false, err
	}
	acquired, err := c.client.SetNX(ctx, redisKey, token, ttl).Result()
	if err != nil {
		return false, mapError(err)
	}
	return acquired, nil
}

func (c *Client) RefreshLease(
	ctx context.Context,
	name string,
	token string,
	ttl time.Duration,
) (bool, error) {
	if ttl <= 0 {
		return false, fmt.Errorf("lease requires a positive ttl")
	}
	key, err := joinKey("lease", name)
	if err != nil {
		return false, err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return false, err
	}
	result, err := refreshLeaseScript.Run(
		ctx,
		c.client,
		[]string{redisKey},
		token,
		ttl.Milliseconds(),
	).Int()
	if err != nil {
		return false, mapError(err)
	}
	return result == 1, nil
}

func (c *Client) ReleaseLease(ctx context.Context, name string, token string) (bool, error) {
	key, err := joinKey("lease", name)
	if err != nil {
		return false, err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return false, err
	}
	result, err := releaseLeaseScript.Run(
		ctx,
		c.client,
		[]string{redisKey},
		token,
	).Int()
	if err != nil {
		return false, mapError(err)
	}
	return result == 1, nil
}

func (c *Client) AllowRateLimit(
	ctx context.Context,
	name string,
	limit int64,
	window time.Duration,
) (bool, int64, error) {
	if limit <= 0 {
		return false, 0, fmt.Errorf("rate limit must be positive")
	}
	if window <= 0 {
		return false, 0, fmt.Errorf("rate limit window must be positive")
	}
	key, err := joinKey("ratelimit", name)
	if err != nil {
		return false, 0, err
	}
	redisKey, err := c.key(key)
	if err != nil {
		return false, 0, err
	}
	count, err := rateLimitScript.Run(
		ctx,
		c.client,
		[]string{redisKey},
		window.Milliseconds(),
	).Int64()
	if err != nil {
		return false, 0, mapError(err)
	}
	return count <= limit, count, nil
}

func (c *Client) key(parts ...string) (string, error) {
	key, err := joinKey(parts...)
	if err != nil {
		return "", err
	}
	return c.prefix + ":" + key, nil
}

func buildOptions(cfg config.RedisConfig) (*goredis.Options, error) {
	if cfg.URL != "" {
		options, err := goredis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("parse REDIS_URL: %w", err)
		}
		applyRedisDefaults(options, cfg)
		return options, nil
	}

	options := &goredis.Options{
		Addr:         net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port)),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.IOTimeout,
		WriteTimeout: cfg.IOTimeout,
	}
	if cfg.TLSEnabled {
		options.TLSConfig = newTLSConfig(cfg)
	}
	return options, nil
}

func applyRedisDefaults(options *goredis.Options, cfg config.RedisConfig) {
	if cfg.PoolSize > 0 {
		options.PoolSize = cfg.PoolSize
	}
	if cfg.DialTimeout > 0 {
		options.DialTimeout = cfg.DialTimeout
	}
	if cfg.IOTimeout > 0 {
		options.ReadTimeout = cfg.IOTimeout
		options.WriteTimeout = cfg.IOTimeout
	}
}

func joinKey(parts ...string) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("redis key requires at least one segment")
	}

	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return "", fmt.Errorf("redis key segment is empty")
		}
		if strings.ContainsAny(part, "\r\n\t ") {
			return "", fmt.Errorf("redis key segment contains an unsupported character")
		}
		normalized = append(normalized, part)
	}

	return strings.Join(normalized, ":"), nil
}

var (
	refreshLeaseScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

	releaseLeaseScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

	rateLimitScript = goredis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)
)
