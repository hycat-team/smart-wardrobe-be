package caching

import (
	"testing"
	"time"

	"smart-wardrobe-be/shared/config"
)

func tlsTestConfig() config.Config {
	var cfg config.Config
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	return cfg
}

// TestRedisTLSOffUsesPlaintext verifies the default self-hosted profile uses
// no TLS (T067, FR-004).
func TestRedisTLSOffUsesPlaintext(t *testing.T) {
	cfg := tlsTestConfig()
	cfg.Redis.TLSEnabled = false

	opts := redisOptions(&cfg, 5*time.Second)
	if opts.TLSConfig != nil {
		t.Fatal("expected nil TLSConfig when redis.tls_enabled=false")
	}
	if opts.Addr != "localhost:6379" {
		t.Fatalf("unexpected addr %q", opts.Addr)
	}
}

// TestRedisTLSOnEnablesTLS verifies managed Redis Cloud uses TLS (T067).
func TestRedisTLSOnEnablesTLS(t *testing.T) {
	cfg := tlsTestConfig()
	cfg.Redis.TLSEnabled = true

	opts := redisOptions(&cfg, 5*time.Second)
	if opts.TLSConfig == nil {
		t.Fatal("expected non-nil TLSConfig when redis.tls_enabled=true")
	}
}
