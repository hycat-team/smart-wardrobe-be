package eventconstants

// Queue names — shared routing vocabulary giữa publisher/consumer
// (application layer) và broker topology (infrastructure layer).
// Giữ package này thuần constants: không behavior, không import.
const (
	QueueElasticsearchSync                = "elasticsearch_sync_queue"
	QueueFashionInit                      = "fashion_init_queue"
	QueueFashionAnalyze                   = "fashion_analyze_queue"
	QueueSubscriptionPlanChanged          = "subscription_plan_changed_queue"
	QueueNotificationEmail                = "notification_email_queue"
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
)
