package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ServiceAPI             = "go-api"
	ServiceRealtimeRuntime = "go-realtime-runtime"
	ServiceWorker          = "go-worker"
	ServiceMigrator        = "go-migrator"
)

type Config struct {
	Environment     string
	HTTPAddr        string
	LogLevel        string
	ServiceName     string
	ShutdownTimeout time.Duration
	WorkerInterval  time.Duration
	MySQL           MySQLConfig
	Redis           RedisConfig
	Auth            AuthConfig
	Secrets         SecretConfig
}

type MySQLConfig struct {
	DSN             string
	Host            string
	Port            int
	Database        string
	User            string
	Password        string
	TLSMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

type RedisConfig struct {
	URL         string
	Host        string
	Port        int
	Password    string
	DB          int
	TLSEnabled  bool
	TLSInsecure bool
	PoolSize    int
	DialTimeout time.Duration
	IOTimeout   time.Duration
	PingTimeout time.Duration
}

type AuthConfig struct {
	IdentityHMACKey    string
	CookieName         string
	SecureCookie       bool
	SessionTTL         time.Duration
	RememberSessionTTL time.Duration
	AllowedOrigins     []string
	RegisterRateLimit  int64
	RegisterRateWindow time.Duration
	LoginRateLimit     int64
	LoginRateWindow    time.Duration
	ChatRateLimit      int64
	ChatRateWindow     time.Duration
	ChatMaxLength      int
	RoomWaitingTTL     time.Duration
}

type SecretConfig struct {
	Key   string
	KeyID string
}

func Load(serviceName string) (Config, error) {
	cfg := Config{
		ServiceName:     serviceName,
		Environment:     envOrDefault("GO_ENV", "development"),
		LogLevel:        strings.ToLower(envOrDefault("GO_LOG_LEVEL", "info")),
		ShutdownTimeout: 10 * time.Second,
		WorkerInterval:  5 * time.Second,
	}

	mysqlCfg, err := loadMySQLConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.MySQL = mysqlCfg

	redisCfg, err := loadRedisConfig()
	if err != nil {
		return Config{}, err
	}
	cfg.Redis = redisCfg

	authCfg, err := loadAuthConfig(cfg.Environment)
	if err != nil {
		return Config{}, err
	}
	cfg.Auth = authCfg

	secretCfg := loadSecretConfig(cfg.Environment)
	if err := secretCfg.Validate(cfg.Environment); err != nil {
		return Config{}, fmt.Errorf("validate secret configuration: %w", err)
	}
	cfg.Secrets = secretCfg

	switch serviceName {
	case ServiceAPI:
		cfg.HTTPAddr = envOrDefault("GO_API_ADDR", ":8080")
	case ServiceRealtimeRuntime:
		cfg.HTTPAddr = envOrDefault("GO_REALTIME_RUNTIME_ADDR", ":8081")
	case ServiceWorker, ServiceMigrator:
		cfg.HTTPAddr = ""
	default:
		return Config{}, fmt.Errorf("unsupported service name %q", serviceName)
	}

	if value := os.Getenv("GO_SHUTDOWN_TIMEOUT"); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse GO_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.ShutdownTimeout = timeout
	}

	if value := os.Getenv("GO_WORKER_INTERVAL"); value != "" {
		interval, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse GO_WORKER_INTERVAL: %w", err)
		}
		cfg.WorkerInterval = interval
	}

	if cfg.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("shutdown timeout must be positive")
	}

	if cfg.WorkerInterval <= 0 {
		return Config{}, fmt.Errorf("worker interval must be positive")
	}

	if !validLogLevel(cfg.LogLevel) {
		return Config{}, fmt.Errorf("unsupported GO_LOG_LEVEL %q", cfg.LogLevel)
	}

	return cfg, nil
}

func loadMySQLConfig() (MySQLConfig, error) {
	port, err := parseEnvInt("MYSQL_PORT", 3306)
	if err != nil {
		return MySQLConfig{}, err
	}

	maxOpenConns, err := parseEnvInt("MYSQL_MAX_OPEN_CONNS", 20)
	if err != nil {
		return MySQLConfig{}, err
	}

	maxIdleConns, err := parseEnvInt("MYSQL_MAX_IDLE_CONNS", 10)
	if err != nil {
		return MySQLConfig{}, err
	}

	connMaxLifetime, err := parseEnvDuration("MYSQL_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return MySQLConfig{}, err
	}

	connMaxIdleTime, err := parseEnvDuration("MYSQL_CONN_MAX_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return MySQLConfig{}, err
	}

	pingTimeout, err := parseEnvDuration("MYSQL_PING_TIMEOUT", 3*time.Second)
	if err != nil {
		return MySQLConfig{}, err
	}

	cfg := MySQLConfig{
		DSN:             os.Getenv("MYSQL_DSN"),
		Host:            envOrDefault("MYSQL_HOST", "127.0.0.1"),
		Port:            port,
		Database:        envOrDefault("MYSQL_DATABASE", "mysteryverse"),
		User:            envOrDefault("MYSQL_USER", "mysteryverse"),
		Password:        os.Getenv("MYSQL_PASSWORD"),
		TLSMode:         envOrDefault("MYSQL_TLS_MODE", "preferred"),
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
		PingTimeout:     pingTimeout,
	}
	if err := cfg.Validate(); err != nil {
		return MySQLConfig{}, fmt.Errorf("validate MYSQL configuration: %w", err)
	}

	return cfg, nil
}

func loadRedisConfig() (RedisConfig, error) {
	port, err := parseEnvInt("REDIS_PORT", 6379)
	if err != nil {
		return RedisConfig{}, err
	}

	db, err := parseEnvInt("REDIS_DB", 0)
	if err != nil {
		return RedisConfig{}, err
	}

	poolSize, err := parseEnvInt("REDIS_POOL_SIZE", 10)
	if err != nil {
		return RedisConfig{}, err
	}

	dialTimeout, err := parseEnvDuration("REDIS_DIAL_TIMEOUT", 3*time.Second)
	if err != nil {
		return RedisConfig{}, err
	}

	ioTimeout, err := parseEnvDuration("REDIS_IO_TIMEOUT", 3*time.Second)
	if err != nil {
		return RedisConfig{}, err
	}

	pingTimeout, err := parseEnvDuration("REDIS_PING_TIMEOUT", 2*time.Second)
	if err != nil {
		return RedisConfig{}, err
	}

	tlsEnabled, err := parseEnvBool("REDIS_TLS_ENABLED", false)
	if err != nil {
		return RedisConfig{}, err
	}

	tlsInsecure, err := parseEnvBool("REDIS_TLS_INSECURE", false)
	if err != nil {
		return RedisConfig{}, err
	}

	cfg := RedisConfig{
		URL:         os.Getenv("REDIS_URL"),
		Host:        envOrDefault("REDIS_HOST", "127.0.0.1"),
		Port:        port,
		Password:    os.Getenv("REDIS_PASSWORD"),
		DB:          db,
		TLSEnabled:  tlsEnabled,
		TLSInsecure: tlsInsecure,
		PoolSize:    poolSize,
		DialTimeout: dialTimeout,
		IOTimeout:   ioTimeout,
		PingTimeout: pingTimeout,
	}
	if err := cfg.Validate(); err != nil {
		return RedisConfig{}, fmt.Errorf("validate REDIS configuration: %w", err)
	}

	return cfg, nil
}

func loadAuthConfig(environment string) (AuthConfig, error) {
	sessionTTL, err := parseEnvDuration("AUTH_SESSION_TTL", 24*time.Hour)
	if err != nil {
		return AuthConfig{}, err
	}
	rememberTTL, err := parseEnvDuration("AUTH_REMEMBER_SESSION_TTL", 30*24*time.Hour)
	if err != nil {
		return AuthConfig{}, err
	}
	registerWindow, err := parseEnvDuration("AUTH_REGISTER_RATE_WINDOW", time.Hour)
	if err != nil {
		return AuthConfig{}, err
	}
	loginWindow, err := parseEnvDuration("AUTH_LOGIN_RATE_WINDOW", 5*time.Minute)
	if err != nil {
		return AuthConfig{}, err
	}
	chatWindow, err := parseEnvDuration("AUTH_CHAT_RATE_WINDOW", time.Minute)
	if err != nil {
		return AuthConfig{}, err
	}
	roomTTL, err := parseEnvDuration("AUTH_ROOM_WAITING_TTL", 2*time.Hour)
	if err != nil {
		return AuthConfig{}, err
	}
	registerLimit, err := parseEnvInt64("AUTH_REGISTER_RATE_LIMIT", 5)
	if err != nil {
		return AuthConfig{}, err
	}
	loginLimit, err := parseEnvInt64("AUTH_LOGIN_RATE_LIMIT", 10)
	if err != nil {
		return AuthConfig{}, err
	}
	chatLimit, err := parseEnvInt64("AUTH_CHAT_RATE_LIMIT", 10)
	if err != nil {
		return AuthConfig{}, err
	}
	chatMaxLength, err := parseEnvInt("AUTH_CHAT_MAX_LENGTH", 4000)
	if err != nil {
		return AuthConfig{}, err
	}
	secureCookie, err := parseEnvBool(
		"AUTH_COOKIE_SECURE",
		environment != "development" && environment != "test",
	)
	if err != nil {
		return AuthConfig{}, err
	}

	hmacKey := envOrDefault(
		"AUTH_IDENTITY_HMAC_KEY",
		"dev-only-change-me-identity-hmac-key",
	)
	origins := splitNonEmpty(envOrDefault(
		"AUTH_ALLOWED_ORIGINS",
		"http://127.0.0.1:5173,http://localhost:5173",
	))

	cfg := AuthConfig{
		IdentityHMACKey:    hmacKey,
		CookieName:         envOrDefault("AUTH_SESSION_COOKIE_NAME", "mv_session"),
		SecureCookie:       secureCookie,
		SessionTTL:         sessionTTL,
		RememberSessionTTL: rememberTTL,
		AllowedOrigins:     origins,
		RegisterRateLimit:  registerLimit,
		RegisterRateWindow: registerWindow,
		LoginRateLimit:     loginLimit,
		LoginRateWindow:    loginWindow,
		ChatRateLimit:      chatLimit,
		ChatRateWindow:     chatWindow,
		ChatMaxLength:      chatMaxLength,
		RoomWaitingTTL:     roomTTL,
	}
	if err := cfg.Validate(environment); err != nil {
		return AuthConfig{}, fmt.Errorf("validate AUTH configuration: %w", err)
	}

	return cfg, nil
}

func loadSecretConfig(environment string) SecretConfig {
	_ = environment
	return SecretConfig{
		Key: envOrDefault(
			"DATA_SECRET_KEY",
			"dev-only-change-me-data-secret-encryption-key",
		),
		KeyID: envOrDefault("DATA_SECRET_KEY_ID", "local-v1"),
	}
}

func (c SecretConfig) Validate(environment string) error {
	if c.Key == "" || c.KeyID == "" {
		return fmt.Errorf("data secret key and key id are required")
	}
	if environment == "production" &&
		(len(c.Key) < 32 || strings.Contains(c.Key, "change-me")) {
		return fmt.Errorf("production data secret key is not configured securely")
	}
	return nil
}

func (c AuthConfig) Validate(environment string) error {
	if c.CookieName == "" {
		return fmt.Errorf("session cookie name is required")
	}
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("at least one allowed origin is required")
	}
	if c.SessionTTL <= 0 || c.RememberSessionTTL < c.SessionTTL {
		return fmt.Errorf("session TTL values are invalid")
	}
	if c.RegisterRateLimit <= 0 || c.LoginRateLimit <= 0 || c.ChatRateLimit <= 0 {
		return fmt.Errorf("rate limits must be positive")
	}
	if c.RegisterRateWindow <= 0 || c.LoginRateWindow <= 0 || c.ChatRateWindow <= 0 {
		return fmt.Errorf("rate limit windows must be positive")
	}
	if c.ChatMaxLength < 1 || c.ChatMaxLength > 10000 {
		return fmt.Errorf("chat max length must be between 1 and 10000")
	}
	if c.RoomWaitingTTL <= 0 {
		return fmt.Errorf("room waiting TTL must be positive")
	}
	if environment == "production" {
		if len(c.IdentityHMACKey) < 32 || strings.Contains(c.IdentityHMACKey, "change-me") {
			return fmt.Errorf("production identity HMAC key is not configured securely")
		}
	}
	return nil
}

func (c MySQLConfig) Validate() error {
	if c.DSN != "" {
		if c.MaxOpenConns <= 0 {
			return fmt.Errorf("max open connections must be positive")
		}
		if c.MaxIdleConns < 0 || c.MaxIdleConns > c.MaxOpenConns {
			return fmt.Errorf("max idle connections must be between 0 and max open connections")
		}
		return nil
	}

	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.MaxOpenConns <= 0 {
		return fmt.Errorf("max open connections must be positive")
	}
	if c.MaxIdleConns < 0 || c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("max idle connections must be between 0 and max open connections")
	}
	if c.ConnMaxLifetime <= 0 || c.ConnMaxIdleTime <= 0 || c.PingTimeout <= 0 {
		return fmt.Errorf("connection timeouts must be positive")
	}

	return nil
}

func (c RedisConfig) Validate() error {
	if c.DSNConfigured() {
		if c.PoolSize <= 0 {
			return fmt.Errorf("pool size must be positive")
		}
		return nil
	}

	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	if c.DB < 0 {
		return fmt.Errorf("database number must not be negative")
	}
	if c.PoolSize <= 0 {
		return fmt.Errorf("pool size must be positive")
	}
	if c.DialTimeout <= 0 || c.IOTimeout <= 0 || c.PingTimeout <= 0 {
		return fmt.Errorf("connection timeouts must be positive")
	}

	return nil
}

func (c RedisConfig) DSNConfigured() bool {
	return c.URL != ""
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func parseEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func parseEnvInt64(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func parseEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func parseEnvBool(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func validLogLevel(value string) bool {
	switch value {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

func splitNonEmpty(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func ParsePort(value string) (int, error) {
	port, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse port: %w", err)
	}

	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port out of range: %d", port)
	}

	return port, nil
}
