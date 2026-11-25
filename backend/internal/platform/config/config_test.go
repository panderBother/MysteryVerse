package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadAPI(t *testing.T) {
	t.Setenv("GO_API_ADDR", ":18080")
	t.Setenv("GO_ENV", "test")
	t.Setenv("GO_LOG_LEVEL", "debug")
	t.Setenv("GO_SHUTDOWN_TIMEOUT", "3s")

	cfg, err := Load(ServiceAPI)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != ":18080" {
		t.Fatalf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":18080")
	}

	if cfg.Environment != "test" {
		t.Fatalf("Environment = %q, want test", cfg.Environment)
	}

	if cfg.ShutdownTimeout != 3*time.Second {
		t.Fatalf("ShutdownTimeout = %s, want 3s", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidInterval(t *testing.T) {
	t.Setenv("GO_WORKER_INTERVAL", "-1s")

	if _, err := Load(ServiceWorker); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadDatabaseAndRedisConfig(t *testing.T) {
	t.Setenv("MYSQL_HOST", "db.internal")
	t.Setenv("MYSQL_PORT", "3307")
	t.Setenv("MYSQL_DATABASE", "mv_test")
	t.Setenv("MYSQL_USER", "mv_user")
	t.Setenv("MYSQL_PASSWORD", "secret")
	t.Setenv("MYSQL_MAX_OPEN_CONNS", "30")
	t.Setenv("MYSQL_MAX_IDLE_CONNS", "15")
	t.Setenv("REDIS_HOST", "redis.internal")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_TLS_ENABLED", "true")

	cfg, err := Load(ServiceAPI)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.MySQL.Host != "db.internal" || cfg.MySQL.Port != 3307 {
		t.Fatalf("MySQL endpoint = %s:%d", cfg.MySQL.Host, cfg.MySQL.Port)
	}
	if cfg.MySQL.MaxOpenConns != 30 || cfg.MySQL.MaxIdleConns != 15 {
		t.Fatalf("MySQL pool = %d/%d", cfg.MySQL.MaxOpenConns, cfg.MySQL.MaxIdleConns)
	}
	if cfg.Redis.Host != "redis.internal" || cfg.Redis.Port != 6380 {
		t.Fatalf("Redis endpoint = %s:%d", cfg.Redis.Host, cfg.Redis.Port)
	}
	if cfg.Redis.DB != 2 || !cfg.Redis.TLSEnabled {
		t.Fatalf("Redis config = db %d, TLS %t", cfg.Redis.DB, cfg.Redis.TLSEnabled)
	}
}

func TestLoadRejectsInvalidDatabasePool(t *testing.T) {
	t.Setenv("MYSQL_MAX_OPEN_CONNS", "2")
	t.Setenv("MYSQL_MAX_IDLE_CONNS", "3")

	if _, err := Load(ServiceAPI); err == nil {
		t.Fatal("Load() error = nil, want invalid pool error")
	}
}

func TestLoadAuthConfig(t *testing.T) {
	t.Setenv("AUTH_SESSION_TTL", "2h")
	t.Setenv("AUTH_REMEMBER_SESSION_TTL", "48h")
	t.Setenv("AUTH_ALLOWED_ORIGINS", "http://localhost:5173,https://app.example")
	t.Setenv("AUTH_CHAT_RATE_LIMIT", "20")
	t.Setenv("AUTH_CHAT_MAX_LENGTH", "2000")

	cfg, err := Load(ServiceAPI)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.SessionTTL != 2*time.Hour || cfg.Auth.RememberSessionTTL != 48*time.Hour {
		t.Fatalf("Auth TTLs = %s/%s", cfg.Auth.SessionTTL, cfg.Auth.RememberSessionTTL)
	}
	if len(cfg.Auth.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins = %#v", cfg.Auth.AllowedOrigins)
	}
	if cfg.Auth.ChatRateLimit != 20 || cfg.Auth.ChatMaxLength != 2000 {
		t.Fatalf("Auth chat config = %d/%d", cfg.Auth.ChatRateLimit, cfg.Auth.ChatMaxLength)
	}
}

func TestProductionRequiresSecureIdentityKey(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	t.Setenv("AUTH_IDENTITY_HMAC_KEY", "change-me")

	if _, err := Load(ServiceAPI); err == nil {
		t.Fatal("Load() accepted a development HMAC key in production")
	}
}

func TestProductionRequiresSecureDataSecretKey(t *testing.T) {
	t.Setenv("GO_ENV", "production")
	t.Setenv("AUTH_IDENTITY_HMAC_KEY", strings.Repeat("i", 32))
	t.Setenv("DATA_SECRET_KEY", "change-me")

	if _, err := Load(ServiceAPI); err == nil {
		t.Fatal("Load() accepted a development data secret key in production")
	}
}

func TestParsePort(t *testing.T) {
	port, err := ParsePort("8080")
	if err != nil {
		t.Fatalf("ParsePort() error = %v", err)
	}

	if port != 8080 {
		t.Fatalf("ParsePort() = %d, want 8080", port)
	}

	if _, err := ParsePort("70000"); err == nil {
		t.Fatal("ParsePort() error = nil, want range error")
	}
}
