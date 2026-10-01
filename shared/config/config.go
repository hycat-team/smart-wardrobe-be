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
	App                 App                        `mapstructure:"app"`
	Database            Database                   `mapstructure:"database"`
	Redis               Redis                      `mapstructure:"redis"`
	Server              Server                     `mapstructure:"server"`
	Startup             Startup                    `mapstructure:"startup"`
	Jwt                 Jwt                        `mapstructure:"jwt"`
	Google              Google                     `mapstructure:"google"`
	Logger              Logger                     `mapstructure:"logger"`
	Quota               Quota                      `mapstructure:"quota"`
	Otp                 Otp                        `mapstructure:"otp"`
	Email               Email                      `mapstructure:"email"`
	RateLimit           RateLimit                  `mapstructure:"rate_limit"`
	ClaimRateLimit      ClaimRateLimit             `mapstructure:"claim_rate_limit"`
	AdminGrantRateLimit AdminGrantRateLimit        `mapstructure:"admin_grant_rate_limit"`
	PayOS               PayOS                      `mapstructure:"payos"`
	Cloudinary          Cloudinary                 `mapstructure:"cloudinary"`
	AI                  AIServiceConfig            `mapstructure:"ai"`
	RabbitMQ            RabbitMQ                   `mapstructure:"rabbitmq"`
	Elasticsearch       Elasticsearch              `mapstructure:"elasticsearch"`
	Community           Community                  `mapstructure:"community"`
	Loyalty             Loyalty                    `mapstructure:"loyalty"`
	BrandMembership     BrandMembershipCacheTTL    `mapstructure:"brand_membership"`
	RAG                 RAG                        `mapstructure:"rag"`
	Subscription        Subscription               `mapstructure:"subscription"`
	Wardrobe            WardrobeProcessing         `mapstructure:"wardrobe"`
	Campaigns           []CampaignConfig           `mapstructure:"campaigns"`
	CampaignSweep       CampaignSweepConfig        `mapstructure:"campaign_sweep"`
	Outbox              OutboxConfig               `mapstructure:"outbox"`
	Admin               AdminConfig                `mapstructure:"admin"`
	BrandFeatureAccess  BrandFeatureAccessCacheTTL `mapstructure:"brand_feature_access"`
	Cache               CacheConfig                `mapstructure:"cache"`
	Ingress             Ingress                    `mapstructure:"ingress"`
}

// App selects the runtime topology of a service binary (FR-044).
//   - http:   HTTP + gRPC only, scale-to-zero, MUST NOT start consumers/cron.
//   - worker: consumers/outbox/cron with CPU always allocated (singleton).
//   - all:    both (local and VPS self-host). Default — safe for non-autoscaled envs.
type App struct {
	Mode string `mapstructure:"mode"`
}

// MaxOverviewTTLSeconds is the hard ceiling on the subscription overview cache
// TTL, from FR-074 (024 T146).
//
// The TTL is how long a freshly granted plan can stay invisible to the customer,
// because entitlement decisions read this cache. It is a ceiling rather than a
// default so that raising it cannot be a configuration mistake: `Validate` rejects
// anything above it at startup.
//
// The zero value means "use the built-in default", which is why this constant is
// not simply the default.
const MaxOverviewTTLSeconds = 60

// CacheConfig holds TTL configuration for application caches.
type CacheConfig struct {
	// OverviewTTLSeconds is the TTL of the subscription overview cache served
	// to AI hot paths via IUserSubscriptionContract (default 60 seconds).
	// May not exceed MaxOverviewTTLSeconds.
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

// DefaultCampaignSweepIntervalSeconds is both the DEFAULT and the FLOOR of
// campaign_sweep.interval_seconds (025, FR-083).
//
// Two roles, one constant, on purpose. The key is absent from a hand-written
// config all the time, so the loader needs a default; and anything below the
// default that somebody wrote down is a misconfiguration that must be refused at
// startup rather than repaired — below 60 seconds a sweep with a large backlog
// starves the grant queue it is supposed to catch up.
const DefaultCampaignSweepIntervalSeconds = 60

// CampaignConfig is ONE entry of the signup campaign grant list (025, FR-010).
//
// A campaign is data, not code: opening a new one is a new list entry and never a
// source change (FR-011, FR-090). The mechanism is named after the feature rather
// than after any single campaign, so the second campaign does not turn the first
// name into a lie.
//
// Every field below is checked at startup by validateCampaigns; the rule is
// repeated on each field so the reason travels with the value.
type CampaignConfig struct {
	// Code identifies the campaign. It is written to the campaign_code column
	// and into the audit trail, so it must be non-empty, unique in the list and
	// match ^[a-z0-9][a-z0-9-]{2,63}$ — a restricted alphabet keeps a code that
	// ends up in a queue or file name from breaking path traversal.
	Code string `mapstructure:"code"`
	// PlanSlug is the plan granted to an eligible account. Must be non-empty
	// here; that the plan EXISTS and is ACTIVE is checked against the database
	// at admission time, because loading configuration must not reach into the
	// database. Granting an inactive plan would grant something with no
	// commercial value (FR-010).
	PlanSlug string `mapstructure:"plan_slug"`
	// Quota is the main budget: the number of accounts that fill the main part
	// and freeze the priority watermark. Must be >= 1 (FR-021) — zero would be a
	// campaign that grants nothing, i.e. meaningless rather than invalid.
	Quota int `mapstructure:"quota"`
	// Reserve is the compensation budget, a CEILING and not a target (FR-032).
	// Must be >= 0 (FR-021); zero means the hard ceiling is exactly Quota.
	Reserve int `mapstructure:"reserve"`
	// StartsAt is the instant the campaign opens, RFC3339 WITH an explicit zone.
	// A zone-less value is rejected: a UTC server would then apply a
	// Vietnam-time boundary 7 hours off (FR-008, R6, constitution §VIII).
	StartsAt string `mapstructure:"starts_at"`
	// EndsAt is the instant the campaign closes, or empty/null for an open-ended
	// campaign (FR-009). When set it must be strictly after StartsAt (FR-013).
	EndsAt string `mapstructure:"ends_at"`

	// StartsAtTime is StartsAt parsed once at load time (025 T007); EndsAtTime is
	// EndsAt parsed the same way, or nil for an open-ended campaign. Neither is
	// mapped from YAML or from the environment: a derived field that had to be
	// kept in sync by hand is a field that will eventually be wrong.
	StartsAtTime time.Time  `mapstructure:"-"`
	EndsAtTime   *time.Time `mapstructure:"-"`
}

// CampaignSweepConfig sets the cadence of the catch-up sweep for EVERY campaign
// at once (025, FR-083).
//
// WHY IT SITS OUTSIDE THE campaigns LIST. One sweep worker runs once for all
// campaigns currently in effect, so putting the interval in each list entry would
// start N workers for N campaigns and force the question "who sweeps first".
// A single action cadence directly under `campaigns` has no such ambiguity.
type CampaignSweepConfig struct {
	// IntervalSeconds is the sweep period. Must be >= 60
	// (DefaultCampaignSweepIntervalSeconds); an absent key falls back to it.
	IntervalSeconds int `mapstructure:"interval_seconds"`
	// RunOnStart sweeps once at startup, which closes the gap left by the last
	// sweep before the service stopped (FR-079, SC-016). Defaults to true.
	RunOnStart bool `mapstructure:"run_on_start"`
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
	VisionPrimary                          APIProviderConfig `mapstructure:"vision_primary"`
	VisionFallback                         APIProviderConfig `mapstructure:"vision_fallback"`
	EmbeddingPrimary                       APIProviderConfig `mapstructure:"embedding_primary"`
	EmbeddingFallback                      APIProviderConfig `mapstructure:"embedding_fallback"`
	ChatTextPrimary                        APIProviderConfig `mapstructure:"chat_text_primary"`
	ChatTextFallback                       APIProviderConfig `mapstructure:"chat_text_fallback"`
	RecommendationTextPrimary              APIProviderConfig `mapstructure:"recommendation_text_primary"`
	RecommendationTextFallback             APIProviderConfig `mapstructure:"recommendation_text_fallback"`
	ChatTextTimeoutSeconds                 int               `mapstructure:"chat_text_timeout_seconds"`
	RecommendationTextTimeoutSeconds       int               `mapstructure:"recommendation_text_timeout_seconds"`
	VisionTimeoutSeconds                   int               `mapstructure:"vision_timeout_seconds"`
	EmbeddingTimeoutSeconds                int               `mapstructure:"embedding_timeout_seconds"`
	ChatTextRPMLimit                       int               `mapstructure:"chat_text_rpm_limit"`
	RecommendationTextRPMLimit             int               `mapstructure:"recommendation_text_rpm_limit"`
	VisionRPMLimit                         int               `mapstructure:"vision_rpm_limit"`
	EmbeddingRPMLimit                      int               `mapstructure:"embedding_rpm_limit"`
	ChatMaxInputCharacters                 int               `mapstructure:"chat_max_input_characters"`
	ChatHistoryMessageMaxCharacters        int               `mapstructure:"chat_history_message_max_characters"`
	ChatMaxOutputTokens                    int               `mapstructure:"chat_max_output_tokens"`
	SummarySourceMaxCharacters             int               `mapstructure:"summary_source_max_characters"`
	SummaryPreviousMaxCharacters           int               `mapstructure:"summary_previous_max_characters"`
	SummaryMaxOutputTokens                 int               `mapstructure:"summary_max_output_tokens"`
	RewriterPromptMaxCharacters            int               `mapstructure:"rewriter_prompt_max_characters"`
	RewriterMaxOutputTokens                int               `mapstructure:"rewriter_max_output_tokens"`
	RecommendationDetailsMaxCharacters     int               `mapstructure:"recommendation_details_max_characters"`
	RecommendationPromptCandidateLimit     int               `mapstructure:"recommendation_prompt_candidate_limit"`
	RecommendationDescriptionMaxCharacters int               `mapstructure:"recommendation_description_max_characters"`
	RecommendationTagsLimit                int               `mapstructure:"recommendation_tags_limit"`
	RecommendationPromptMaxCharacters      int               `mapstructure:"recommendation_prompt_max_characters"`
	RecommendationMaxOutputTokens          int               `mapstructure:"recommendation_max_output_tokens"`
	FreeTextPrimary                        APIProviderConfig `mapstructure:"free_text_primary"`
	FreeTextFallback                       APIProviderConfig `mapstructure:"free_text_fallback"`
	FreeTextTimeoutSeconds                 int               `mapstructure:"free_text_timeout_seconds"`
	FreeTextRPMLimit                       int               `mapstructure:"free_text_rpm_limit"`
	UsageReconcileCron                     string            `mapstructure:"usage_reconcile_cron"`
	UsageReconcileBatchSize                int               `mapstructure:"usage_reconcile_batch_size"`
	// DefaultCostPolicyCode is the aiexecutor.ai_cost_policies.code used as
	// fallback tier 1 when an account has no policy grant row (FR-006). The
	// table has no tier_rank/is_default column, so the default policy can only
	// be selected by code from configuration (research.md R8). Hardcoding a
	// code here would violate constitution §VIII.
	DefaultCostPolicyCode string                `mapstructure:"default_cost_policy_code"`
	TokenEstimation       TokenEstimationConfig `mapstructure:"token_estimation"`
}

type RabbitMQ struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	// Vhost selects the AMQP vhost (managed CloudAMQP uses a dedicated vhost;
	// empty defaults to "/" for self-hosted brokers).
	Vhost               string `mapstructure:"vhost"`
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
	// MigrationTable is the per-service Goose version table (FR-035). Account
	// keeps `goose_db_version`; Commerce/AI use suffixed tables so version
	// ordering never collides across services sharing one database.
	MigrationTable string `mapstructure:"migration_table"`
	// MigrationAllowOutOfOrder tolerates late-merged migrations within one
	// service (FR-035). Enabled on managed.
	MigrationAllowOutOfOrder bool `mapstructure:"migration_allow_out_of_order"`
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
	Port           string `mapstructure:"port"`
	FrontEndOrigin string `mapstructure:"front_end_origin"`
	// FrontEndOrigins is the normalized list derived from FrontEndOrigin after
	// load (FRONT_END_ORIGIN may hold several comma-separated origins per
	// environment). It is NOT mapped from env/YAML; consumers MUST use this
	// slice instead of splitting FrontEndOrigin themselves.
	FrontEndOrigins []string `mapstructure:"-"`
	// CookieDomain is the Domain attribute for session cookies. Empty = host-only
	// (local/dev). In production set it to the shared parent domain so the
	// frontend and API subdomains receive the same session cookie.
	CookieDomain        string    `mapstructure:"cookie_domain"`
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
	// ReplayBufferSize caps how many recent events are kept per analysis session
	// for SSE replay on reconnect (FR-006, feature 023).
	ReplayBufferSize int `mapstructure:"replay_buffer_size"`
	// ReplayBufferTTLSeconds bounds how long a per-session replay buffer is kept
	// after its last event so late reconnects still replay (FR-006, feature 023).
	ReplayBufferTTLSeconds int `mapstructure:"replay_buffer_ttl_seconds"`
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

// Google holds the Google OAuth/OIDC client configuration for social login.
// ClientIDs is derived from the comma-separated ClientIDRaw after load; it is
// NOT mapped directly from env/YAML.
type Google struct {
	ClientIDRaw     string   `mapstructure:"client_ids"`
	ClientSecret    string   `mapstructure:"client_secret"`
	RedirectURI     string   `mapstructure:"redirect_uri"`
	StateTTLMinutes int      `mapstructure:"state_ttl_minutes"`
	ClientIDs       []string `mapstructure:"-"`
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

// AdminGrantRateLimit caps the admin batch-grant endpoint. Two independent
// tiers, both counted per administrator rather than per IP (FR-078):
// AllPerMinute covers every request, MultiPerMinute covers only requests whose
// item count is greater than one. MultiPerMinute must stay below
// AllPerMinute, otherwise the stricter tier is dead configuration.
type AdminGrantRateLimit struct {
	AllPerMinute   int `mapstructure:"all_per_minute"`
	MultiPerMinute int `mapstructure:"multi_per_minute"`
	WindowSeconds  int `mapstructure:"window_seconds"`
}

type Community struct {
	HotWindowDays            int     `mapstructure:"hot_window_days"`
	HotGravity               float64 `mapstructure:"hot_gravity"`
	LikeWeight               int     `mapstructure:"like_weight"`
	CommentWeight            int     `mapstructure:"comment_weight"`
	SearchContentPrefixChars int     `mapstructure:"search_content_prefix_chars"`
	SearchUserCandidateCap   int     `mapstructure:"search_user_candidate_cap"`
	MaxMediaPerPost          int     `mapstructure:"max_media_per_post"`
	MaxImageBytes            int64   `mapstructure:"max_image_bytes"`
	MaxVideoBytes            int64   `mapstructure:"max_video_bytes"`
	MaxVideoDurationSeconds  int     `mapstructure:"max_video_duration_seconds"`
	MaxTitleLength           int     `mapstructure:"max_title_length"`
	MaxContentLength         int     `mapstructure:"max_content_length"`
	MaxCommentLength         int     `mapstructure:"max_comment_length"`
	PostRateLimitPerHour     int     `mapstructure:"post_rate_limit_per_hour"`
	CommentRateLimitPerHour  int     `mapstructure:"comment_rate_limit_per_hour"`
	FollowRateLimitPerHour   int     `mapstructure:"follow_rate_limit_per_hour"`
	LikeRateLimitPerHour     int     `mapstructure:"like_rate_limit_per_hour"`
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
