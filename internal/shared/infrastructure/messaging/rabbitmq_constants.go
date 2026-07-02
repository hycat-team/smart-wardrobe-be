package messaging

const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"

	// Queues
	QueueElasticsearchSync  = "elasticsearch_sync_queue"
	QueueFashionAnalyzeItem = "fashion_analyze_item_queue"

	// Routing Keys
	RoutingKeyElasticsearchSyncWildcard = "wardrobe.event.*"
	RoutingKeyFashionAnalyzeItem        = "fashion.event.analyze_item"
)
