package config

import (
	"os"
	"strconv"
	"time"
)

func loadSecrets(cfg *Config) {
	// Server and authentication
	cfg.Server.SwaggerAccessCode = os.Getenv("SWAGGER_ACCESS_CODE")
	cfg.Jwt.Secret = os.Getenv("JWT_SECRET")

	// Database and infrastructure
	cfg.Database.User = os.Getenv("DB_USER")
	cfg.Database.Password = os.Getenv("DB_PASSWORD")
	cfg.Database.DbName = os.Getenv("DB_NAME")
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")
	cfg.RabbitMQ.User = os.Getenv("RABBITMQ_USER")
	cfg.RabbitMQ.Password = os.Getenv("RABBITMQ_PASSWORD")
	cfg.Elasticsearch.User = os.Getenv("ELASTICSEARCH_USER")
	cfg.Elasticsearch.Password = os.Getenv("ELASTICSEARCH_PASSWORD")

	// Email
	cfg.Email.SenderEmail = os.Getenv("EMAIL_SENDER_EMAIL")
	cfg.Email.AppPassword = os.Getenv("EMAIL_APP_PASSWORD")

	// PayOS
	cfg.PayOS.ClientID = os.Getenv("PAYOS_CLIENT_ID")
	cfg.PayOS.ApiKey = os.Getenv("PAYOS_API_KEY")
	cfg.PayOS.ChecksumKey = os.Getenv("PAYOS_CHECKSUM_KEY")

	// Cloudinary
	cfg.Cloudinary.CloudName = os.Getenv("CLOUDINARY_CLOUD_NAME")
	cfg.Cloudinary.ApiKey = os.Getenv("CLOUDINARY_API_KEY")
	cfg.Cloudinary.ApiSecret = os.Getenv("CLOUDINARY_API_SECRET")

	if value := os.Getenv("LOYALTY_EXPIRY_WORKER_ENABLED"); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			cfg.Loyalty.ExpiryWorkerEnabled = parsed
		}
	}
	if value := os.Getenv("LOYALTY_EXPIRY_WORKER_INTERVAL"); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			cfg.Loyalty.ExpiryWorkerInterval = parsed
		}
	}
	if value := os.Getenv("LOYALTY_EXPIRY_WORKER_BATCH_SIZE"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			cfg.Loyalty.ExpiryWorkerBatchSize = parsed
		}
	}

	// AI providers
	loadAIProvider("VISION_PRIMARY", &cfg.AI.VisionPrimary)
	loadAIProvider("VISION_FALLBACK", &cfg.AI.VisionFallback)
	loadAIProvider("EMBEDDING_PRIMARY", &cfg.AI.EmbeddingPrimary)
	loadAIProvider("EMBEDDING_FALLBACK", &cfg.AI.EmbeddingFallback)
	loadAIProvider("CHAT_TEXT_PRIMARY", &cfg.AI.ChatTextPrimary)
	loadAIProvider("CHAT_TEXT_FALLBACK", &cfg.AI.ChatTextFallback)
	loadAIProvider("RECOMMENDATION_TEXT_PRIMARY", &cfg.AI.RecommendationTextPrimary)
	loadAIProvider("RECOMMENDATION_TEXT_FALLBACK", &cfg.AI.RecommendationTextFallback)
	loadAIProvider("FREE_TEXT_PRIMARY", &cfg.AI.FreeTextPrimary)
	loadAIProvider("FREE_TEXT_FALLBACK", &cfg.AI.FreeTextFallback)

	// AI API key fallbacks
	cfg.AI.ChatTextPrimary.ApiKey = firstEnv("CHAT_TEXT_PRIMARY_API_KEY", "VISION_PRIMARY_API_KEY")
	cfg.AI.ChatTextFallback.ApiKey = firstEnv("CHAT_TEXT_FALLBACK_API_KEY", "VISION_FALLBACK_API_KEY")
	cfg.AI.RecommendationTextPrimary.ApiKey = firstEnv("RECOMMENDATION_TEXT_PRIMARY_API_KEY", "VISION_PRIMARY_API_KEY")
	cfg.AI.RecommendationTextFallback.ApiKey = firstEnv("RECOMMENDATION_TEXT_FALLBACK_API_KEY", "VISION_FALLBACK_API_KEY")
}

func loadAIProvider(prefix string, target *APIProviderConfig) {
	target.Provider = os.Getenv(prefix + "_PROVIDER")
	target.ApiKey = os.Getenv(prefix + "_API_KEY")
	target.Endpoint = os.Getenv(prefix + "_ENDPOINT")
	target.Model = os.Getenv(prefix + "_MODEL")
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
