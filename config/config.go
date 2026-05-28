package config

type Config struct {
	Database  Database
	Redis     Redis
	Server    Server
	Jwt       Jwt
	Logger    Logger
	Quota     Quota
	Otp       Otp
	Email     Email
	RateLimit RateLimit
	PayOS     PayOS
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
