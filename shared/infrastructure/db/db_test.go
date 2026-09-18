package db

import (
	"strings"
	"testing"

	"smart-wardrobe-be/shared/config"
)

// DirectDSN must prefer the listener (direct) fields and fall back to runtime
// fields only when they are unset.
func TestDirectDSN(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.Host = "pooler.supabase.com"
	cfg.Database.Port = 6543
	cfg.Database.User = "u"
	cfg.Database.Password = "p"
	cfg.Database.DbName = "d"
	cfg.Database.SslMode = "require"

	fallback := DirectDSN(cfg)
	if !strings.Contains(fallback, "host=pooler.supabase.com") || !strings.Contains(fallback, "port=6543") {
		t.Fatalf("fallback DirectDSN = %q", fallback)
	}

	cfg.Database.ListenerHost = "db.ref.supabase.co"
	cfg.Database.ListenerPort = 5432
	cfg.Database.ListenerSslMode = "require"

	direct := DirectDSN(cfg)
	if !strings.Contains(direct, "host=db.ref.supabase.co") || !strings.Contains(direct, "port=5432") {
		t.Fatalf("listener DirectDSN = %q", direct)
	}
}

// queryExecMode must force simple_protocol only for the Supavisor transaction
// pooler (FR-046); direct connections keep pgx's prepared-statement default.
func TestQueryExecMode(t *testing.T) {
	cases := []struct {
		name string
		port int
		mode string
		want string
	}{
		{"pooler forces simple_protocol", supavisorPoolerPort, "", "simple_protocol"},
		{"pooler overrides explicit mode", supavisorPoolerPort, "exec", "simple_protocol"},
		{"direct uses pgx default (empty)", 5432, "", ""},
		{"direct honours explicit override", 5432, "exec", "exec"},
	}
	for _, c := range cases {
		cfg := &config.Config{}
		cfg.Database.Port = c.port
		cfg.Database.QueryExecMode = c.mode
		if got := queryExecMode(cfg); got != c.want {
			t.Errorf("%s: queryExecMode = %q, want %q", c.name, got, c.want)
		}
	}
}
