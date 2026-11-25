package redis

import (
	"crypto/tls"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/config"
)

func newTLSConfig(cfg config.RedisConfig) *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cfg.Host,
		InsecureSkipVerify: cfg.TLSInsecure,
	}
}
