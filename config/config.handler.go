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
			DefaultWardrobeLimit: getEnvInt("QUOTA_DEFAULT_WARDROBE_LIMIT", 50),
			DefaultAiOutfitLimit: getEnvInt("QUOTA_DEFAULT_AI_OUTFIT_LIMIT", 10),
			DefaultAiChatLimit:   getEnvInt("QUOTA_DEFAULT_AI_CHAT_LIMIT", 10),
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
			ClientID:    getEnv("PAYOS_CLIENT_ID", ""),
			ApiKey:      getEnv("PAYOS_API_KEY", ""),
			ChecksumKey: getEnv("PAYOS_CHECKSUM_KEY", ""),
			ReturnUrl:   getEnv("PAYOS_RETURN_URL", "http://localhost:3000"),
			CancelUrl:   getEnv("PAYOS_CANCEL_URL", "http://localhost:3000"),
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
			TextPrimary: APIProviderConfig{
				Provider: getEnv("TEXT_PRIMARY_PROVIDER", getEnv("VISION_PRIMARY_PROVIDER", "")),
				ApiKey:   getEnv("TEXT_PRIMARY_API_KEY", getEnv("VISION_PRIMARY_API_KEY", "")),
				Endpoint: getEnv("TEXT_PRIMARY_ENDPOINT", getEnv("VISION_PRIMARY_ENDPOINT", "")),
				Model:    getEnv("TEXT_PRIMARY_MODEL", getEnv("VISION_PRIMARY_MODEL", "")),
			},
			TextFallback: APIProviderConfig{
				Provider: getEnv("TEXT_FALLBACK_PROVIDER", getEnv("VISION_FALLBACK_PROVIDER", "")),
				ApiKey:   getEnv("TEXT_FALLBACK_API_KEY", getEnv("VISION_FALLBACK_API_KEY", "")),
				Endpoint: getEnv("TEXT_FALLBACK_ENDPOINT", getEnv("VISION_FALLBACK_ENDPOINT", "")),
				Model:    getEnv("TEXT_FALLBACK_MODEL", getEnv("VISION_FALLBACK_MODEL", "")),
			},
			RpmLimit: getEnvInt("AI_RPM_LIMIT", 5),
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

	return nil
}
