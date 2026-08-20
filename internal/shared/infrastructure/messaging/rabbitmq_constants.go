package messaging

import "smart-wardrobe-be/internal/shared/application/constants/eventconstants"

const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"

	// Queues
	QueueElasticsearchSync = "elasticsearch_sync_queue"
	QueueFashionInit       = "fashion_init_queue"
	QueueFashionAnalyze    = "fashion_analyze_queue"

	// Queues
	QueueSubscriptionPlanChanged = "subscription_plan_changed_queue"
	QueueNotificationEmail       = "notification_email_queue"
	QueueAuditLog                = "audit_log_queue"

	// Routing Keys
	RoutingKeyElasticsearchSyncWildcard = "wardrobe.event.*"
	RoutingKeyFashionInit               = eventconstants.TopicFashionInit
	RoutingKeyFashionAnalyze            = eventconstants.TopicFashionAnalyze
	RoutingKeySubscriptionPlanChanged   = eventconstants.TopicSubscriptionPlanChanged
	RoutingKeyNotificationEmail         = eventconstants.TopicNotificationEmail + ".#"
	RoutingKeyAuditLog                  = eventconstants.TopicAuditLog
)
