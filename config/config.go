package config

type Config struct {
	Database      Database
	Redis         Redis
	Server        Server
	Startup       Startup
	Jwt           Jwt
	Logger        Logger
	Quota         Quota
	Otp           Otp
	Email         Email
	RateLimit     RateLimit
	PayOS         PayOS
	Cloudinary    Cloudinary
	AI            AIServiceConfig
	RabbitMQ      RabbitMQ
	Elasticsearch Elasticsearch
	Community     Community
	RAG           RAG
	Wardrobe      WardrobeProcessing
}

type WardrobeProcessing struct {
	RetryDelay1Seconds int
	RetryDelay2Seconds int
	RetryDelay3Seconds int
	StaleMinutes       int
	MaxRetryCount      int
	RecoveryScanCron   string
	CategoryCacheTTLSeconds int
}

type Startup struct {
	RetryAttempt1Seconds int
	RetryAttempt2Seconds int
	RetryAttempt3Seconds int
}

type Elasticsearch struct {
	Addresses []string
	User      string
	Password  string
}

type APIProviderConfig struct {
	Provider string
	ApiKey   string
	Endpoint string
	Model    string
}

type AIServiceConfig struct {
	VisionPrimary     APIProviderConfig
	VisionFallback    APIProviderConfig
	EmbeddingPrimary  APIProviderConfig
	EmbeddingFallback APIProviderConfig
	TextPrimary       APIProviderConfig
	TextFallback      APIProviderConfig
	RpmLimit          int
}

type RabbitMQ struct {
	Host     string
	Port     int
	User     string
	Password string
}

type Cloudinary struct {
	CloudName    string
	ApiKey       string
	ApiSecret    string
	AvatarFolder string
	ItemFolder   string
	OutfitFolder string
	PostFolder   string
}

type PayOS struct {
	ClientID    string
	ApiKey      string
	ChecksumKey string
	ReturnUrl   string
	CancelUrl   string
}

type Database struct {
	Host     string
	Port     int
	User     string
	Password string
	DbName   string
	SslMode  string
	TimeZone string
}

type Redis struct {
	Host     string
	Port     int
	Password string
	Db       int
}

type Server struct {
	Port           string
	FrontEndOrigin string
	TimeoutSeconds int
	Env            string
}

type Jwt struct {
	Secret                          string
	Issuer                          string
	Audience                        string
	AccessExpirationMinutes         int
	RefreshExpirationDays           int
	ForgotPasswordExpirationMinutes int
}

type Logger struct {
	LogLevel  string
	FilePath  string
	LogToFile bool
}

type Quota struct {
	DefaultWardrobeLimit int
	DefaultAiOutfitLimit int
	DefaultAiChatLimit   int
}

type Otp struct {
	MaxAttempts           int
	ExpiryMinutes         int
	ResendIntervalSeconds int
}

type Email struct {
	Host        string
	Port        int
	SenderName  string
	SenderEmail string
	AppPassword string
}

type RateLimit struct {
	TokenLimit           int
	TokensPerPeriod      int
	ReplenishmentSeconds int
}

type Community struct {
	MaxPersonalizedWindow int
}

type RAG struct {
	RecentlyWornPenaltyDays int
	LongUnwornBonusDays     int
	RrfKParameter           int
}
