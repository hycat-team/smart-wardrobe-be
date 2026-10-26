package config

import (
	"sort"
	"strings"
	"testing"
)

// clearAITrialEnv keeps the tests hermetic: the developer's shell may export
// any of the trial variables, and an empty value means "leave the loaded value
// alone" everywhere they are read.
//
// The ten provider names are cleared as well: the provider-name tests mutate one
// block at a time, so a provider left over from the developer's .env would make
// the "empty is allowed" case meaningless.
func clearAITrialEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"AI_TEXT_MAX_CONCURRENCY",
		"AI_TEXT_PRIMARY_TIMEOUT_SECONDS",
		"AI_TEXT_FALLBACK_TIMEOUT_SECONDS",
		"AI_CHAT_CHUNK_DEADLINE_SECONDS",
		"AI_OPENAI_REASONING_EFFORT",
		"AI_OPENAI_TRIAL_EXPECTED_HOSTS",
		"VISION_PRIMARY_PROVIDER",
		"VISION_FALLBACK_PROVIDER",
		"EMBEDDING_PRIMARY_PROVIDER",
		"EMBEDDING_FALLBACK_PROVIDER",
		"CHAT_TEXT_PRIMARY_PROVIDER",
		"CHAT_TEXT_FALLBACK_PROVIDER",
		"RECOMMENDATION_TEXT_PRIMARY_PROVIDER",
		"RECOMMENDATION_TEXT_FALLBACK_PROVIDER",
		"FREE_TEXT_PRIMARY_PROVIDER",
		"FREE_TEXT_FALLBACK_PROVIDER",
	} {
		t.Setenv(key, "")
	}
}

func loadAITrialConfig(t *testing.T, path string) *Config {
	t.Helper()
	baseEnv(t)
	clearAITrialEnv(t)
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("SERVER_ENV", "local")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("config %s must load: %v", path, err)
	}
	return cfg
}

func TestAITrialDefaultsFromShippedConfigs(t *testing.T) {
	local, prod := repoConfigs(t)

	for _, path := range []string{local, prod} {
		t.Run(path, func(t *testing.T) {
			cfg := loadAITrialConfig(t, path)

			if cfg.AI.TextMaxConcurrency != 2 {
				t.Errorf("text_max_concurrency = %d, want 2", cfg.AI.TextMaxConcurrency)
			}
			if cfg.AI.TextPrimaryTimeoutSeconds != 30 || cfg.AI.TextFallbackTimeoutSeconds != 30 {
				t.Errorf("text timeouts = %d/%d, want 30/30", cfg.AI.TextPrimaryTimeoutSeconds, cfg.AI.TextFallbackTimeoutSeconds)
			}
			if cfg.Server.SSE.AIChatChunkDeadlineSeconds != 30 {
				t.Errorf("ai_chat_chunk_deadline_seconds = %d, want 30", cfg.Server.SSE.AIChatChunkDeadlineSeconds)
			}
			if len(cfg.AI.OpenAITrialExpectedHosts) != 0 {
				t.Errorf("openai_trial_expected_hosts = %v, want empty", cfg.AI.OpenAITrialExpectedHosts)
			}
			for name, provider := range map[string]APIProviderConfig{
				"chat_text_primary":           cfg.AI.ChatTextPrimary,
				"recommendation_text_primary": cfg.AI.RecommendationTextPrimary,
				"free_text_primary":           cfg.AI.FreeTextPrimary,
			} {
				if provider.ReasoningEffort != "" {
					t.Errorf("%s.reasoning_effort = %q, want empty", name, provider.ReasoningEffort)
				}
			}
		})
	}
}

func TestAITrialEnvOverrides(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	clearAITrialEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_ENV", "local")
	t.Setenv("AI_TEXT_MAX_CONCURRENCY", "3")
	t.Setenv("AI_TEXT_PRIMARY_TIMEOUT_SECONDS", "90")
	t.Setenv("AI_TEXT_FALLBACK_TIMEOUT_SECONDS", "30")
	t.Setenv("AI_CHAT_CHUNK_DEADLINE_SECONDS", "90")
	t.Setenv("AI_OPENAI_REASONING_EFFORT", "none")
	t.Setenv("AI_OPENAI_TRIAL_EXPECTED_HOSTS", " ollama.example.com, edge.example.com ,,")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("trial env must load: %v", err)
	}

	if cfg.AI.TextMaxConcurrency != 3 {
		t.Errorf("text_max_concurrency = %d, want 3", cfg.AI.TextMaxConcurrency)
	}
	if cfg.AI.TextPrimaryTimeoutSeconds != 90 || cfg.AI.TextFallbackTimeoutSeconds != 30 {
		t.Errorf("text timeouts = %d/%d, want 90/30", cfg.AI.TextPrimaryTimeoutSeconds, cfg.AI.TextFallbackTimeoutSeconds)
	}
	if cfg.Server.SSE.AIChatChunkDeadlineSeconds != 90 {
		t.Errorf("ai_chat_chunk_deadline_seconds = %d, want 90 (AI_CHAT_CHUNK_DEADLINE_SECONDS)", cfg.Server.SSE.AIChatChunkDeadlineSeconds)
	}
	for name, provider := range map[string]APIProviderConfig{
		"chat_text_primary":           cfg.AI.ChatTextPrimary,
		"recommendation_text_primary": cfg.AI.RecommendationTextPrimary,
		"free_text_primary":           cfg.AI.FreeTextPrimary,
	} {
		if provider.ReasoningEffort != "none" {
			t.Errorf("%s.reasoning_effort = %q, want none", name, provider.ReasoningEffort)
		}
	}
	want := []string{"ollama.example.com", "edge.example.com"}
	if len(cfg.AI.OpenAITrialExpectedHosts) != len(want) {
		t.Fatalf("openai_trial_expected_hosts = %v, want %v", cfg.AI.OpenAITrialExpectedHosts, want)
	}
	for i := range want {
		if cfg.AI.OpenAITrialExpectedHosts[i] != want[i] {
			t.Fatalf("openai_trial_expected_hosts = %v, want %v", cfg.AI.OpenAITrialExpectedHosts, want)
		}
	}
}

func TestAITrialValidationRules(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		name    string
		mutate  func(*Config)
		wantSub string
	}{
		{
			name:    "concurrency below one",
			mutate:  func(c *Config) { c.AI.TextMaxConcurrency = 0 },
			wantSub: "text_max_concurrency must be at least 1",
		},
		{
			name:    "primary timeout not positive",
			mutate:  func(c *Config) { c.AI.TextPrimaryTimeoutSeconds = 0 },
			wantSub: "text attempt timeouts must be greater than 0",
		},
		{
			name:    "fallback timeout not positive",
			mutate:  func(c *Config) { c.AI.TextFallbackTimeoutSeconds = 0 },
			wantSub: "text attempt timeouts must be greater than 0",
		},
		{
			name: "timeout sum above write budget",
			mutate: func(c *Config) {
				c.AI.TextPrimaryTimeoutSeconds = 91
				c.AI.TextFallbackTimeoutSeconds = 30
				c.Server.SSE.AIChatChunkDeadlineSeconds = 91
			},
			wantSub: "must not exceed 120 seconds",
		},
		{
			name: "primary deadline above sse chunk deadline",
			mutate: func(c *Config) {
				c.AI.TextPrimaryTimeoutSeconds = 31
				c.Server.SSE.AIChatChunkDeadlineSeconds = 30
			},
			wantSub: "must not exceed server.sse.ai_chat_chunk_deadline_seconds",
		},
		{
			name:    "sse chunk deadline not positive",
			mutate:  func(c *Config) { c.Server.SSE.AIChatChunkDeadlineSeconds = 0 },
			wantSub: "ai_chat_chunk_deadline_seconds must be greater than 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadAITrialConfig(t, local)
			tc.mutate(cfg)

			err := validateConfig(cfg)
			if err == nil {
				t.Fatal("expected the configuration to be rejected")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("error %q must contain %q", err, tc.wantSub)
			}
		})
	}
}

func TestAITrialOpenAIEndpointValidation(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		name     string
		endpoint string
		wantErr  bool
	}{
		{name: "https full chat path", endpoint: "https://ollama.example.com/v1/chat/completions"},
		{name: "loopback ipv4 over http", endpoint: "http://127.0.0.1:11434/v1/chat/completions"},
		{name: "loopback localhost over http", endpoint: "http://localhost:11434/v1/chat/completions"},
		{name: "plain http is not loopback", endpoint: "http://ollama.example.com/v1/chat/completions", wantErr: true},
		{name: "missing full path", endpoint: "https://ollama.example.com/v1/chat", wantErr: true},
		{name: "host only", endpoint: "https://ollama.example.com", wantErr: true},
		{name: "empty", endpoint: "", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadAITrialConfig(t, local)
			cfg.AI.ChatTextPrimary = APIProviderConfig{
				Provider: "openai",
				Endpoint: tc.endpoint,
				Model:    "qwen3:4b",
			}

			err := validateConfig(cfg)
			if tc.wantErr && err == nil {
				t.Fatal("expected the endpoint to be rejected")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("endpoint %q must be accepted: %v", tc.endpoint, err)
			}
		})
	}
}

func TestAITrialReasoningEffortValidation(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		value   string
		wantErr bool
	}{
		{value: ""},
		{value: "none"},
		{value: "low"},
		{value: "medium"},
		{value: "high"},
		{value: "HIGH"},
		{value: "minimal", wantErr: true},
		{value: "none-ish", wantErr: true},
	}

	for _, tc := range cases {
		t.Run("reasoning_effort="+tc.value, func(t *testing.T) {
			cfg := loadAITrialConfig(t, local)
			cfg.AI.ChatTextPrimary.ReasoningEffort = tc.value

			err := validateConfig(cfg)
			if tc.wantErr && err == nil {
				t.Fatal("expected the reasoning effort to be rejected")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("reasoning_effort %q must be accepted: %v", tc.value, err)
			}
		})
	}
}

func TestAIChatChunkDeadlineEnvOverride(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	clearAITrialEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_ENV", "local")
	t.Setenv("AI_CHAT_CHUNK_DEADLINE_SECONDS", "90")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("must load: %v", err)
	}
	if cfg.Server.SSE.AIChatChunkDeadlineSeconds != 90 {
		t.Fatalf("ai_chat_chunk_deadline_seconds = %d, want 90", cfg.Server.SSE.AIChatChunkDeadlineSeconds)
	}
}

// allAPIProviderBlocks returns every block the startup provider-name check must
// cover, keyed by the name used in the error message. The provider tests mutate
// one block at a time so the text-endpoint rule cannot mask a name failure.
func allAPIProviderBlocks(cfg *Config) map[string]*APIProviderConfig {
	return map[string]*APIProviderConfig{
		"vision_primary":               &cfg.AI.VisionPrimary,
		"vision_fallback":              &cfg.AI.VisionFallback,
		"embedding_primary":            &cfg.AI.EmbeddingPrimary,
		"embedding_fallback":           &cfg.AI.EmbeddingFallback,
		"chat_text_primary":            &cfg.AI.ChatTextPrimary,
		"chat_text_fallback":           &cfg.AI.ChatTextFallback,
		"recommendation_text_primary":  &cfg.AI.RecommendationTextPrimary,
		"recommendation_text_fallback": &cfg.AI.RecommendationTextFallback,
		"free_text_primary":            &cfg.AI.FreeTextPrimary,
		"free_text_fallback":           &cfg.AI.FreeTextFallback,
	}
}

func TestAIProviderNameValidation(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "empty is allowed", value: ""},
		{name: "openai", value: "openai"},
		{name: "google", value: "google"},
		{name: "unknown provider", value: "anthropic", wantErr: true},
		{name: "gemini is not a provider name", value: "gemini", wantErr: true},
		{name: "wrong case", value: "OpenAI", wantErr: true},
	}

	blockNames := sortedAPIProviderBlockNames()
	for _, tc := range cases {
		for _, blockName := range blockNames {
			t.Run(tc.name+"/"+blockName, func(t *testing.T) {
				cfg := loadAITrialConfig(t, local)
				block := allAPIProviderBlocks(cfg)[blockName]
				block.Provider = tc.value
				if tc.value == "openai" && strings.Contains(blockName, "_text_") {
					// The openai text blocks also require a full chat endpoint;
					// supply a valid one so this test only exercises the name.
					block.Endpoint = "https://ollama.example.com/v1/chat/completions"
				}

				err := validateConfig(cfg)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("provider %q on %s must be rejected", tc.value, blockName)
					}
					if !strings.Contains(err.Error(), "provider") || !strings.Contains(err.Error(), blockName) {
						t.Fatalf("error %q must name the invalid provider on %s", err, blockName)
					}
					return
				}
				if err != nil {
					t.Fatalf("provider %q on %s must be accepted: %v", tc.value, blockName, err)
				}
			})
		}
	}
}

func sortedAPIProviderBlockNames() []string {
	names := make([]string, 0, 10)
	for name := range allAPIProviderBlocks(&Config{}) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func TestAITrialExpectedHostValidation(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		name    string
		hosts   []string
		wantErr bool
	}{
		{name: "empty list is allowed"},
		{name: "single bare host", hosts: []string{"ollama.example.com"}},
		{name: "several bare hosts", hosts: []string{"ollama.example.com", "edge.example.com"}},
		{name: "scheme is rejected", hosts: []string{"https://ollama.example.com"}, wantErr: true},
		{name: "path is rejected", hosts: []string{"ollama.example.com/v1"}, wantErr: true},
		{
			name:    "scheme and path are rejected",
			hosts:   []string{"https://ollama.example.com/v1/chat/completions"},
			wantErr: true,
		},
		{
			name:    "one bad entry among valid hosts",
			hosts:   []string{"ollama.example.com", "http://edge.example.com"},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := loadAITrialConfig(t, local)
			cfg.AI.OpenAITrialExpectedHosts = tc.hosts

			err := validateConfig(cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected hosts %v to be rejected", tc.hosts)
				}
				if !strings.Contains(err.Error(), "openai_trial_expected_hosts") {
					t.Fatalf("error %q must name openai_trial_expected_hosts", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected hosts %v must be accepted: %v", tc.hosts, err)
			}
		})
	}
}

// TestAITrialLoadRejectsBadProviderAndHosts proves the checks run on the real
// load path (env -> YAML -> validateConfig), not only when a test calls
// validateConfig directly.
func TestAITrialLoadRejectsBadProviderAndHosts(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "unknown provider", key: "CHAT_TEXT_PRIMARY_PROVIDER", value: "anthropic"},
		{name: "expected host with scheme", key: "AI_OPENAI_TRIAL_EXPECTED_HOSTS", value: "https://ollama.example.com"},
		{name: "expected host with path", key: "AI_OPENAI_TRIAL_EXPECTED_HOSTS", value: "ollama.example.com/v1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseEnv(t)
			clearAITrialEnv(t)
			t.Setenv("CONFIG_FILE", local)
			t.Setenv("SERVER_ENV", "local")
			t.Setenv(tc.key, tc.value)

			if _, err := loadConfig(); err == nil {
				t.Fatalf("%s=%q must be rejected at load time", tc.key, tc.value)
			}
		})
	}
}
