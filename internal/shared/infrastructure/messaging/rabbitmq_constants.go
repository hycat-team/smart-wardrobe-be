package messaging

const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"

	// Queues
	QueueWardrobeBatchUpload = "wardrobe_batch_upload_queue"
	QueueElasticsearchSync   = "elasticsearch_sync_queue"
	QueueFashionAnalyzeItem  = "fashion_analyze_item_queue"

	// Routing Keys
	RoutingKeyWardrobeBatchUpload       = "wardrobe.event.batch-upload"
	RoutingKeyElasticsearchSyncWildcard = "wardrobe.event.*"
	RoutingKeyFashionAnalyzeItem        = "fashion.event.analyze_item"
)
