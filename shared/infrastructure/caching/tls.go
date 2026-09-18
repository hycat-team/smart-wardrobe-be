package caching

import (
	"crypto/tls"
	"fmt"
	"time"

	"smart-wardrobe-be/shared/config"

	"github.com/redis/go-redis/v9"
)

// redisOptions builds the go-redis options from configuration (T064/FR-004).
// Managed Redis Cloud requires TLS (`rediss`); self-hosted keeps it off for
// VPS compatibility. The zero TLSConfig means plaintext.
func redisOptions(cfg *config.Config, timeout time.Duration) *redis.Options {
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.Db,
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}
	if cfg.Redis.TLSEnabled {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return opts
}
