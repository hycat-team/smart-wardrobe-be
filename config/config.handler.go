package config

import (
	"log"
	"os"
	"strconv"

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

	return &Config{
		Database: Database{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnvInt("POSTGRES_PORT", 5432),
			User:     getEnv("POSTGRES_USER", "admin"),
			Password: getEnv("POSTGRES_PASSWORD", "123456"),
			DbName:   getEnv("POSTGRES_DB", "smart_wardrobe_db"),
			SslMode:  getEnv("POSTGRES_SSLMODE", "disable"),
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
		},
		Jwt: Jwt{
			Secret:                          getEnv("JWT_SECRET", "default_secret_key_change_me_in_production"),
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
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if val, err := strconv.Atoi(value); err == nil {
			return val
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if val, err := strconv.ParseBool(value); err == nil {
			return val
		}
	}
	return fallback
}
