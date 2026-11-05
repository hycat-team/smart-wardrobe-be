package eventconstants

// Queue names — shared routing vocabulary giữa publisher/consumer
// (application layer) và broker topology (infrastructure layer).
// Giữ package này thuần constants: không behavior, không import.
const (
	QueueElasticsearchSync = "elasticsearch_sync_queue"
	QueueFashionInit       = "fashion_init_queue"
	QueueFashionAnalyze    = "fashion_analyze_queue"
	// QueueFashionAnalyzeRetry* hold a retry job for a fixed delay and then
	// dead-letter it back to fashion_analyze_queue. They have no consumer: the
	// broker's x-message-ttl IS the delay, which keeps the backoff durable
	// across restarts instead of living in a worker timer.
	QueueFashionAnalyzeRetry1    = "fashion_analyze_retry_1_queue"
	QueueFashionAnalyzeRetry2    = "fashion_analyze_retry_2_queue"
	QueueFashionAnalyzeRetry3    = "fashion_analyze_retry_3_queue"
	QueueSubscriptionPlanChanged = "subscription_plan_changed_queue"
	QueueNotificationEmail       = "notification_email_queue"
	// QueueNotificationEmailRetry holds a failed delivery for a fixed delay, then
	// dead-letters it back to the main exchange. One queue, no consumer: the
	// message returns to notification_email_queue by itself after the TTL.
	QueueNotificationEmailRetry           = "notification_email_retry_queue"
	QueueAuditLog                         = "audit_log_queue"
	QueueWardrobeFashionAnalyzed          = "wardrobe_fashion_analyzed_queue"
	QueueBrandFashionAnalyzed             = "brand_fashion_analyzed_queue"
	QueueWardrobeCleanup                  = "wardrobe_cleanup_queue"
	QueueWardrobeProcessingExhausted      = "wardrobe_processing_exhausted_queue"
	QueueBrandProcessingExhausted         = "brand_processing_exhausted_queue"
	QueueWardrobeFashionBind              = "wardrobe_fashion_bind_queue"
	QueueBrandFashionBind                 = "brand_fashion_bind_queue"
	QueueElasticsearchProcessingExhausted = "elasticsearch_processing_exhausted_queue"
	QueueCommunityOutfitCascade           = "community_outfit_cascade_queue"
	QueueCommunityUserStatus              = "community_user_status_queue"
	QueueStyleLearning                    = "style_learning_queue"

	// QueueCampaignGrantUserCreated consumes TopicUserCreated for the signup
	// campaign grant (025, FR-093). Named after the FEATURE, never after a
	// campaign code: the queue outlives every campaign it serves (FR-090).
	QueueCampaignGrantUserCreated = "campaign_grant_user_created_queue"
)

// Fashion analyze retry delays, in milliseconds. These are constants rather
// than configuration because RabbitMQ cannot change a queue argument in place:
// a configurable TTL would invite an env change that makes every service fail
// at startup with 406 PRECONDITION_FAILED. The config knobs
// (wardrobe.retry_delay_*_seconds) are pinned to these values by validation.go
// so an operator cannot set a delay the broker will not honour.
const (
	FashionAnalyzeRetryDelay1Millis = 60_000  // 1 minute
	FashionAnalyzeRetryDelay2Millis = 300_000 // 5 minutes
	FashionAnalyzeRetryDelay3Millis = 900_000 // 15 minutes
)
