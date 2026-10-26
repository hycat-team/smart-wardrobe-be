package config

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// drillSchemaPattern confines drills to clearly-marked temporary schemas so a
// misconfigured drill can never resolve to a production schema (FR-027).
var drillSchemaPattern = regexp.MustCompile(`^drill_[a-z0-9_]{1,48}$`)

// campaignCodePattern is GONE (026 T011), and the reason is worth one line
// because the rule it enforced is NOT.
//
// The PATTERN itself survives — it is `CHECK chk_campaigns_code_format` on
// `subscription.campaigns` now, plus `400` on the admin create endpoint
// (FR-011/FR-054). What disappeared is the reason it was HERE: 025 validated the
// campaign list at STARTUP because the list was the configuration, and a bad
// campaign had to be refused before it could grant a plan to a real account.
//
// After 026 the campaign list is data read at RUN time, so there is no startup
// moment at which a campaign could be checked, and pretending otherwise would
// reintroduce the dependency the move removed. `plan_slug` naming a missing plan
// stays a runtime check for the same structural reason (FR-014).

// validAppModes mirrors FR-044.
var validAppModes = []string{"http", "worker", "all"}

func validateConfig(cfg *Config) error {
	required := map[string]string{"JWT secret": cfg.Jwt.Secret, "DB host": cfg.Database.Host, "DB user": cfg.Database.User, "DB name": cfg.Database.DbName}
	for label, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	if cfg.Database.MaxIdleConns <= 0 || cfg.Database.MaxOpenConns <= 0 || cfg.Database.ConnMaxLifetime <= 0 {
		return fmt.Errorf("database connection pool configuration is invalid")
	}
	if IsProductionEnv(cfg.Server.Env) && strings.Contains(cfg.Server.FrontEndOrigin, "*") {
		return fmt.Errorf("wildcard FRONTEND_ORIGIN is not allowed in production")
	}
	if strings.TrimSpace(cfg.Server.Port) == "" {
		return fmt.Errorf("server port is required (set PORT, SERVER_PORT, or server.port)")
	}
	if err := validateAppMode(cfg.App.Mode); err != nil {
		return err
	}
	if err := validateJWT(cfg); err != nil {
		return err
	}
	if err := validateDrillSchema(cfg.Database.DrillSchema); err != nil {
		return err
	}
	if IsManagedEnv(cfg.Server.Env) {
		if err := validateManaged(cfg); err != nil {
			return err
		}
	}
	if cfg.Startup.RetryAttempt1Seconds <= 0 || cfg.Startup.RetryAttempt2Seconds <= 0 || cfg.Startup.RetryAttempt3Seconds <= 0 {
		return fmt.Errorf("startup retry durations must be greater than 0")
	}
	if cfg.Wardrobe.RetryDelay1Seconds <= 0 || cfg.Wardrobe.RetryDelay2Seconds <= 0 || cfg.Wardrobe.RetryDelay3Seconds <= 0 {
		return fmt.Errorf("wardrobe retry delays must be greater than 0")
	}
	if cfg.Wardrobe.StaleMinutes <= 0 || cfg.Wardrobe.MaxRetryCount <= 0 || strings.TrimSpace(cfg.Wardrobe.RecoveryScanCron) == "" {
		return fmt.Errorf("wardrobe processing recovery configuration is invalid")
	}
	if cfg.Wardrobe.CategoryCacheTTLSeconds <= 0 {
		return fmt.Errorf("wardrobe category cache ttl must be greater than 0")
	}
	if cfg.AI.ChatTextTimeoutSeconds <= 0 || cfg.AI.RecommendationTextTimeoutSeconds <= 0 || cfg.AI.VisionTimeoutSeconds <= 0 || cfg.AI.EmbeddingTimeoutSeconds <= 0 {
		return fmt.Errorf("ai timeouts must be greater than 0")
	}
	if cfg.AI.FreeTextTimeoutSeconds <= 0 || cfg.AI.FreeTextRPMLimit <= 0 {
		return fmt.Errorf("free ai text limits must be greater than 0")
	}
	if strings.TrimSpace(cfg.AI.UsageReconcileCron) == "" || cfg.AI.UsageReconcileBatchSize <= 0 {
		return fmt.Errorf("ai usage reconciliation configuration is invalid")
	}
	if cfg.AI.ChatTextRPMLimit <= 0 || cfg.AI.RecommendationTextRPMLimit <= 0 || cfg.AI.VisionRPMLimit <= 0 || cfg.AI.EmbeddingRPMLimit <= 0 {
		return fmt.Errorf("ai rpm limits must be greater than 0")
	}
	if err := validateAIProviderBlocks(cfg); err != nil {
		return err
	}
	if err := validateAITrialSettings(cfg); err != nil {
		return err
	}
	if cfg.AI.ChatMaxInputCharacters <= 0 || cfg.AI.ChatHistoryMessageMaxCharacters <= 0 || cfg.AI.ChatMaxOutputTokens <= 0 || cfg.AI.SummarySourceMaxCharacters <= 0 || cfg.AI.SummaryPreviousMaxCharacters <= 0 || cfg.AI.SummaryMaxOutputTokens <= 0 || cfg.AI.RewriterPromptMaxCharacters <= 0 || cfg.AI.RewriterMaxOutputTokens <= 0 || cfg.AI.RecommendationDetailsMaxCharacters <= 0 || cfg.AI.RecommendationPromptCandidateLimit <= 0 || cfg.AI.RecommendationDescriptionMaxCharacters <= 0 || cfg.AI.RecommendationTagsLimit <= 0 || cfg.AI.RecommendationPromptMaxCharacters <= 0 || cfg.AI.RecommendationMaxOutputTokens <= 0 {
		return fmt.Errorf("ai input and output limits must be greater than 0")
	}
	if cfg.RAG.RecommendationCandidateLimit <= 0 || cfg.RAG.RecommendationMinimumCandidatePool <= 0 || cfg.RAG.RecommendationEmbeddingDimension <= 0 || cfg.RAG.RecommendationEmbeddingTimeoutSeconds <= 0 {
		return fmt.Errorf("rag recommendation retrieval configuration is invalid")
	}
	if cfg.PayOS.ExpiredMinutes <= 0 {
		return fmt.Errorf("payos expiration must be greater than 0")
	}
	if cfg.ClaimRateLimit.IPLimit <= 0 || cfg.ClaimRateLimit.UserLimit <= 0 || cfg.ClaimRateLimit.TokenLimit <= 0 || cfg.ClaimRateLimit.WindowSeconds <= 0 {
		return fmt.Errorf("claim rate limit configuration is invalid")
	}
	if cfg.RateLimit.TokenLimit <= 0 || cfg.RateLimit.TokensPerPeriod <= 0 || cfg.RateLimit.ReplenishmentSeconds <= 0 {
		// A zero token_limit means zero burst — every request would be rejected.
		return fmt.Errorf("global rate limit configuration is invalid")
	}
	if cfg.AdminGrantRateLimit.AllPerMinute <= 0 || cfg.AdminGrantRateLimit.MultiPerMinute <= 0 || cfg.AdminGrantRateLimit.WindowSeconds <= 0 {
		// A zero quota here locks administrators out of batch grant entirely.
		return fmt.Errorf("admin grant rate limit configuration is invalid")
	}
	if cfg.AdminGrantRateLimit.MultiPerMinute > cfg.AdminGrantRateLimit.AllPerMinute {
		// The stricter tier would never fire - multi requests would be counted
		// only by the looser tier, so the batch cap would be dead config.
		return fmt.Errorf("admin grant multi_per_minute must not exceed all_per_minute")
	}
	// 024 T146 (FR-074): the overview cache TTL is capped at 60 seconds, not just
	// set to it.
	//
	// FR-074 requires the grant to be visible to a customer on their next request,
	// and this TTL is how long it can take to arrive. Nothing else bounds it, so a
	// deployment could raise it and silently break a customer-facing guarantee -
	// the customer would be charged Premium and refused by the AI service until the
	// cache expired.
	//
	// Zero is allowed and means "use the built-in default", so the setting stays
	// optional; anything ABOVE the ceiling is rejected at startup rather than
	// accepted and discovered later.
	if cfg.Cache.OverviewTTLSeconds < 0 {
		return fmt.Errorf("cache overview_ttl_seconds must not be negative")
	}
	if cfg.Cache.OverviewTTLSeconds > MaxOverviewTTLSeconds {
		return fmt.Errorf("cache overview_ttl_seconds must not exceed %d seconds (FR-074): a longer cache would keep a freshly granted plan invisible to the customer", MaxOverviewTTLSeconds)
	}
	// 026 (FR-004, T011): the campaign LIST is no longer validated here, because
	// it is no longer in the configuration. `subscription.campaigns` is read at run
	// time and its shape is enforced where it is written — by `CHECK`/`PRIMARY KEY`
	// on the table and by `400`/`422` on the admin endpoints (FR-010..FR-016,
	// spec.md Câu 2).
	//
	// What is still validated at startup is the sweep CADENCE, because the cadence
	// is still configuration: it is the knob of the mechanism, not campaign data, and
	// a sweep running every ten seconds starves the grant queue it exists to catch
	// up (025, FR-083).
	if err := validateCampaignSweep(cfg.CampaignSweep); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.PayOS.ReconciliationCron) == "" || cfg.PayOS.ReconciliationBatchSize <= 0 || cfg.PayOS.ReconciliationLeaseSeconds <= 0 || cfg.PayOS.ReconciliationMaxAttempts <= 0 || cfg.PayOS.ReconciliationMaxAgeHours <= 0 {
		return fmt.Errorf("payos reconciliation configuration is invalid")
	}
	if cfg.Loyalty.ExpiryWorkerInterval <= 0 || cfg.Loyalty.ExpiryWorkerBatchSize <= 0 {
		return fmt.Errorf("loyalty expiry worker configuration is invalid")
	}
	if cfg.Outbox.RetryMaxAttempts <= 0 || cfg.Outbox.RetryIntervalMinutes <= 0 || cfg.Outbox.CleanupRetentionDays <= 0 || cfg.Outbox.CleanupIntervalMinutes <= 0 || cfg.Outbox.BatchSize <= 0 {
		return fmt.Errorf("outbox worker configuration is invalid")
	}
	// Validate Token Estimation Configurations
	est := cfg.AI.TokenEstimation
	if est.CharsPerToken <= 0 {
		return fmt.Errorf("chars_per_token must be greater than 0")
	}
	if est.LocalSafetyMultiplier < 1.0 {
		return fmt.Errorf("local_safety_multiplier must be at least 1.0")
	}
	if est.CountTokensThresholdRatio <= 0 || est.CountTokensThresholdRatio > 1.0 {
		return fmt.Errorf("count_tokens_threshold_ratio must be between 0 and 1.0")
	}
	if est.CountTokensTimeout <= 0 {
		return fmt.Errorf("count_tokens_timeout must be greater than 0")
	}
	if cfg.Server.SSE.HeartbeatSeconds <= 0 {
		return fmt.Errorf("server sse heartbeat_seconds must be greater than 0")
	}
	if cfg.Server.SSE.ReplayBufferSize <= 0 {
		return fmt.Errorf("server sse replay_buffer_size must be greater than 0")
	}
	if cfg.Server.SSE.ReplayBufferTTLSeconds <= 0 {
		return fmt.Errorf("server sse replay_buffer_ttl_seconds must be greater than 0")
	}
	if err := validateGoogle(cfg); err != nil {
		return err
	}
	return nil
}

// validateCampaignSweep enforces the catch-up sweep floor (025, FR-083).
//
// An ABSENT interval_seconds never reaches this function as zero: the loader
// applies the default before decoding, and a viper default only applies when the
// key is missing. So a value below the floor here is always a number somebody
// wrote down, and it is refused rather than repaired — silently turning it into
// 60 would hide the misconfiguration until someone wondered why the sweep cadence
// was not the one they set.
func validateCampaignSweep(sweep CampaignSweepConfig) error {
	if sweep.IntervalSeconds < DefaultCampaignSweepIntervalSeconds {
		return fmt.Errorf("campaign_sweep.interval_seconds must be at least %d seconds (FR-083), got %d", DefaultCampaignSweepIntervalSeconds, sweep.IntervalSeconds)
	}
	return nil
}

// validAIProviders mirrors contracts/env-ai-trial.md §1.4. The adapters are
// selected by exact name (shared/infrastructure/ai switches on it), so any other
// non-empty value would surface later as an "unsupported provider" error on a
// live request instead of at startup.
var validAIProviders = []string{"openai", "google"}

// validateAIProviderBlocks rejects a provider name the adapters do not
// implement on every AI block, plus an expected-host allowlist entry that can
// never match the host it is compared with.
//
// Empty is allowed on purpose: the shipped .env examples leave most blocks
// blank, and an unconfigured capability must not stop a service from booting.
// Only a value somebody wrote down is refused, so the check cannot be dodged by
// leaving a block out of a deployment profile.
func validateAIProviderBlocks(cfg *Config) error {
	ai := cfg.AI
	providers := []struct {
		name string
		cfg  APIProviderConfig
	}{
		{"vision_primary", ai.VisionPrimary},
		{"vision_fallback", ai.VisionFallback},
		{"embedding_primary", ai.EmbeddingPrimary},
		{"embedding_fallback", ai.EmbeddingFallback},
		{"chat_text_primary", ai.ChatTextPrimary},
		{"chat_text_fallback", ai.ChatTextFallback},
		{"recommendation_text_primary", ai.RecommendationTextPrimary},
		{"recommendation_text_fallback", ai.RecommendationTextFallback},
		{"free_text_primary", ai.FreeTextPrimary},
		{"free_text_fallback", ai.FreeTextFallback},
	}
	for _, provider := range providers {
		if err := validateAIProviderName(provider.name, provider.cfg.Provider); err != nil {
			return err
		}
	}
	for _, host := range ai.OpenAITrialExpectedHosts {
		if err := validateTrialExpectedHost(host); err != nil {
			return err
		}
	}
	return nil
}

// validateAIProviderName accepts empty (block not configured) or one of
// validAIProviders. Surrounding whitespace is tolerated because it is invisible
// in an env file; the name itself must still be exact, since a differently-cased
// value would not match the adapter switch either.
func validateAIProviderName(name, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	for _, valid := range validAIProviders {
		if trimmed == valid {
			return nil
		}
	}
	return fmt.Errorf("ai.%s.provider %q is invalid (want openai|google or empty)", name, value)
}

// validateTrialExpectedHost rejects an allowlist entry the runtime can never
// match. OpenAIEndpointHostMismatch compares the parsed URL's Hostname() — bare,
// no scheme, no path — against each entry, so an entry carrying either would
// leave the "openai points at the wrong host" warning permanently wrong with no
// hint why. An empty entry is dropped by the CSV splitter and never reaches here.
func validateTrialExpectedHost(host string) error {
	trimmed := strings.TrimSpace(host)
	if strings.Contains(trimmed, "://") || strings.Contains(trimmed, "/") {
		return fmt.Errorf("ai.openai_trial_expected_hosts entry %q must be a bare host without scheme or path", host)
	}
	return nil
}

// validateAITrialSettings enforces the per-attempt text limits added by feature
// 032. These are startup rules, not operational checks: a deadline that cannot
// fit inside the server write timeout, or a primary deadline longer than the SSE
// chunk deadline, is a configuration mistake that would only show up as a
// truncated stream or a runtime fallback under load.
func validateAITrialSettings(cfg *Config) error {
	ai := cfg.AI
	if ai.TextMaxConcurrency < 1 {
		return fmt.Errorf("ai.text_max_concurrency must be at least 1")
	}
	if ai.TextPrimaryTimeoutSeconds <= 0 || ai.TextFallbackTimeoutSeconds <= 0 {
		return fmt.Errorf("ai text attempt timeouts must be greater than 0")
	}
	if cfg.Server.SSE.AIChatChunkDeadlineSeconds <= 0 {
		return fmt.Errorf("server sse ai_chat_chunk_deadline_seconds must be greater than 0")
	}
	if ai.TextPrimaryTimeoutSeconds+ai.TextFallbackTimeoutSeconds > MaxTextAttemptTimeoutSeconds {
		return fmt.Errorf("ai.text_primary_timeout_seconds + ai.text_fallback_timeout_seconds must not exceed %d seconds (server write timeout)", MaxTextAttemptTimeoutSeconds)
	}
	if ai.TextPrimaryTimeoutSeconds > cfg.Server.SSE.AIChatChunkDeadlineSeconds {
		return fmt.Errorf("ai.text_primary_timeout_seconds must not exceed server.sse.ai_chat_chunk_deadline_seconds")
	}

	textProviders := []struct {
		name string
		cfg  APIProviderConfig
	}{
		{"chat_text_primary", ai.ChatTextPrimary},
		{"chat_text_fallback", ai.ChatTextFallback},
		{"recommendation_text_primary", ai.RecommendationTextPrimary},
		{"recommendation_text_fallback", ai.RecommendationTextFallback},
		{"free_text_primary", ai.FreeTextPrimary},
		{"free_text_fallback", ai.FreeTextFallback},
	}
	for _, provider := range textProviders {
		if provider.cfg.Provider == "openai" {
			if err := validateOpenAITextEndpoint(provider.name, provider.cfg.Endpoint); err != nil {
				return err
			}
		}
	}

	reasoningProviders := []struct {
		name string
		cfg  APIProviderConfig
	}{
		{"vision_primary", ai.VisionPrimary},
		{"vision_fallback", ai.VisionFallback},
		{"embedding_primary", ai.EmbeddingPrimary},
		{"embedding_fallback", ai.EmbeddingFallback},
		{"chat_text_primary", ai.ChatTextPrimary},
		{"chat_text_fallback", ai.ChatTextFallback},
		{"recommendation_text_primary", ai.RecommendationTextPrimary},
		{"recommendation_text_fallback", ai.RecommendationTextFallback},
		{"free_text_primary", ai.FreeTextPrimary},
		{"free_text_fallback", ai.FreeTextFallback},
	}
	for _, provider := range reasoningProviders {
		if err := validateReasoningEffort(provider.name, provider.cfg.ReasoningEffort); err != nil {
			return err
		}
	}
	return nil
}

// validateOpenAITextEndpoint requires the full OpenAI-compatible chat URL. The
// adapter POSTs provider.Endpoint verbatim, so joining a path later would send a
// request to /v1/chat/completions/v1/chat/completions. Plain HTTP is accepted
// only for a loopback local drill, and even there the path is mandatory.
func validateOpenAITextEndpoint(name, endpoint string) error {
	trimmed := strings.TrimSpace(endpoint)
	u, err := url.Parse(trimmed)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("ai.%s.endpoint %q must be a full URL ending in /v1/chat/completions", name, endpoint)
	}
	if u.Path != "/v1/chat/completions" {
		return fmt.Errorf("ai.%s.endpoint %q must end in /v1/chat/completions (the adapter does not append a path)", name, endpoint)
	}
	host := strings.ToLower(u.Hostname())
	loopback := host == "127.0.0.1" || host == "localhost"
	if u.Scheme == "https" || (u.Scheme == "http" && loopback) {
		return nil
	}
	return fmt.Errorf("ai.%s.endpoint %q must use https (http is allowed only for 127.0.0.1 or localhost)", name, endpoint)
}

// validateReasoningEffort rejects values the OpenAI-compatible providers do not
// define. Empty is valid and means "do not send the field".
func validateReasoningEffort(name, value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none", "low", "medium", "high":
		return nil
	default:
		return fmt.Errorf("ai.%s.reasoning_effort %q is invalid (want none|low|medium|high or empty)", name, value)
	}
}

// validateGoogle enforces Google OAuth config consistency when the feature is
// configured. Empty config is allowed (feature disabled in that environment);
// a partial config is rejected. The web client is named explicitly so a mobile
// client ID can never be mistaken for it — an ordering mistake here only
// surfaces later as an opaque redirect_uri_mismatch from Google.
func validateGoogle(cfg *Config) error {
	g := cfg.Google
	if strings.TrimSpace(g.WebClientID) == "" &&
		strings.TrimSpace(g.ClientIDRaw) == "" &&
		strings.TrimSpace(g.RedirectURI) == "" {
		return nil
	}
	if strings.TrimSpace(g.RedirectURI) == "" {
		return fmt.Errorf("google redirect_uri is required when Google login is configured")
	}
	if strings.TrimSpace(g.WebClientID) == "" {
		return fmt.Errorf("google web_client_id (GOOGLE_WEB_CLIENT_ID) is required when Google login is configured")
	}
	if strings.TrimSpace(g.WebClientSecret) == "" {
		return fmt.Errorf("google web_client_secret (GOOGLE_WEB_CLIENT_SECRET) is required when Google login is configured")
	}
	if len(g.ClientIDs) == 0 {
		return fmt.Errorf("google client_ids is required when Google login is configured")
	}
	if g.StateTTLMinutes <= 0 {
		return fmt.Errorf("google state_ttl_minutes must be greater than 0")
	}
	return nil
}

func validateAppMode(mode string) error {
	for _, m := range validAppModes {
		if mode == m {
			return nil
		}
	}
	return fmt.Errorf("app mode %q is invalid (want one of http|worker|all)", mode)
}

// validateJWT enforces key material per algorithm (FR-022): ES256 needs both
// PEM key paths, HS256 needs the shared secret.
func validateJWT(cfg *Config) error {
	switch strings.ToUpper(strings.TrimSpace(cfg.Jwt.Algorithm)) {
	case "", "HS256":
		if strings.TrimSpace(cfg.Jwt.Secret) == "" {
			return fmt.Errorf("JWT secret is required for HS256")
		}
	case "ES256":
		if strings.TrimSpace(cfg.Jwt.PrivateKey) == "" {
			return fmt.Errorf("JWT private key path is required for ES256")
		}
		if strings.TrimSpace(cfg.Jwt.PublicKey) == "" {
			return fmt.Errorf("JWT public key path is required for ES256")
		}
	default:
		return fmt.Errorf("JWT algorithm %q is unsupported (want HS256|ES256)", cfg.Jwt.Algorithm)
	}
	return nil
}

// validateDrillSchema allows empty (no drill) or a drill_-prefixed temporary
// schema name — anything else risks touching production schemas (FR-027).
func validateDrillSchema(schema string) error {
	if strings.TrimSpace(schema) == "" {
		return nil
	}
	if !drillSchemaPattern.MatchString(schema) {
		return fmt.Errorf("drill schema %q is invalid (want drill_<a-z0-9_>, max 54 chars)", schema)
	}
	return nil
}

// validateManaged enforces managed-profile requirements: each service migrates
// on start (FR-035/FR-011) and ingress routing is fully addressed.
func validateManaged(cfg *Config) error {
	if !cfg.Database.MigrateOnStart {
		return fmt.Errorf("database migrate_on_start must be true in managed environments")
	}
	required := map[string]string{
		"ingress account_base_url":  cfg.Ingress.AccountBaseURL,
		"ingress commerce_base_url": cfg.Ingress.CommerceBaseURL,
		"ingress ai_base_url":       cfg.Ingress.AIBaseURL,
		"ingress token_audience":    cfg.Ingress.TokenAudience,
	}
	for label, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required in managed environments", label)
		}
	}
	return nil
}
