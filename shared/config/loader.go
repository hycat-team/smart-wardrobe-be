package config

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// ManagedEnvs lists Server.Env values that run on managed platforms, where
// managed-only requirements (migrate-on-start, ingress URLs) are enforced.
var ManagedEnvs = []string{"managed", "managed-drill"}

func LoadConfig() *Config {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	return cfg
}

func loadConfig() (*Config, error) {
	configPath := envOrDefault("CONFIG_FILE", filepath.Join("configs", "config.yaml"))
	if err := loadLocalDotEnv(configPath); err != nil {
		return nil, err
	}

	rawBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", configPath, err)
	}

	expanded := os.ExpandEnv(string(rawBytes))

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(expanded)); err != nil {
		return nil, fmt.Errorf("read config file %q: %w", configPath, err)
	}

	var cfg Config
	if err := v.UnmarshalExact(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file %q: %w", configPath, err)
	}
	applyEnvOverrides(&cfg)
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyEnvOverrides layers environment on top of YAML (FR-005).
// Precedence per field: explicit env var > YAML > built-in default.
// Booleans are only overridden when the env var is present AND parses;
// an invalid boolean value is a hard error (fail fast, no silent default).
func applyEnvOverrides(cfg *Config) {
	// FR-049: platform port wins over everything, then SERVER_PORT, then YAML.
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		cfg.Server.Port = port
	} else if port := strings.TrimSpace(os.Getenv("SERVER_PORT")); port != "" {
		cfg.Server.Port = port
	}

	// FR-044: APP_MODE http|worker|all (default all).
	cfg.App.Mode = strings.ToLower(strings.TrimSpace(envOrDefault("APP_MODE", cfg.App.Mode)))
	if cfg.App.Mode == "" {
		cfg.App.Mode = "all"
	}

	// Profile selector: local / vps-legacy / vps / managed / managed-drill
	// (Phase 2 checkpoint) without multiplying YAML files.
	if v := strings.TrimSpace(os.Getenv("SERVER_ENV")); v != "" {
		cfg.Server.Env = v
	}

	overrideBool("ELASTICSEARCH_ENABLED", &cfg.Elasticsearch.Enabled)
	overrideBool("DATABASE_MIGRATE_ON_START", &cfg.Database.MigrateOnStart)
	overrideBool("DATABASE_MIGRATION_ALLOW_OUT_OF_ORDER", &cfg.Database.MigrationAllowOutOfOrder)
	overrideBool("REDIS_TLS_ENABLED", &cfg.Redis.TLSEnabled)
	overrideBool("RABBITMQ_TLS_ENABLED", &cfg.RabbitMQ.TLSEnabled)
	overrideBool("RABBITMQ_MANAGEMENT_TLS_ENABLED", &cfg.RabbitMQ.ManagementTLSEnabled)
	overrideBool("OUTBOX_LISTENER_ENABLED", &cfg.Outbox.ListenerEnabled)

	// QUERY_EXEC_MODE is resolved by the DB layer, which forces simple_protocol
	// when connecting through the Supavisor transaction pooler (FR-046) and
	// otherwise uses pgx's default (prepared statements). An explicit value here
	// only overrides the non-pooler case.
	if v := strings.TrimSpace(os.Getenv("DATABASE_QUERY_EXEC_MODE")); v != "" {
		cfg.Database.QueryExecMode = v
	}
	// ssl_mode/listener_ssl_mode come from YAML ${...}; default to disable when
	// unset so the DSN stays valid (FR-005).
	if v := strings.TrimSpace(os.Getenv("DB_SSLMODE")); v != "" {
		cfg.Database.SslMode = v
	}
	if cfg.Database.SslMode == "" {
		cfg.Database.SslMode = "disable"
	}
	if v := strings.TrimSpace(os.Getenv("DB_LISTENER_SSLMODE")); v != "" {
		cfg.Database.ListenerSslMode = v
	}
	if cfg.Database.ListenerSslMode == "" {
		cfg.Database.ListenerSslMode = "disable"
	}
	// DB_PORT / DB_LISTENER_PORT let managed point runtime at the pooler (6543)
	// while migrations/LISTEN/advisory locks keep the direct 5432.
	if v := strings.TrimSpace(os.Getenv("DB_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.Database.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("DB_LISTENER_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.Database.ListenerPort = p
		}
	}
	// Managed Redis Cloud / CloudAMQP expose non-default ports (FR-005).
	if v := strings.TrimSpace(os.Getenv("REDIS_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.Redis.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("RABBITMQ_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			cfg.RabbitMQ.Port = p
		}
	}
	if v := strings.TrimSpace(os.Getenv("RABBITMQ_VHOST")); v != "" {
		cfg.RabbitMQ.Vhost = v
	}
	if v := strings.TrimSpace(os.Getenv("DATABASE_DRILL_SCHEMA")); v != "" {
		cfg.Database.DrillSchema = v
	}
	if v := strings.TrimSpace(os.Getenv("DATABASE_MIGRATION_TABLE")); v != "" {
		cfg.Database.MigrationTable = v
	}
	if v := strings.TrimSpace(os.Getenv("JWT_ALGORITHM")); v != "" {
		cfg.Jwt.Algorithm = strings.ToUpper(v)
	}
	if v := strings.TrimSpace(os.Getenv("JWT_PRIVATE_KEY")); v != "" {
		cfg.Jwt.PrivateKey = v
	}
	if v := strings.TrimSpace(os.Getenv("JWT_PUBLIC_KEY")); v != "" {
		cfg.Jwt.PublicKey = v
	}
}

// overrideBool applies an env var to *dst only when present. Invalid values
// are fatal — a misconfigured toggle must never silently fall back.
func overrideBool(envKey string, dst *bool) {
	raw, ok := os.LookupEnv(envKey)
	if !ok || strings.TrimSpace(raw) == "" {
		return
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		log.Fatalf("Invalid boolean for %s=%q: %v", envKey, raw, err)
	}
	*dst = parsed
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func loadLocalDotEnv(configPath string) error {
	// Search order: CWD, config dir, repo root (config dir's parent — needed when
	// the binary runs from services/<svc> with CONFIG_FILE=configs/config.yaml),
	// then CWD's parent.
	paths := []string{
		".env",
		filepath.Join(filepath.Dir(configPath), ".env"),
		filepath.Join(filepath.Dir(configPath), "..", ".env"),
		"../.env",
	}
	for _, path := range paths {
		err := godotenv.Load(path)
		if err == nil {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("load environment file %q: %w", path, err)
		}
	}
	return nil
}

// IsManagedEnv reports whether Server.Env targets managed platforms.
func IsManagedEnv(env string) bool {
	for _, m := range ManagedEnvs {
		if strings.EqualFold(strings.TrimSpace(env), m) {
			return true
		}
	}
	return false
}
