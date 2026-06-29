package config

import "time"

type Config struct {
	Database       Database           `mapstructure:"database"`
	Redis          Redis              `mapstructure:"redis"`
	Server         Server             `mapstructure:"server"`
	Startup        Startup            `mapstructure:"startup"`
	Jwt            Jwt                `mapstructure:"jwt"`
	Logger         Logger             `mapstructure:"logger"`
	Quota          Quota              `mapstructure:"quota"`
	Otp            Otp                `mapstructure:"otp"`
	Email          Email              `mapstructure:"email"`
	RateLimit      RateLimit          `mapstructure:"rate_limit"`
	ClaimRateLimit ClaimRateLimit     `mapstructure:"claim_rate_limit"`
	PayOS          PayOS              `mapstructure:"payos"`
	Cloudinary     Cloudinary         `mapstructure:"cloudinary"`
	AI             AIServiceConfig    `mapstructure:"ai"`
	RabbitMQ       RabbitMQ           `mapstructure:"rabbitmq"`
	Elasticsearch  Elasticsearch      `mapstructure:"elasticsearch"`
	Community      Community          `mapstructure:"community"`
	Loyalty        Loyalty            `mapstructure:"loyalty"`
	RAG            RAG                `mapstructure:"rag"`
	Wardrobe       WardrobeProcessing `mapstructure:"wardrobe"`
}

type Loyalty struct {
	ExpiryWorkerEnabled   bool          `mapstructure:"expiry_worker_enabled"`
	ExpiryWorkerInterval  time.Duration `mapstructure:"expiry_worker_interval"`
	ExpiryWorkerBatchSize int           `mapstructure:"expiry_worker_batch_size"`
}

type WardrobeProcessing struct {
	RetryDelay1Seconds      int    `mapstructure:"retry_delay_1_seconds"`
	RetryDelay2Seconds      int    `mapstructure:"retry_delay_2_seconds"`
	RetryDelay3Seconds      int    `mapstructure:"retry_delay_3_seconds"`
	StaleMinutes            int    `mapstructure:"stale_minutes"`
	MaxRetryCount           int    `mapstructure:"max_retry_count"`
	RecoveryScanCron        string `mapstructure:"recovery_scan_cron"`
	CategoryCacheTTLSeconds int    `mapstructure:"category_cache_ttl_seconds"`
}

type Startup struct {
	RetryAttempt1Seconds int `mapstructure:"retry_attempt_1_seconds"`
	RetryAttempt2Seconds int `mapstructure:"retry_attempt_2_seconds"`
	RetryAttempt3Seconds int `mapstructure:"retry_attempt_3_seconds"`
}

type Elasticsearch struct {
	Addresses []string `mapstructure:"addresses"`
	User      string   `mapstructure:"-"`
	Password  string   `mapstructure:"-"`
}

type APIProviderConfig struct {
	Provider string `mapstructure:"-"`
	ApiKey   string `mapstructure:"-"`
	Endpoint string `mapstructure:"-"`
	Model    string `mapstructure:"-"`
}

type AIServiceConfig struct {
	VisionPrimary                          APIProviderConfig     `mapstructure:"vision_primary"`
	VisionFallback                         APIProviderConfig     `mapstructure:"vision_fallback"`
	EmbeddingPrimary                       APIProviderConfig     `mapstructure:"embedding_primary"`
	EmbeddingFallback                      APIProviderConfig     `mapstructure:"embedding_fallback"`
	ChatTextPrimary                        APIProviderConfig     `mapstructure:"chat_text_primary"`
	ChatTextFallback                       APIProviderConfig     `mapstructure:"chat_text_fallback"`
	RecommendationTextPrimary              APIProviderConfig     `mapstructure:"recommendation_text_primary"`
	RecommendationTextFallback             APIProviderConfig     `mapstructure:"recommendation_text_fallback"`
	ChatTextTimeoutSeconds                 int                   `mapstructure:"chat_text_timeout_seconds"`
	RecommendationTextTimeoutSeconds       int                   `mapstructure:"recommendation_text_timeout_seconds"`
	VisionTimeoutSeconds                   int                   `mapstructure:"vision_timeout_seconds"`
	EmbeddingTimeoutSeconds                int                   `mapstructure:"embedding_timeout_seconds"`
	ChatTextRPMLimit                       int                   `mapstructure:"chat_text_rpm_limit"`
	RecommendationTextRPMLimit             int                   `mapstructure:"recommendation_text_rpm_limit"`
	VisionRPMLimit                         int                   `mapstructure:"vision_rpm_limit"`
	EmbeddingRPMLimit                      int                   `mapstructure:"embedding_rpm_limit"`
	ChatMaxInputCharacters                 int                   `mapstructure:"chat_max_input_characters"`
	ChatHistoryMessageMaxCharacters        int                   `mapstructure:"chat_history_message_max_characters"`
	ChatMaxOutputTokens                    int                   `mapstructure:"chat_max_output_tokens"`
	SummarySourceMaxCharacters             int                   `mapstructure:"summary_source_max_characters"`
	SummaryPreviousMaxCharacters           int                   `mapstructure:"summary_previous_max_characters"`
	SummaryMaxOutputTokens                 int                   `mapstructure:"summary_max_output_tokens"`
	RewriterPromptMaxCharacters            int                   `mapstructure:"rewriter_prompt_max_characters"`
	RewriterMaxOutputTokens                int                   `mapstructure:"rewriter_max_output_tokens"`
	RecommendationDetailsMaxCharacters     int                   `mapstructure:"recommendation_details_max_characters"`
	RecommendationPromptCandidateLimit     int                   `mapstructure:"recommendation_prompt_candidate_limit"`
	RecommendationDescriptionMaxCharacters int                   `mapstructure:"recommendation_description_max_characters"`
	RecommendationTagsLimit                int                   `mapstructure:"recommendation_tags_limit"`
	RecommendationPromptMaxCharacters      int                   `mapstructure:"recommendation_prompt_max_characters"`
	RecommendationMaxOutputTokens          int                   `mapstructure:"recommendation_max_output_tokens"`
	FreeTextPrimary                        APIProviderConfig     `mapstructure:"free_text_primary"`
	FreeTextFallback                       APIProviderConfig     `mapstructure:"free_text_fallback"`
	FreeTextTimeoutSeconds                 int                   `mapstructure:"free_text_timeout_seconds"`
	FreeTextRPMLimit                       int                   `mapstructure:"free_text_rpm_limit"`
	Pricing                                AIPricingConfig       `mapstructure:"pricing"`
	UsageReconcileCron                     string                `mapstructure:"usage_reconcile_cron"`
	UsageReconcileBatchSize                int                   `mapstructure:"usage_reconcile_batch_size"`
	TokenEstimation                        TokenEstimationConfig `mapstructure:"token_estimation"`
}

type AIPricingConfig struct {
	Version  string               `mapstructure:"version"`
	Currency string               `mapstructure:"currency"`
	USDToVND string               `mapstructure:"usd_to_vnd"`
	Paid     AIModelPricingConfig `mapstructure:"paid"`
}

type AIModelPricingConfig struct {
	InputUSDPerMillionTokens  string `mapstructure:"input_usd_per_million_tokens"`
	OutputUSDPerMillionTokens string `mapstructure:"output_usd_per_million_tokens"`
}

type RabbitMQ struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"-"`
	Password string `mapstructure:"-"`
}

type Cloudinary struct {
	CloudName    string `mapstructure:"-"`
	ApiKey       string `mapstructure:"-"`
	ApiSecret    string `mapstructure:"-"`
	AvatarFolder string `mapstructure:"avatar_folder"`
	ItemFolder   string `mapstructure:"item_folder"`
	OutfitFolder string `mapstructure:"outfit_folder"`
	PostFolder   string `mapstructure:"post_folder"`
}

type PayOS struct {
	ClientID                   string `mapstructure:"-"`
	ApiKey                     string `mapstructure:"-"`
	ChecksumKey                string `mapstructure:"-"`
	ReturnUrl                  string `mapstructure:"return_url"`
	CancelUrl                  string `mapstructure:"cancel_url"`
	ExpiredMinutes             int    `mapstructure:"expired_minutes"`
	ReconciliationCron         string `mapstructure:"reconciliation_cron"`
	ReconciliationBatchSize    int    `mapstructure:"reconciliation_batch_size"`
	ReconciliationLeaseSeconds int    `mapstructure:"reconciliation_lease_seconds"`
	ReconciliationMaxAttempts  int    `mapstructure:"reconciliation_max_attempts"`
	ReconciliationMaxAgeHours  int    `mapstructure:"reconciliation_max_age_hours"`
}

type Database struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"-"`
	Password string `mapstructure:"-"`
	DbName   string `mapstructure:"-"`
	SslMode  string `mapstructure:"ssl_mode"`
	TimeZone string `mapstructure:"time_zone"`
}

type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"-"`
	Db       int    `mapstructure:"db"`
}

type Server struct {
	Port              string `mapstructure:"port"`
	FrontEndOrigin    string `mapstructure:"front_end_origin"`
	TimeoutSeconds    int    `mapstructure:"timeout_seconds"`
	Env               string `mapstructure:"env"`
	SwaggerAccessCode string `mapstructure:"-"`
}

type Jwt struct {
	Secret                          string `mapstructure:"-"`
	Issuer                          string `mapstructure:"issuer"`
	Audience                        string `mapstructure:"audience"`
	AccessExpirationMinutes         int    `mapstructure:"access_expiration_minutes"`
	RefreshExpirationDays           int    `mapstructure:"refresh_expiration_days"`
	ForgotPasswordExpirationMinutes int    `mapstructure:"forgot_password_expiration_minutes"`
}

type Logger struct {
	LogLevel  string `mapstructure:"log_level"`
	FilePath  string `mapstructure:"file_path"`
	LogToFile bool   `mapstructure:"log_to_file"`
}

type Quota struct {
	DefaultWardrobeLimit int `mapstructure:"default_wardrobe_limit"`
	DefaultAiOutfitLimit int `mapstructure:"default_ai_outfit_limit"`
	DefaultAiChatLimit   int `mapstructure:"default_ai_chat_limit"`
}

type Otp struct {
	MaxAttempts           int `mapstructure:"max_attempts"`
	ExpiryMinutes         int `mapstructure:"expiry_minutes"`
	ResendIntervalSeconds int `mapstructure:"resend_interval_seconds"`
}

type Email struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	SenderName  string `mapstructure:"sender_name"`
	SenderEmail string `mapstructure:"-"`
	AppPassword string `mapstructure:"-"`
}

type RateLimit struct {
	TokenLimit           int `mapstructure:"token_limit"`
	TokensPerPeriod      int `mapstructure:"tokens_per_period"`
	ReplenishmentSeconds int `mapstructure:"replenishment_seconds"`
}

type ClaimRateLimit struct {
	IPLimit       int `mapstructure:"ip_limit"`
	UserLimit     int `mapstructure:"user_limit"`
	TokenLimit    int `mapstructure:"token_limit"`
	WindowSeconds int `mapstructure:"window_seconds"`
}

type Community struct {
	MaxPersonalizedWindow int `mapstructure:"max_personalized_window"`
}

type RAG struct {
	RecentlyWornPenaltyDays                 int  `mapstructure:"recently_worn_penalty_days"`
	LongUnwornBonusDays                     int  `mapstructure:"long_unworn_bonus_days"`
	RrfKParameter                           int  `mapstructure:"rrf_k_parameter"`
	RecommendationCandidateLimit            int  `mapstructure:"recommendation_candidate_limit"`
	RecommendationMinimumCandidatePool      int  `mapstructure:"recommendation_minimum_candidate_pool"`
	RecommendationEmbeddingDimension        int  `mapstructure:"recommendation_embedding_dimension"`
	RecommendationEmbeddingTimeoutSeconds   int  `mapstructure:"recommendation_embedding_timeout_seconds"`
	RecommendationLLMRewriterEnabled        bool `mapstructure:"recommendation_llm_rewriter_enabled"`
	RecommendationLLMRewriterTimeoutSeconds int  `mapstructure:"recommendation_llm_rewriter_timeout_seconds"`
	RecommendationRewriterMaxSemanticLength int  `mapstructure:"recommendation_rewriter_max_semantic_length"`
	RecommendationRewriterMaxLexicalTerms   int  `mapstructure:"recommendation_rewriter_max_lexical_terms"`
	RecommendationRewriterMaxExcludedTerms  int  `mapstructure:"recommendation_rewriter_max_excluded_terms"`
}

type TokenEstimationConfig struct {
	CharsPerToken             float64       `mapstructure:"chars_per_token"`
	LocalSafetyMultiplier     float64       `mapstructure:"local_safety_multiplier"`
	CountTokensThresholdRatio float64       `mapstructure:"count_tokens_threshold_ratio"`
	CountTokensTimeout        time.Duration `mapstructure:"count_tokens_timeout"`
}
