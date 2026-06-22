package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func prepareConfigTest(t *testing.T, mutate func(string) string) string {
	t.Helper()
	source, err := os.ReadFile("config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	content := string(source)
	if mutate != nil {
		content = mutate(content)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("ENV", "development")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_PASSWORD", "test-password")
	t.Setenv("DB_NAME", "test-db")
	return path
}

func TestLoadConfigBindsNestedYAMLAndSecrets(t *testing.T) {
	prepareConfigTest(t, nil)
	t.Setenv("VISION_PRIMARY_API_KEY", "vision-secret")
	t.Setenv("VISION_PRIMARY_PROVIDER", "test-provider")
	t.Setenv("VISION_PRIMARY_ENDPOINT", "https://example.test")
	t.Setenv("VISION_PRIMARY_MODEL", "test-model")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Wardrobe.RetryDelay2Seconds != 300 {
		t.Fatalf("unexpected nested YAML value: %d", cfg.Wardrobe.RetryDelay2Seconds)
	}
	if cfg.Jwt.Secret != "test-secret" || cfg.AI.VisionPrimary.ApiKey != "vision-secret" {
		t.Fatal("environment secrets were not loaded")
	}
	if cfg.AI.VisionPrimary.Provider != "test-provider" || cfg.AI.VisionPrimary.Endpoint != "https://example.test" || cfg.AI.VisionPrimary.Model != "test-model" {
		t.Fatal("private AI provider configuration was not loaded")
	}
}

func TestLoadConfigRejectsUnknownKey(t *testing.T) {
	prepareConfigTest(t, func(value string) string { return value + "\nunknown_section: true\n" })
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "unknown_section") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestLoadConfigRejectsSecretInYAML(t *testing.T) {
	prepareConfigTest(t, func(value string) string {
		return strings.Replace(value, "    host: localhost", "    host: localhost\n    password: \"forbidden\"", 1)
	})
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "password") {
		t.Fatalf("expected secret key error, got %v", err)
	}
}

func TestLoadConfigUsesCustomPathAndIgnoresNonSecretEnvironment(t *testing.T) {
	path := prepareConfigTest(t, nil)
	t.Setenv("SERVER_PORT", "9090")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(os.Getenv("CONFIG_FILE")) != filepath.Clean(path) || cfg.Server.Port != "8080" {
		t.Fatal("custom path was ignored or non-secret environment unexpectedly overrode YAML")
	}
}

func TestLoadConfigValidatesRequiredSecret(t *testing.T) {
	prepareConfigTest(t, nil)
	t.Setenv("JWT_SECRET", "")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "JWT secret") {
		t.Fatalf("expected required secret error, got %v", err)
	}
}

func TestLoadConfigRejectsProductionWildcardOrigin(t *testing.T) {
	prepareConfigTest(t, func(value string) string {
		value = strings.Replace(value, "env: development", "env: production", 1)
		lines := strings.Split(value, "\n")
		for i, line := range lines {
			if strings.Contains(line, "front_end_origin:") {
				lines[i] = "    front_end_origin: '*'"
				break
			}
		}
		return strings.Join(lines, "\n")
	})
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "wildcard") {
		t.Fatalf("expected production validation error, got %v", err)
	}
}

func TestProductionConfigIsValid(t *testing.T) {
	path, err := filepath.Abs("config.production.yaml")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DB_USER", "test-user")
	t.Setenv("DB_PASSWORD", "test-password")
	t.Setenv("DB_NAME", "test-db")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Env != "production" || cfg.Database.Host != "postgres" {
		t.Fatal("production config was not loaded")
	}
}
