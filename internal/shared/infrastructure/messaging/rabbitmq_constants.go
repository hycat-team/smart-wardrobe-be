package messaging

const (
	ExchangeName = "smart_wardrobe_exchange"
	ExchangeType = "topic"

	// Queues
	QueueBatchCropJobs     = "batch_crop_jobs"
	QueueElasticsearchSync = "elasticsearch_sync_queue"

	// Routing Keys
	RoutingKeyBatchCropJobs             = "batch_crop_jobs"
	RoutingKeyElasticsearchSyncWildcard = "wardrobe.event.*"
)
