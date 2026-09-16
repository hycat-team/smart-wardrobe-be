// Package config is the canonical configuration model for all 3 services
// (020-platform-migration, FR-005/FR-008).
//
// It mirrors internal/config (monolith) field-for-field so every service keeps
// the full behavior surface, and adds the 020-managed sections: App (APP_MODE),
// pooler/listener database settings, TLS toggles, asymmetric JWT, Ingress and
// extended SSE. The monolith copy is removed with cmd/server at T034; from then
// on this package is the single source of truth.
package config

import "time"

type Config struct {
	App                App                        `mapstructure:"app"`
	Database           Database                   `mapstructure:"database"`
	Redis              Redis                      `mapstructure:"redis"`
	Server             Server                     `mapstructure:"server"`
	Startup            Startup                    `mapstructure:"startup"`
	Jwt                Jwt                        `mapstructure:"jwt"`
	Logger             Logger                     `mapstructure:"logger"`
	Quota              Quota                      `mapstructure:"quota"`
	Otp                Otp                        `mapstructure:"otp"`
	Email              Email                      `mapstructure:"email"`
	RateLimit          RateLimit                  `mapstructure:"rate_limit"`
	ClaimRateLimit     ClaimRateLimit             `mapstructure:"claim_rate_limit"`
	PayOS              PayOS                      `mapstructure:"payos"`
	Cloudinary         Cloudinary                 `mapstructure:"cloudinary"`
	AI                 AIServiceConfig            `mapstructure:"ai"`
	RabbitMQ           RabbitMQ                   `mapstructure:"rabbitmq"`
	Elasticsearch      Elasticsearch              `mapstructure:"elasticsearch"`
	Community          Community                  `mapstructure:"community"`
	Loyalty            Loyalty                    `mapstructure:"loyalty"`
	BrandMembership    BrandMembershipCacheTTL    `mapstructure:"brand_membership"`
	RAG                RAG                        `mapstructure:"rag"`
	Subscription       Subscription               `mapstructure:"subscription"`
	Wardrobe           WardrobeProcessing         `mapstructure:"wardrobe"`
	Outbox             OutboxConfig               `mapstructure:"outbox"`
	Admin              AdminConfig                `mapstructure:"admin"`
	BrandFeatureAccess BrandFeatureAccessCacheTTL `mapstructure:"brand_feature_access"`
	Cache              CacheConfig                `mapstructure:"cache"`
	Ingress            Ingress                    `mapstructure:"ingress"`
}

// App selects the runtime topology of a service binary (FR-044).
//   - http:   HTTP + gRPC only, scale-to-zero, MUST NOT start consumers/cron.
//   - worker: consumers/outbox/cron with CPU always allocated (singleton).
//   - all:    both (local and VPS self-host). Default — safe for non-autoscaled envs.
type App struct {
	Mode string `mapstructure:"mode"`
}

// CacheConfig holds TTL configuration for application caches.
type CacheConfig struct {
	// OverviewTTLSeconds is the TTL of the subscription overview cache served
	// to AI hot paths via IUserSubscriptionContract (default 60 seconds).
	OverviewTTLSeconds int `mapstructure:"overview_ttl_seconds"`
}

type AdminConfig struct {
	AuditRetentionMonths int `mapstructure:"audit_retention_months"`
}

type OutboxConfig struct {
	RetryMaxAttempts       int `mapstructure:"retry_max_attempts"`
	RetryIntervalMinutes   int `mapstructure:"retry_interval_minutes"`
	CleanupRetentionDays   int `mapstructure:"cleanup_retention_days"`
	CleanupIntervalMinutes int `mapstructure:"cleanup_interval_minutes"`
	BatchSize              int `mapstructure:"batch_size"`
	// ListenerEnabled toggles the LISTEN/NOTIFY push listener. It MUST be false
	// on Supavisor transaction pooler (no LISTEN support) so boot falls back to
	// poll mode instead of failing (FR-009/FR-010).
	ListenerEnabled bool `mapstructure:"listener_enabled"`
}

type Loyalty struct {
	ExpiryWorkerEnabled   bool          `mapstructure:"expiry_worker_enabled"`
	ExpiryWorkerInterval  time.Duration `mapstructure:"expiry_worker_interval"`
	ExpiryWorkerBatchSize int           `mapstructure:"expiry_worker_batch_size"`
}

// BrandMembershipCacheTTL is the TTL of the Redis cache that stores a user's
// active brand memberships used by brand RBAC middleware. Default 5 minutes.
type BrandMembershipCacheTTL struct {
	TTL time.Duration `mapstructure:"ttl"`
}

// BrandFeatureAccessCacheTTL is the TTL of the Redis cache that stores a user's
// feature access list per brand. Default 15 minutes.
type BrandFeatureAccessCacheTTL struct {
	TTL time.Duration `mapstructure:"ttl"`
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
	// Enabled gates the whole engine: false makes index/delete no-op, search
	// fall back to database, and skips sync workers (FR-001/FR-002/FR-003).
	Enabled   bool     `mapstructure:"enabled"`
	Addresses []string `mapstructure:"addresses"`
	User      string   `mapstructure:"user"`
	Password  string   `mapstructure:"password"`
}

type APIProviderConfig struct {
	Provider string `mapstructure:"provider"`
	ApiKey   string `mapstructure:"api_key"`
	Endpoint string `mapstructure:"endpoint"`
	Model    string `mapstructure:"model"`
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
	UsageReconcileCron                     string                `mapstructure:"usage_reconcile_cron"`
	UsageReconcileBatchSize                int                   `mapstructure:"usage_reconcile_batch_size"`
	TokenEstimation                        TokenEstimationConfig `mapstructure:"token_estimation"`
}

type RabbitMQ struct {
	Host                string `mapstructure:"host"`
	Port                int    `mapstructure:"port"`
	User                string `mapstructure:"user"`
	Password            string `mapstructure:"password"`
	Prefetch            int    `mapstructure:"prefetch"`
	ConsumerConcurrency int    `mapstructure:"consumer_concurrency"`
	DeliveryLimit       int    `mapstructure:"delivery_limit"`
	ManagementPort      int    `mapstructure:"management_port"`
	// TLSEnabled switches AMQP to amqps (managed CloudAMQP requires TLS).
	TLSEnabled bool `mapstructure:"tls_enabled"`
	// ManagementTLSEnabled switches the gauge/management API to HTTPS.
	ManagementTLSEnabled bool `mapstructure:"management_tls_enabled"`
}

type Cloudinary struct {
	CloudName    string `mapstructure:"cloud_name"`
	ApiKey       string `mapstructure:"api_key"`
	ApiSecret    string `mapstructure:"api_secret"`
	APIUrl       string `mapstructure:"api_url"`
	AvatarFolder string `mapstructure:"avatar_folder"`
	BrandsFolder string `mapstructure:"brands_folder"`
	ItemFolder   string `mapstructure:"item_folder"`
	OutfitFolder string `mapstructure:"outfit_folder"`
	PostFolder   string `mapstructure:"post_folder"`
}

type PayOS struct {
	ClientID                   string `mapstructure:"client_id"`
	ApiKey                     string `mapstructure:"api_key"`
	ChecksumKey                string `mapstructure:"checksum_key"`
	Endpoint                   string `mapstructure:"endpoint"`
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
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DbName          string        `mapstructure:"db_name"`
	SslMode         string        `mapstructure:"ssl_mode"`
	TimeZone        string        `mapstructure:"time_zone"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	SearchPath      string        `mapstructure:"search_path"`
	// MigrateOnStart lets each service run its own Goose migrations at boot
	// (FR-035). Enabled on managed, safe default per profile.
	MigrateOnStart bool `mapstructure:"migrate_on_start"`
	// Listener* is the session-scoped direct DSN used for migration,
	// LISTEN/NOTIFY and advisory locks — never the transaction pooler (FR-046).
	ListenerHost    string `mapstructure:"listener_host"`
	ListenerPort    int    `mapstructure:"listener_port"`
	ListenerSslMode string `mapstructure:"listener_ssl_mode"`
	// DrillSchema scopes runbook drills to a temporary schema that is dropped
	// afterwards, never touching production schemas (FR-027).
	DrillSchema string `mapstructure:"drill_schema"`
	// QueryExecMode is the pgx query exec mode for runtime pooler traffic.
	// MUST be simple_protocol on Supavisor transaction pooler (no named
	// prepared statements); direct connections keep their own mode (FR-046).
	QueryExecMode string `mapstructure:"query_exec_mode"`
}

type Redis struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Db       int    `mapstructure:"db"`
	// TLSEnabled switches to rediss (Redis Cloud requires TLS).
	TLSEnabled bool `mapstructure:"tls_enabled"`
}

type Server struct {
	Port                string    `mapstructure:"port"`
	FrontEndOrigin      string    `mapstructure:"front_end_origin"`
	TimeoutSeconds      int       `mapstructure:"timeout_seconds"`
	Env                 string    `mapstructure:"env"`
	SwaggerAccessCode   string    `mapstructure:"swagger_access_code"`
	ReadTimeoutSeconds  int       `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int       `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds  int       `mapstructure:"idle_timeout_seconds"`
	GzipEnabled         bool      `mapstructure:"gzip_enabled"`
	BodyLimit           BodyLimit `mapstructure:"body_limit"`
	SSE                 SSE       `mapstructure:"sse"`
}

// SSE configures per-event write deadlines for server-sent event streams.
// AI chat streams get a looser window because LLM providers can stall longer
// between tokens than local progress broadcasters.
type SSE struct {
	ChunkDeadlineSeconds       int `mapstructure:"chunk_deadline_seconds"`
	AIChatChunkDeadlineSeconds int `mapstructure:"ai_chat_chunk_deadline_seconds"`
	// HeartbeatSeconds is the SSE comment heartbeat keeping Nginx → Cloud Run
	// streams alive (FR-036). MUST be identical in every environment (shared configs/).
	HeartbeatSeconds int `mapstructure:"heartbeat_seconds"`
}

// BodyLimit configures the tiered request body size limit:
// DefaultMB applies to every route, ElevatedMB to the routes matched by the
// elevated tier patterns, and FreeRoutes lists `METHOD prefix` entries that
// bypass the limit entirely.
type BodyLimit struct {
	DefaultMB  int      `mapstructure:"default_mb"`
	ElevatedMB int      `mapstructure:"elevated_mb"`
	FreeRoutes []string `mapstructure:"free_routes"`
}

type Jwt struct {
	Secret                          string `mapstructure:"secret"`
	Issuer                          string `mapstructure:"issuer"`
	Audience                        string `mapstructure:"audience"`
	AccessExpirationMinutes         int    `mapstructure:"access_expiration_minutes"`
	RefreshExpirationDays           int    `mapstructure:"refresh_expiration_days"`
	ForgotPasswordExpirationMinutes int    `mapstructure:"forgot_password_expiration_minutes"`
	// Algorithm selects the inter-service signing scheme (FR-022): ES256
	// (asymmetric, identity signs with PrivateKey, others verify with
	// PublicKey) with HS256 kept only as a transitional legacy fallback.
	Algorithm string `mapstructure:"algorithm"`
	// PrivateKey/PublicKey are filesystem paths to the PEM keys (ES256).
	PrivateKey string `mapstructure:"private_key"`
	PublicKey  string `mapstructure:"public_key"`
}

// Ingress holds the managed-editions routing addresses (FR-005/FR-008).
// Empty in self-host profiles; required in managed profiles.
type Ingress struct {
	AccountBaseURL  string `mapstructure:"account_base_url"`
	CommerceBaseURL string `mapstructure:"commerce_base_url"`
	AIBaseURL       string `mapstructure:"ai_base_url"`
	TokenHelperURL  string `mapstructure:"token_helper_url"`
	TokenAudience   string `mapstructure:"token_audience"`
	AuthHeader      string `mapstructure:"auth_header"`
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
	SenderEmail string `mapstructure:"sender_email"`
	// SmtpUser is the SMTP auth username; falls back to SenderEmail when empty
	SmtpUser string `mapstructure:"smtp_user"`
	SmtpKey  string `mapstructure:"smtp_key"`
	// ReplyTo is optional — when empty, no Reply-To header is emitted.
	ReplyTo string `mapstructure:"reply_to"`
}

type Subscription struct {
	MaxRetryAttempts         int `mapstructure:"max_retry_attempts"`
	WalletBalanceWarningDays int `mapstructure:"wallet_balance_warning_days"`
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
