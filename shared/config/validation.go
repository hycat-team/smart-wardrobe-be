package config

import (
	"fmt"
	"regexp"
	"strings"
)

// drillSchemaPattern confines drills to clearly-marked temporary schemas so a
// misconfigured drill can never resolve to a production schema (FR-027).
var drillSchemaPattern = regexp.MustCompile(`^drill_[a-z0-9_]{1,48}$`)

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
	if cfg.Server.Env == "production" && strings.Contains(cfg.Server.FrontEndOrigin, "*") {
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
