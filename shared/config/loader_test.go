package config

import (
	"os"
	"path/filepath"
	"testing"
)

func repoConfigs(t *testing.T) (local, prod string) {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, "configs", "config.yaml"), filepath.Join(root, "configs", "config.production.yaml")
}

// baseEnv satisfies every ${} placeholder the YAMLs reference so tests are
// hermetic regardless of the developer's real .env.
func baseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("JWT_ALGORITHM", "HS256")
	t.Setenv("SWAGGER_ACCESS_CODE", "test-code")
	t.Setenv("FRONT_END_ORIGIN", "http://localhost:3000")
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_USER", "admin")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "smart_wardrobe_db")
	t.Setenv("DB_LISTENER_HOST", "localhost")
	t.Setenv("REDIS_HOST", "localhost")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("RABBITMQ_HOST", "localhost")
	t.Setenv("RABBITMQ_USER", "guest")
	t.Setenv("RABBITMQ_PASSWORD", "guest")
	t.Setenv("ELASTICSEARCH_ADDRESSES", "http://localhost:9200")
	t.Setenv("ELASTICSEARCH_USER", "")
	t.Setenv("ELASTICSEARCH_PASSWORD", "")
}

func TestLoadLocalProfile(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_ENV", "local")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("local profile must load: %v", err)
	}
	if cfg.App.Mode != "all" {
		t.Errorf("App.Mode = %q, want all", cfg.App.Mode)
	}
	if cfg.Server.SSE.AIChatChunkDeadlineSeconds != 30 || cfg.Server.SSE.HeartbeatSeconds != 15 {
		t.Errorf("SSE = %+v, want ai_chat 30s + heartbeat 15s", cfg.Server.SSE)
	}
	if cfg.Elasticsearch.Enabled {
		t.Error("Elasticsearch must default to disabled")
	}
	if cfg.Database.QueryExecMode != "" {
		t.Errorf("QueryExecMode = %q, want empty (resolved pooler-only in the db layer)", cfg.Database.QueryExecMode)
	}
	if cfg.Database.SslMode != "disable" || cfg.Database.ListenerSslMode != "disable" {
		t.Errorf("ssl modes = %q/%q, want disable/disable by default", cfg.Database.SslMode, cfg.Database.ListenerSslMode)
	}
	if !cfg.Database.MigrationAllowOutOfOrder {
		t.Error("MigrationAllowOutOfOrder must default to true (FR-035)")
	}
}

func TestPortPrecedence(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_PORT", "9001")
	t.Setenv("PORT", "9002")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("must load: %v", err)
	}
	if cfg.Server.Port != "9002" {
		t.Errorf("Server.Port = %q, want 9002 (PORT wins)", cfg.Server.Port)
	}
}

func TestManagedRequiresIngressAndMigrateOnStart(t *testing.T) {
	_, prod := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", prod)
	t.Setenv("SERVER_ENV", "managed")

	if _, err := loadConfig(); err == nil {
		t.Fatal("managed without ingress URLs must fail")
	}

	t.Setenv("INGRESS_ACCOUNT_URL", "https://account.run.app")
	t.Setenv("INGRESS_COMMERCE_URL", "https://commerce.run.app")
	t.Setenv("INGRESS_AI_URL", "https://ai.run.app")
	t.Setenv("INGRESS_TOKEN_AUDIENCE", "https://account.run.app")
	if _, err := loadConfig(); err != nil {
		t.Fatalf("managed with ingress must load: %v", err)
	}
}

func TestJWTAlgorithmRules(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)

	t.Setenv("JWT_ALGORITHM", "ES256")
	if _, err := loadConfig(); err == nil {
		t.Fatal("ES256 without key paths must fail")
	}

	t.Setenv("JWT_PRIVATE_KEY", "testdata/es256-private.pem")
	t.Setenv("JWT_PUBLIC_KEY", "testdata/es256-public.pem")
	if _, err := loadConfig(); err != nil {
		t.Fatalf("ES256 with key paths must load: %v", err)
	}

	t.Setenv("JWT_ALGORITHM", "RS999")
	if _, err := loadConfig(); err == nil {
		t.Fatal("unknown algorithm must fail")
	}
}

func TestDrillSchemaGuard(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)

	t.Setenv("DB_DRILL_SCHEMA", "public")
	if _, err := loadConfig(); err == nil {
		t.Fatal("non-drill schema must fail")
	}
	t.Setenv("DB_DRILL_SCHEMA", "drill_20260915 Smoke")
	if _, err := loadConfig(); err == nil {
		t.Fatal("drill schema with illegal chars must fail")
	}
	t.Setenv("DB_DRILL_SCHEMA", "drill_20260915_01")
	if _, err := loadConfig(); err != nil {
		t.Fatalf("valid drill schema must load: %v", err)
	}
}

func TestInvalidAppMode(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("APP_MODE", "sometimes")

	if _, err := loadConfig(); err == nil {
		t.Fatal("invalid APP_MODE must fail")
	}
}

func TestMain(m *testing.M) {
	// Never pick up the developer's real .env: tests set everything explicitly.
	os.Unsetenv("CONFIG_FILE")
	os.Exit(m.Run())
}
