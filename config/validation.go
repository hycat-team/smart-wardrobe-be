package config

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

func validateConfig(cfg *Config) error {
	required := map[string]string{"JWT secret": cfg.Jwt.Secret, "DB host": cfg.Database.Host, "DB user": cfg.Database.User, "DB name": cfg.Database.DbName}
	for label, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}
	if cfg.Server.Env == "production" && strings.Contains(cfg.Server.FrontEndOrigin, "*") {
		return fmt.Errorf("wildcard FRONTEND_ORIGIN is not allowed in production")
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
	if strings.TrimSpace(cfg.AI.Pricing.Version) == "" || strings.ToUpper(strings.TrimSpace(cfg.AI.Pricing.Currency)) != "VND" {
		return fmt.Errorf("ai pricing version and VND currency are required")
	}
	for label, raw := range map[string]string{"usd_to_vnd": cfg.AI.Pricing.USDToVND, "input_price": cfg.AI.Pricing.Paid.InputUSDPerMillionTokens, "output_price": cfg.AI.Pricing.Paid.OutputUSDPerMillionTokens} {
		value, err := decimal.NewFromString(raw)
		if err != nil || !value.IsPositive() {
			return fmt.Errorf("ai pricing %s must be a positive decimal", label)
		}
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
	if strings.TrimSpace(cfg.PayOS.ReconciliationCron) == "" || cfg.PayOS.ReconciliationBatchSize <= 0 || cfg.PayOS.ReconciliationLeaseSeconds <= 0 || cfg.PayOS.ReconciliationMaxAttempts <= 0 || cfg.PayOS.ReconciliationMaxAgeHours <= 0 {
		return fmt.Errorf("payos reconciliation configuration is invalid")
	}
	if cfg.Loyalty.ExpiryWorkerInterval <= 0 || cfg.Loyalty.ExpiryWorkerBatchSize <= 0 {
		return fmt.Errorf("loyalty expiry worker configuration is invalid")
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
	return nil
}
