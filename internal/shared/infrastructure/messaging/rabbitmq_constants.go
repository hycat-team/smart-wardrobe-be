package messaging

const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"

	// Queues
	QueueElasticsearchSync = "elasticsearch_sync_queue"
	QueueFashionInit       = "fashion_init_queue"
	QueueFashionAnalyze    = "fashion_analyze_queue"

	// Queues
	QueueSubscriptionPlanChanged = "subscription_plan_changed_queue"

	// Routing Keys
	RoutingKeyElasticsearchSyncWildcard = "wardrobe.event.*"
	RoutingKeyFashionInit               = "fashion.event.init"
	RoutingKeyFashionAnalyze            = "fashion.event.analyze"
	RoutingKeySubscriptionPlanChanged   = "subscription.plan.changed"
)
