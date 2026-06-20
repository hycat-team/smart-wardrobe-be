package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/joho/godotenv"
)

func LoadConfig() *Config {
	err := godotenv.Load(".env")
	if err != nil {
		errRoot := godotenv.Load("../.env")
		if errRoot != nil {
			log.Println("Warning: No .env file found. Using OS environment variables.")
		} else {
			log.Println("Loaded .env from root directory")
		}
	}

	cfg := &Config{
		Database: Database{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "admin"),
			Password: getEnv("DB_PASSWORD", "123456"),
			DbName:   getEnv("DB_NAME", "smart_wardrobe_db"),
			SslMode:  getEnv("DB_SSLMODE", "disable"),
			TimeZone: getEnv("DB_TIMEZONE", "Asia/Ho_Chi_Minh"),
		},
		Redis: Redis{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			Db:       getEnvInt("REDIS_DB", 0),
		},
		Server: Server{
			Port:           getEnv("SERVER_PORT", "8080"),
			FrontEndOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
			TimeoutSeconds: getEnvInt("REQUEST_TIMEOUT_SECONDS", 30),
			Env:            getEnv("ENV", "development"),
		},
		Startup: Startup{
			RetryAttempt1Seconds: getEnvInt("STARTUP_RETRY_ATTEMPT_1_SECONDS", 5),
			RetryAttempt2Seconds: getEnvInt("STARTUP_RETRY_ATTEMPT_2_SECONDS", 15),
			RetryAttempt3Seconds: getEnvInt("STARTUP_RETRY_ATTEMPT_3_SECONDS", 30),
		},
		Jwt: Jwt{
			Secret:                          getEnv("JWT_SECRET", ""),
			Issuer:                          getEnv("JWT_ISSUER", "SmartWardrobe"),
			Audience:                        getEnv("JWT_AUDIENCE", "SmartWardrobeUsers"),
			AccessExpirationMinutes:         getEnvInt("JWT_ACCESS_EXPIRATION_MINUTES", 60),
			RefreshExpirationDays:           getEnvInt("JWT_REFRESH_EXPIRATION_DAYS", 30),
			ForgotPasswordExpirationMinutes: getEnvInt("JWT_FORGOT_PASSWORD_EXPIRATION_MINUTES", 15),
		},
		Logger: Logger{
			LogLevel:  getEnv("LOG_LEVEL", "debug"),
			FilePath:  getEnv("LOG_FILE_PATH", "./logs/app.log"),
			LogToFile: getEnvBool("LOG_TO_FILE", false),
		},
		Quota: Quota{
			DefaultWardrobeLimit: getEnvInt("QUOTA_DEFAULT_WARDROBE_LIMIT", 100),
			DefaultAiOutfitLimit: getEnvInt("QUOTA_DEFAULT_AI_OUTFIT_LIMIT", 3),
			DefaultAiChatLimit:   getEnvInt("QUOTA_DEFAULT_AI_CHAT_LIMIT", 3),
		},
		Otp: Otp{
			MaxAttempts:           getEnvInt("OTP_MAX_ATTEMPTS", 5),
			ExpiryMinutes:         getEnvInt("OTP_EXPIRY_MINUTES", 5),
			ResendIntervalSeconds: getEnvInt("OTP_RESEND_INTERVAL_SECONDS", 60),
		},
		Email: Email{
			Host:        getEnv("EMAIL_HOST", "smtp.gmail.com"),
			Port:        getEnvInt("EMAIL_PORT", 587),
			SenderName:  getEnv("EMAIL_SENDER_NAME", "SmartWardrobe"),
			SenderEmail: getEnv("EMAIL_SENDER_EMAIL", ""),
			AppPassword: getEnv("EMAIL_APP_PASSWORD", ""),
		},
		RateLimit: RateLimit{
			TokenLimit:           getEnvInt("RATE_LIMIT_TOKEN_LIMIT", 100),
			TokensPerPeriod:      getEnvInt("RATE_LIMIT_TOKENS_PER_PERIOD", 20),
			ReplenishmentSeconds: getEnvInt("RATE_LIMIT_REPLENISHMENT_SECONDS", 10),
		},
		PayOS: PayOS{
			ClientID:                   getEnv("PAYOS_CLIENT_ID", ""),
			ApiKey:                     getEnv("PAYOS_API_KEY", ""),
			ChecksumKey:                getEnv("PAYOS_CHECKSUM_KEY", ""),
			ReturnUrl:                  getEnv("PAYOS_RETURN_URL", "http://localhost:3000"),
			CancelUrl:                  getEnv("PAYOS_CANCEL_URL", "http://localhost:3000"),
			ExpiredMinutes:             getEnvInt("PAYOS_EXPIRED_MINUTES", 15),
			ReconciliationCron:         getEnv("PAYOS_RECONCILIATION_CRON", "0 */10 * * * *"),
			ReconciliationBatchSize:    getEnvInt("PAYOS_RECONCILIATION_BATCH_SIZE", 50),
			ReconciliationLeaseSeconds: getEnvInt("PAYOS_RECONCILIATION_LEASE_SECONDS", 45),
			ReconciliationMaxAttempts:  getEnvInt("PAYOS_RECONCILIATION_MAX_ATTEMPTS", 20),
			ReconciliationMaxAgeHours:  getEnvInt("PAYOS_RECONCILIATION_MAX_AGE_HOURS", 168),
		},
		Cloudinary: Cloudinary{
			CloudName:    getEnv("CLOUDINARY_CLOUD_NAME", ""),
			ApiKey:       getEnv("CLOUDINARY_API_KEY", ""),
			ApiSecret:    getEnv("CLOUDINARY_API_SECRET", ""),
			AvatarFolder: getEnv("CLOUDINARY_AVATAR_FOLDER", "smart_wardrobe/avatars"),
			ItemFolder:   getEnv("CLOUDINARY_ITEM_FOLDER", "smart_wardrobe/items"),
			OutfitFolder: getEnv("CLOUDINARY_OUTFIT_FOLDER", "smart_wardrobe/outfits"),
			PostFolder:   getEnv("CLOUDINARY_POST_FOLDER", "smart_wardrobe/posts"),
		},
		AI: AIServiceConfig{
			VisionPrimary: APIProviderConfig{
				Provider: getEnv("VISION_PRIMARY_PROVIDER", ""),
				ApiKey:   getEnv("VISION_PRIMARY_API_KEY", ""),
				Endpoint: getEnv("VISION_PRIMARY_ENDPOINT", ""),
				Model:    getEnv("VISION_PRIMARY_MODEL", ""),
			},
			VisionFallback: APIProviderConfig{
				Provider: getEnv("VISION_FALLBACK_PROVIDER", ""),
				ApiKey:   getEnv("VISION_FALLBACK_API_KEY", ""),
				Endpoint: getEnv("VISION_FALLBACK_ENDPOINT", ""),
				Model:    getEnv("VISION_FALLBACK_MODEL", ""),
			},
			EmbeddingPrimary: APIProviderConfig{
				Provider: getEnv("EMBEDDING_PRIMARY_PROVIDER", ""),
				ApiKey:   getEnv("EMBEDDING_PRIMARY_API_KEY", ""),
				Endpoint: getEnv("EMBEDDING_PRIMARY_ENDPOINT", ""),
				Model:    getEnv("EMBEDDING_PRIMARY_MODEL", ""),
			},
			EmbeddingFallback: APIProviderConfig{
				Provider: getEnv("EMBEDDING_FALLBACK_PROVIDER", ""),
				ApiKey:   getEnv("EMBEDDING_FALLBACK_API_KEY", ""),
				Endpoint: getEnv("EMBEDDING_FALLBACK_ENDPOINT", ""),
				Model:    getEnv("EMBEDDING_FALLBACK_MODEL", ""),
			},
			ChatTextPrimary: APIProviderConfig{
				Provider: getEnv("CHAT_TEXT_PRIMARY_PROVIDER", getEnv("TEXT_PRIMARY_PROVIDER", getEnv("VISION_PRIMARY_PROVIDER", ""))),
				ApiKey:   getEnv("CHAT_TEXT_PRIMARY_API_KEY", getEnv("TEXT_PRIMARY_API_KEY", getEnv("VISION_PRIMARY_API_KEY", ""))),
				Endpoint: getEnv("CHAT_TEXT_PRIMARY_ENDPOINT", getEnv("TEXT_PRIMARY_ENDPOINT", getEnv("VISION_PRIMARY_ENDPOINT", ""))),
				Model:    getEnv("CHAT_TEXT_PRIMARY_MODEL", getEnv("TEXT_PRIMARY_MODEL", getEnv("VISION_PRIMARY_MODEL", ""))),
			},
			ChatTextFallback: APIProviderConfig{
				Provider: getEnv("CHAT_TEXT_FALLBACK_PROVIDER", getEnv("TEXT_FALLBACK_PROVIDER", getEnv("VISION_FALLBACK_PROVIDER", ""))),
				ApiKey:   getEnv("CHAT_TEXT_FALLBACK_API_KEY", getEnv("TEXT_FALLBACK_API_KEY", getEnv("VISION_FALLBACK_API_KEY", ""))),
				Endpoint: getEnv("CHAT_TEXT_FALLBACK_ENDPOINT", getEnv("TEXT_FALLBACK_ENDPOINT", getEnv("VISION_FALLBACK_ENDPOINT", ""))),
				Model:    getEnv("CHAT_TEXT_FALLBACK_MODEL", getEnv("TEXT_FALLBACK_MODEL", getEnv("VISION_FALLBACK_MODEL", ""))),
			},
			RecommendationTextPrimary: APIProviderConfig{
				Provider: getEnv("RECOMMENDATION_TEXT_PRIMARY_PROVIDER", getEnv("TEXT_PRIMARY_PROVIDER", getEnv("VISION_PRIMARY_PROVIDER", ""))),
				ApiKey:   getEnv("RECOMMENDATION_TEXT_PRIMARY_API_KEY", getEnv("TEXT_PRIMARY_API_KEY", getEnv("VISION_PRIMARY_API_KEY", ""))),
				Endpoint: getEnv("RECOMMENDATION_TEXT_PRIMARY_ENDPOINT", getEnv("TEXT_PRIMARY_ENDPOINT", getEnv("VISION_PRIMARY_ENDPOINT", ""))),
				Model:    getEnv("RECOMMENDATION_TEXT_PRIMARY_MODEL", getEnv("TEXT_PRIMARY_MODEL", getEnv("VISION_PRIMARY_MODEL", ""))),
			},
			RecommendationTextFallback: APIProviderConfig{
				Provider: getEnv("RECOMMENDATION_TEXT_FALLBACK_PROVIDER", getEnv("TEXT_FALLBACK_PROVIDER", getEnv("VISION_FALLBACK_PROVIDER", ""))),
				ApiKey:   getEnv("RECOMMENDATION_TEXT_FALLBACK_API_KEY", getEnv("TEXT_FALLBACK_API_KEY", getEnv("VISION_FALLBACK_API_KEY", ""))),
				Endpoint: getEnv("RECOMMENDATION_TEXT_FALLBACK_ENDPOINT", getEnv("TEXT_FALLBACK_ENDPOINT", getEnv("VISION_FALLBACK_ENDPOINT", ""))),
				Model:    getEnv("RECOMMENDATION_TEXT_FALLBACK_MODEL", getEnv("TEXT_FALLBACK_MODEL", getEnv("VISION_FALLBACK_MODEL", ""))),
			},
			ChatTextTimeoutSeconds:           getEnvInt("AI_CHAT_TEXT_TIMEOUT_SECONDS", getEnvInt("AI_TEXT_TIMEOUT_SECONDS", 30)),
			RecommendationTextTimeoutSeconds: getEnvInt("AI_RECOMMENDATION_TEXT_TIMEOUT_SECONDS", getEnvInt("AI_TEXT_TIMEOUT_SECONDS", 30)),
			VisionTimeoutSeconds:             getEnvInt("AI_VISION_TIMEOUT_SECONDS", 20),
			EmbeddingTimeoutSeconds:          getEnvInt("AI_EMBEDDING_TIMEOUT_SECONDS", 20),
			ChatTextRPMLimit:                 getEnvInt("AI_CHAT_TEXT_RPM_LIMIT", getEnvInt("AI_TEXT_RPM_LIMIT", 5)),
			RecommendationTextRPMLimit:       getEnvInt("AI_RECOMMENDATION_TEXT_RPM_LIMIT", getEnvInt("AI_TEXT_RPM_LIMIT", 5)),
			VisionRPMLimit:                   getEnvInt("AI_VISION_RPM_LIMIT", 5),
			EmbeddingRPMLimit:                getEnvInt("AI_EMBEDDING_RPM_LIMIT", 5),
		},
		RabbitMQ: RabbitMQ{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnvInt("RABBITMQ_PORT", 5672),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "123456"),
		},
		Elasticsearch: Elasticsearch{
			Addresses: []string{getEnv("ELASTICSEARCH_ADDRESS", "http://localhost:9200")},
			User:      getEnv("ELASTICSEARCH_USER", ""),
			Password:  getEnv("ELASTICSEARCH_PASSWORD", ""),
		},
		Community: Community{
			MaxPersonalizedWindow: getEnvInt("COMMUNITY_MAX_PERSONALIZED_WINDOW", 100),
		},
		RAG: RAG{
			RecentlyWornPenaltyDays:                 getEnvInt("RAG_RECENTLY_WORN_PENALTY_DAYS", 3),
			LongUnwornBonusDays:                     getEnvInt("RAG_LONG_UNWORN_BONUS_DAYS", 14),
			RrfKParameter:                           getEnvInt("RAG_RRF_K_PARAMETER", 30),
			RecommendationCandidateLimit:            getEnvInt("RAG_RECOMMENDATION_CANDIDATE_LIMIT", 40),
			RecommendationMinimumCandidatePool:      getEnvInt("RAG_RECOMMENDATION_MINIMUM_CANDIDATE_POOL", 20),
			RecommendationEmbeddingDimension:        getEnvInt("RAG_RECOMMENDATION_EMBEDDING_DIMENSION", 768),
			RecommendationEmbeddingTimeoutSeconds:   getEnvInt("RAG_RECOMMENDATION_EMBEDDING_TIMEOUT_SECONDS", 2),
			RecommendationLLMRewriterEnabled:        getEnvBool("RAG_RECOMMENDATION_LLM_REWRITER_ENABLED", false),
			RecommendationLLMRewriterTimeoutSeconds: getEnvInt("RAG_RECOMMENDATION_LLM_REWRITER_TIMEOUT_SECONDS", 2),
			RecommendationRewriterMaxSemanticLength: getEnvInt("RAG_RECOMMENDATION_REWRITER_MAX_SEMANTIC_LENGTH", 512),
			RecommendationRewriterMaxLexicalTerms:   getEnvInt("RAG_RECOMMENDATION_REWRITER_MAX_LEXICAL_TERMS", 24),
			RecommendationRewriterMaxExcludedTerms:  getEnvInt("RAG_RECOMMENDATION_REWRITER_MAX_EXCLUDED_TERMS", 24),
		},
		Wardrobe: WardrobeProcessing{
			RetryDelay1Seconds:      getEnvInt("WARDROBE_RETRY_DELAY_1_SECONDS", 60),
			RetryDelay2Seconds:      getEnvInt("WARDROBE_RETRY_DELAY_2_SECONDS", 300),
			RetryDelay3Seconds:      getEnvInt("WARDROBE_RETRY_DELAY_3_SECONDS", 900),
			StaleMinutes:            getEnvInt("WARDROBE_PROCESSING_STALE_MINUTES", 20),
			MaxRetryCount:           getEnvInt("WARDROBE_PROCESSING_MAX_RETRIES", 3),
			RecoveryScanCron:        getEnv("WARDROBE_RECOVERY_SCAN_CRON", "0 */5 * * * *"),
			CategoryCacheTTLSeconds: getEnvInt("WARDROBE_CATEGORY_CACHE_TTL_SECONDS", 300),
		},
	}

	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	return cfg
}

func validateConfig(cfg *Config) error {
	required := map[string]string{
		"JWT secret": cfg.Jwt.Secret,
		"DB host":    cfg.Database.Host,
		"DB user":    cfg.Database.User,
		"DB name":    cfg.Database.DbName,
	}

	for label, value := range required {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", label)
		}
	}

	if cfg.Server.Env == "production" {
		if cfg.Jwt.Secret == "" {
			return fmt.Errorf("default JWT secret is not allowed in production")
		}
		if strings.Contains(cfg.Server.FrontEndOrigin, "*") {
			return fmt.Errorf("wildcard FRONTEND_ORIGIN is not allowed in production")
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
	if cfg.AI.ChatTextRPMLimit <= 0 || cfg.AI.RecommendationTextRPMLimit <= 0 || cfg.AI.VisionRPMLimit <= 0 || cfg.AI.EmbeddingRPMLimit <= 0 {
		return fmt.Errorf("ai rpm limits must be greater than 0")
	}
	if cfg.RAG.RecommendationCandidateLimit <= 0 || cfg.RAG.RecommendationMinimumCandidatePool <= 0 || cfg.RAG.RecommendationEmbeddingDimension <= 0 || cfg.RAG.RecommendationEmbeddingTimeoutSeconds <= 0 {
		return fmt.Errorf("rag recommendation retrieval configuration is invalid")
	}
	if cfg.PayOS.ExpiredMinutes <= 0 {
		return fmt.Errorf("payos expiration must be greater than 0")
	}
	if strings.TrimSpace(cfg.PayOS.ReconciliationCron) == "" || cfg.PayOS.ReconciliationBatchSize <= 0 || cfg.PayOS.ReconciliationLeaseSeconds <= 0 || cfg.PayOS.ReconciliationMaxAttempts <= 0 || cfg.PayOS.ReconciliationMaxAgeHours <= 0 {
		return fmt.Errorf("payos reconciliation configuration is invalid")
	}

	return nil
}
